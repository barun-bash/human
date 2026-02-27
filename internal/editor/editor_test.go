package editor

import (
	"strings"
	"testing"
)

// ── Buffer tests ──

func TestNewBuffer(t *testing.T) {
	b := NewBuffer("hello\nworld\n")
	if b.LineCount() != 2 {
		t.Errorf("expected 2 lines, got %d", b.LineCount())
	}
	if string(b.Line(0)) != "hello" {
		t.Errorf("expected 'hello', got '%s'", string(b.Line(0)))
	}
	if string(b.Line(1)) != "world" {
		t.Errorf("expected 'world', got '%s'", string(b.Line(1)))
	}
}

func TestNewBufferEmpty(t *testing.T) {
	b := NewBuffer("")
	if b.LineCount() != 1 {
		t.Errorf("expected 1 line, got %d", b.LineCount())
	}
}

func TestInsertChar(t *testing.T) {
	b := NewBuffer("abc")
	b.SetCursor(1, 0) // between a and b
	b.InsertChar('X')
	if string(b.Line(0)) != "aXbc" {
		t.Errorf("expected 'aXbc', got '%s'", string(b.Line(0)))
	}
	cx, _ := b.Cursor()
	if cx != 2 {
		t.Errorf("expected cursor at 2, got %d", cx)
	}
}

func TestBackspace(t *testing.T) {
	b := NewBuffer("abc")
	b.SetCursor(2, 0)
	b.Backspace()
	if string(b.Line(0)) != "ac" {
		t.Errorf("expected 'ac', got '%s'", string(b.Line(0)))
	}
}

func TestBackspaceJoinLines(t *testing.T) {
	b := NewBuffer("hello\nworld")
	b.SetCursor(0, 1) // start of "world"
	b.Backspace()
	if b.LineCount() != 1 {
		t.Errorf("expected 1 line, got %d", b.LineCount())
	}
	if string(b.Line(0)) != "helloworld" {
		t.Errorf("expected 'helloworld', got '%s'", string(b.Line(0)))
	}
}

func TestNewLine(t *testing.T) {
	b := NewBuffer("hello world")
	b.SetCursor(5, 0) // after "hello"
	b.NewLine()
	if b.LineCount() != 2 {
		t.Errorf("expected 2 lines, got %d", b.LineCount())
	}
	if string(b.Line(0)) != "hello" {
		t.Errorf("expected 'hello', got '%s'", string(b.Line(0)))
	}
	if string(b.Line(1)) != " world" {
		t.Errorf("expected ' world', got '%s'", string(b.Line(1)))
	}
}

func TestAutoIndent(t *testing.T) {
	b := NewBuffer("  indented")
	b.SetCursor(10, 0) // end of line
	b.NewLine()
	if b.LineCount() != 2 {
		t.Errorf("expected 2 lines, got %d", b.LineCount())
	}
	if !strings.HasPrefix(string(b.Line(1)), "  ") {
		t.Errorf("expected auto-indent, got '%s'", string(b.Line(1)))
	}
}

func TestDeleteChar(t *testing.T) {
	b := NewBuffer("abc")
	b.SetCursor(1, 0)
	b.DeleteChar()
	if string(b.Line(0)) != "ac" {
		t.Errorf("expected 'ac', got '%s'", string(b.Line(0)))
	}
}

func TestDeleteCharJoinLines(t *testing.T) {
	b := NewBuffer("hello\nworld")
	b.SetCursor(5, 0) // end of "hello"
	b.DeleteChar()
	if b.LineCount() != 1 {
		t.Errorf("expected 1 line, got %d", b.LineCount())
	}
	if string(b.Line(0)) != "helloworld" {
		t.Errorf("expected 'helloworld', got '%s'", string(b.Line(0)))
	}
}

func TestUndo(t *testing.T) {
	b := NewBuffer("abc")
	b.SetCursor(3, 0)
	b.InsertChar('d')
	if string(b.Line(0)) != "abcd" {
		t.Errorf("expected 'abcd', got '%s'", string(b.Line(0)))
	}
	b.Undo()
	if string(b.Line(0)) != "abc" {
		t.Errorf("after undo expected 'abc', got '%s'", string(b.Line(0)))
	}
}

