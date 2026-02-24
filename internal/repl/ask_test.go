package repl

import (
	"os"
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/cli"
	"github.com/barun-bash/human/internal/config"
)

func TestAsk_NoLLMConfigured(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	cli.ColorEnabled = false

	r, _, errOut := newTestREPL("/ask build me a blog\n/quit\n")
	r.Run()
	output := errOut.String()

	if !strings.Contains(output, "/connect") {
		t.Errorf("expected error message pointing to /connect, got: %s", output)
	}
}

func TestAsk_NoDescription(t *testing.T) {
	cli.ColorEnabled = false

	// /ask with no args prompts for description; the next line is empty.
	r, _, errOut := newTestREPL("/ask\n\n/quit\n")
	r.Run()
	output := errOut.String()

	if !strings.Contains(output, "No description provided") {
		t.Errorf("expected 'No description provided' error, got: %s", output)
	}
}

func TestAsk_NoDescriptionEOF(t *testing.T) {
	cli.ColorEnabled = false

	// /ask with no args, then EOF.
	r, _, errOut := newTestREPL("/ask\n")
	r.Run()
	output := errOut.String()

	if !strings.Contains(output, "No description provided") {
		t.Errorf("expected 'No description provided' error on EOF, got: %s", output)
	}
}

func TestAsk_PromptShown(t *testing.T) {
	cli.ColorEnabled = false

	// /ask with no args should show the prompt.
	r, out, _ := newTestREPL("/ask\n\n/quit\n")
	r.Run()
	output := out.String()

	if !strings.Contains(output, "Describe the app") {
		t.Errorf("expected description prompt, got: %s", output)
	}
}

func TestAsk_HelpListing(t *testing.T) {
	cli.ColorEnabled = false
	r, out, _ := newTestREPL("/help\n/quit\n")
	r.Run()
	output := out.String()

	if !strings.Contains(output, "/ask") {
		t.Error("expected /help output to list /ask")
	}
}

func TestAsk_HelpOrder(t *testing.T) {
	cli.ColorEnabled = false
	r, out, _ := newTestREPL("/help\n/quit\n")
	r.Run()
	output := out.String()

	// Search within the help listing section only (after "Available Commands")
	// to avoid false matches in the banner/tips area.
	helpStart := strings.Index(output, "Available Commands")
	if helpStart < 0 {
		t.Fatal("expected 'Available Commands' heading in output")
	}
	helpSection := output[helpStart:]

	askIdx := strings.Index(helpSection, "/ask")
	buildIdx := strings.Index(helpSection, "/build")
	if askIdx < 0 || buildIdx < 0 {
		t.Fatal("expected both /ask and /build in help output")
	}
	// /ask should appear before /build in the help listing.
	if askIdx > buildIdx {
		t.Error("expected /ask to appear before /build in help")
	}
}

func TestAsk_WithGlobalConfig_NoEnv(t *testing.T) {
	// Test that loadREPLConnector finds the global config.
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")

	gc := &config.GlobalConfig{
		LLM: &config.GlobalLLMConfig{
			Provider: "anthropic",
			Model:    "claude-sonnet-4-20250514",
			APIKey:   "sk-test-key",
		},
	}
	if err := config.SaveGlobalConfig(gc); err != nil {
		t.Fatal(err)
	}

	// loadREPLConnector should find it (will fail at NewProvider since
	// providers aren't registered in tests, but it proves resolution works).
	_, _, err := loadREPLConnector()
	// We expect either success or a "unknown LLM provider" error (if providers
	// package init() hasn't run). Either way, NOT the "no LLM provider" error.
	if err != nil && strings.Contains(err.Error(), "no LLM provider configured") {
		t.Errorf("expected global config to be found, got: %v", err)
	}
}

func TestAsk_EnvVarFallback(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("ANTHROPIC_API_KEY", "sk-env-key")
	t.Setenv("OPENAI_API_KEY", "")

	// No global config, no project config — should find env var.
	_, _, err := loadREPLConnector()
	// Should NOT be the "no LLM provider" error.
	if err != nil && strings.Contains(err.Error(), "no LLM provider configured") {
		t.Errorf("expected env var to be detected, got: %v", err)
	}
}

// ── Filename derivation tests ──

func TestDeriveFilename(t *testing.T) {
	tests := []struct {
		name string
		code string
		want string
	}{
		{
			name: "simple app name",
			code: "app TaskFlow is a web application",
			want: "task-flow.human",
		},
		{
			name: "single word",
			code: "app Blog is a web application",
			want: "blog.human",
		},
		{
			name: "multi word camel",
			code: "app ProjectManagementTool is a web application",
			want: "project-management-tool.human",
		},
		{
			name: "no app declaration",
			code: "page Home:\n  show \"hello\"",
			want: "app.human",
		},
		{
			name: "empty code",
			code: "",
			want: "app.human",
		},
		{
			name: "app in indented line",
			code: "  app MyApp is a web application",
			want: "my-app.human",
		},
		{
			name: "case insensitive",
			code: "App MyBlog is a web application",
			want: "my-blog.human",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveFilename(tt.code)
			if got != tt.want {
				t.Errorf("deriveFilename() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCamelToKebab(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"KanbanFlow", "kanban-flow"},
		{"Blog", "blog"},
		{"HTTPServer", "httpserver"},
		{"myApp", "my-app"},
		{"ABC", "abc"},
		{"task123Flow", "task123-flow"},
	}

	for _, tt := range tests {
		got := camelToKebab(tt.input)
		if got != tt.want {
			t.Errorf("camelToKebab(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"my-app", "my-app"},
		{"my-app!@#", "my-app"},
		{"-leading-", "leading"},
		{"hello_world", "helloworld"},
		{"", ""},
	}

	for _, tt := range tests {
		got := sanitizeFilename(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIsYes(t *testing.T) {
	for _, input := range []string{"y", "Y", "yes", "Yes", "YES", " y "} {
		if !isYes(input) {
			t.Errorf("isYes(%q) = false, want true", input)
		}
	}
	for _, input := range []string{"n", "no", "", "maybe", "yep"} {
		if isYes(input) {
			t.Errorf("isYes(%q) = true, want false", input)
		}
	}
}

func TestAsk_OverwriteDeclined(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	cli.ColorEnabled = false

	// Create a pre-existing file that would be the target.
	if err := os.WriteFile("app.human", []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove("app.human")

	// The /ask command will fail at LLM loading (no provider) before it
	// gets to overwrite logic. This test just verifies the file is untouched.
	r, _, _ := newTestREPL("/ask build a blog\n/quit\n")
	r.Run()

	data, err := os.ReadFile("app.human")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "existing" {
		t.Error("pre-existing file should not be modified when LLM is not configured")
	}
}
