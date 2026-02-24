package providers

import (
	"github.com/barun-bash/human/internal/config"
	"github.com/barun-bash/human/internal/llm"
)

const groqDefaultURL = "https://api.groq.com/openai/v1/chat/completions"

func init() {
	llm.RegisterProvider("groq", newGroq)
}

func newGroq(cfg *config.LLMConfig) (llm.Provider, error) {
	if cfg.APIKey == "" {
		return nil, llm.ErrNoAPIKey("groq")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = groqDefaultURL
	}

	return &OpenAI{
		apiKey:  cfg.APIKey,
		model:   cfg.Model,
		baseURL: baseURL,
		client:  defaultHTTPClient(),
		name:    "groq",
	}, nil
}
