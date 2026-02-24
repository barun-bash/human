package readline

import (
	"io"
	"os"
	"strings"
	"testing"
)

// ── partialWord tests ──

func TestPartialWord_Empty(t *testing.T) {
	if got := partialWord("", 0); got != "" {
		t.Errorf("partialWord(\"\", 0) = %q, want \"\"", got)
	}
}

func TestPartialWord_SingleWord(t *testing.T) {
	if got := partialWord("/build", 6); got != "/build" {
		t.Errorf("partialWord(\"/build\", 6) = %q, want \"/build\"", got)
	}
}

func TestPartialWord_SecondWord(t *testing.T) {
	if got := partialWord("/open ex", 8); got != "ex" {
		t.Errorf("partialWord(\"/open ex\", 8) = %q, want \"ex\"", got)
	}
}

func TestPartialWord_MidWord(t *testing.T) {
	if got := partialWord("/open examples/", 15); got != "examples/" {
		t.Errorf("partialWord(\"/open examples/\", 15) = %q, want \"examples/\"", got)
	}
}

func TestPartialWord_AtSpace(t *testing.T) {
	if got := partialWord("/open ", 6); got != "" {
		t.Errorf("partialWord(\"/open \", 6) = %q, want \"\"", got)
	}
}

// ── longestCommonPrefix tests ──

func TestLongestCommonPrefix_Empty(t *testing.T) {
	if got := longestCommonPrefix(nil); got != "" {
		t.Errorf("got %q, want \"\"", got)
	}
}

func TestLongestCommonPrefix_Single(t *testing.T) {
	if got := longestCommonPrefix([]string{"/build"}); got != "/build" {
		t.Errorf("got %q, want \"/build\"", got)
	}
}

func TestLongestCommonPrefix_Common(t *testing.T) {
	got := longestCommonPrefix([]string{"/build", "/browse"})
	if got != "/b" {
		t.Errorf("got %q, want \"/b\"", got)
	}
}

func TestLongestCommonPrefix_NoCommon(t *testing.T) {
	got := longestCommonPrefix([]string{"abc", "def"})
	if got != "" {
		t.Errorf("got %q, want \"\"", got)
	}
}

func TestLongestCommonPrefix_Full(t *testing.T) {
	got := longestCommonPrefix([]string{"edit", "edit"})
	if got != "edit" {
		t.Errorf("got %q, want \"edit\"", got)
	}
}

// ── visibleWidth tests ──

func TestVisibleWidth_Plain(t *testing.T) {
	if got := visibleWidth("hello"); got != 5 {
		t.Errorf("got %d, want 5", got)
	}
}

func TestVisibleWidth_WithANSI(t *testing.T) {
	// "\033[1;34mhello\033[0m" = bold blue "hello" + reset
	s := "\033[1;34mhello\033[0m"
	if got := visibleWidth(s); got != 5 {
		t.Errorf("got %d, want 5", got)
	}
}

func TestVisibleWidth_Empty(t *testing.T) {
	if got := visibleWidth(""); got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}

func TestVisibleWidth_OnlyANSI(t *testing.T) {
	if got := visibleWidth("\033[1m\033[0m"); got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}

// ── Line buffer operations ──

func TestInsertChar(t *testing.T) {
	rl := &Instance{out: io.Discard}
	rl.buf = []rune("hllo")
	rl.pos = 1
	rl.insertChar('e')
	if got := string(rl.buf); got != "hello" {
		t.Errorf("buf = %q, want \"hello\"", got)
	}
	if rl.pos != 2 {
		t.Errorf("pos = %d, want 2", rl.pos)
	}
}

func TestInsertChar_AtEnd(t *testing.T) {
	rl := &Instance{out: io.Discard}
	rl.buf = []rune("hell")
	rl.pos = 4
	rl.insertChar('o')
	if got := string(rl.buf); got != "hello" {
		t.Errorf("buf = %q, want \"hello\"", got)
	}
	if rl.pos != 5 {
		t.Errorf("pos = %d, want 5", rl.pos)
	}
}

