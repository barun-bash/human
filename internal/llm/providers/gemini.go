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

const geminiDefaultBaseURL = "https://generativelanguage.googleapis.com/v1beta"

// Gemini implements the llm.Provider interface for the Google AI Gemini API.
type Gemini struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

func init() {
	llm.RegisterProvider("gemini", newGemini)
}

func newGemini(cfg *config.LLMConfig) (llm.Provider, error) {
	if cfg.APIKey == "" {
		return nil, llm.ErrNoAPIKey("gemini")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = geminiDefaultBaseURL
	}

	return &Gemini{
		apiKey:  cfg.APIKey,
		model:   cfg.Model,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{},
	}, nil
}

func (g *Gemini) Name() string { return "gemini" }

// Gemini API request/response types.

type geminiRequest struct {
	Contents         []geminiContent        `json:"contents"`
	SystemInstruction *geminiContent        `json:"systemInstruction,omitempty"`
	GenerationConfig *geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text       string          `json:"text,omitempty"`
	InlineData *geminiBlobPart `json:"inlineData,omitempty"`
}

type geminiBlobPart struct {
	MIMEType string `json:"mimeType"`
	Data     string `json:"data"`
}

type geminiGenerationConfig struct {
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
	Temperature     float64 `json:"temperature"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []geminiPart `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata *struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
	} `json:"usageMetadata"`
}

type geminiError struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

func (g *Gemini) Complete(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	model := req.Model
	if model == "" {
		model = g.model
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", g.baseURL, model, g.apiKey)
	body := g.buildRequest(req)

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return nil, llm.ErrNetworkFailure("Gemini", err.Error())
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if err := g.checkError(resp.StatusCode, respBody); err != nil {
		return nil, err
	}

	var apiResp geminiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	content := ""
	stopReason := ""
	if len(apiResp.Candidates) > 0 {
		for _, part := range apiResp.Candidates[0].Content.Parts {
			content += part.Text
		}
		stopReason = apiResp.Candidates[0].FinishReason
	}

	usage := llm.TokenUsage{}
	if apiResp.UsageMetadata != nil {
		usage.InputTokens = apiResp.UsageMetadata.PromptTokenCount
		usage.OutputTokens = apiResp.UsageMetadata.CandidatesTokenCount
	}

	return &llm.Response{
		Content:    content,
		Model:      model,
		StopReason: stopReason,
		TokenUsage: usage,
	}, nil
}

func (g *Gemini) Stream(ctx context.Context, req *llm.Request) (<-chan llm.StreamChunk, error) {
	model := req.Model
	if model == "" {
		model = g.model
	}

	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse&key=%s", g.baseURL, model, g.apiKey)
	body := g.buildRequest(req)

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return nil, llm.ErrNetworkFailure("Gemini", err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, g.checkError(resp.StatusCode, respBody)
	}

	ch := make(chan llm.StreamChunk, 64)
	go g.readSSE(resp.Body, ch)
	return ch, nil
}

func (g *Gemini) buildRequest(req *llm.Request) geminiRequest {
	gr := geminiRequest{
		GenerationConfig: &geminiGenerationConfig{
			MaxOutputTokens: req.MaxTokens,
			Temperature:     req.Temperature,
		},
	}

	for _, msg := range req.Messages {
		if msg.Role == llm.RoleSystem {
			gr.SystemInstruction = &geminiContent{
				Parts: []geminiPart{{Text: msg.Content}},
			}
			continue
		}

		role := "user"
		if msg.Role == llm.RoleAssistant {
			role = "model"
		}
		gr.Contents = append(gr.Contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: msg.Content}},
		})
	}

	// If images are provided, add inline data parts to the last user content.
	if len(req.Images) > 0 && len(gr.Contents) > 0 {
		last := len(gr.Contents) - 1
		if gr.Contents[last].Role == "user" {
			for _, img := range req.Images {
				gr.Contents[last].Parts = append(gr.Contents[last].Parts, geminiPart{
					InlineData: &geminiBlobPart{
						MIMEType: img.MIMEType,
						Data:     img.Data,
					},
				})
			}
		}
	}

	return gr
}

func (g *Gemini) checkError(statusCode int, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	switch statusCode {
	case 401, 403:
		return llm.ErrAuthFailed("Gemini")
	case 429:
		return llm.ErrRateLimit("Gemini")
	default:
		var apiErr geminiError
		if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Error.Message != "" {
			return llm.ErrProviderError("Gemini", statusCode, apiErr.Error.Message)
		}
		return llm.ErrProviderError("Gemini", statusCode, string(body))
	}
}

// readSSE parses Server-Sent Events from the Gemini streaming API.
func (g *Gemini) readSSE(body io.ReadCloser, ch chan<- llm.StreamChunk) {
	defer close(ch)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		var resp geminiResponse
		if err := json.Unmarshal([]byte(data), &resp); err != nil {
			continue
		}

		if len(resp.Candidates) > 0 {
			for _, part := range resp.Candidates[0].Content.Parts {
				if part.Text != "" {
					ch <- llm.StreamChunk{Delta: part.Text}
				}
			}
		}

		if resp.UsageMetadata != nil {
			ch <- llm.StreamChunk{
				Usage: &llm.TokenUsage{
					InputTokens:  resp.UsageMetadata.PromptTokenCount,
					OutputTokens: resp.UsageMetadata.CandidatesTokenCount,
				},
			}
		}
	}

	if err := scanner.Err(); err != nil {
		ch <- llm.StreamChunk{Err: err, Done: true}
		return
	}

	ch <- llm.StreamChunk{Done: true}
}
