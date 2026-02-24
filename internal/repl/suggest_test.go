package repl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/cli"
	"github.com/barun-bash/human/internal/llm/prompts"
)

func TestSuggest_RequiresProject(t *testing.T) {
	cli.ColorEnabled = false
	r, _, errOut := newTestREPL("/suggest\n/quit\n")
	r.Run()
	output := errOut.String()

	if !strings.Contains(output, "No project loaded") {
		t.Errorf("expected 'No project loaded' error, got: %s", output)
	}
}

func TestSuggest_RequiresLLM(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	cli.ColorEnabled = false

	tmpFile := filepath.Join(t.TempDir(), "test.human")
	if err := os.WriteFile(tmpFile, []byte("app Test is a web application\n"), 0644); err != nil {
		t.Fatal(err)
	}

	r, _, errOut := newTestREPL("/open " + tmpFile + "\n/suggest\n/quit\n")
	r.Run()
	output := errOut.String()

	if !strings.Contains(output, "/connect") {
		t.Errorf("expected error pointing to /connect, got: %s", output)
	}
}

func TestSuggest_HelpListing(t *testing.T) {
	cli.ColorEnabled = false
	r, out, _ := newTestREPL("/help\n/quit\n")
	r.Run()
	output := out.String()

	if !strings.Contains(output, "/suggest") {
		t.Error("expected /help output to list /suggest")
	}
}

func TestSuggest_HelpOrder(t *testing.T) {
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

	suggestIdx := strings.Index(helpSection, "/suggest")
	buildIdx := strings.Index(helpSection, "/build")
	editIdx := strings.Index(helpSection, "/edit")

	if suggestIdx < 0 || buildIdx < 0 || editIdx < 0 {
		t.Fatal("expected /suggest, /edit, and /build in help output")
	}

	// /suggest should appear after /edit but before /build.
	if suggestIdx < editIdx {
		t.Error("expected /suggest to appear after /edit in help")
	}
	if suggestIdx > buildIdx {
		t.Error("expected /suggest to appear before /build in help")
	}
}

// ── Apply subcommand tests ──

func TestSuggestApply_NoSuggestions(t *testing.T) {
	cli.ColorEnabled = false

	tmpFile := filepath.Join(t.TempDir(), "test.human")
	if err := os.WriteFile(tmpFile, []byte("app Test is a web application\n"), 0644); err != nil {
		t.Fatal(err)
	}

	r, _, errOut := newTestREPL("/open " + tmpFile + "\n/suggest apply 1\n/quit\n")
	r.Run()
	output := errOut.String()

	if !strings.Contains(output, "No suggestions available") {
		t.Errorf("expected 'No suggestions available' error, got: %s", output)
	}
}

func TestSuggestApply_InvalidNumber(t *testing.T) {
	cli.ColorEnabled = false

	tmpFile := filepath.Join(t.TempDir(), "test.human")
	if err := os.WriteFile(tmpFile, []byte("app Test is a web application\n"), 0644); err != nil {
		t.Fatal(err)
	}

	r, _, errOut := newTestREPL("")
	r.setProject(tmpFile)
	r.lastSuggestions = []prompts.Suggestion{
		{Category: "security", Text: "Add rate limiting"},
	}

	cmdSuggest(r, []string{"apply", "5"})
	output := errOut.String()

	if !strings.Contains(output, "Invalid suggestion number") {
		t.Errorf("expected 'Invalid suggestion number' error, got: %s", output)
	}
}

func TestSuggestApply_InvalidText(t *testing.T) {
	cli.ColorEnabled = false

	tmpFile := filepath.Join(t.TempDir(), "test.human")
	if err := os.WriteFile(tmpFile, []byte("app Test is a web application\n"), 0644); err != nil {
		t.Fatal(err)
	}

	r, _, errOut := newTestREPL("")
	r.setProject(tmpFile)
	r.lastSuggestions = []prompts.Suggestion{
		{Category: "security", Text: "Add rate limiting"},
	}

	cmdSuggest(r, []string{"apply", "abc"})
	output := errOut.String()

	if !strings.Contains(output, "Invalid suggestion number") {
		t.Errorf("expected 'Invalid suggestion number' error, got: %s", output)
	}
}

