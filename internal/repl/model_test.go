package repl

import (
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/cli"
	"github.com/barun-bash/human/internal/config"
)

func TestModel_NoProvider(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cli.ColorEnabled = false

	r, _, errOut := newTestREPL("")
	cmdModel(r, nil)

	if !strings.Contains(errOut.String(), "No LLM provider configured") {
		t.Errorf("expected no-provider error, got: %s", errOut.String())
	}
}

func TestModel_ShowCurrent(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	cli.ColorEnabled = false

	gc := &config.GlobalConfig{
		LLM: &config.GlobalLLMConfig{
			Provider: "anthropic",
			Model:    "claude-sonnet-4-20250514",
			APIKey:   "sk-test",
		},
	}
	config.SaveGlobalConfig(gc)

	r, out, _ := newTestREPL("")
	cmdModel(r, nil)

	if !strings.Contains(out.String(), "claude-sonnet-4-20250514") {
		t.Errorf("expected current model in output, got: %s", out.String())
	}
	if !strings.Contains(out.String(), "anthropic") {
		t.Errorf("expected provider name in output, got: %s", out.String())
	}
}

func TestModel_Switch(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	cli.ColorEnabled = false

	gc := &config.GlobalConfig{
		LLM: &config.GlobalLLMConfig{
			Provider: "anthropic",
			Model:    "claude-sonnet-4-20250514",
			APIKey:   "sk-test",
		},
	}
	config.SaveGlobalConfig(gc)

	r, out, _ := newTestREPL("")
	cmdModel(r, []string{"claude-opus-4-20250514"})

	if !strings.Contains(out.String(), "Model set to claude-opus-4-20250514") {
		t.Errorf("expected success message, got: %s", out.String())
	}

	// Verify it was saved.
	loaded, _ := config.LoadGlobalConfig()
	if loaded.LLM.Model != "claude-opus-4-20250514" {
		t.Errorf("model = %q, want claude-opus-4-20250514", loaded.LLM.Model)
	}
}

func TestModel_List(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	cli.ColorEnabled = false

	gc := &config.GlobalConfig{
		LLM: &config.GlobalLLMConfig{
			Provider: "openai",
			Model:    "gpt-4o",
			APIKey:   "sk-test",
		},
	}
	config.SaveGlobalConfig(gc)

	r, out, _ := newTestREPL("")
	cmdModel(r, []string{"list"})
	output := out.String()

	if !strings.Contains(output, "gpt-4o") {
		t.Errorf("expected gpt-4o in model list, got: %s", output)
	}
	if !strings.Contains(output, "gpt-4o-mini") {
		t.Errorf("expected gpt-4o-mini in model list, got: %s", output)
	}
}

func TestModel_HelpOrder(t *testing.T) {
	cli.ColorEnabled = false
	r, out, _ := newTestREPL("/help\n/quit\n")
	r.Run()
	output := out.String()

	helpStart := strings.Index(output, "Available Commands")
	if helpStart < 0 {
		t.Fatal("expected 'Available Commands' heading")
	}
	helpSection := output[helpStart:]

	modelIdx := strings.Index(helpSection, "/model")
	if modelIdx < 0 {
		t.Error("expected /model in help listing")
	}
}
