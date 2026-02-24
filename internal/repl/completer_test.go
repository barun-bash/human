package repl

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/cli"
	"github.com/barun-bash/human/internal/llm/prompts"
)

func TestCompleteCommandName_Prefix(t *testing.T) {
	cli.ColorEnabled = false
	r, _, _ := newTestREPL("")
	matches := r.completeCommandName("/bu")
	sort.Strings(matches)
	if len(matches) != 1 || matches[0] != "/build" {
		t.Errorf("expected [/build], got %v", matches)
	}
}

func TestCompleteCommandName_MultipleMatches(t *testing.T) {
	cli.ColorEnabled = false
	r, _, _ := newTestREPL("")
	matches := r.completeCommandName("/c")
	sort.Strings(matches)

	// Should match /cd, /check, /clear, /config, /connect
	found := map[string]bool{}
	for _, m := range matches {
		found[m] = true
	}
	for _, expected := range []string{"/cd", "/check", "/clear", "/config", "/connect"} {
		if !found[expected] {
			t.Errorf("expected %s in matches, got %v", expected, matches)
		}
	}
}

func TestCompleteCommandName_NoMatch(t *testing.T) {
	cli.ColorEnabled = false
	r, _, _ := newTestREPL("")
	matches := r.completeCommandName("/xyz")
	if len(matches) != 0 {
		t.Errorf("expected no matches, got %v", matches)
	}
}

func TestCompleteCommandName_IncludesAliases(t *testing.T) {
	cli.ColorEnabled = false
	r, _, _ := newTestREPL("")
	matches := r.completeCommandName("/e")
	// Should include /edit, /examples, /exit (alias for /quit), and /e (alias for /edit)
	found := map[string]bool{}
	for _, m := range matches {
		found[m] = true
	}
	if !found["/edit"] {
		t.Errorf("expected /edit in matches, got %v", matches)
	}
	if !found["/examples"] {
		t.Errorf("expected /examples in matches, got %v", matches)
	}
}

func TestBuildCompleter_CommandCompletion(t *testing.T) {
	cli.ColorEnabled = false
	r, _, _ := newTestREPL("")
	completer := r.buildCompleter()

	// Typing "/bu" should complete to "/build".
	line := "/bu"
	results := completer(line, len(line))
	if len(results) != 1 || results[0] != "/build" {
		t.Errorf("expected [/build], got %v", results)
	}
}

func TestBuildCompleter_SubcommandCompletion(t *testing.T) {
	cli.ColorEnabled = false
	r, _, _ := newTestREPL("")
	completer := r.buildCompleter()

	// /instructions <tab> should show edit, init.
	line := "/instructions "
	results := completer(line, len(line))
	sort.Strings(results)
	expected := []string{"edit", "init"}
	if len(results) != 2 || results[0] != expected[0] || results[1] != expected[1] {
		t.Errorf("expected %v, got %v", expected, results)
	}
}

func TestBuildCompleter_ConnectProviders(t *testing.T) {
	cli.ColorEnabled = false
	r, _, _ := newTestREPL("")
	completer := r.buildCompleter()

	line := "/connect "
	results := completer(line, len(line))
	sort.Strings(results)
	// Should include status + all supported providers.
	found := map[string]bool{}
	for _, r := range results {
		found[r] = true
	}
	for _, expected := range []string{"anthropic", "openai", "ollama", "groq", "openrouter", "gemini", "custom", "status"} {
		if !found[expected] {
			t.Errorf("expected %s in results, got %v", expected, results)
		}
	}
}

func TestBuildCompleter_MCPSubcommands(t *testing.T) {
	cli.ColorEnabled = false
	r, _, _ := newTestREPL("")
	completer := r.buildCompleter()

	line := "/mcp "
	results := completer(line, len(line))
	sort.Strings(results)
	found := map[string]bool{}
	for _, r := range results {
		found[r] = true
	}
	for _, expected := range []string{"list", "add", "remove", "status"} {
		if !found[expected] {
			t.Errorf("expected %s in results", expected)
		}
	}
}

func TestBuildCompleter_MCPAddKnownServers(t *testing.T) {
	cli.ColorEnabled = false
	r, _, _ := newTestREPL("")
	completer := r.buildCompleter()

	line := "/mcp add "
	results := completer(line, len(line))
	found := map[string]bool{}
	for _, r := range results {
		found[r] = true
	}
	if !found["figma"] {
		t.Errorf("expected figma in known servers, got %v", results)
	}
	if !found["github"] {
		t.Errorf("expected github in known servers, got %v", results)
	}
}

func TestBuildCompleter_ThemeNames(t *testing.T) {
	cli.ColorEnabled = false
	r, _, _ := newTestREPL("")
	completer := r.buildCompleter()

	line := "/theme "
	results := completer(line, len(line))
	found := map[string]bool{}
	for _, r := range results {
		found[r] = true
	}
	if !found["list"] {
		t.Errorf("expected 'list' in theme completions, got %v", results)
	}
	// Check at least one theme name is present.
	themeNames := cli.ThemeNames()
	if !found[themeNames[0]] {
		t.Errorf("expected theme %s in completions, got %v", themeNames[0], results)
	}
}

