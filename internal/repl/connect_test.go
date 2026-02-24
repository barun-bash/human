package repl

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/cli"
	"github.com/barun-bash/human/internal/config"
)

func TestConnect_Status_NoProvider(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cli.ColorEnabled = false

	r, out, _ := newTestREPL("/connect\n/quit\n")
	r.Run()
	output := out.String()

	if !strings.Contains(output, "Not configured") {
		t.Errorf("expected 'Not configured' in output, got: %s", output)
	}
	if !strings.Contains(output, "/connect <provider>") {
		t.Errorf("expected setup hint in output, got: %s", output)
	}
}

func TestConnect_Status_WithProvider(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	cli.ColorEnabled = false

	// Pre-populate global config.
	gc := &config.GlobalConfig{
		LLM: &config.GlobalLLMConfig{
			Provider: "anthropic",
			Model:    "claude-sonnet-4-20250514",
			APIKey:   "sk-ant-test-key-abcdef1234",
		},
	}
	if err := config.SaveGlobalConfig(gc); err != nil {
		t.Fatal(err)
	}

	r, out, _ := newTestREPL("/connect status\n/quit\n")
	r.Run()
	output := out.String()

	if !strings.Contains(output, "anthropic") {
		t.Errorf("expected 'anthropic' in status output, got: %s", output)
	}
	if !strings.Contains(output, "...1234") {
		t.Errorf("expected masked key '...1234' in status output, got: %s", output)
	}
	if strings.Contains(output, "sk-ant-test-key-abcdef1234") {
		t.Error("full API key should NOT appear in status output")
	}
}

func TestConnect_UnknownProvider(t *testing.T) {
	cli.ColorEnabled = false

	r, _, errOut := newTestREPL("/connect gemini\n/quit\n")
	r.Run()
	output := errOut.String()

	if !strings.Contains(output, "Unknown provider") {
		t.Errorf("expected 'Unknown provider' error, got: %s", output)
	}
	if !strings.Contains(output, "anthropic, openai, ollama") {
		t.Errorf("expected supported provider list, got: %s", output)
	}
}

func TestConnect_MaskAPIKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"sk-ant-api03-abcdefgh1234", "...1234"},
		{"abcd", "****"},
		{"ab", "****"},
		{"", "****"},
		{"12345", "...2345"},
	}

	for _, tt := range tests {
		got := maskAPIKey(tt.input)
		if got != tt.want {
			t.Errorf("maskAPIKey(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestConnect_Ollama_SavesConfig(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	cli.ColorEnabled = false

	// We can't actually test the full flow because validateProvider makes a
	// network call. Instead, test that the global config save/load works for ollama.
	gc := &config.GlobalConfig{
		LLM: &config.GlobalLLMConfig{
			Provider: "ollama",
			Model:    "llama3",
			BaseURL:  "http://localhost:11434",
		},
	}
	if err := config.SaveGlobalConfig(gc); err != nil {
		t.Fatal(err)
	}

	// Verify it was saved.
	loaded, err := config.LoadGlobalConfig()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.LLM == nil {
		t.Fatal("expected LLM config")
	}
	if loaded.LLM.Provider != "ollama" {
		t.Errorf("provider = %q, want %q", loaded.LLM.Provider, "ollama")
	}
	if loaded.LLM.BaseURL != "http://localhost:11434" {
		t.Errorf("base_url = %q, want %q", loaded.LLM.BaseURL, "http://localhost:11434")
	}
	if loaded.LLM.APIKey != "" {
		t.Errorf("ollama should not have an API key, got %q", loaded.LLM.APIKey)
	}
}

func TestConnect_ScanLine(t *testing.T) {
	cli.ColorEnabled = false
	r, _, _ := newTestREPL("  hello world  \nignored\n")
	got, ok := r.scanLine()
	if !ok {
		t.Error("expected scanLine to succeed")
	}
	if got != "hello world" {
		t.Errorf("scanLine = %q, want %q", got, "hello world")
	}
}

func TestConnect_ScanLineEOF(t *testing.T) {
	cli.ColorEnabled = false
	r, _, _ := newTestREPL("")
	got, ok := r.scanLine()
	if ok {
		t.Error("expected scanLine to return false on EOF")
	}
	if got != "" {
		t.Errorf("scanLine on empty = %q, want %q", got, "")
	}
}

func TestConnect_APIKey_EmptyInput(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	cli.ColorEnabled = false

	// With the shared scanner, /connect anthropic reads the NEXT line from
	// the same scanner as Run(). Provide: command, empty key, quit.
	r, _, errOut := newTestREPL("/connect anthropic\n\n/quit\n")
	r.Run()

	if !strings.Contains(errOut.String(), "No API key provided") {
		t.Errorf("expected 'No API key provided' error, got errOut=%q", errOut.String())
	}
}

func TestConnect_HelpOrder(t *testing.T) {
	cli.ColorEnabled = false
	r, out, _ := newTestREPL("/help\n/quit\n")
	r.Run()
	output := out.String()

	if !strings.Contains(output, "/connect") {
		t.Error("expected /help output to list /connect")
	}

	// Search within the help listing section only (after "Available Commands")
	// to avoid false matches in the banner/tips area.
	helpStart := strings.Index(output, "Available Commands")
	if helpStart < 0 {
		t.Fatal("expected 'Available Commands' heading in output")
	}
	helpSection := output[helpStart:]

	// /connect should appear before /theme in the help listing.
	connectIdx := strings.Index(helpSection, "/connect")
	themeIdx := strings.Index(helpSection, "/theme")
	if connectIdx > themeIdx {
		t.Error("expected /connect to appear before /theme in help")
	}
}

func TestConnect_StatusAfterSave(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	cli.ColorEnabled = false

	// Save config then check status shows it.
	gc := &config.GlobalConfig{
		LLM: &config.GlobalLLMConfig{
			Provider: "openai",
			Model:    "gpt-4o",
			APIKey:   "sk-openai-xyz-9999",
		},
	}
	if err := config.SaveGlobalConfig(gc); err != nil {
		t.Fatal(err)
	}

	// Verify file exists at the right path.
	path := filepath.Join(tmpHome, ".human", "config.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("global config file not created: %v", err)
	}

	// Call status directly.
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	r := New("0.4.0-test",
		WithInput(strings.NewReader("")),
		WithOutput(out),
		WithErrOutput(errOut),
	)

	cmdConnect(r, []string{"status"})
	output := out.String()

	if !strings.Contains(output, "openai") {
		t.Errorf("expected 'openai' in status, got: %s", output)
	}
	if !strings.Contains(output, "gpt-4o") {
		t.Errorf("expected 'gpt-4o' in status, got: %s", output)
	}
	if !strings.Contains(output, "...9999") {
		t.Errorf("expected masked key '...9999', got: %s", output)
	}
}
