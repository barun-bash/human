package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/barun-bash/human/internal/config"
	"github.com/barun-bash/human/internal/llm"
)

const ollamaDefaultURL = "http://localhost:11434/v1/chat/completions"

// Ollama implements the llm.Provider interface using Ollama's OpenAI-compatible
// endpoint. No authentication is required.
type Ollama struct {
	model   string
	baseURL string
	client  *http.Client
}

func init() {
	llm.RegisterProvider("ollama", newOllama)
}

func newOllama(cfg *config.LLMConfig) (llm.Provider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	// Ensure the URL points to the OpenAI-compatible endpoint.
	if !strings.HasSuffix(baseURL, "/v1/chat/completions") {
		baseURL = strings.TrimRight(baseURL, "/") + "/v1/chat/completions"
	}

	return &Ollama{
		model:   cfg.Model,
		baseURL: baseURL,
		client:  &http.Client{},
	}, nil
}

func (o *Ollama) Name() string { return "ollama" }

func (o *Ollama) Complete(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	body := o.buildRequest(req, false)

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.baseURL, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(httpReq)
	if err != nil {
		if isConnectionRefused(err) {
			return nil, llm.ErrOllamaNotRunning()
		}
		return nil, llm.ErrNetworkFailure("Ollama", err.Error())
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, llm.ErrProviderError("Ollama", resp.StatusCode, string(respBody))
	}

	// Ollama uses OpenAI response format.
	var apiResp openaiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	content := ""
	stopReason := ""
	if len(apiResp.Choices) > 0 {
		content = apiResp.Choices[0].Message.Content
		stopReason = apiResp.Choices[0].FinishReason
	}

	return &llm.Response{
		Content:    content,
		Model:      apiResp.Model,
		StopReason: stopReason,
		TokenUsage: llm.TokenUsage{
			InputTokens:  apiResp.Usage.PromptTokens,
			OutputTokens: apiResp.Usage.CompletionTokens,
		},
	}, nil
}

func (o *Ollama) Stream(ctx context.Context, req *llm.Request) (<-chan llm.StreamChunk, error) {
	body := o.buildRequest(req, true)

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.baseURL, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(httpReq)
	if err != nil {
		if isConnectionRefused(err) {
			return nil, llm.ErrOllamaNotRunning()
		}
		return nil, llm.ErrNetworkFailure("Ollama", err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, llm.ErrProviderError("Ollama", resp.StatusCode, string(respBody))
	}

	ch := make(chan llm.StreamChunk, 64)
	go readOpenAISSE(resp.Body, ch)
	return ch, nil
}

func (o *Ollama) buildRequest(req *llm.Request, stream bool) openaiRequest {
	or := openaiRequest{
		Model:       req.Model,
		Temperature: req.Temperature,
		Stream:      stream,
	}

	if or.Model == "" {
		or.Model = o.model
	}

	for _, msg := range req.Messages {
		or.Messages = append(or.Messages, openaiMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}

	return or
}

// isConnectionRefused checks if an error is a connection refused error,
// which indicates Ollama is not running.
func isConnectionRefused(err error) bool {
	return strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "dial tcp")
}
