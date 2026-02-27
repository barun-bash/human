package editor

import (
	"os"
	"strings"
	"testing"
)

// TestSmokeTaskflow loads the real taskflow example and exercises all editor
// components against it: buffer, highlighting, annotations, validation, and
// rendering helpers.
func TestSmokeTaskflow(t *testing.T) {
	// Load the real example file.
	path := "../../examples/taskflow/app.human"
	content, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("skipping smoke test: %v", err)
	}

	src := string(content)

	// ── Buffer ──
	buf := NewBuffer(src)

	if buf.LineCount() < 300 {
		t.Errorf("expected 300+ lines in taskflow, got %d", buf.LineCount())
	}

	// First line.
	first := string(buf.Line(0))
	if first != "app TaskFlow is a web application" {
		t.Errorf("unexpected first line: %q", first)
	}

	// Content round-trip.
	roundTrip := buf.Content()
	if roundTrip != src {
		// Allow trailing newline difference.
		if strings.TrimRight(roundTrip, "\n") != strings.TrimRight(src, "\n") {
			t.Error("buffer content round-trip mismatch")
		}
	}

	// Navigate to a known line.
	buf.SetCursor(0, 82) // "data Task:" line (0-indexed 82 = line 83)
	_, cy := buf.Cursor()
	if cy != 82 {
		t.Errorf("expected cursor at line 82, got %d", cy)
	}

	// ── Syntax Highlighting ──

	// Comment line.
	commentLine := buf.Line(2) // "# ───..."
	tokens := HighlightLine(commentLine)
	if len(tokens) == 0 || tokens[0].Type != tokComment {
		t.Error("expected comment token for line 3")
	}

	// Section header: "data User:" (line 72, 0-indexed)
	dataLine := buf.Line(72)
	dataTokens := HighlightLine(dataLine)
	foundSection := false
	for _, tok := range dataTokens {
		if tok.Type == tokSection {
			foundSection = true
			break
		}
	}
	if !foundSection {
		t.Errorf("expected section token in 'data User:', got tokens: %v for line %q", dataTokens, string(dataLine))
	}

	// String literal: line with "Get Started".
	stringLine := buf.Line(21) // 'show a "Get Started" button'
	stringTokens := HighlightLine(stringLine)
	foundString := false
	for _, tok := range stringTokens {
		if tok.Type == tokString {
			foundString = true
			break
		}
	}
	if !foundString {
		t.Errorf("expected string token in %q", string(stringLine))
	}

	// Keyword: "has a name which is text" (line 73)
	kwLine := buf.Line(73)
	kwTokens := HighlightLine(kwLine)
	foundKW := false
	foundType := false
	for _, tok := range kwTokens {
		if tok.Type == tokKeyword {
			foundKW = true
		}
		if tok.Type == tokType {
			foundType = true
		}
	}
	if !foundKW {
		t.Errorf("expected keyword token in %q", string(kwLine))
	}
	if !foundType {
		t.Errorf("expected type token in %q", string(kwLine))
	}

	// ── RenderHighlighted produces ANSI ──
	highlighted := RenderHighlighted(dataLine)
	if !strings.Contains(highlighted, "\033[") {
		t.Error("expected ANSI codes in highlighted data line")
	}
	if !strings.Contains(highlighted, "data") {
		t.Error("expected 'data' in highlighted output")
	}

	// ── Annotations ──
	tests := []struct {
		lineIdx  int
		ctx      string
		expected string
	}{
		{0, "", "app"},           // "app TaskFlow is a web application"
		{19, "", "page"},         // "page Home:"
		{27, "page", "page"},     // "  show a greeting..." inside page block
		{72, "", "data"},         // "data User:"
		{82, "", "data"},         // "data Task:"
		{73, "data", "data"},     // "  has a name..." inherits "data"
		{341, "", "build"},       // "build with:"
		{342, "build", "build"},  // "  frontend using React..." inherits "build"
	}
	for _, tt := range tests {
		if tt.lineIdx >= buf.LineCount() {
			continue
		}
		got := LineCategory(buf.Line(tt.lineIdx), tt.ctx)
		if got != tt.expected {
			t.Errorf("LineCategory(line %d %q, %q) = %q, want %q",
				tt.lineIdx, string(buf.Line(tt.lineIdx)), tt.ctx, got, tt.expected)
		}
	}

	// ── Block context resolution ──
	// Simulate the renderer's block context walk.
	blockCtx := ""
	categorySeen := map[string]bool{}
	for i := 0; i < buf.LineCount(); i++ {
		cat := LineCategory(buf.Line(i), blockCtx)
		if cat != "" {
			blockCtx = cat
			categorySeen[cat] = true
		}
	}

	// The file should have all major categories.
	for _, expected := range []string{"app", "page", "data", "build"} {
		if !categorySeen[expected] {
			t.Errorf("expected category %q in block context walk", expected)
		}
	}

	// ── Editing operations ──

	// Insert + undo round-trip.
	buf.SetCursor(0, 0)
	originalLine0 := string(buf.Line(0))
	buf.InsertChar('X')
	if string(buf.Line(0)) != "X"+originalLine0 {
		t.Error("insert at start of file failed")
	}
	buf.Undo()
	if string(buf.Line(0)) != originalLine0 {
		t.Error("undo after insert failed")
	}

	// Newline + undo.
	lineCount := buf.LineCount()
	buf.SetCursor(3, 0) // start of "app TaskFlow..."
	buf.NewLine()
	if buf.LineCount() != lineCount+1 {
		t.Error("newline didn't add a line")
	}
	buf.Undo()
	if buf.LineCount() != lineCount {
		t.Error("undo newline didn't restore line count")
	}

	// ── Completer ──
	comp := NewCompleter()

	// Complete "sho" → should find "show".
	comp.TriggerComplete([]rune("sho"), 3)
	if !comp.Active() {
		t.Error("completer should be active for 'sho'")
	}
	foundShow := false
	for _, item := range comp.Items() {
		if strings.Contains(item, "show") {
			foundShow = true
		}
	}
	if !foundShow {
		t.Errorf("expected 'show' completion, got %v", comp.Items())
	}

	// Complete "da" → should find "data".
	comp.TriggerComplete([]rune("da"), 2)
	if !comp.Active() {
		t.Error("completer should be active for 'da'")
	}
	foundData := false
	for _, item := range comp.Items() {
		if strings.Contains(item, "data") {
			foundData = true
		}
	}
	if !foundData {
		t.Errorf("expected 'data' completion, got %v", comp.Items())
	}

	// ── Render helpers ──
	ansiStr := "\033[31mhello\033[0m world"
	if visLen(ansiStr) != 11 { // "hello world"
		t.Errorf("visLen(%q) = %d, want 11", ansiStr, visLen(ansiStr))
	}

	truncated := truncateVisible("hello world", 5)
	if !strings.Contains(truncated, "hello") {
		t.Errorf("truncateVisible failed: %q", truncated)
	}

	t.Logf("Smoke test passed: %d lines, %d categories detected, highlight/annotations/complete all working", buf.LineCount(), len(categorySeen))
}

// TestSmokeHighlightCoverage verifies every major line type in taskflow gets highlighted.
func TestSmokeHighlightCoverage(t *testing.T) {
	path := "../../examples/taskflow/app.human"
	content, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("skipping: %v", err)
	}

	buf := NewBuffer(string(content))

	tokenTypeSeen := map[tokenType]bool{}
	for i := 0; i < buf.LineCount(); i++ {
		tokens := HighlightLine(buf.Line(i))
		for _, tok := range tokens {
			tokenTypeSeen[tok.Type] = true
		}
	}

	expected := []tokenType{tokComment, tokSection, tokKeyword, tokType, tokString}
	for _, tt := range expected {
		if !tokenTypeSeen[tt] {
			t.Errorf("token type %d never seen in taskflow highlighting", tt)
		}
	}

	t.Logf("Token types seen: %d different types across %d lines", len(tokenTypeSeen), buf.LineCount())
}
