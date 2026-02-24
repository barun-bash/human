package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/barun-bash/human/internal/config"
	"github.com/barun-bash/human/internal/llm/prompts"
	"github.com/barun-bash/human/internal/parser"
)

// Connector orchestrates LLM operations: ask, suggest, and edit.
// It wraps a Provider and adds validation via the parser.
type Connector struct {
	provider     Provider
	config       *config.LLMConfig
	Instructions string // optional project instructions from HUMAN.md
}

// NewConnector creates a Connector with the given provider and config.
func NewConnector(provider Provider, cfg *config.LLMConfig) *Connector {
	return &Connector{
		provider: provider,
		config:   cfg,
	}
}

// AskResult is the result of an Ask operation.
type AskResult struct {
	// RawResponse is the full LLM response.
	RawResponse string

	// Code is the extracted .human code (fences stripped).
	Code string

	// Valid is true if the extracted code parses successfully.
	Valid bool

	// ParseError describes why the code is invalid (empty if Valid).
	ParseError string

	// Usage tracks token consumption.
	Usage TokenUsage
}

// Ask sends a freeform query to the LLM and returns generated .human code.
// The response is validated against the parser.
func (c *Connector) Ask(ctx context.Context, query string) (*AskResult, error) {
	pMsgs := prompts.AskPrompt(query, c.Instructions)

	resp, err := c.provider.Complete(ctx, &Request{
		Messages:    convertMessages(pMsgs),
		Model:       c.config.Model,
		MaxTokens:   c.config.MaxTokens,
		Temperature: c.config.Temperature,
	})
	if err != nil {
		return nil, err
	}

	code := prompts.ExtractHumanCode(resp.Content)
	valid, parseErr := validateCode(code)

	return &AskResult{
		RawResponse: resp.Content,
		Code:        code,
		Valid:       valid,
		ParseError:  parseErr,
		Usage:       resp.TokenUsage,
	}, nil
}

// AskStream sends a query and streams the response. It returns a channel of
// StreamChunks for real-time output. The caller should collect the full text
// and call ValidateCode() separately after the stream completes.
func (c *Connector) AskStream(ctx context.Context, query string) (<-chan StreamChunk, error) {
	pMsgs := prompts.AskPrompt(query, c.Instructions)

	return c.provider.Stream(ctx, &Request{
		Messages:    convertMessages(pMsgs),
		Model:       c.config.Model,
		MaxTokens:   c.config.MaxTokens,
		Temperature: c.config.Temperature,
		Stream:      true,
	})
}

// SuggestResult is the result of a Suggest operation.
type SuggestResult struct {
	RawResponse string
	Suggestions []prompts.Suggestion
	Usage       TokenUsage
}

// Suggest analyzes a .human source file and returns improvement suggestions.
func (c *Connector) Suggest(ctx context.Context, source string) (*SuggestResult, error) {
	// Check token estimate against context window.
	tokens := prompts.EstimateTokens(source)
	window := prompts.ContextWindowSize(c.config.Model)
	if tokens > int(float64(window)*0.8) {
		return nil, fmt.Errorf("source file is too large (%d estimated tokens) for the model's context window (%d tokens). Consider splitting into smaller files", tokens, window)
	}

	pMsgs := prompts.SuggestPrompt(source, c.Instructions)

	resp, err := c.provider.Complete(ctx, &Request{
		Messages:    convertMessages(pMsgs),
		Model:       c.config.Model,
		MaxTokens:   c.config.MaxTokens,
		Temperature: c.config.Temperature,
	})
	if err != nil {
		return nil, err
	}

	suggestions := prompts.ExtractSuggestions(resp.Content)

	return &SuggestResult{
		RawResponse: resp.Content,
		Suggestions: suggestions,
		Usage:       resp.TokenUsage,
	}, nil
}

// EditResult is the result of an Edit operation.
type EditResult struct {
	RawResponse string
	Code        string
	Valid       bool
	ParseError  string
	Usage       TokenUsage
}

// Edit applies an instruction to existing .human source, with optional
// conversation history for multi-turn editing.
func (c *Connector) Edit(ctx context.Context, source, instruction string, history []Message) (*EditResult, error) {
	// Convert llm.Message history to prompts.Message history.
	pHistory := make([]prompts.Message, len(history))
	for i, m := range history {
		pHistory[i] = prompts.Message{
			Role:    prompts.Role(m.Role),
			Content: m.Content,
		}
	}

	pMsgs := prompts.EditPrompt(source, instruction, pHistory, c.Instructions)

	resp, err := c.provider.Complete(ctx, &Request{
		Messages:    convertMessages(pMsgs),
		Model:       c.config.Model,
		MaxTokens:   c.config.MaxTokens,
		Temperature: c.config.Temperature,
	})
	if err != nil {
		return nil, err
	}

	code := prompts.ExtractHumanCode(resp.Content)
	valid, parseErr := validateCode(code)

	return &EditResult{
		RawResponse: resp.Content,
		Code:        code,
		Valid:       valid,
		ParseError:  parseErr,
		Usage:       resp.TokenUsage,
	}, nil
}

// ExtractHumanCode strips markdown code fences from an LLM response and
// returns the raw .human code. Useful for post-processing streamed output.
func ExtractHumanCode(response string) string {
	return prompts.ExtractHumanCode(response)
}

// ValidateCode checks if a string is valid .human code by running it through
// the parser. Returns (true, "") if valid, (false, errorMessage) if not.
func ValidateCode(code string) (bool, string) {
	return validateCode(code)
}

// ExtractAndValidate extracts .human code from an LLM response (stripping
// fences) and validates it. Returns the extracted code, validity, and any
// parse error. This is the correct function to call on streamed output.
func ExtractAndValidate(response string) (code string, valid bool, parseErr string) {
	code = prompts.ExtractHumanCode(response)
	valid, parseErr = validateCode(code)
	return
}

func validateCode(code string) (bool, string) {
	if strings.TrimSpace(code) == "" {
		return false, "empty code"
	}

	_, err := parser.Parse(code)
	if err != nil {
		return false, err.Error()
	}
	return true, ""
}

// convertMessages converts prompts.Message to llm.Message.
func convertMessages(pMsgs []prompts.Message) []Message {
	msgs := make([]Message, len(pMsgs))
	for i, m := range pMsgs {
		msgs[i] = Message{
			Role:    Role(m.Role),
			Content: m.Content,
		}
	}
	return msgs
}