func TestBackspace(t *testing.T) {
	rl := &Instance{out: io.Discard}
	rl.buf = []rune("hello")
	rl.pos = 5
	rl.backspace()
	if got := string(rl.buf); got != "hell" {
		t.Errorf("buf = %q, want \"hell\"", got)
	}
	if rl.pos != 4 {
		t.Errorf("pos = %d, want 4", rl.pos)
	}
}

func TestBackspace_AtStart(t *testing.T) {
	rl := &Instance{out: io.Discard}
	rl.buf = []rune("hello")
	rl.pos = 0
	rl.backspace()
	if got := string(rl.buf); got != "hello" {
		t.Errorf("buf should be unchanged, got %q", got)
	}
}

func TestDeleteChar(t *testing.T) {
	rl := &Instance{out: io.Discard}
	rl.buf = []rune("heello")
	rl.pos = 2
	rl.deleteChar()
	if got := string(rl.buf); got != "hello" {
		t.Errorf("buf = %q, want \"hello\"", got)
	}
}

func TestDeleteChar_AtEnd(t *testing.T) {
	rl := &Instance{out: io.Discard}
	rl.buf = []rune("hello")
	rl.pos = 5
	rl.deleteChar() // should be no-op
	if got := string(rl.buf); got != "hello" {
		t.Errorf("buf should be unchanged, got %q", got)
	}
}

func TestKillToEnd(t *testing.T) {
	rl := &Instance{out: io.Discard}
	rl.buf = []rune("hello world")
	rl.pos = 5
	rl.killToEnd()
	if got := string(rl.buf); got != "hello" {
		t.Errorf("buf = %q, want \"hello\"", got)
	}
}

func TestKillToStart(t *testing.T) {
	rl := &Instance{out: io.Discard}
	rl.buf = []rune("hello world")
	rl.pos = 6
	rl.killToStart()
	if got := string(rl.buf); got != "world" {
		t.Errorf("buf = %q, want \"world\"", got)
	}
	if rl.pos != 0 {
		t.Errorf("pos = %d, want 0", rl.pos)
	}
}

func TestDeleteWordBackward(t *testing.T) {
	rl := &Instance{out: io.Discard}
	rl.buf = []rune("/open examples/task")
	rl.pos = 19
	rl.deleteWordBackward()
	if got := string(rl.buf); got != "/open " {
		t.Errorf("buf = %q, want \"/open \"", got)
	}
}

func TestDeleteWordBackward_MultipleSpaces(t *testing.T) {
	rl := &Instance{out: io.Discard}
	rl.buf = []rune("hello   world")
	rl.pos = 8 // at 'w'
	// First delete spaces, then "hello"
	rl.deleteWordBackward()
	if got := string(rl.buf); got != "world" {
		t.Errorf("buf = %q, want \"world\"", got)
	}
}

// ── Cursor movement ──

func TestMoveCursorLeft(t *testing.T) {
	rl := &Instance{out: io.Discard}
	rl.buf = []rune("hello")
	rl.pos = 3
	rl.moveCursorLeft()
	if rl.pos != 2 {
		t.Errorf("pos = %d, want 2", rl.pos)
	}
}

func TestMoveCursorLeft_AtStart(t *testing.T) {
	rl := &Instance{out: io.Discard}
	rl.buf = []rune("hello")
	rl.pos = 0
	rl.moveCursorLeft()
	if rl.pos != 0 {
		t.Errorf("pos = %d, want 0", rl.pos)
	}
}

func TestMoveCursorRight(t *testing.T) {
	rl := &Instance{out: io.Discard}
	rl.buf = []rune("hello")
	rl.pos = 3
	rl.moveCursorRight()
	if rl.pos != 4 {
		t.Errorf("pos = %d, want 4", rl.pos)
	}
}

func TestMoveCursorRight_AtEnd(t *testing.T) {
	rl := &Instance{out: io.Discard}
	rl.buf = []rune("hello")
	rl.pos = 5
	rl.moveCursorRight()
	if rl.pos != 5 {
		t.Errorf("pos = %d, want 5", rl.pos)
	}
}

