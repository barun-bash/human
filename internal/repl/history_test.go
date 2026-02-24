package repl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/cli"
)

func TestHistory_AddAndDedup(t *testing.T) {
	h := &History{}
	h.Add("/build")
	h.Add("/build") // consecutive duplicate — should be skipped
	h.Add("/check")
	h.Add("/build") // non-consecutive — should be kept

	entries := h.Entries()
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d: %v", len(entries), entries)
	}
	if entries[0] != "/build" || entries[1] != "/check" || entries[2] != "/build" {
		t.Errorf("unexpected entries: %v", entries)
	}
}

func TestHistory_AddEmpty(t *testing.T) {
	h := &History{}
	h.Add("")
	h.Add("   ")
	if len(h.Entries()) != 0 {
		t.Errorf("expected empty history, got %d entries", len(h.Entries()))
	}
}

func TestHistory_MaxLines(t *testing.T) {
	h := &History{}
	for i := 0; i < 600; i++ {
		h.Add(strings.Repeat("x", i+1)) // unique entries
	}
	if len(h.Entries()) != maxHistoryLines {
		t.Errorf("expected %d entries, got %d", maxHistoryLines, len(h.Entries()))
	}
}

func TestHistory_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history")

	// Create and save
	h1 := NewHistoryWithPath(path)
	h1.Add("/build")
	h1.Add("/check")
	h1.Add("/quit")
	h1.Save()

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("history file was not created")
	}

	// Load into new History
	h2 := NewHistoryWithPath(path)
	entries := h2.Entries()
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries after load, got %d", len(entries))
	}
	if entries[0] != "/build" || entries[1] != "/check" || entries[2] != "/quit" {
		t.Errorf("unexpected entries after load: %v", entries)
	}
}

func TestHistory_LoadMissingFile(t *testing.T) {
	h := NewHistoryWithPath("/nonexistent/path/history")
	// Should not panic or error
	if len(h.Entries()) != 0 {
		t.Errorf("expected empty history for missing file, got %d entries", len(h.Entries()))
	}
}

func TestHistory_Clear(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history")

	h := NewHistoryWithPath(path)
	h.Add("/build")
	h.Add("/check")
	h.Save()

	// Verify file exists.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("history file should exist")
	}

	h.Clear()
	if h.Len() != 0 {
		t.Errorf("expected 0 entries after clear, got %d", h.Len())
	}
	// File should be deleted.
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("history file should be deleted after clear")
	}
}

func TestHistory_Len(t *testing.T) {
	h := &History{}
	if h.Len() != 0 {
		t.Errorf("expected 0, got %d", h.Len())
	}
	h.Add("/build")
	h.Add("/check")
	if h.Len() != 2 {
		t.Errorf("expected 2, got %d", h.Len())
	}
}

func TestHistory_SaveCreatesDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "history")

	h := NewHistoryWithPath(path)
	h.Add("/test")
	h.Save()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("Save should have created the parent directory and file")
	}
}

// ── /history command tests ──

func TestHistoryCommand_ShowRecent(t *testing.T) {
	cli.ColorEnabled = false
	r, out, _ := newTestREPL("/build\n/check\n/history\n/quit\n")
	r.Run()
	output := out.String()

	// Should show the commands we typed (minus /quit and /history itself which are after).
	if !strings.Contains(output, "/build") {
		t.Errorf("expected /build in history output, got: %s", output)
	}
	if !strings.Contains(output, "/check") {
		t.Errorf("expected /check in history output, got: %s", output)
	}
}

func TestHistoryCommand_ReExecute(t *testing.T) {
	cli.ColorEnabled = false
	r, out, _ := newTestREPL("")
	r.history = &History{}
	r.history.Add("/version")
	r.history.Add("/pwd")

	cmdHistory(r, []string{"1"})
	output := out.String()

	// /history 1 should re-execute /version, which prints the version.
	if !strings.Contains(output, "Re-executing: /version") {
		t.Errorf("expected re-execute message for /version, got: %s", output)
	}
}

func TestHistoryCommand_ReExecuteOutOfRange(t *testing.T) {
	cli.ColorEnabled = false
	r, _, errOut := newTestREPL("")
	r.history = &History{}
	r.history.Add("/version")

	cmdHistory(r, []string{"5"})

	if !strings.Contains(errOut.String(), "does not exist") {
		t.Errorf("expected 'does not exist' error, got: %s", errOut.String())
	}
}

func TestHistoryCommand_Clear(t *testing.T) {
	cli.ColorEnabled = false
	r, out, _ := newTestREPL("/build\n/history clear\n/history\n/quit\n")
	r.Run()
	output := out.String()

	if !strings.Contains(output, "History cleared") {
		t.Errorf("expected 'History cleared' message, got: %s", output)
	}
	// After clearing, /history should show only /history clear and /history.
	// Actually the clear happens after /build and /history clear are in history.
	// Clear removes everything, then /history is added after. So it shows just /history.
}

func TestHistoryCommand_Empty(t *testing.T) {
	cli.ColorEnabled = false
	r, out, _ := newTestREPL("")
	// Replace history with a fresh empty one (the REPL constructor may load from disk).
	r.history = &History{}
	cmdHistory(r, nil)
	if !strings.Contains(out.String(), "No history") {
		t.Errorf("expected 'No history' message, got: %s", out.String())
	}
}

func TestHistoryCommand_HelpOrder(t *testing.T) {
	cli.ColorEnabled = false
	r, out, _ := newTestREPL("/help\n/quit\n")
	r.Run()
	output := out.String()

	helpStart := strings.Index(output, "Available Commands")
	if helpStart < 0 {
		t.Fatal("expected 'Available Commands' heading")
	}
	helpSection := output[helpStart:]

	historyIdx := strings.Index(helpSection, "/history")
	if historyIdx < 0 {
		t.Error("expected /history in help listing")
	}
}
