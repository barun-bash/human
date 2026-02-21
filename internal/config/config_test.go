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
