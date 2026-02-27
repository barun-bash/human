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

// ── NewLine edge case tests ──

func TestNewLineAtEndOfLine(t *testing.T) {
	b := NewBuffer("  hello world")
	b.SetCursor(13, 0) // end of line (len("  hello world") = 13)
	b.NewLine()

	if b.LineCount() != 2 {
		t.Fatalf("expected 2 lines, got %d", b.LineCount())
	}
	if string(b.Line(0)) != "  hello world" {
		t.Errorf("line 0 = %q, want %q", string(b.Line(0)), "  hello world")
	}
	// New line should have auto-indent (2 spaces).
	if string(b.Line(1)) != "  " {
		t.Errorf("line 1 = %q, want %q (auto-indent)", string(b.Line(1)), "  ")
	}
	cx, cy := b.Cursor()
	if cy != 1 || cx != 2 {
		t.Errorf("cursor = (%d, %d), want (2, 1)", cx, cy)
	}
}

func TestNewLineAtStartOfLine(t *testing.T) {
	b := NewBuffer("  hello world")
	b.SetCursor(0, 0) // position 0 — before indentation
	b.NewLine()

	if b.LineCount() != 2 {
		t.Fatalf("expected 2 lines, got %d", b.LineCount())
	}
	// Line 0 should be empty (before part).
	if string(b.Line(0)) != "" {
		t.Errorf("line 0 = %q, want empty", string(b.Line(0)))
	}
	// Line 1 should have original content UNCHANGED (no double-indent).
	if string(b.Line(1)) != "  hello world" {
		t.Errorf("line 1 = %q, want %q (no double-indent)", string(b.Line(1)), "  hello world")
	}
	// Cursor on line 1, col 0.
	cx, cy := b.Cursor()
	if cy != 1 || cx != 0 {
		t.Errorf("cursor = (%d, %d), want (0, 1)", cx, cy)
	}
}

func TestNewLineInMiddleOfLine(t *testing.T) {
	b := NewBuffer("  hello world")
	b.SetCursor(7, 0) // after "  hello" (position 7)
	b.NewLine()

	if b.LineCount() != 2 {
		t.Fatalf("expected 2 lines, got %d", b.LineCount())
	}
	if string(b.Line(0)) != "  hello" {
		t.Errorf("line 0 = %q, want %q", string(b.Line(0)), "  hello")
	}
	// After split: " world", with auto-indent prepended: "   world"
	if string(b.Line(1)) != "   world" {
		t.Errorf("line 1 = %q, want %q", string(b.Line(1)), "   world")
	}
	cx, cy := b.Cursor()
	if cy != 1 || cx != 2 {
		t.Errorf("cursor = (%d, %d), want (2, 1)", cx, cy)
	}
}

func TestNewLineAtStartThenType(t *testing.T) {
	// Simulates the user's bug: Enter at col 0, then type text.
	b := NewBuffer("  there is a date range picker")
	b.SetCursor(0, 0)
	b.NewLine()

	// Now type "helloworld" on the current line.
	for _, r := range "helloworld" {
		b.InsertChar(r)
	}

	if b.LineCount() != 2 {
		t.Fatalf("expected 2 lines, got %d", b.LineCount())
	}
	// Line 0: empty (the blank line created by Enter).
	if string(b.Line(0)) != "" {
		t.Errorf("line 0 = %q, want empty", string(b.Line(0)))
	}
	// Line 1: "helloworld" followed by original content.
	// Cursor was at col 0, so text goes before the indentation.
	expected := "helloworld  there is a date range picker"
	if string(b.Line(1)) != expected {
		t.Errorf("line 1 = %q, want %q", string(b.Line(1)), expected)
	}
}

func TestNewLineUndoPreservesAutoIndent(t *testing.T) {
	b := NewBuffer("  indented line")
	b.SetCursor(15, 0) // end of line
	b.NewLine()
	b.InsertChar('x')

	// Line 1 should be "  x" (auto-indent + typed char).
	if string(b.Line(1)) != "  x" {
		t.Errorf("line 1 = %q, want %q", string(b.Line(1)), "  x")
	}

	// Undo the char.
	b.Undo()
	if string(b.Line(1)) != "  " {
		t.Errorf("after undo char: line 1 = %q, want %q", string(b.Line(1)), "  ")
	}

	// Undo the newline.
	b.Undo()
	if b.LineCount() != 1 {
		t.Errorf("after undo newline: expected 1 line, got %d", b.LineCount())
	}
	if string(b.Line(0)) != "  indented line" {
		t.Errorf("after undo: line 0 = %q, want %q", string(b.Line(0)), "  indented line")
	}
}

// ── CR+LF input tests ──

func TestCRLFProducesSingleEnter(t *testing.T) {
	ir := &inputReader{
		byteCh: make(chan byte, 16),
	}
	// Send CR+LF (the pair some terminals send).
	ir.byteCh <- 13
	ir.byteCh <- 10

	key := ir.ReadKey()
	if key.Key != KeyEnter {
		t.Errorf("expected KeyEnter, got %v", key.Key)
	}

	// Should not produce a second KeyEnter — the LF was consumed.
	ir.byteCh <- 'a'
	key = ir.ReadKey()
	if key.Key != KeyChar || key.Rune != 'a' {
		t.Errorf("expected KeyChar 'a', got key=%v rune=%c", key.Key, key.Rune)
	}
}

