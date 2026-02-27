package editor

import (
	"strings"

	"github.com/barun-bash/human/internal/syntax"
)

// Completer provides context-aware autocomplete for the editor.
type Completer struct {
	items    []string
	selected int
	active   bool
	startCol int // column where completion prefix starts
}

// NewCompleter creates a new completer.
func NewCompleter() *Completer {
	return &Completer{}
}

// TriggerComplete computes completions for the current cursor position.
func (c *Completer) TriggerComplete(line []rune, cx int) {
	// Extract the word being typed.
	start := cx
	for start > 0 && !isDelimiter(line[start-1]) {
		start--
	}

	prefix := ""
	if start < cx {
		prefix = string(line[start:cx])
	}

	c.startCol = start
	c.items = nil
	c.selected = 0

	if prefix == "" {
		c.active = false
		return
	}

	// Get completions from syntax database.
	patterns := syntax.Autocomplete(prefix)
	seen := make(map[string]bool)
	for _, p := range patterns {
		// Extract first meaningful word from template.
		label := extractCompletionLabel(p.Template)
		if label != "" && !seen[label] {
			seen[label] = true
			c.items = append(c.items, label)
		}
	}

	// Add keyword completions.
	for _, kw := range keywordCompletions {
		if strings.HasPrefix(kw, strings.ToLower(prefix)) && !seen[kw] {
			seen[kw] = true
			c.items = append(c.items, kw)
		}
	}

	c.active = len(c.items) > 0
}

// Accept applies the selected completion.
func (c *Completer) Accept(buf *Buffer) {
	if !c.active || c.selected >= len(c.items) {
		return
	}

	item := c.items[c.selected]
	// Delete the prefix that was typed.
	for buf.cx > c.startCol {
		buf.Backspace()
	}
	// Insert the completion.
	for _, r := range item {
		buf.InsertChar(r)
	}
	c.active = false
}

// Next moves selection down.
func (c *Completer) Next() {
	if !c.active {
		return
	}
	c.selected++
	max := len(c.items)
	if max > 8 {
		max = 8
	}
	if c.selected >= max {
		c.selected = 0
	}
}

// Prev moves selection up.
func (c *Completer) Prev() {
	if !c.active {
		return
	}
	c.selected--
	if c.selected < 0 {
		max := len(c.items)
		if max > 8 {
			max = 8
		}
		c.selected = max - 1
	}
}

// Dismiss hides the popup.
func (c *Completer) Dismiss() {
	c.active = false
	c.items = nil
}

// Active returns whether completions are showing.
func (c *Completer) Active() bool { return c.active }

// Items returns current completion items.
func (c *Completer) Items() []string { return c.items }

// Selected returns the current selection index.
func (c *Completer) Selected() int { return c.selected }

func isDelimiter(r rune) bool {
	return r == ' ' || r == '\t' || r == ':' || r == ',' || r == '"' || r == '(' || r == ')'
}

func extractCompletionLabel(template string) string {
	// Templates look like "show a list of <data>". Extract the actionable phrase.
	parts := strings.Fields(template)
	if len(parts) == 0 {
		return ""
	}
	// Return the template itself (trimmed to reasonable length).
	label := template
	if len(label) > 50 {
		label = label[:50]
	}
	return label
}

// Common keyword completions for Human language.
var keywordCompletions = []string{
	"show", "has a", "has an", "has many", "belongs to",
	"which is", "clicking", "navigates to", "while loading",
	"there is a", "requires authentication",
	"frontend", "backend", "database", "deploy with",
	"required", "optional", "unique", "encrypted",
	"accepts", "returns", "validates", "sends", "triggers",
	"index", "restrict", "allow", "deny",
	"app", "data", "page", "component", "api", "build",
	"text", "number", "decimal", "boolean", "date", "email",
}
