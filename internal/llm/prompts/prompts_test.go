package prompts

import (
	"strings"
	"testing"
)

// ── Prompt Structure Tests ──

func TestAskPrompt(t *testing.T) {
	msgs := AskPrompt("describe a blog application", "")

	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Role != RoleSystem {
		t.Errorf("first message role = %q, want system", msgs[0].Role)
	}
	if msgs[1].Role != RoleUser {
		t.Errorf("second message role = %q, want user", msgs[1].Role)
	}
	if msgs[1].Content != "describe a blog application" {
		t.Errorf("user message = %q", msgs[1].Content)
	}
}

func TestAskPromptWithInstructions(t *testing.T) {
	msgs := AskPrompt("build a todo app", "Always use React and TypeScript")

	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if !strings.Contains(msgs[0].Content, "PROJECT INSTRUCTIONS") {
		t.Error("system prompt should contain instructions header")
	}
	if !strings.Contains(msgs[0].Content, "Always use React and TypeScript") {
		t.Error("system prompt should contain the instructions text")
	}
}

func TestSuggestPrompt(t *testing.T) {
	source := "app Test is a web application\n\ndata User:\n  name is text"
	msgs := SuggestPrompt(source, "")

	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if !strings.Contains(msgs[1].Content, source) {
		t.Error("suggest prompt should include the source code")
	}
	if !strings.Contains(msgs[1].Content, "[performance]") {
		t.Error("suggest prompt should mention category tags")
	}
}

func TestSuggestPromptWithInstructions(t *testing.T) {
	msgs := SuggestPrompt("app Test is a web application", "Focus on security")

	if !strings.Contains(msgs[0].Content, "Focus on security") {
		t.Error("suggest system prompt should contain instructions")
	}
}

func TestEditPrompt(t *testing.T) {
	source := "app Test is a web application"
	instruction := "add a User model"

	msgs := EditPrompt(source, instruction, nil, "")

	// System + user = 2 messages.
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if !strings.Contains(msgs[1].Content, source) {
		t.Error("edit prompt should include the source code")
	}
	if !strings.Contains(msgs[1].Content, instruction) {
		t.Error("edit prompt should include the instruction")
	}
}

func TestEditPromptWithInstructions(t *testing.T) {
	msgs := EditPrompt("source", "instruction", nil, "Use Shadcn components")

	if !strings.Contains(msgs[0].Content, "Use Shadcn components") {
		t.Error("edit system prompt should contain instructions")
	}
	if !strings.Contains(msgs[0].Content, "You are editing") {
		t.Error("edit system prompt should still contain editing context")
	}
}

func TestEditPromptWithHistory(t *testing.T) {
	history := []Message{
		{Role: RoleUser, Content: "add a User model"},
		{Role: RoleAssistant, Content: "```human\ndata User:\n  name is text\n```"},
	}

	msgs := EditPrompt("data User:\n  name is text", "add email field", history, "")

	// System + 2 history + user = 4 messages.
	if len(msgs) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(msgs))
	}
}

func TestEditPromptHistoryCap(t *testing.T) {
	// Create 20 history messages.
	var history []Message
	for i := 0; i < 20; i++ {
		history = append(history, Message{Role: RoleUser, Content: "edit"})
	}

	msgs := EditPrompt("source", "instruction", history, "")

	// System + 10 (capped history) + user = 12.
	if len(msgs) != 12 {
		t.Fatalf("expected 12 messages (10 capped history), got %d", len(msgs))
	}
}

func TestBuildSystemPrompt_Empty(t *testing.T) {
	result := buildSystemPrompt("base prompt", "")
	if result != "base prompt" {
		t.Errorf("expected unchanged base prompt, got: %q", result)
	}
}

