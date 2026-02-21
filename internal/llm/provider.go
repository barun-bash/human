package llm

import "context"

// Provider is the interface that LLM backends must implement.
// Each provider (Anthropic, OpenAI, Ollama) implements this interface
// using stdlib net/http — no external dependencies.
type Provider interface {
	// Name returns the provider identifier (e.g. "anthropic", "openai", "ollama").
	Name() string

	// Complete sends a request and returns the full response.
	Complete(ctx context.Context, req *Request) (*Response, error)

	// Stream sends a request and returns a channel of incremental chunks.
	// The channel is closed when the response is complete or an error occurs.
	// Callers should check StreamChunk.Err for per-chunk errors.
	Stream(ctx context.Context, req *Request) (<-chan StreamChunk, error)
}

// Role identifies the sender of a message.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Message is a single message in a conversation.
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

// Request is the input to a Provider.
type Request struct {
	Messages    []Message `json:"messages"`
	Model       string    `json:"model"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature"`
	Stream      bool      `json:"stream"`
}

// Response is the output from a non-streaming completion.
type Response struct {
	Content     string     `json:"content"`
	Model       string     `json:"model"`
	TokenUsage  TokenUsage `json:"usage"`
	StopReason  string     `json:"stop_reason"`
}

// TokenUsage tracks token consumption for cost awareness.
type TokenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// StreamChunk is one piece of a streaming response.
type StreamChunk struct {
	// Delta is the new text fragment. Empty on final chunk.
	Delta string

	// Done is true when the stream is complete.
	Done bool

	// Usage is populated on the final chunk (if the provider reports it).
	Usage *TokenUsage

	// Err is set if an error occurred during streaming.
	Err error
}
