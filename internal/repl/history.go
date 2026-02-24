package repl

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const maxHistoryLines = 500

// History manages persistent command history.
type History struct {
	entries []string
	path    string
}

// NewHistory creates a History that loads from and saves to ~/.human/history.
func NewHistory() *History {
	h := &History{}
	h.path = historyPath()
	h.load()
	return h
}

// NewHistoryWithPath creates a History backed by a specific file path (for testing).
func NewHistoryWithPath(path string) *History {
	h := &History{path: path}
	h.load()
	return h
}

// Add appends a command to history, skipping consecutive duplicates.
func (h *History) Add(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	// Skip consecutive duplicates
	if len(h.entries) > 0 && h.entries[len(h.entries)-1] == line {
		return
	}
	h.entries = append(h.entries, line)

	// Truncate to max
	if len(h.entries) > maxHistoryLines {
		h.entries = h.entries[len(h.entries)-maxHistoryLines:]
	}
}

// Entries returns all history entries.
func (h *History) Entries() []string {
	return h.entries
}

// Save writes history to disk.
func (h *History) Save() {
	if h.path == "" {
		return
	}

	dir := filepath.Dir(h.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not create history directory: %v\n", err)
		return
	}

	f, err := os.Create(h.path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not save history: %v\n", err)
		return
	}
	defer f.Close()

	for _, entry := range h.entries {
		fmt.Fprintln(f, entry)
	}
}

// load reads history from disk.
func (h *History) load() {
	if h.path == "" {
		return
	}

	f, err := os.Open(h.path)
	if err != nil {
		return // File doesn't exist yet, that's fine
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			h.entries = append(h.entries, line)
		}
	}

	// Truncate if file was too long
	if len(h.entries) > maxHistoryLines {
		h.entries = h.entries[len(h.entries)-maxHistoryLines:]
	}
}

// historyPath returns the default history file path (~/.human/history).
func historyPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".human", "history")
}