func TestBuildSystemPrompt_WithInstructions(t *testing.T) {
	result := buildSystemPrompt("base prompt", "my instructions")
	if !strings.Contains(result, "base prompt") {
		t.Error("should contain base prompt")
	}
	if !strings.Contains(result, "PROJECT INSTRUCTIONS") {
		t.Error("should contain instructions header")
	}
	if !strings.Contains(result, "my instructions") {
		t.Error("should contain instructions content")
	}
}

func TestSystemPromptContainsKeyElements(t *testing.T) {
	checks := []string{
		"data <Name>:",
		"page <Name>:",
		"api <Name>:",
		"build with:",
		"text, number",
		"```human",
	}

	for _, check := range checks {
		if !strings.Contains(SystemPrompt, check) {
			t.Errorf("system prompt missing %q", check)
		}
	}
}

// ── Extraction Tests ──

func TestExtractHumanCode(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "with human fence",
			input: `Here is the code:

` + "```human" + `
app Blog is a web application

data Post:
  title is text
` + "```" + `

Hope this helps!`,
			want: "app Blog is a web application\n\ndata Post:\n  title is text",
		},
		{
			name: "with generic fence",
			input: "```\napp Test is a web application\n```",
			want:  "app Test is a web application",
		},
		{
			name:  "no fence",
			input: "app Test is a web application",
			want:  "app Test is a web application",
		},
		{
			name: "unclosed fence",
			input: "```human\napp Test is a web application\ndata User:\n  name is text",
			want:  "app Test is a web application\ndata User:\n  name is text",
		},
		{
			name:  "empty input",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractHumanCode(tt.input)
			if got != tt.want {
				t.Errorf("ExtractHumanCode:\n  got:  %q\n  want: %q", got, tt.want)
			}
		})
	}
}

func TestExtractSuggestions(t *testing.T) {
	input := `Here are my suggestions:

[security] Add rate limiting to all API endpoints
[performance] Consider indexing the email field on User
[usability] Add a loading state to the Dashboard page
[structure] Extract the sidebar into a reusable component
Not a suggestion line
[feature] Add password reset workflow`

	suggestions := ExtractSuggestions(input)

	if len(suggestions) != 5 {
		t.Fatalf("expected 5 suggestions, got %d", len(suggestions))
	}

	expected := []struct {
		category string
		prefix   string
	}{
		{"security", "Add rate limiting"},
		{"performance", "Consider indexing"},
		{"usability", "Add a loading"},
		{"structure", "Extract the sidebar"},
		{"feature", "Add password reset"},
	}

	for i, e := range expected {
		if suggestions[i].Category != e.category {
			t.Errorf("suggestion %d category = %q, want %q", i, suggestions[i].Category, e.category)
		}
		if !strings.HasPrefix(suggestions[i].Text, e.prefix) {
			t.Errorf("suggestion %d text = %q, want prefix %q", i, suggestions[i].Text, e.prefix)
		}
	}
}

func TestExtractSuggestionsEmpty(t *testing.T) {
	suggestions := ExtractSuggestions("no suggestions here")
	if len(suggestions) != 0 {
		t.Errorf("expected 0 suggestions, got %d", len(suggestions))
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		text string
		min  int
		max  int
	}{
		{"", 0, 0},
		{"hello", 1, 2},
		{"this is a longer text that should be more tokens", 10, 20},
	}

	for _, tt := range tests {
		tokens := EstimateTokens(tt.text)
		if tokens < tt.min || tokens > tt.max {
			t.Errorf("EstimateTokens(%q) = %d, expected between %d and %d", tt.text, tokens, tt.min, tt.max)
		}
	}
}

func TestContextWindowSize(t *testing.T) {
	tests := []struct {
		model string
		want  int
	}{
		{"claude-sonnet-4-20250514", 200000},
		{"gpt-4o", 128000},
		{"llama3", 8000},
		{"unknown-model", 8000},
	}

	for _, tt := range tests {
		got := ContextWindowSize(tt.model)
		if got != tt.want {
			t.Errorf("ContextWindowSize(%q) = %d, want %d", tt.model, got, tt.want)
		}
	}
}
