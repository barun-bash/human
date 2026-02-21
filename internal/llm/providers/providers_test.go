package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/config"
	"github.com/barun-bash/human/internal/llm"
)

// ── Anthropic Tests ──

func TestAnthropicComplete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("x-api-key") != "test-key" {
			t.Error("missing or wrong x-api-key header")
		}
		if r.Header.Get("anthropic-version") != anthropicVersion {
			t.Error("missing or wrong anthropic-version header")
		}

		// Verify request body
		var req anthropicRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decoding request: %v", err)
		}
		if req.System == "" {
			t.Error("expected system message")
		}

		// Return mock response
		resp := anthropicResponse{
			Model:      "claude-sonnet-4-20250514",
			StopReason: "end_turn",
			Content: []anthropicContent{
				{Type: "text", Text: "app TodoApp is a web application"},
			},
		}
		resp.Usage.InputTokens = 100
		resp.Usage.OutputTokens = 50
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.LLMConfig{
		Provider: "anthropic",
		APIKey:   "test-key",
		Model:    "claude-sonnet-4-20250514",
		BaseURL:  server.URL,
	}

	provider, err := newAnthropic(cfg)
	if err != nil {
		t.Fatalf("creating provider: %v", err)
	}

	resp, err := provider.Complete(context.Background(), &llm.Request{
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: "You are a helpful assistant."},
			{Role: llm.RoleUser, Content: "describe a todo app"},
		},
		MaxTokens: 4096,
	})
	if err != nil {
		t.Fatalf("complete error: %v", err)
	}

	if resp.Content != "app TodoApp is a web application" {
		t.Errorf("content = %q, want %q", resp.Content, "app TodoApp is a web application")
	}
	if resp.TokenUsage.InputTokens != 100 {
		t.Errorf("input tokens = %d, want 100", resp.TokenUsage.InputTokens)
	}
	if resp.TokenUsage.OutputTokens != 50 {
		t.Errorf("output tokens = %d, want 50", resp.TokenUsage.OutputTokens)
	}
}

func TestAnthropicAuthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		fmt.Fprint(w, `{"error":{"type":"authentication_error","message":"invalid api key"}}`)
	}))
	defer server.Close()

	provider := &Anthropic{
		apiKey:  "bad-key",
		model:   "claude-sonnet-4-20250514",
		baseURL: server.URL,
		client:  &http.Client{},
	}

	_, err := provider.Complete(context.Background(), &llm.Request{
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "hello"}},
	})
	if err == nil {
		t.Fatal("expected auth error")
	}

	llmErr, ok := err.(*llm.LLMError)
	if !ok {
		t.Fatalf("expected LLMError, got %T", err)
	}
	if llmErr.Code != "auth_failed" {
		t.Errorf("error code = %q, want %q", llmErr.Code, "auth_failed")
	}
}

func TestAnthropicRateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
		fmt.Fprint(w, `{"error":{"type":"rate_limit_error","message":"too many requests"}}`)
	}))
	defer server.Close()

	provider := &Anthropic{
		apiKey:  "test-key",
		model:   "claude-sonnet-4-20250514",
		baseURL: server.URL,
		client:  &http.Client{},
	}

	_, err := provider.Complete(context.Background(), &llm.Request{
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "hello"}},
	})
	if err == nil {
		t.Fatal("expected rate limit error")
	}

	llmErr, ok := err.(*llm.LLMError)
	if !ok {
		t.Fatalf("expected LLMError, got %T", err)
	}
	if llmErr.Code != "rate_limit" {
		t.Errorf("error code = %q, want %q", llmErr.Code, "rate_limit")
	}
}

