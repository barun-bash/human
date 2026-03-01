package figma

import (
	"context"
	"testing"

	"github.com/barun-bash/human/internal/llm"
)

func TestSupportsVision(t *testing.T) {
	tests := []struct {
		name   string
		expect bool
	}{
		{"anthropic", true},
		{"openai", true},
		{"gemini", true},
		{"ollama", false},
		{"groq", false},
	}

	for _, tt := range tests {
		provider := &mockProvider{name: tt.name}
		got := SupportsVision(provider)
		if got != tt.expect {
			t.Errorf("SupportsVision(%q) = %v, want %v", tt.name, got, tt.expect)
		}
	}
}

func TestIsImageFile(t *testing.T) {
	tests := []struct {
		path   string
		expect bool
	}{
		{"screenshot.png", true},
		{"mockup.jpg", true},
		{"design.jpeg", true},
		{"photo.webp", true},
		{"document.pdf", false},
		{"code.go", false},
		{"README.md", false},
		{"image.PNG", true}, // case insensitive
	}

	for _, tt := range tests {
		got := IsImageFile(tt.path)
		if got != tt.expect {
			t.Errorf("IsImageFile(%q) = %v, want %v", tt.path, got, tt.expect)
		}
	}
}

func TestDetectMIMEType(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"file.png", "image/png"},
		{"file.jpg", "image/jpeg"},
		{"file.jpeg", "image/jpeg"},
		{"file.webp", "image/webp"},
		{"file.gif", "image/gif"},
		{"file.unknown", "image/png"},
	}

	for _, tt := range tests {
		got := detectMIMEType(tt.path)
		if got != tt.expected {
			t.Errorf("detectMIMEType(%q) = %q, want %q", tt.path, got, tt.expected)
		}
	}
}

func TestExtractHumanCode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text",
			input:    "app MyApp is a web application",
			expected: "app MyApp is a web application",
		},
		{
			name:     "fenced code block",
			input:    "```\napp MyApp is a web application\n```",
			expected: "app MyApp is a web application",
		},
		{
			name:     "fenced with language hint",
			input:    "```human\napp MyApp is a web application\n```",
			expected: "app MyApp is a web application",
		},
		{
			name:     "whitespace around",
			input:    "  \n  app MyApp is a web application  \n  ",
			expected: "app MyApp is a web application",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractHumanCode(tt.input)
			if got != tt.expected {
				t.Errorf("extractHumanCode() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// mockProvider implements llm.Provider for testing.
type mockProvider struct {
	name string
}

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) Complete(_ context.Context, _ *llm.Request) (*llm.Response, error) {
	return nil, nil
}
func (m *mockProvider) Stream(_ context.Context, _ *llm.Request) (<-chan llm.StreamChunk, error) {
	return nil, nil
}
