package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFile(t *testing.T) {
	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if cfg.LLM != nil {
		t.Fatalf("expected nil LLM config, got: %+v", cfg.LLM)
	}
}

func TestLoadValidConfig(t *testing.T) {
	dir := t.TempDir()
	humanDir := filepath.Join(dir, ".human")
	if err := os.MkdirAll(humanDir, 0755); err != nil {
		t.Fatal(err)
	}

	data := `{
  "llm": {
    "provider": "ollama",
    "model": "codellama",
    "base_url": "http://localhost:11434",
    "max_tokens": 2048,
    "temperature": 0.2
  }
}`
	if err := os.WriteFile(filepath.Join(humanDir, "config.json"), []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.LLM == nil {
		t.Fatal("expected LLM config to be populated")
	}
	if cfg.LLM.Provider != "ollama" {
		t.Errorf("provider = %q, want %q", cfg.LLM.Provider, "ollama")
	}
	if cfg.LLM.Model != "codellama" {
		t.Errorf("model = %q, want %q", cfg.LLM.Model, "codellama")
	}
	if cfg.LLM.MaxTokens != 2048 {
		t.Errorf("max_tokens = %d, want 2048", cfg.LLM.MaxTokens)
	}
	if cfg.LLM.Temperature != 0.2 {
		t.Errorf("temperature = %f, want 0.2", cfg.LLM.Temperature)
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	humanDir := filepath.Join(dir, ".human")
	if err := os.MkdirAll(humanDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(humanDir, "config.json"), []byte("{bad json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestAPIKeyNeverSerialized(t *testing.T) {
	cfg := &LLMConfig{
		Provider: "anthropic",
		APIKey:   "sk-secret-key-12345",
		Model:    "claude-sonnet-4-20250514",
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}

	jsonStr := string(data)
	if contains(jsonStr, "sk-secret") {
		t.Errorf("API key leaked into JSON output: %s", jsonStr)
	}
	if contains(jsonStr, "api_key") {
		t.Errorf("api_key field present in JSON output: %s", jsonStr)
	}
}

func TestEnvVarOverride(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key-abc")

	key, err := ResolveAPIKey("anthropic")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "test-key-abc" {
		t.Errorf("key = %q, want %q", key, "test-key-abc")
	}
}

func TestEnvVarMissing(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	// Redirect HOME so resolveAPIKeyFromGlobal won't find real ~/.human/config.json
	t.Setenv("HOME", t.TempDir())

	_, err := ResolveAPIKey("anthropic")
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
}

func TestOllamaNoKeyRequired(t *testing.T) {
	key, err := ResolveAPIKey("ollama")
	if err != nil {
		t.Fatalf("unexpected error for ollama: %v", err)
	}
	if key != "" {
		t.Errorf("expected empty key for ollama, got %q", key)
	}
}

func TestUnknownProvider(t *testing.T) {
	_, err := ResolveAPIKey("gemini")
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		LLM: &LLMConfig{
			Provider:  "openai",
			Model:     "gpt-4o",
			MaxTokens: 4096,
		},
	}

	if err := Save(dir, cfg); err != nil {
		t.Fatalf("save error: %v", err)
	}

	// Verify file exists
	path := filepath.Join(dir, ".human", "config.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	// Load it back
	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if loaded.LLM == nil {
		t.Fatal("expected LLM config after load")
	}
	if loaded.LLM.Provider != "openai" {
		t.Errorf("provider = %q, want %q", loaded.LLM.Provider, "openai")
	}
}

func TestDefaultLLMConfig(t *testing.T) {
	tests := []struct {
		provider string
		model    string
		baseURL  string
	}{
		{"anthropic", "claude-sonnet-4-20250514", ""},
		{"openai", "gpt-4o", ""},
		{"ollama", "llama3", "http://localhost:11434"},
	}

	for _, tt := range tests {
		cfg := DefaultLLMConfig(tt.provider)
		if cfg.Model != tt.model {
			t.Errorf("%s: model = %q, want %q", tt.provider, cfg.Model, tt.model)
		}
		if cfg.BaseURL != tt.baseURL {
			t.Errorf("%s: base_url = %q, want %q", tt.provider, cfg.BaseURL, tt.baseURL)
		}
		if cfg.MaxTokens != 4096 {
			t.Errorf("%s: max_tokens = %d, want 4096", tt.provider, cfg.MaxTokens)
		}
	}
}

// ── GlobalSettings Tests ──

func TestGlobalSettingsDefaults(t *testing.T) {
	s := &GlobalSettings{}
	if !s.AnimateEnabled() {
		t.Error("default animate should be true")
	}
	if s.EffectivePlanMode() != "always" {
		t.Errorf("default plan_mode = %q, want %q", s.EffectivePlanMode(), "always")
	}
}

func TestGlobalSettingsSetAnimate(t *testing.T) {
	s := &GlobalSettings{}
	s.SetAnimate(false)
	if s.AnimateEnabled() {
		t.Error("expected animate to be false after SetAnimate(false)")
	}
	s.SetAnimate(true)
	if !s.AnimateEnabled() {
		t.Error("expected animate to be true after SetAnimate(true)")
	}
}

func TestGlobalSettingsPlanMode(t *testing.T) {
	s := &GlobalSettings{PlanMode: "off"}
	if s.EffectivePlanMode() != "off" {
		t.Errorf("plan_mode = %q, want %q", s.EffectivePlanMode(), "off")
	}
	s.PlanMode = "auto"
	if s.EffectivePlanMode() != "auto" {
		t.Errorf("plan_mode = %q, want %q", s.EffectivePlanMode(), "auto")
	}
}

func TestGlobalSettingsRoundTrip(t *testing.T) {
	// Override HOME to a temp directory so we don't write to the real home.
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	original := &GlobalSettings{
		Theme:        "ocean",
		PlanMode:     "auto",
		FirstRunDone: true,
	}
	original.SetAnimate(false)

	if err := SaveGlobal(original); err != nil {
		t.Fatalf("SaveGlobal error: %v", err)
	}

	loaded, err := LoadGlobal()
	if err != nil {
		t.Fatalf("LoadGlobal error: %v", err)
	}

	if loaded.Theme != "ocean" {
		t.Errorf("theme = %q, want %q", loaded.Theme, "ocean")
	}
	if loaded.PlanMode != "auto" {
		t.Errorf("plan_mode = %q, want %q", loaded.PlanMode, "auto")
	}
	if !loaded.FirstRunDone {
		t.Error("first_run_done should be true")
	}
	if loaded.AnimateEnabled() {
		t.Error("animate should be false")
	}
}

func TestLoadGlobalMissing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	s, err := LoadGlobal()
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	// Should return defaults.
	if !s.AnimateEnabled() {
		t.Error("default animate should be true")
	}
}

// ── GlobalConfig Tests ──

func TestGlobalConfigRoundTrip(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	original := &GlobalConfig{
		LLM: &GlobalLLMConfig{
			Provider: "anthropic",
			Model:    "claude-sonnet-4-20250514",
			APIKey:   "sk-ant-test-key-12345",
		},
	}

	if err := SaveGlobalConfig(original); err != nil {
		t.Fatalf("SaveGlobalConfig error: %v", err)
	}

	loaded, err := LoadGlobalConfig()
	if err != nil {
		t.Fatalf("LoadGlobalConfig error: %v", err)
	}

	if loaded.LLM == nil {
		t.Fatal("expected LLM config after load")
	}
	if loaded.LLM.Provider != "anthropic" {
		t.Errorf("provider = %q, want %q", loaded.LLM.Provider, "anthropic")
	}
	if loaded.LLM.Model != "claude-sonnet-4-20250514" {
		t.Errorf("model = %q, want %q", loaded.LLM.Model, "claude-sonnet-4-20250514")
	}
	if loaded.LLM.APIKey != "sk-ant-test-key-12345" {
		t.Errorf("api_key = %q, want %q", loaded.LLM.APIKey, "sk-ant-test-key-12345")
	}
}

func TestGlobalConfigMissing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg, err := LoadGlobalConfig()
	if err != nil {
		t.Fatalf("expected no error for missing global config, got: %v", err)
	}
	if cfg.LLM != nil {
		t.Errorf("expected nil LLM config, got: %+v", cfg.LLM)
	}
}

func TestGlobalConfigAPIKeyPersisted(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	cfg := &GlobalConfig{
		LLM: &GlobalLLMConfig{
			Provider: "openai",
			APIKey:   "sk-openai-secret-999",
		},
	}

	if err := SaveGlobalConfig(cfg); err != nil {
		t.Fatal(err)
	}

	// Read raw file to verify API key IS in the JSON (unlike project LLMConfig).
	path := filepath.Join(tmpHome, ".human", "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	jsonStr := string(data)
	if !contains(jsonStr, "sk-openai-secret-999") {
		t.Error("API key should be persisted in global config JSON")
	}
	if !contains(jsonStr, "api_key") {
		t.Error("api_key field should be present in global config JSON")
	}
}

func TestGlobalConfigFilePermissions(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	cfg := &GlobalConfig{
		LLM: &GlobalLLMConfig{
			Provider: "anthropic",
			APIKey:   "sk-secret",
		},
	}

	if err := SaveGlobalConfig(cfg); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(tmpHome, ".human", "config.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("file permissions = %o, want 0600", perm)
	}
}

func TestResolveAPIKeyFromGlobal(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Clear env vars so they don't interfere.
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")

	// Save a global config with an Anthropic key.
	cfg := &GlobalConfig{
		LLM: &GlobalLLMConfig{
			Provider: "anthropic",
			APIKey:   "sk-global-key-abc",
		},
	}
	if err := SaveGlobalConfig(cfg); err != nil {
		t.Fatal(err)
	}

	// ResolveAPIKey should find the global key.
	key, err := ResolveAPIKey("anthropic")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "sk-global-key-abc" {
		t.Errorf("key = %q, want %q", key, "sk-global-key-abc")
	}

	// Env var should take priority.
	t.Setenv("ANTHROPIC_API_KEY", "sk-env-key-xyz")
	key, err = ResolveAPIKey("anthropic")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "sk-env-key-xyz" {
		t.Errorf("key = %q, want %q (env var should take priority)", key, "sk-env-key-xyz")
	}
}

func TestResolveAPIKeyGlobalWrongProvider(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("OPENAI_API_KEY", "")

	// Save global config with Anthropic key.
	cfg := &GlobalConfig{
		LLM: &GlobalLLMConfig{
			Provider: "anthropic",
			APIKey:   "sk-ant-key",
		},
	}
	if err := SaveGlobalConfig(cfg); err != nil {
		t.Fatal(err)
	}

	// OpenAI should NOT find the Anthropic key.
	_, err := ResolveAPIKey("openai")
	if err == nil {
		t.Error("expected error when global config has different provider")
	}
}

func TestGlobalConfigMCPRoundTrip(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	gc := &GlobalConfig{
		LLM: &GlobalLLMConfig{
			Provider: "anthropic",
			Model:    "claude-sonnet-4-20250514",
			APIKey:   "sk-test",
		},
		MCP: []*MCPServerConfig{
			{
				Name:    "figma",
				Command: "npx",
				Args:    []string{"-y", "@anthropic/mcp-server-figma"},
				Env:     map[string]string{"FIGMA_ACCESS_TOKEN": "tok-123"},
			},
			{
				Name:    "github",
				Command: "npx",
				Args:    []string{"-y", "@anthropic/mcp-server-github"},
				Env:     map[string]string{"GITHUB_TOKEN": "ghp-456"},
			},
		},
	}

	if err := SaveGlobalConfig(gc); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadGlobalConfig()
	if err != nil {
		t.Fatal(err)
	}

	if loaded.LLM == nil || loaded.LLM.Provider != "anthropic" {
		t.Error("LLM config lost during MCP round-trip")
	}
	if len(loaded.MCP) != 2 {
		t.Fatalf("expected 2 MCP servers, got %d", len(loaded.MCP))
	}
	if loaded.MCP[0].Name != "figma" {
		t.Errorf("MCP[0].Name = %q, want figma", loaded.MCP[0].Name)
	}
	if loaded.MCP[0].Command != "npx" {
		t.Errorf("MCP[0].Command = %q, want npx", loaded.MCP[0].Command)
	}
	if loaded.MCP[0].Env["FIGMA_ACCESS_TOKEN"] != "tok-123" {
		t.Errorf("MCP[0] env token not persisted")
	}
	if loaded.MCP[1].Name != "github" {
		t.Errorf("MCP[1].Name = %q, want github", loaded.MCP[1].Name)
	}
}

func TestGlobalConfigMCPPreservesLLM(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Save LLM config first.
	gc1 := &GlobalConfig{
		LLM: &GlobalLLMConfig{
			Provider: "openai",
			Model:    "gpt-4o",
			APIKey:   "sk-openai-key",
		},
	}
	if err := SaveGlobalConfig(gc1); err != nil {
		t.Fatal(err)
	}

	// Load, add MCP, save again.
	loaded, _ := LoadGlobalConfig()
	loaded.MCP = []*MCPServerConfig{
		{Name: "figma", Command: "npx", Args: []string{"-y", "@anthropic/mcp-server-figma"}},
	}
	if err := SaveGlobalConfig(loaded); err != nil {
		t.Fatal(err)
	}

	// Reload and verify LLM wasn't clobbered.
	final, _ := LoadGlobalConfig()
	if final.LLM == nil || final.LLM.APIKey != "sk-openai-key" {
		t.Error("LLM config was clobbered when saving MCP config")
	}
	if len(final.MCP) != 1 || final.MCP[0].Name != "figma" {
		t.Error("MCP config not saved correctly")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstr(s, substr)
}

func searchSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
