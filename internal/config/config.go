package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds all project configuration loaded from .human/config.json.
type Config struct {
	LLM *LLMConfig `json:"llm,omitempty"`
}

// LLMConfig holds configuration for the LLM connector.
type LLMConfig struct {
	Provider    string  `json:"provider"`            // "anthropic", "openai", "ollama"
	Model       string  `json:"model,omitempty"`     // e.g. "claude-sonnet-4-20250514"
	APIKey      string  `json:"-"`                   // NEVER serialized — env vars only
	BaseURL     string  `json:"base_url,omitempty"`  // override for Ollama/proxies
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
}

// configFileName is the configuration file path relative to the project root.
const configFileName = ".human/config.json"

// Load reads the project configuration from .human/config.json in the given
// project directory. If the file doesn't exist, it returns a zero Config (not
// an error). Environment variables override file values for API keys.
func Load(projectDir string) (*Config, error) {
	cfg := &Config{}

	path := filepath.Join(projectDir, configFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", configFileName, err)
	}

	// Resolve API key from environment if LLM section exists.
	if cfg.LLM != nil {
		key, _ := ResolveAPIKey(cfg.LLM.Provider)
		cfg.LLM.APIKey = key
	}

	return cfg, nil
}

// Save writes the config to .human/config.json, creating the directory if
// needed. API keys are never written to disk (json:"-" tag on LLMConfig).
func Save(projectDir string, cfg *Config) error {
	dir := filepath.Join(projectDir, ".human")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating .human directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	path := filepath.Join(projectDir, configFileName)
	if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", configFileName, err)
	}

	return nil
}

// ResolveAPIKey looks up the API key for a provider.
// Resolution order: environment variable → global config (~/.human/config.json) → error.
// Returns ("", nil) for providers that don't need keys (ollama).
func ResolveAPIKey(provider string) (string, error) {
	switch provider {
	case "anthropic":
		if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
			return key, nil
		}
		if key := resolveAPIKeyFromGlobal(provider); key != "" {
			return key, nil
		}
		return "", fmt.Errorf("no API key found for Anthropic. Set ANTHROPIC_API_KEY or run /connect anthropic")
	case "openai":
		if key := os.Getenv("OPENAI_API_KEY"); key != "" {
			return key, nil
		}
		if key := resolveAPIKeyFromGlobal(provider); key != "" {
			return key, nil
		}
		return "", fmt.Errorf("no API key found for OpenAI. Set OPENAI_API_KEY or run /connect openai")
	case "ollama":
		return "", nil
	default:
		return "", fmt.Errorf("unknown provider %q. Supported: anthropic, openai, ollama", provider)
	}
}

// resolveAPIKeyFromGlobal reads the global config and returns the API key
// if the provider matches. Returns "" on any error or mismatch.
func resolveAPIKeyFromGlobal(provider string) string {
	gc, err := LoadGlobalConfig()
	if err != nil || gc.LLM == nil {
		return ""
	}
	if gc.LLM.Provider == provider {
		return gc.LLM.APIKey
	}
	return ""
}

// DefaultLLMConfig returns a sensible starting config for the given provider.
func DefaultLLMConfig(provider string) *LLMConfig {
	cfg := &LLMConfig{
		Provider:    provider,
		MaxTokens:   4096,
		Temperature: 0.0,
	}

	switch provider {
	case "anthropic":
		cfg.Model = "claude-sonnet-4-20250514"
	case "openai":
		cfg.Model = "gpt-4o"
	case "ollama":
		cfg.Model = "llama3"
		cfg.BaseURL = "http://localhost:11434"
	}

	return cfg
}

// ── Global Config (user-wide LLM credentials, stored in ~/.human/config.json) ──

// GlobalConfig holds user-wide configuration stored at ~/.human/config.json.
// Unlike project config, this persists API keys locally.
type GlobalConfig struct {
	LLM *GlobalLLMConfig  `json:"llm,omitempty"`
	MCP []*MCPServerConfig `json:"mcp,omitempty"`
}

