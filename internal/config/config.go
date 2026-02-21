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

// ResolveAPIKey looks up the API key for a provider from environment variables.
// Returns ("", nil) for providers that don't need keys (ollama).
// Returns ("", error) when a required key is missing.
func ResolveAPIKey(provider string) (string, error) {
	switch provider {
	case "anthropic":
		key := os.Getenv("ANTHROPIC_API_KEY")
		if key == "" {
			return "", fmt.Errorf("no API key found for Anthropic. Set the ANTHROPIC_API_KEY environment variable")
		}
		return key, nil
	case "openai":
		key := os.Getenv("OPENAI_API_KEY")
		if key == "" {
			return "", fmt.Errorf("no API key found for OpenAI. Set the OPENAI_API_KEY environment variable")
		}
		return key, nil
	case "ollama":
		return "", nil
	default:
		return "", fmt.Errorf("unknown provider %q. Supported: anthropic, openai, ollama", provider)
	}
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
