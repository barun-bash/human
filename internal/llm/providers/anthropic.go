package providers

import (
	"bufio"
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

const (
	anthropicDefaultURL = "https://api.anthropic.com/v1/messages"
	anthropicVersion    = "2023-06-01"
)

// Anthropic implements the llm.Provider interface for the Anthropic Messages API.
type Anthropic struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

func init() {
	llm.RegisterProvider("anthropic", newAnthropic)
}

func newAnthropic(cfg *config.LLMConfig) (llm.Provider, error) {
	if cfg.APIKey == "" {
		return nil, llm.ErrNoAPIKey("anthropic")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = anthropicDefaultURL
	}

	return &Anthropic{
		apiKey:  cfg.APIKey,
		model:   cfg.Model,
		baseURL: baseURL,
		client:  &http.Client{},
	}, nil
}

func (a *Anthropic) Name() string { return "anthropic" }

// anthropicRequest is the Anthropic Messages API request format.
type anthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float64            `json:"temperature"`
	System      string             `json:"system,omitempty"`
	Messages    []anthropicMessage `json:"messages"`
	Stream      bool               `json:"stream,omitempty"`
}

type anthropicMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // string or []anthropicContentBlock for vision
}

type anthropicContentBlock struct {
	Type   string                     `json:"type"`
	Text   string                     `json:"text,omitempty"`
	Source *anthropicContentBlockImage `json:"source,omitempty"`
}

type anthropicContentBlockImage struct {
	Type      string `json:"type"`       // "base64"
	MediaType string `json:"media_type"` // e.g. "image/png"
	Data      string `json:"data"`
}

// anthropicResponse is the Anthropic Messages API non-streaming response.
type anthropicResponse struct {
	Content    []anthropicContent `json:"content"`
	Model      string             `json:"model"`
	StopReason string             `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type anthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicError struct {
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

func (a *Anthropic) Complete(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	body := a.buildRequest(req, false)

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	a.setHeaders(httpReq)

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, llm.ErrNetworkFailure("Anthropic", err.Error())
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if err := a.checkError(resp.StatusCode, respBody); err != nil {
		return nil, err
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	content := ""
	for _, c := range apiResp.Content {
		if c.Type == "text" {
			content += c.Text
		}
	}

	return &llm.Response{
		Content:    content,
		Model:      apiResp.Model,
		StopReason: apiResp.StopReason,
		TokenUsage: llm.TokenUsage{
			InputTokens:  apiResp.Usage.InputTokens,
			OutputTokens: apiResp.Usage.OutputTokens,
		},
	}, nil
}

func (a *Anthropic) Stream(ctx context.Context, req *llm.Request) (<-chan llm.StreamChunk, error) {
	body := a.buildRequest(req, true)

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	a.setHeaders(httpReq)

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, llm.ErrNetworkFailure("Anthropic", err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, a.checkError(resp.StatusCode, respBody)
	}

	ch := make(chan llm.StreamChunk, 64)
	go a.readSSE(resp.Body, ch)
	return ch, nil
}

func (a *Anthropic) buildRequest(req *llm.Request, stream bool) anthropicRequest {
	ar := anthropicRequest{
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      stream,
	}

	if ar.Model == "" {
		ar.Model = a.model
	}
	if ar.MaxTokens == 0 {
		ar.MaxTokens = 4096
	}

	for _, msg := range req.Messages {
		if msg.Role == llm.RoleSystem {
			ar.System = msg.Content
		} else {
			ar.Messages = append(ar.Messages, anthropicMessage{
				Role:    string(msg.Role),
				Content: msg.Content,
			})
		}
	}

	// If images are provided, convert the last user message to multi-modal content blocks.
	if len(req.Images) > 0 && len(ar.Messages) > 0 {
		last := len(ar.Messages) - 1
		if ar.Messages[last].Role == "user" {
			var blocks []anthropicContentBlock
			// Add image blocks first.
			for _, img := range req.Images {
				blocks = append(blocks, anthropicContentBlock{
					Type: "image",
					Source: &anthropicContentBlockImage{
						Type:      "base64",
						MediaType: img.MIMEType,
						Data:      img.Data,
					},
				})
			}
			// Then the text block with the original content.
			if text, ok := ar.Messages[last].Content.(string); ok && text != "" {
				blocks = append(blocks, anthropicContentBlock{
					Type: "text",
					Text: text,
				})
			}
			ar.Messages[last].Content = blocks
		}
	}

	return ar
}

func (a *Anthropic) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)
}

func (a *Anthropic) checkError(statusCode int, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	switch statusCode {
	case 401:
		return llm.ErrAuthFailed("Anthropic")
	case 429:
		return llm.ErrRateLimit("Anthropic")
	default:
		var apiErr anthropicError
		if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Error.Message != "" {
			return llm.ErrProviderError("Anthropic", statusCode, apiErr.Error.Message)
		}
		return llm.ErrProviderError("Anthropic", statusCode, string(body))
	}
}

// readSSE parses Server-Sent Events from the Anthropic streaming API.
func (a *Anthropic) readSSE(body io.ReadCloser, ch chan<- llm.StreamChunk) {
	defer close(ch)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		if data == "[DONE]" {
			ch <- llm.StreamChunk{Done: true}
			return
		}

		var event struct {
			Type  string `json:"type"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
			Usage *struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		switch event.Type {
		case "content_block_delta":
			if event.Delta.Text != "" {
				ch <- llm.StreamChunk{Delta: event.Delta.Text}
			}
		case "message_delta":
			// Final message with usage stats.
			chunk := llm.StreamChunk{}
			if event.Usage != nil {
				chunk.Usage = &llm.TokenUsage{
					InputTokens:  event.Usage.InputTokens,
					OutputTokens: event.Usage.OutputTokens,
				}
			}
			ch <- chunk
		case "message_stop":
			ch <- llm.StreamChunk{Done: true}
			return
		}
	}

	if err := scanner.Err(); err != nil {
		ch <- llm.StreamChunk{Err: err, Done: true}
	}
}