func TestAnthropicStreaming(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected ResponseWriter to implement Flusher")
		}

		events := []string{
			`data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"app "}}`,
			`data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"Todo"}}`,
			`data: {"type":"message_delta","usage":{"input_tokens":10,"output_tokens":5}}`,
			`data: {"type":"message_stop"}`,
		}
		for _, event := range events {
			fmt.Fprintln(w, event)
			fmt.Fprintln(w)
			flusher.Flush()
		}
	}))
	defer server.Close()

	provider := &Anthropic{
		apiKey:  "test-key",
		model:   "claude-sonnet-4-20250514",
		baseURL: server.URL,
		client:  &http.Client{},
	}

	ch, err := provider.Stream(context.Background(), &llm.Request{
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("stream error: %v", err)
	}

	var text strings.Builder
	for chunk := range ch {
		if chunk.Err != nil {
			t.Fatalf("chunk error: %v", chunk.Err)
		}
		text.WriteString(chunk.Delta)
	}

	if text.String() != "app Todo" {
		t.Errorf("streamed text = %q, want %q", text.String(), "app Todo")
	}
}

func TestAnthropicNoKey(t *testing.T) {
	_, err := newAnthropic(&config.LLMConfig{
		Provider: "anthropic",
		APIKey:   "",
	})
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
}

// ── OpenAI Tests ──

func TestOpenAIComplete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify auth header
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-key" {
			t.Errorf("auth header = %q, want %q", auth, "Bearer test-key")
		}

		resp := openaiResponse{
			Model: "gpt-4o",
		}
		resp.Choices = []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}{
			{
				Message:      struct{ Content string `json:"content"` }{"app BlogApp is a web application"},
				FinishReason: "stop",
			},
		}
		resp.Usage.PromptTokens = 80
		resp.Usage.CompletionTokens = 40
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.LLMConfig{
		Provider: "openai",
		APIKey:   "test-key",
		Model:    "gpt-4o",
		BaseURL:  server.URL,
	}

	provider, err := newOpenAI(cfg)
	if err != nil {
		t.Fatalf("creating provider: %v", err)
	}

	resp, err := provider.Complete(context.Background(), &llm.Request{
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: "You are a helpful assistant."},
			{Role: llm.RoleUser, Content: "describe a blog"},
		},
		MaxTokens: 4096,
	})
	if err != nil {
		t.Fatalf("complete error: %v", err)
	}

	if resp.Content != "app BlogApp is a web application" {
		t.Errorf("content = %q", resp.Content)
	}
	if resp.TokenUsage.InputTokens != 80 {
		t.Errorf("input tokens = %d, want 80", resp.TokenUsage.InputTokens)
	}
}

func TestOpenAIAuthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		fmt.Fprint(w, `{"error":{"message":"invalid api key","type":"authentication_error"}}`)
	}))
	defer server.Close()

	provider := &OpenAI{
		apiKey:  "bad-key",
		model:   "gpt-4o",
		baseURL: server.URL,
		client:  &http.Client{},
	}

	_, err := provider.Complete(context.Background(), &llm.Request{
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "hello"}},
	})
	if err == nil {
		t.Fatal("expected auth error")
	}

	llmErr, ok := err.(*llm.LLMError)
	if !ok {
		t.Fatalf("expected LLMError, got %T", err)
	}
	if llmErr.Code != "auth_failed" {
		t.Errorf("error code = %q, want %q", llmErr.Code, "auth_failed")
	}
}

func TestOpenAIStreaming(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)

		events := []string{
			`data: {"choices":[{"delta":{"content":"page "}}]}`,
			`data: {"choices":[{"delta":{"content":"Home"}}]}`,
			`data: [DONE]`,
		}
		for _, event := range events {
			fmt.Fprintln(w, event)
			fmt.Fprintln(w)
			flusher.Flush()
		}
	}))
	defer server.Close()

	provider := &OpenAI{
		apiKey:  "test-key",
		model:   "gpt-4o",
		baseURL: server.URL,
		client:  &http.Client{},
	}

	ch, err := provider.Stream(context.Background(), &llm.Request{
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("stream error: %v", err)
	}

	var text strings.Builder
	for chunk := range ch {
		if chunk.Err != nil {
			t.Fatalf("chunk error: %v", chunk.Err)
		}
		text.WriteString(chunk.Delta)
	}

	if text.String() != "page Home" {
		t.Errorf("streamed text = %q, want %q", text.String(), "page Home")
	}
}

