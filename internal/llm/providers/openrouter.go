package providers

import (
	"github.com/barun-bash/human/internal/config"
	"github.com/barun-bash/human/internal/llm"
)

const openrouterDefaultURL = "https://openrouter.ai/api/v1/chat/completions"

func init() {
	llm.RegisterProvider("openrouter", newOpenRouter)
}

func newOpenRouter(cfg *config.LLMConfig) (llm.Provider, error) {
	if cfg.APIKey == "" {
		return nil, llm.ErrNoAPIKey("openrouter")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = openrouterDefaultURL
	}

	return &OpenAI{
		apiKey:  cfg.APIKey,
		model:   cfg.Model,
		baseURL: baseURL,
		client:  defaultHTTPClient(),
		name:    "openrouter",
	}, nil
}