func TestMoveCursorHome(t *testing.T) {
	rl := &Instance{out: io.Discard}
	rl.buf = []rune("hello")
	rl.pos = 3
	rl.moveCursorHome()
	if rl.pos != 0 {
		t.Errorf("pos = %d, want 0", rl.pos)
	}
}

func TestMoveCursorEnd(t *testing.T) {
	rl := &Instance{out: io.Discard}
	rl.buf = []rune("hello")
	rl.pos = 2
	rl.moveCursorEnd()
	if rl.pos != 5 {
		t.Errorf("pos = %d, want 5", rl.pos)
	}
}

// ── History navigation ──

func TestHistoryUp(t *testing.T) {
	rl := &Instance{out: io.Discard}
	rl.history = []string{"/build", "/check", "/help"}
	rl.histIdx = 3 // at end
	rl.buf = []rune("/b")
	rl.pos = 2

	rl.historyUp()
	if got := string(rl.buf); got != "/help" {
		t.Errorf("buf = %q, want \"/help\"", got)
	}
	if rl.histIdx != 2 {
		t.Errorf("histIdx = %d, want 2", rl.histIdx)
	}

	rl.historyUp()
	if got := string(rl.buf); got != "/check" {
		t.Errorf("buf = %q, want \"/check\"", got)
	}

	rl.historyUp()
	if got := string(rl.buf); got != "/build" {
		t.Errorf("buf = %q, want \"/build\"", got)
	}

	// At top — should not change.
	rl.historyUp()
	if got := string(rl.buf); got != "/build" {
		t.Errorf("buf should stay \"/build\", got %q", got)
	}
}

func TestHistoryDown(t *testing.T) {
	rl := &Instance{out: io.Discard}
	rl.history = []string{"/build", "/check"}
	rl.histIdx = 2
	rl.buf = []rune("current")
	rl.pos = 7

	// Go up twice.
	rl.historyUp()
	rl.historyUp()
	if got := string(rl.buf); got != "/build" {
		t.Errorf("buf = %q, want \"/build\"", got)
	}

	// Go back down.
	rl.historyDown()
	if got := string(rl.buf); got != "/check" {
		t.Errorf("buf = %q, want \"/check\"", got)
	}

	// Down to restored saved line.
	rl.historyDown()
	if got := string(rl.buf); got != "current" {
		t.Errorf("buf = %q, want \"current\" (restored)", got)
	}

	// Already at bottom — no-op.
	rl.historyDown()
	if got := string(rl.buf); got != "current" {
		t.Errorf("buf = %q, want \"current\"", got)
	}
}

func TestHistoryUp_EmptyHistory(t *testing.T) {
	rl := &Instance{out: io.Discard}
	rl.history = nil
	rl.histIdx = 0
	rl.buf = []rune("test")
	rl.pos = 4

	rl.historyUp()
	if got := string(rl.buf); got != "test" {
		t.Errorf("buf should be unchanged, got %q", got)
	}
}

// ── Tab completion ──

func TestHandleTab_NoCompleter(t *testing.T) {
	rl := &Instance{out: io.Discard}
	rl.buf = []rune("/b")
	rl.pos = 2
	rl.handleTab() // should not panic
}

func TestHandleTab_SingleMatch(t *testing.T) {
	rl := &Instance{out: io.Discard}
	rl.complete = func(line string, pos int) []string {
		return []string{"/build"}
	}
	rl.buf = []rune("/b")
	rl.pos = 2

	rl.handleTab()
	if got := string(rl.buf); got != "/build" {
		t.Errorf("buf = %q, want \"/build\"", got)
	}
}

func TestHandleTab_CommonPrefix(t *testing.T) {
	rl := &Instance{out: io.Discard}
	rl.complete = func(line string, pos int) []string {
		return []string{"/connect", "/config"}
	}
	rl.buf = []rune("/c")
	rl.pos = 2

	rl.handleTab()
	if got := string(rl.buf); got != "/con" {
		t.Errorf("buf = %q, want \"/con\"", got)
	}
}

func TestHandleTab_NoMatch(t *testing.T) {
	rl := &Instance{out: io.Discard}
	rl.complete = func(line string, pos int) []string {
		return nil
	}
	rl.buf = []rune("/xyz")
	rl.pos = 4

	rl.handleTab()
	if got := string(rl.buf); got != "/xyz" {
		t.Errorf("buf should be unchanged, got %q", got)
	}
}