// MCPServerConfig stores configuration for an external MCP server.
type MCPServerConfig struct {
	Name    string            `json:"name"`              // display name (e.g. "figma")
	Command string            `json:"command"`           // executable (e.g. "npx")
	Args    []string          `json:"args,omitempty"`    // command arguments
	Env     map[string]string `json:"env,omitempty"`     // env vars (e.g. FIGMA_ACCESS_TOKEN)
}

// GlobalLLMConfig stores LLM credentials globally.
// This is a separate type from LLMConfig because LLMConfig.APIKey has json:"-".
type GlobalLLMConfig struct {
	Provider string `json:"provider"`
	Model    string `json:"model,omitempty"`
	APIKey   string `json:"api_key,omitempty"`
	BaseURL  string `json:"base_url,omitempty"`
}

const globalConfigFile = ".human/config.json"

// LoadGlobalConfig reads user-wide LLM config from ~/.human/config.json.
// Returns a zero GlobalConfig if the file doesn't exist.
func LoadGlobalConfig() (*GlobalConfig, error) {
	cfg := &GlobalConfig{}

	home, err := os.UserHomeDir()
	if err != nil {
		return cfg, nil
	}

	path := filepath.Join(home, globalConfigFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("reading global config: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", globalConfigFile, err)
	}

	return cfg, nil
}

// SaveGlobalConfig writes user-wide LLM config to ~/.human/config.json.
// The file is written with 0600 permissions since it may contain API keys.
func SaveGlobalConfig(cfg *GlobalConfig) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not find home directory: %w", err)
	}

	dir := filepath.Join(home, ".human")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating ~/.human directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling global config: %w", err)
	}

	path := filepath.Join(home, globalConfigFile)
	if err := os.WriteFile(path, append(data, '\n'), 0600); err != nil {
		return fmt.Errorf("writing %s: %w", globalConfigFile, err)
	}

	return nil
}

// ── Global Settings (user-wide, stored in ~/.human/settings.json) ──

// GlobalSettings holds user-wide preferences that persist across projects.
type GlobalSettings struct {
	Theme        string `json:"theme,omitempty"`      // "default", "dark", "light", etc.
	Animate      *bool  `json:"animate,omitempty"`    // nil = true (default)
	PlanMode     string `json:"plan_mode,omitempty"`  // "always" (default), "auto", "off"
	FirstRunDone bool   `json:"first_run_done"`
}

// globalSettingsFile is the path relative to the user's home directory.
const globalSettingsFile = ".human/settings.json"

// LoadGlobal reads user-wide settings from ~/.human/settings.json.
// Returns default settings if the file doesn't exist.
func LoadGlobal() (*GlobalSettings, error) {
	s := &GlobalSettings{}

	home, err := os.UserHomeDir()
	if err != nil {
		return s, nil
	}

	path := filepath.Join(home, globalSettingsFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, fmt.Errorf("reading global settings: %w", err)
	}

	if err := json.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", globalSettingsFile, err)
	}

	return s, nil
}

// SaveGlobal writes user-wide settings to ~/.human/settings.json.
func SaveGlobal(s *GlobalSettings) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not find home directory: %w", err)
	}

	dir := filepath.Join(home, ".human")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating ~/.human directory: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling settings: %w", err)
	}

	path := filepath.Join(home, globalSettingsFile)
	if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", globalSettingsFile, err)
	}

	return nil
}

// AnimateEnabled returns whether startup animation is enabled.
// Defaults to true when the Animate field is nil.
func (s *GlobalSettings) AnimateEnabled() bool {
	if s.Animate == nil {
		return true
	}
	return *s.Animate
}

// SetAnimate sets the animation preference.
func (s *GlobalSettings) SetAnimate(enabled bool) {
	s.Animate = &enabled
}

// EffectivePlanMode returns the plan mode, defaulting to "always".
func (s *GlobalSettings) EffectivePlanMode() string {
	if s.PlanMode == "" {
		return "always"
	}
	return s.PlanMode
}
