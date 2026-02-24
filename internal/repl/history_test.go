package repl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