func TestRedo(t *testing.T) {
	b := NewBuffer("abc")
	b.SetCursor(3, 0)
	b.InsertChar('d')
	b.Undo()
	b.Redo()
	if string(b.Line(0)) != "abcd" {
		t.Errorf("after redo expected 'abcd', got '%s'", string(b.Line(0)))
	}
}

func TestUndoNewLine(t *testing.T) {
	b := NewBuffer("hello world")
	b.SetCursor(5, 0)
	b.NewLine()
	if b.LineCount() != 2 {
		t.Errorf("expected 2 lines after newline, got %d", b.LineCount())
	}
	b.Undo()
	if b.LineCount() != 1 {
		t.Errorf("expected 1 line after undo, got %d", b.LineCount())
	}
	if string(b.Line(0)) != "hello world" {
		t.Errorf("expected 'hello world' after undo, got '%s'", string(b.Line(0)))
	}
}

func TestUndoBackspaceJoin(t *testing.T) {
	b := NewBuffer("hello\nworld")
	b.SetCursor(0, 1)
	b.Backspace() // joins lines
	if b.LineCount() != 1 {
		t.Errorf("expected 1 line after join, got %d", b.LineCount())
	}
	b.Undo()
	if b.LineCount() != 2 {
		t.Errorf("expected 2 lines after undo, got %d", b.LineCount())
	}
}