func TestCRWithoutLF(t *testing.T) {
	ir := &inputReader{
		byteCh: make(chan byte, 16),
	}
	// Send CR followed by a different character.
	ir.byteCh <- 13
	ir.byteCh <- 'b'

	key := ir.ReadKey()
	if key.Key != KeyEnter {
		t.Errorf("expected KeyEnter from CR, got %v", key.Key)
	}

	// 'b' should be preserved via pushback.
	key = ir.ReadKey()
	if key.Key != KeyChar || key.Rune != 'b' {
		t.Errorf("expected KeyChar 'b' after CR pushback, got key=%v rune=%c", key.Key, key.Rune)
	}
}

func TestBareLF(t *testing.T) {
	ir := &inputReader{
		byteCh: make(chan byte, 16),
	}
	ir.byteCh <- 10 // bare LF

	key := ir.ReadKey()
	if key.Key != KeyEnter {
		t.Errorf("expected KeyEnter from LF, got %v", key.Key)
	}
}

// ── KillToEnd / KillToStart tests ──

func TestKillToEnd(t *testing.T) {
	b := NewBuffer("hello world")
	b.SetCursor(5, 0) // after "hello"
	b.KillToEnd()

	if string(b.Line(0)) != "hello" {
		t.Errorf("after KillToEnd: line = %q, want %q", string(b.Line(0)), "hello")
	}
	cx, _ := b.Cursor()
	if cx != 5 {
		t.Errorf("cursor col = %d, want 5", cx)
	}
}

func TestKillToStart(t *testing.T) {
	b := NewBuffer("hello world")
	b.SetCursor(5, 0) // after "hello"
	b.KillToStart()

	if string(b.Line(0)) != " world" {
		t.Errorf("after KillToStart: line = %q, want %q", string(b.Line(0)), " world")
	}
	cx, _ := b.Cursor()
	if cx != 0 {
		t.Errorf("cursor col = %d, want 0", cx)
	}
}

func TestKillToEndUndo(t *testing.T) {
	b := NewBuffer("hello world")
	b.SetCursor(5, 0)
	b.KillToEnd()
	b.Undo()

	if string(b.Line(0)) != "hello world" {
		t.Errorf("after undo KillToEnd: line = %q, want %q", string(b.Line(0)), "hello world")
	}
	cx, _ := b.Cursor()
	if cx != 5 {
		t.Errorf("cursor col after undo = %d, want 5", cx)
	}
}

func TestKillToStartUndo(t *testing.T) {
	b := NewBuffer("hello world")
	b.SetCursor(5, 0)
	b.KillToStart()
	b.Undo()

	if string(b.Line(0)) != "hello world" {
		t.Errorf("after undo KillToStart: line = %q, want %q", string(b.Line(0)), "hello world")
	}
	cx, _ := b.Cursor()
	if cx != 5 {
		t.Errorf("cursor col after undo = %d, want 5", cx)
	}
}

func TestKillToEndAtEOL(t *testing.T) {
	b := NewBuffer("hello")
	b.SetCursor(5, 0) // at end of line
	b.KillToEnd()

	// Should be a no-op.
	if string(b.Line(0)) != "hello" {
		t.Errorf("KillToEnd at EOL should be no-op, got %q", string(b.Line(0)))
	}
}

func TestKillToStartAtBOL(t *testing.T) {
	b := NewBuffer("hello")
	b.SetCursor(0, 0) // at start of line
	b.KillToStart()

	// Should be a no-op.
	if string(b.Line(0)) != "hello" {
		t.Errorf("KillToStart at BOL should be no-op, got %q", string(b.Line(0)))
	}
}

// ── Dynamic gutter width test ──

func TestDynamicGutterWidth(t *testing.T) {
	// Build a buffer with > 9999 lines to trigger wider gutter.
	lines := make([]string, 10001)
	for i := range lines {
		lines[i] = "x"
	}
	buf := NewBuffer(strings.Join(lines, "\n"))

	var out strings.Builder
	r := NewRenderer(&out, 80, 24)
	r.updateGutter(buf.LineCount())

	// 10001 lines = 5 digits → gutterWidth should be 7 (5 + 2).
	if r.digitWidth != 5 {
		t.Errorf("digitWidth = %d, want 5 for %d lines", r.digitWidth, buf.LineCount())
	}
	if r.gutterWidth != 7 {
		t.Errorf("gutterWidth = %d, want 7", r.gutterWidth)
	}

	// Small file: 100 lines → digitWidth=4 (minimum), gutterWidth=6.
	r2 := NewRenderer(&out, 80, 24)
	r2.updateGutter(100)
	if r2.digitWidth != 4 {
		t.Errorf("digitWidth = %d, want 4 for 100 lines", r2.digitWidth)
	}
	if r2.gutterWidth != 6 {
		t.Errorf("gutterWidth = %d, want 6", r2.gutterWidth)
	}
}