func TestSuggestApply_NoArgs(t *testing.T) {
	cli.ColorEnabled = false

	tmpFile := filepath.Join(t.TempDir(), "test.human")
	if err := os.WriteFile(tmpFile, []byte("app Test is a web application\n"), 0644); err != nil {
		t.Fatal(err)
	}

	r, _, errOut := newTestREPL("")
	r.setProject(tmpFile)
	r.lastSuggestions = []prompts.Suggestion{
		{Category: "security", Text: "Add rate limiting"},
	}

	cmdSuggest(r, []string{"apply"})
	output := errOut.String()

	if !strings.Contains(output, "Usage") {
		t.Errorf("expected usage hint, got: %s", output)
	}
}

func TestSuggestApply_AllNoSuggestions(t *testing.T) {
	cli.ColorEnabled = false

	tmpFile := filepath.Join(t.TempDir(), "test.human")
	if err := os.WriteFile(tmpFile, []byte("app Test is a web application\n"), 0644); err != nil {
		t.Fatal(err)
	}

	r, _, errOut := newTestREPL("")
	r.setProject(tmpFile)
	// lastSuggestions is nil

	cmdSuggest(r, []string{"apply", "all"})
	output := errOut.String()

	if !strings.Contains(output, "No suggestions available") {
		t.Errorf("expected 'No suggestions available', got: %s", output)
	}
}

// ── Staleness clearing tests ──

func TestSuggest_ClearedOnOpen(t *testing.T) {
	cli.ColorEnabled = false

	tmpFile1 := filepath.Join(t.TempDir(), "a.human")
	tmpFile2 := filepath.Join(t.TempDir(), "b.human")
	os.WriteFile(tmpFile1, []byte("app A is a web application\n"), 0644)
	os.WriteFile(tmpFile2, []byte("app B is a web application\n"), 0644)

	r, _, _ := newTestREPL("")
	r.setProject(tmpFile1)
	r.lastSuggestions = []prompts.Suggestion{
		{Category: "test", Text: "something"},
	}

	// /open a different file should clear suggestions.
	cmdOpen(r, []string{tmpFile2})

	if len(r.lastSuggestions) != 0 {
		t.Error("expected suggestions to be cleared after /open")
	}
}

func TestSuggest_ClearedOnUndo(t *testing.T) {
	cli.ColorEnabled = false

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.human")
	os.WriteFile(tmpFile, []byte("app Modified is a web application\n"), 0644)

	// Create backup.
	backupDirPath := filepath.Join(tmpDir, ".human", "backup")
	os.MkdirAll(backupDirPath, 0755)
	os.WriteFile(filepath.Join(backupDirPath, "test.human.bak"), []byte("app Original is a web application\n"), 0644)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	r, _, _ := newTestREPL("")
	r.setProject(tmpFile)
	r.lastSuggestions = []prompts.Suggestion{
		{Category: "test", Text: "something"},
	}

	cmdUndo(r, nil)

	if len(r.lastSuggestions) != 0 {
		t.Error("expected suggestions to be cleared after /undo")
	}
}

// ── Direct function tests ──

func TestSuggestApply_ZeroIndex(t *testing.T) {
	cli.ColorEnabled = false

	tmpFile := filepath.Join(t.TempDir(), "test.human")
	if err := os.WriteFile(tmpFile, []byte("app Test is a web application\n"), 0644); err != nil {
		t.Fatal(err)
	}

	r, _, errOut := newTestREPL("")
	r.setProject(tmpFile)
	r.lastSuggestions = []prompts.Suggestion{
		{Category: "security", Text: "Add rate limiting"},
	}

	// Suggestion number 0 is invalid (1-indexed).
	cmdSuggest(r, []string{"apply", "0"})
	output := errOut.String()

	if !strings.Contains(output, "Invalid suggestion number") {
		t.Errorf("expected 'Invalid suggestion number' for 0, got: %s", output)
	}
}

func TestSuggestApply_NegativeNumber(t *testing.T) {
	cli.ColorEnabled = false

	tmpFile := filepath.Join(t.TempDir(), "test.human")
	if err := os.WriteFile(tmpFile, []byte("app Test is a web application\n"), 0644); err != nil {
		t.Fatal(err)
	}

	r, _, errOut := newTestREPL("")
	r.setProject(tmpFile)
	r.lastSuggestions = []prompts.Suggestion{
		{Category: "security", Text: "Add rate limiting"},
	}

	cmdSuggest(r, []string{"apply", "-1"})
	output := errOut.String()

	if !strings.Contains(output, "Invalid suggestion number") {
		t.Errorf("expected 'Invalid suggestion number' for -1, got: %s", output)
	}
}