func TestContent(t *testing.T) {
	b := NewBuffer("hello\nworld\n")
	got := b.Content()
	expected := "hello\nworld\n"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestNavigation(t *testing.T) {
	b := NewBuffer("abc\ndef\nghi")
	b.SetCursor(1, 1) // middle of "def"

	b.MoveUp()
	_, cy := b.Cursor()
	if cy != 0 {
		t.Errorf("expected row 0 after MoveUp, got %d", cy)
	}

	b.MoveDown()
	b.MoveDown()
	_, cy = b.Cursor()
	if cy != 2 {
		t.Errorf("expected row 2 after 2x MoveDown, got %d", cy)
	}

	b.Home()
	cx, _ := b.Cursor()
	if cx != 0 {
		t.Errorf("expected col 0 after Home, got %d", cx)
	}

	b.End()
	cx, _ = b.Cursor()
	if cx != 3 {
		t.Errorf("expected col 3 after End, got %d", cx)
	}
}

func TestPageUpDown(t *testing.T) {
	lines := make([]string, 50)
	for i := range lines {
		lines[i] = "line"
	}
	b := NewBuffer(strings.Join(lines, "\n"))
	b.SetCursor(0, 0)
	b.PageDown(20)
	_, cy := b.Cursor()
	if cy != 20 {
		t.Errorf("expected row 20 after PageDown(20), got %d", cy)
	}
	b.PageUp(10)
	_, cy = b.Cursor()
	if cy != 10 {
		t.Errorf("expected row 10 after PageUp(10), got %d", cy)
	}
}

func TestSetCursorClamp(t *testing.T) {
	b := NewBuffer("short\nlonger line")
	b.SetCursor(100, 100) // way out of bounds
	cx, cy := b.Cursor()
	if cy != 1 {
		t.Errorf("expected row clamped to 1, got %d", cy)
	}
	if cx != 11 { // length of "longer line"
		t.Errorf("expected col clamped to 11, got %d", cx)
	}
}

func TestInsertTab(t *testing.T) {
	b := NewBuffer("")
	b.InsertTab()
	if string(b.Line(0)) != "  " {
		t.Errorf("expected 2 spaces, got '%s'", string(b.Line(0)))
	}
}

// ── Highlight tests ──

func TestHighlightComment(t *testing.T) {
	tokens := HighlightLine([]rune("# this is a comment"))
	if len(tokens) != 1 || tokens[0].Type != tokComment {
		t.Errorf("expected single comment token, got %v", tokens)
	}
}

func TestHighlightString(t *testing.T) {
	tokens := HighlightLine([]rune(`show "hello world"`))
	found := false
	for _, tok := range tokens {
		if tok.Type == tokString {
			found = true
		}
	}
	if !found {
		t.Error("expected a string token")
	}
}

func TestHighlightSection(t *testing.T) {
	tokens := HighlightLine([]rune("data Task:"))
	found := false
	for _, tok := range tokens {
		if tok.Type == tokSection {
			found = true
		}
	}
	if !found {
		t.Error("expected a section token for 'data'")
	}
}

func TestHighlightKeyword(t *testing.T) {
	tokens := HighlightLine([]rune("  has a title which is text"))
	foundKW := false
	foundType := false
	for _, tok := range tokens {
		if tok.Type == tokKeyword {
			foundKW = true
		}
		if tok.Type == tokType {
			foundType = true
		}
	}
	if !foundKW {
		t.Error("expected keyword tokens")
	}
	if !foundType {
		t.Error("expected type token for 'text'")
	}
}

func TestRenderHighlighted(t *testing.T) {
	result := RenderHighlighted([]rune("# comment"))
	if !strings.Contains(result, "\033[") {
		t.Error("expected ANSI codes in highlighted output")
	}
	if !strings.Contains(result, "comment") {
		t.Error("expected 'comment' in output")
	}
}

// ── Annotation tests ──

func TestLineCategory(t *testing.T) {
	tests := []struct {
		line     string
		ctx      string
		expected string
	}{
		{"app TaskFlow is a task manager", "", "app"},
		{"data Task:", "", "data"},
		{"page Dashboard:", "", "page"},
		{"  has a title which is text", "data", "data"},
		{"build with:", "", "build"},
		{"frontend using React", "", "build"},
	}
	for _, tt := range tests {
		got := LineCategory([]rune(tt.line), tt.ctx)
		if got != tt.expected {
			t.Errorf("LineCategory(%q, %q) = %q, want %q", tt.line, tt.ctx, got, tt.expected)
		}
	}
}

// ── Render helper tests ──

func TestVisLen(t *testing.T) {
	plain := "hello"
	if visLen(plain) != 5 {
		t.Errorf("visLen(%q) = %d, want 5", plain, visLen(plain))
	}

	colored := "\033[31mhello\033[0m"
	if visLen(colored) != 5 {
		t.Errorf("visLen(%q) = %d, want 5", colored, visLen(colored))
	}
}

func TestTruncateVisible(t *testing.T) {
	s := "hello world"
	got := truncateVisible(s, 5)
	if !strings.Contains(got, "hello") {
		t.Errorf("expected 'hello' in truncated output, got %q", got)
	}
	if strings.Contains(got, "world") {
		t.Errorf("expected 'world' to be truncated, got %q", got)
	}
}

// ── Completer tests ──

func TestCompleterTrigger(t *testing.T) {
	c := NewCompleter()
	line := []rune("sho")
	c.TriggerComplete(line, 3)
	if !c.Active() {
		t.Error("expected completer to be active for 'sho'")
	}
	items := c.Items()
	if len(items) == 0 {
		t.Error("expected completion items for 'sho'")
	}
	// Should contain "show" as one of the items.
	found := false
	for _, item := range items {
		if strings.Contains(item, "show") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'show' in completions, got %v", items)
	}
}

func TestCompleterDismiss(t *testing.T) {
	c := NewCompleter()
	c.TriggerComplete([]rune("sho"), 3)
	c.Dismiss()
	if c.Active() {
		t.Error("expected completer to be inactive after dismiss")
	}
}

func TestCompleterNextPrev(t *testing.T) {
	c := NewCompleter()
	c.TriggerComplete([]rune("sho"), 3)
	if !c.Active() {
		t.Skip("no completions to test nav")
	}
	c.Next()
	if c.Selected() != 1 {
		t.Errorf("expected selected=1 after Next, got %d", c.Selected())
	}
	c.Prev()
	if c.Selected() != 0 {
		t.Errorf("expected selected=0 after Prev, got %d", c.Selected())
	}
}
