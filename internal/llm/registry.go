package llm

import (
	"fmt"
	"strings"

	"github.com/barun-bash/human/internal/config"
)

// SupportedProviders lists all available LLM provider names.
var SupportedProviders = []string{"anthropic", "openai", "ollama", "groq", "openrouter", "gemini", "custom"}

// ProviderFactory is a function that creates a Provider from config.
// Registered by each provider package via RegisterProvider.
type ProviderFactory func(cfg *config.LLMConfig) (Provider, error)

// registry holds the registered provider factories.
var registry = map[string]ProviderFactory{}

// RegisterProvider registers a factory for a named provider.
// Called by provider init() functions.
func RegisterProvider(name string, factory ProviderFactory) {
	registry[name] = factory
}

// NewProvider creates a Provider from the given LLM config.
// It resolves the API key, selects the right implementation, and applies defaults.
func NewProvider(cfg *config.LLMConfig) (Provider, error) {
	if cfg == nil {
		return nil, ErrNoProvider()
	}

	// Apply default model if not specified.
	if cfg.Model == "" {
		defaults := config.DefaultLLMConfig(cfg.Provider)
		cfg.Model = defaults.Model
	}

	// Apply default max tokens if not specified.
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = 4096
	}

	// Resolve API key from environment if not already set.
	if cfg.APIKey == "" {
		key, err := config.ResolveAPIKey(cfg.Provider)
		if err != nil {
			return nil, err
		}
		cfg.APIKey = key
	}

	factory, ok := registry[cfg.Provider]
	if !ok {
		return nil, fmt.Errorf("unknown LLM provider %q. Supported: %s", cfg.Provider, strings.Join(SupportedProviders, ", "))
	}

	return factory(cfg)
}
