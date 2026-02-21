package llm

import (
	"context"
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/config"
)

// mockProvider is a test double for the Provider interface.
type mockProvider struct {
	name       string
	response   *Response
	err        error
	chunks     []StreamChunk
	streamErr  error
}

func (m *mockProvider) Name() string { return m.name }

func (m *mockProvider) Complete(ctx context.Context, req *Request) (*Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func (m *mockProvider) Stream(ctx context.Context, req *Request) (<-chan StreamChunk, error) {
	if m.streamErr != nil {
		return nil, m.streamErr
	}
	ch := make(chan StreamChunk, len(m.chunks))
	for _, chunk := range m.chunks {
		ch <- chunk
	}
	close(ch)
	return ch, nil
}

func TestConnectorAskValid(t *testing.T) {
	mock := &mockProvider{
		name: "mock",
		response: &Response{
			Content: "```human\napp Blog is a web application\n```",
			TokenUsage: TokenUsage{
				InputTokens:  100,
				OutputTokens: 20,
			},
		},
	}

	cfg := &config.LLMConfig{
		Provider:  "mock",
		Model:     "test-model",
		MaxTokens: 4096,
	}

	connector := NewConnector(mock, cfg)
	result, err := connector.Ask(context.Background(), "create a blog app")
	if err != nil {
		t.Fatalf("ask error: %v", err)
	}

	if result.Code != "app Blog is a web application" {
		t.Errorf("code = %q", result.Code)
	}
	if !result.Valid {
		t.Errorf("expected valid code, got parse error: %s", result.ParseError)
	}
	if result.Usage.InputTokens != 100 {
		t.Errorf("input tokens = %d, want 100", result.Usage.InputTokens)
	}
}

func TestConnectorAskEmpty(t *testing.T) {
	mock := &mockProvider{
		name: "mock",
		response: &Response{
			Content: "I don't know how to do that.",
		},
	}

	cfg := &config.LLMConfig{
		Provider:  "mock",
		Model:     "test-model",
		MaxTokens: 4096,
	}

	connector := NewConnector(mock, cfg)
	result, err := connector.Ask(context.Background(), "generate something")
	if err != nil {
		t.Fatalf("ask error: %v", err)
	}

	// When the LLM doesn't produce code in a fence, the raw text is still returned.
	if result.Code == "" {
		t.Error("expected code to be returned (raw response)")
	}
	// Validation: the parser may or may not accept raw English. What matters
	// is that the result is populated and no panic occurs.
	_ = result.Valid
	_ = result.ParseError
}

func TestConnectorAskProviderError(t *testing.T) {
	mock := &mockProvider{
		name: "mock",
		err:  ErrRateLimit("mock"),
	}

	cfg := &config.LLMConfig{
		Provider:  "mock",
		Model:     "test-model",
		MaxTokens: 4096,
	}

	connector := NewConnector(mock, cfg)
	_, err := connector.Ask(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error from provider")
	}

	llmErr, ok := err.(*LLMError)
	if !ok {
		t.Fatalf("expected LLMError, got %T", err)
	}
	if llmErr.Code != "rate_limit" {
		t.Errorf("error code = %q, want rate_limit", llmErr.Code)
	}
}

func TestConnectorAskStream(t *testing.T) {
	mock := &mockProvider{
		name: "mock",
		chunks: []StreamChunk{
			{Delta: "app "},
			{Delta: "Test"},
			{Done: true},
		},
	}

	cfg := &config.LLMConfig{
		Provider:  "mock",
		Model:     "test-model",
		MaxTokens: 4096,
	}

	connector := NewConnector(mock, cfg)
	ch, err := connector.AskStream(context.Background(), "describe an app")
	if err != nil {
		t.Fatalf("stream error: %v", err)
	}

	var text strings.Builder
	for chunk := range ch {
		text.WriteString(chunk.Delta)
	}

	if text.String() != "app Test" {
		t.Errorf("streamed text = %q, want %q", text.String(), "app Test")
	}
}

func TestConnectorSuggest(t *testing.T) {
	mock := &mockProvider{
		name: "mock",
		response: &Response{
			Content: "[security] Add rate limiting to API endpoints\n[performance] Index the email field",
			TokenUsage: TokenUsage{
				InputTokens:  200,
				OutputTokens: 30,
			},
		},
	}

	cfg := &config.LLMConfig{
		Provider:  "mock",
		Model:     "test-model",
		MaxTokens: 4096,
	}

	connector := NewConnector(mock, cfg)
	result, err := connector.Suggest(context.Background(), "app Test is a web application")
	if err != nil {
		t.Fatalf("suggest error: %v", err)
	}

	if len(result.Suggestions) != 2 {
		t.Fatalf("expected 2 suggestions, got %d", len(result.Suggestions))
	}
	if result.Suggestions[0].Category != "security" {
		t.Errorf("suggestion 0 category = %q", result.Suggestions[0].Category)
	}
}

func TestConnectorSuggestTooLarge(t *testing.T) {
	mock := &mockProvider{name: "mock"}

	cfg := &config.LLMConfig{
		Provider:  "mock",
		Model:     "llama3", // 8K context window
		MaxTokens: 4096,
	}

	// Create source larger than 80% of 8K tokens (~6400 tokens ~= 25600 chars).
	largeSource := strings.Repeat("x", 30000)

	connector := NewConnector(mock, cfg)
	_, err := connector.Suggest(context.Background(), largeSource)
	if err == nil {
		t.Fatal("expected error for oversized source")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Errorf("error = %q, expected 'too large'", err.Error())
	}
}

func TestConnectorEdit(t *testing.T) {
	mock := &mockProvider{
		name: "mock",
		response: &Response{
			Content: "```human\napp Blog is a web application\n\ndata User:\n  name is text\n  email is email\n```",
		},
	}

	cfg := &config.LLMConfig{
		Provider:  "mock",
		Model:     "test-model",
		MaxTokens: 4096,
	}

	connector := NewConnector(mock, cfg)
	result, err := connector.Edit(context.Background(), "app Blog is a web application", "add a User model", nil)
	if err != nil {
		t.Fatalf("edit error: %v", err)
	}

	if !strings.Contains(result.Code, "data User:") {
		t.Errorf("code = %q, expected to contain User model", result.Code)
	}
	if !result.Valid {
		t.Errorf("expected valid code, got parse error: %s", result.ParseError)
	}
}

func TestValidateCode(t *testing.T) {
	tests := []struct {
		name  string
		code  string
		valid bool
	}{
		{"valid app", "app Test is a web application", true},
		{"empty", "", false},
		{"whitespace", "   \n  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, _ := ValidateCode(tt.code)
			if valid != tt.valid {
				t.Errorf("ValidateCode(%q) = %v, want %v", tt.code, valid, tt.valid)
			}
		})
	}
}

func TestExtractAndValidate(t *testing.T) {
	tests := []struct {
		name     string
		response string
		wantCode string
		valid    bool
	}{
		{
			name:     "fenced valid code",
			response: "Here is the code:\n\n```human\napp Blog is a web application\n```\n\nHope this helps!",
			wantCode: "app Blog is a web application",
			valid:    true,
		},
		{
			name:     "raw valid code",
			response: "app Blog is a web application",
			wantCode: "app Blog is a web application",
			valid:    true,
		},
		{
			name:     "empty response",
			response: "",
			wantCode: "",
			valid:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, valid, _ := ExtractAndValidate(tt.response)
			if code != tt.wantCode {
				t.Errorf("code = %q, want %q", code, tt.wantCode)
			}
			if valid != tt.valid {
				t.Errorf("valid = %v, want %v", valid, tt.valid)
			}
		})
	}
}

func TestExtractHumanCode(t *testing.T) {
	response := "Sure!\n\n```human\napp Test is a web application\n```\n"
	code := ExtractHumanCode(response)
	if code != "app Test is a web application" {
		t.Errorf("ExtractHumanCode = %q, want %q", code, "app Test is a web application")
	}
}