// ── Non-TTY fallback ──

func TestReadLine_NonTTY(t *testing.T) {
	r, w, _ := pipeFile(t)
	defer r.Close()

	rl := New(r, io.Discard)
	if rl.IsTTY() {
		t.Fatal("pipe should not be a TTY")
	}

	// Write input and close.
	go func() {
		w.WriteString("/build\n")
		w.Close()
	}()

	line, err := rl.ReadLine()
	if err != nil {
		t.Fatalf("ReadLine error: %v", err)
	}
	if line != "/build" {
		t.Errorf("got %q, want \"/build\"", line)
	}
}

func TestReadLine_NonTTY_EOF(t *testing.T) {
	r, w, _ := pipeFile(t)
	defer r.Close()

	rl := New(r, io.Discard)

	go func() {
		w.Close()
	}()

	_, err := rl.ReadLine()
	if err != io.EOF {
		t.Errorf("expected io.EOF, got %v", err)
	}
}

func TestReadSimpleLine_NonTTY(t *testing.T) {
	r, w, _ := pipeFile(t)
	defer r.Close()

	rl := New(r, io.Discard)

	go func() {
		w.WriteString("yes\n")
		w.Close()
	}()

	line, err := rl.ReadSimpleLine()
	if err != nil {
		t.Fatalf("ReadSimpleLine error: %v", err)
	}
	if line != "yes" {
		t.Errorf("got %q, want \"yes\"", line)
	}
}

// ── Escape sequence parsing ──

func TestCollectEscapeBytes_CSI(t *testing.T) {
	// CSI sequence: [A (up arrow)
	buf := []byte{'[', 'A', 'x'}
	seq := collectEscapeBytes(buf)
	if string(seq) != "[A" {
		t.Errorf("got %q, want \"[A\"", string(seq))
	}
}

func TestCollectEscapeBytes_CSI_Extended(t *testing.T) {
	// CSI sequence: [3~ (delete)
	buf := []byte{'[', '3', '~'}
	seq := collectEscapeBytes(buf)
	if string(seq) != "[3~" {
		t.Errorf("got %q, want \"[3~\"", string(seq))
	}
}

func TestCollectEscapeBytes_SS3(t *testing.T) {
	// SS3 sequence: OA (up arrow on some terminals)
	buf := []byte{'O', 'A'}
	seq := collectEscapeBytes(buf)
	if string(seq) != "OA" {
		t.Errorf("got %q, want \"OA\"", string(seq))
	}
}

// ── Apply completion ──

func TestApplyCompletion_FromPartial(t *testing.T) {
	rl := &Instance{out: io.Discard}
	rl.buf = []rune("/open exa")
	rl.pos = 9

	rl.applyCompletion("examples/")
	if got := string(rl.buf); got != "/open examples/" {
		t.Errorf("buf = %q, want \"/open examples/\"", got)
	}
}

func TestApplyCompletion_CommandName(t *testing.T) {
	rl := &Instance{out: io.Discard}
	rl.buf = []rune("/bu")
	rl.pos = 3

	rl.applyCompletion("/build")
	if got := string(rl.buf); got != "/build" {
		t.Errorf("buf = %q, want \"/build\"", got)
	}
}

// ── SetLine ──

func TestSetLine(t *testing.T) {
	rl := &Instance{out: io.Discard}
	rl.buf = []rune("old")
	rl.pos = 3

	rl.setLine([]rune("new content"))
	if got := string(rl.buf); got != "new content" {
		t.Errorf("buf = %q, want \"new content\"", got)
	}
	if rl.pos != 11 {
		t.Errorf("pos = %d, want 11", rl.pos)
	}
}

// ── Helpers ──

// pipeFile creates an os.File-based pipe for testing non-TTY readline.
func pipeFile(t *testing.T) (*os.File, *os.File, error) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe failed: %v", err)
	}
	return r, w, nil
}

// Ensure string import is used
var _ = strings.HasPrefix
