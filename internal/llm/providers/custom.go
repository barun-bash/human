package providers

import (
	"fmt"

	"github.com/barun-bash/human/internal/config"
	"github.com/barun-bash/human/internal/llm"
)

func init() {
	llm.RegisterProvider("custom", newCustom)
}

func newCustom(cfg *config.LLMConfig) (llm.Provider, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("custom provider requires a base URL. Run /connect custom to configure")
	}

	return &OpenAI{
		apiKey:  cfg.APIKey,
		model:   cfg.Model,
		baseURL: cfg.BaseURL,
		client:  defaultHTTPClient(),
		name:    "custom",
	}, nil
}
