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

const openaiDefaultURL = "https://api.openai.com/v1/chat/completions"

// OpenAI implements the llm.Provider interface for the OpenAI Chat Completions API.
// It also serves as the base for OpenAI-compatible providers (Groq, OpenRouter, Custom).
type OpenAI struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
	name    string // provider name, defaults to "openai"
}

func init() {
	llm.RegisterProvider("openai", newOpenAI)
}

func newOpenAI(cfg *config.LLMConfig) (llm.Provider, error) {
	if cfg.APIKey == "" {
		return nil, llm.ErrNoAPIKey("openai")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = openaiDefaultURL
	}

	return &OpenAI{
		apiKey:  cfg.APIKey,
		model:   cfg.Model,
		baseURL: baseURL,
		client:  defaultHTTPClient(),
		name:    "openai",
	}, nil
}

// defaultHTTPClient returns a shared HTTP client for OpenAI-compatible providers.
func defaultHTTPClient() *http.Client {
	return &http.Client{}
}

func (o *OpenAI) Name() string {
	if o.name != "" {
		return o.name
	}
	return "openai"
}

// openaiRequest is the OpenAI Chat Completions request format.
type openaiRequest struct {
	Model       string          `json:"model"`
	Messages    []openaiMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature"`
	Stream      bool            `json:"stream,omitempty"`
}

type openaiMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // string or []openaiContentPart for vision
}

type openaiContentPart struct {
	Type     string              `json:"type"`
	Text     string              `json:"text,omitempty"`
	ImageURL *openaiImageURLPart `json:"image_url,omitempty"`
}

type openaiImageURLPart struct {
	URL string `json:"url"`
}

// openaiResponse is the OpenAI Chat Completions non-streaming response.
type openaiResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

type openaiError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

func (o *OpenAI) Complete(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	body := o.buildRequest(req, false)

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.baseURL, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	o.setHeaders(httpReq)

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, llm.ErrNetworkFailure("OpenAI", err.Error())
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if err := o.checkError(resp.StatusCode, respBody); err != nil {
		return nil, err
	}

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

func (o *OpenAI) Stream(ctx context.Context, req *llm.Request) (<-chan llm.StreamChunk, error) {
	body := o.buildRequest(req, true)

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.baseURL, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	o.setHeaders(httpReq)

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, llm.ErrNetworkFailure("OpenAI", err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, o.checkError(resp.StatusCode, respBody)
	}

	ch := make(chan llm.StreamChunk, 64)
	go readOpenAISSE(resp.Body, ch)
	return ch, nil
}

func (o *OpenAI) buildRequest(req *llm.Request, stream bool) openaiRequest {
	or := openaiRequest{
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
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

	// If images are provided, convert the last user message to multi-part content.
	if len(req.Images) > 0 && len(or.Messages) > 0 {
		last := len(or.Messages) - 1
		if or.Messages[last].Role == "user" {
			var parts []openaiContentPart
			// Text part first.
			if text, ok := or.Messages[last].Content.(string); ok && text != "" {
				parts = append(parts, openaiContentPart{
					Type: "text",
					Text: text,
				})
			}
			// Image parts.
			for _, img := range req.Images {
				dataURL := fmt.Sprintf("data:%s;base64,%s", img.MIMEType, img.Data)
				parts = append(parts, openaiContentPart{
					Type:     "image_url",
					ImageURL: &openaiImageURLPart{URL: dataURL},
				})
			}
			or.Messages[last].Content = parts
		}
	}

	return or
}

func (o *OpenAI) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)
}

func (o *OpenAI) checkError(statusCode int, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	switch statusCode {
	case 401:
		return llm.ErrAuthFailed("OpenAI")
	case 429:
		return llm.ErrRateLimit("OpenAI")
	default:
		var apiErr openaiError
		if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Error.Message != "" {
			return llm.ErrProviderError("OpenAI", statusCode, apiErr.Error.Message)
		}
		return llm.ErrProviderError("OpenAI", statusCode, string(body))
	}
}

// readOpenAISSE parses Server-Sent Events from the OpenAI streaming API.
// Exported for reuse by the Ollama provider (OpenAI-compatible endpoint).
func readOpenAISSE(body io.ReadCloser, ch chan<- llm.StreamChunk) {
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
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		if len(event.Choices) > 0 {
			delta := event.Choices[0].Delta.Content
			if delta != "" {
				ch <- llm.StreamChunk{Delta: delta}
			}
		}

		if event.Usage != nil {
			ch <- llm.StreamChunk{
				Usage: &llm.TokenUsage{
					InputTokens:  event.Usage.PromptTokens,
					OutputTokens: event.Usage.CompletionTokens,
				},
			}
		}
	}

	if err := scanner.Err(); err != nil {
		ch <- llm.StreamChunk{Err: err, Done: true}
		return
	}

	// If no explicit [DONE], send final.
	ch <- llm.StreamChunk{Done: true}
}