func TestOpenAINoKey(t *testing.T) {
	_, err := newOpenAI(&config.LLMConfig{
		Provider: "openai",
		APIKey:   "",
	})
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
}

// ── Ollama Tests ──

func TestOllamaComplete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify NO auth header
		if auth := r.Header.Get("Authorization"); auth != "" {
			t.Errorf("ollama should not send auth header, got %q", auth)
		}

		// Verify path
		if !strings.HasSuffix(r.URL.Path, "/v1/chat/completions") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		resp := openaiResponse{
			Model: "llama3",
		}
		resp.Choices = []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}{
			{
				Message:      struct{ Content string `json:"content"` }{"data User:\n  name is text"},
				FinishReason: "stop",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.LLMConfig{
		Provider: "ollama",
		Model:    "llama3",
		BaseURL:  server.URL,
	}

	provider, err := newOllama(cfg)
	if err != nil {
		t.Fatalf("creating provider: %v", err)
	}

	resp, err := provider.Complete(context.Background(), &llm.Request{
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "create a user model"}},
	})
	if err != nil {
		t.Fatalf("complete error: %v", err)
	}

	if resp.Content != "data User:\n  name is text" {
		t.Errorf("content = %q", resp.Content)
	}
}

func TestOllamaConnectionRefused(t *testing.T) {
	// Use a port that nothing is listening on.
	cfg := &config.LLMConfig{
		Provider: "ollama",
		Model:    "llama3",
		BaseURL:  "http://127.0.0.1:19999",
	}

	provider, err := newOllama(cfg)
	if err != nil {
		t.Fatalf("creating provider: %v", err)
	}

	_, err = provider.Complete(context.Background(), &llm.Request{
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "hello"}},
	})
	if err == nil {
		t.Fatal("expected connection error")
	}

	llmErr, ok := err.(*llm.LLMError)
	if !ok {
		t.Fatalf("expected LLMError, got %T: %v", err, err)
	}
	if llmErr.Code != "ollama_not_running" && llmErr.Code != "network_failure" {
		t.Errorf("error code = %q, want ollama_not_running or network_failure", llmErr.Code)
	}
}

func TestOllamaURLNormalization(t *testing.T) {
	// The Ollama provider should append /v1/chat/completions to bare URLs.
	cfg := &config.LLMConfig{
		Provider: "ollama",
		Model:    "llama3",
		BaseURL:  "http://localhost:11434",
	}

	provider, err := newOllama(cfg)
	if err != nil {
		t.Fatalf("creating provider: %v", err)
	}

	o := provider.(*Ollama)
	if !strings.HasSuffix(o.baseURL, "/v1/chat/completions") {
		t.Errorf("baseURL = %q, should end with /v1/chat/completions", o.baseURL)
	}
}

// ── Registry Tests ──

func TestRegistryCreatesProviders(t *testing.T) {
	tests := []struct {
		provider string
		apiKey   string
	}{
		{"anthropic", "test-key"},
		{"openai", "test-key"},
		{"ollama", ""},
	}

	for _, tt := range tests {
		cfg := &config.LLMConfig{
			Provider: tt.provider,
			APIKey:   tt.apiKey,
			Model:    "test-model",
		}
		p, err := llm.NewProvider(cfg)
		if err != nil {
			t.Errorf("%s: %v", tt.provider, err)
			continue
		}
		if p.Name() != tt.provider {
			t.Errorf("Name() = %q, want %q", p.Name(), tt.provider)
		}
	}
}

func TestRegistryUnknownProvider(t *testing.T) {
	cfg := &config.LLMConfig{
		Provider: "gemini",
		APIKey:   "test-key",
	}
	_, err := llm.NewProvider(cfg)
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestRegistryNilConfig(t *testing.T) {
	_, err := llm.NewProvider(nil)
	if err == nil {
		t.Fatal("expected error for nil config")
	}
}