func TestBuildCompleter_ConfigSet(t *testing.T) {
	cli.ColorEnabled = false
	r, _, _ := newTestREPL("")
	completer := r.buildCompleter()

	// /config <tab> → set
	line := "/config "
	results := completer(line, len(line))
	if len(results) != 1 || results[0] != "set" {
		t.Errorf("expected [set], got %v", results)
	}

	// /config set <tab> → keys
	line = "/config set "
	results = completer(line, len(line))
	sort.Strings(results)
	expected := []string{"animate", "auto_accept", "plan_mode", "theme"}
	if len(results) != 4 {
		t.Errorf("expected 4 config keys, got %v", results)
	}
	for i, e := range expected {
		if i < len(results) && results[i] != e {
			t.Errorf("results[%d] = %s, want %s", i, results[i], e)
		}
	}

	// /config set animate <tab> → on, off
	line = "/config set animate "
	results = completer(line, len(line))
	sort.Strings(results)
	if len(results) != 2 || results[0] != "off" || results[1] != "on" {
		t.Errorf("expected [off, on], got %v", results)
	}
}

func TestBuildCompleter_SuggestApply(t *testing.T) {
	cli.ColorEnabled = false
	r, _, _ := newTestREPL("")

	// No suggestions cached.
	completer := r.buildCompleter()
	line := "/suggest "
	results := completer(line, len(line))
	if len(results) != 1 || results[0] != "apply" {
		t.Errorf("expected [apply], got %v", results)
	}

	// Cache some suggestions.
	r.lastSuggestions = []prompts.Suggestion{
		{Category: "style", Text: "fix spacing"},
		{Category: "logic", Text: "add validation"},
	}
	line = "/suggest apply "
	results = completer(line, len(line))
	sort.Strings(results)
	found := map[string]bool{}
	for _, r := range results {
		found[r] = true
	}
	if !found["all"] {
		t.Error("expected 'all' in suggestions")
	}
	if !found["1"] || !found["2"] {
		t.Errorf("expected '1' and '2' in suggestions, got %v", results)
	}
}

func TestBuildCompleter_Deploy(t *testing.T) {
	cli.ColorEnabled = false
	r, _, _ := newTestREPL("")
	completer := r.buildCompleter()

	line := "/deploy "
	results := completer(line, len(line))
	if len(results) != 1 || results[0] != "--dry-run" {
		t.Errorf("expected [--dry-run], got %v", results)
	}
}

func TestBuildCompleter_EmptyLine(t *testing.T) {
	cli.ColorEnabled = false
	r, _, _ := newTestREPL("")
	completer := r.buildCompleter()

	results := completer("", 0)
	if len(results) == 0 {
		t.Error("expected command names for empty input")
	}
	// Should contain at least /help.
	found := false
	for _, r := range results {
		if r == "/help" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected /help in results for empty input")
	}
}

func TestBuildCompleter_UnknownCommand(t *testing.T) {
	cli.ColorEnabled = false
	r, _, _ := newTestREPL("")
	completer := r.buildCompleter()

	// Unknown command with args — no completions.
	line := "/unknown arg"
	results := completer(line, len(line))
	if len(results) != 0 {
		t.Errorf("expected no completions for unknown command, got %v", results)
	}
}

func TestBuildCompleter_AliasCompletion(t *testing.T) {
	cli.ColorEnabled = false
	r, _, _ := newTestREPL("")
	completer := r.buildCompleter()

	// /b is an alias for /build — subcommand completion should work.
	line := "/b "
	results := completer(line, len(line))
	if len(results) != 1 || results[0] != "--dry-run" {
		t.Errorf("expected [--dry-run] via alias, got %v", results)
	}
}

func TestCompleteFromList_Empty(t *testing.T) {
	results := completeFromList([]string{"edit", "init"}, "")
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %v", results)
	}
}

func TestCompleteFromList_Partial(t *testing.T) {
	results := completeFromList([]string{"edit", "init"}, "e")
	if len(results) != 1 || results[0] != "edit" {
		t.Errorf("expected [edit], got %v", results)
	}
}

func TestCompleteFromList_NoMatch(t *testing.T) {
	results := completeFromList([]string{"edit", "init"}, "xyz")
	if len(results) != 0 {
		t.Errorf("expected no matches, got %v", results)
	}
}

func TestCompleteFromList_CaseInsensitive(t *testing.T) {
	results := completeFromList([]string{"Edit", "Init"}, "e")
	if len(results) != 1 || results[0] != "Edit" {
		t.Errorf("expected [Edit], got %v", results)
	}
}

func TestCompleteFiles_HumanFiles(t *testing.T) {
	tmpDir := t.TempDir()
	// Create test files.
	os.WriteFile(filepath.Join(tmpDir, "app.human"), []byte("app Test\n"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "other.txt"), []byte("text\n"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "examples"), 0755)

	// Save and restore CWD.
	old, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(old)

	results := completeFiles("")
	found := map[string]bool{}
	for _, r := range results {
		found[r] = true
	}

	if !found["app.human"] {
		t.Errorf("expected app.human in results, got %v", results)
	}
	if found["other.txt"] {
		t.Error("should not include .txt files")
	}
	if !found["examples/"] {
		t.Errorf("expected examples/ directory in results, got %v", results)
	}
}

func TestCompleteFiles_WithPrefix(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "app.human"), []byte("app Test\n"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "api.human"), []byte("app API\n"), 0644)

	old, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(old)

	results := completeFiles("ap")
	if len(results) != 2 {
		t.Errorf("expected 2 matches for 'ap', got %v", results)
	}
	for _, r := range results {
		if !strings.HasPrefix(r, "ap") {
			t.Errorf("result %q doesn't start with 'ap'", r)
		}
	}
}

func TestCommandNames(t *testing.T) {
	cli.ColorEnabled = false
	r, _, _ := newTestREPL("")
	names := r.commandNames()
	if len(names) == 0 {
		t.Error("expected non-empty command names")
	}
	found := false
	for _, n := range names {
		if n == "/help" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected /help in command names")
	}
}
