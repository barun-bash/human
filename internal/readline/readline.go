// Package readline provides interactive line editing with history navigation
// and tab completion for terminal input. It uses golang.org/x/term for raw
// terminal mode and falls back to bufio.Scanner when stdin is not a terminal.
package readline

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/term"
)

// CompleteFunc returns completions for the given line at the given cursor position.
type CompleteFunc func(line string, pos int) []string

// Instance is a readline instance that provides line editing, history
// navigation, and tab completion.
type Instance struct {
	in      *os.File
	out     io.Writer
	isTTY   bool
	scanner *bufio.Scanner // fallback for non-TTY

	// Configuration (set between ReadLine calls).
	prompt      string
	promptWidth int // visible width (excluding ANSI escapes)
	history     []string
	complete    CompleteFunc

	// Per-line state (reset each ReadLine call).
	buf       []rune // current line buffer
	pos       int    // cursor position in buf
	histIdx   int    // current history position
	savedLine []rune // saved input when browsing history
}

// New creates a readline instance. Pass os.Stdin as in for interactive use.
// If in is not a terminal, ReadLine falls back to simple line reading.
func New(in *os.File, out io.Writer) *Instance {
	rl := &Instance{
		in:  in,
		out: out,
	}
	if in != nil {
		rl.isTTY = term.IsTerminal(int(in.Fd()))
	}
	if !rl.isTTY {
		rl.scanner = bufio.NewScanner(in)
	}
	return rl
}

// SetPrompt sets the prompt displayed before user input.
func (rl *Instance) SetPrompt(p string) {
	rl.prompt = p
	rl.promptWidth = visibleWidth(p)
}

// SetHistory sets the history entries for up/down navigation.
func (rl *Instance) SetHistory(entries []string) {
	rl.history = entries
}

// SetCompleter sets the tab completion function.
func (rl *Instance) SetCompleter(fn CompleteFunc) {
	rl.complete = fn
}

// IsTTY returns whether the input is a terminal.
func (rl *Instance) IsTTY() bool {
	return rl.isTTY
}

// ReadLine reads a line with editing, history, and completion support.
// Returns the line (without newline) and any error. Returns io.EOF on Ctrl+D
// with an empty line.
func (rl *Instance) ReadLine() (string, error) {
	if !rl.isTTY {
		return rl.readSimple()
	}
	return rl.readRaw()
}

// ReadSimpleLine reads a line without completion (for sub-prompts like y/n).
// Still provides basic line editing if in TTY mode.
func (rl *Instance) ReadSimpleLine() (string, error) {
	return rl.readSimple()
}

// ── Simple (non-TTY) reading ──

func (rl *Instance) readSimple() (string, error) {
	if rl.scanner == nil {
		rl.scanner = bufio.NewScanner(rl.in)
	}
	if !rl.scanner.Scan() {
		if err := rl.scanner.Err(); err != nil {
			return "", err
		}
		return "", io.EOF
	}
	return rl.scanner.Text(), nil
}

// ── Raw terminal reading ──

func (rl *Instance) readRaw() (string, error) {
	// Enter raw mode.
	oldState, err := term.MakeRaw(int(rl.in.Fd()))
	if err != nil {
		// Fallback to simple reading.
		return rl.readSimple()
	}
	defer term.Restore(int(rl.in.Fd()), oldState)

	// Reset per-line state.
	rl.buf = nil
	rl.pos = 0
	rl.histIdx = len(rl.history)
	rl.savedLine = nil

	// Print prompt.
	fmt.Fprint(rl.out, rl.prompt)

	buf := make([]byte, 64)
	for {
		n, err := rl.in.Read(buf)
		if err != nil {
			return "", err
		}

		for i := 0; i < n; {
			b := buf[i]
			i++

			switch {
			case b == ctrlA: // Home
				rl.moveCursorHome()

			case b == ctrlB: // Left
				rl.moveCursorLeft()

			case b == ctrlC: // Cancel line
				fmt.Fprint(rl.out, "^C\r\n")
				return "", nil // empty line, not EOF

			case b == ctrlD: // EOF or delete
				if len(rl.buf) == 0 {
					fmt.Fprint(rl.out, "\r\n")
					return "", io.EOF
				}
				rl.deleteChar()

			case b == ctrlE: // End
				rl.moveCursorEnd()

			case b == ctrlF: // Right
				rl.moveCursorRight()

			case b == ctrlK: // Kill to end of line
				rl.killToEnd()

			case b == ctrlL: // Clear screen
				fmt.Fprint(rl.out, "\033[2J\033[H") // clear + home
				rl.redraw()

			case b == ctrlU: // Kill to start of line
				rl.killToStart()

			case b == ctrlW: // Delete word backward
				rl.deleteWordBackward()

			case b == tab: // Tab completion
				rl.handleTab()

			case b == enter || b == ctrlJ: // Submit line
				fmt.Fprint(rl.out, "\r\n")
				return string(rl.buf), nil

			case b == backspace || b == ctrlH: // Backspace
				rl.backspace()

			case b == escape: // Escape sequence
				// Read rest of escape sequence.
				if i < n {
					seq := collectEscapeBytes(buf[i:n])
					i += len(seq)
					rl.handleEscape(seq)
				}

			default:
				// Regular character (possibly multi-byte UTF-8).
				// Collect full rune.
				rBuf := []byte{b}
				for utf8.RuneCount(rBuf) == 0 && i < n {
					rBuf = append(rBuf, buf[i])
					i++
				}
				if r, size := utf8.DecodeRune(rBuf); size > 0 && r != utf8.RuneError {
					rl.insertChar(r)
				}
			}
		}
	}
}

// collectEscapeBytes reads the bytes of an escape sequence from the buffer.
func collectEscapeBytes(buf []byte) []byte {
	if len(buf) == 0 {
		return nil
	}
	// CSI sequences: ESC [ ... final_byte (0x40-0x7E)
	if buf[0] == '[' {
		for i := 1; i < len(buf); i++ {
			if buf[i] >= 0x40 && buf[i] <= 0x7E {
				return buf[:i+1]
			}
		}
		return buf // incomplete, take what we have
	}
	// SS3 sequences: ESC O ... (e.g., arrow keys on some terminals)
	if buf[0] == 'O' && len(buf) >= 2 {
		return buf[:2]
	}
	return buf[:1]
}

// handleEscape dispatches an escape sequence.
func (rl *Instance) handleEscape(seq []byte) {
	if len(seq) < 2 {
		return
	}

	if seq[0] == '[' {
		switch seq[1] {
		case 'A': // Up arrow
			rl.historyUp()
		case 'B': // Down arrow
			rl.historyDown()
		case 'C': // Right arrow
			rl.moveCursorRight()
		case 'D': // Left arrow
			rl.moveCursorLeft()
		case 'H': // Home
			rl.moveCursorHome()
		case 'F': // End
			rl.moveCursorEnd()
		case '3': // Delete key: ESC[3~
			if len(seq) >= 3 && seq[2] == '~' {
				rl.deleteChar()
			}
		case '1': // Home: ESC[1~
			if len(seq) >= 3 && seq[2] == '~' {
				rl.moveCursorHome()
			}
		case '4': // End: ESC[4~
			if len(seq) >= 3 && seq[2] == '~' {
				rl.moveCursorEnd()
			}
		}
	} else if seq[0] == 'O' {
		// SS3 sequences (some terminals use these for arrow keys).
		switch seq[1] {
		case 'A':
			rl.historyUp()
		case 'B':
			rl.historyDown()
		case 'C':
			rl.moveCursorRight()
		case 'D':
			rl.moveCursorLeft()
		case 'H':
			rl.moveCursorHome()
		case 'F':
			rl.moveCursorEnd()
		}
	}
}

// ── Line editing operations ──

func (rl *Instance) insertChar(r rune) {
	if rl.pos == len(rl.buf) {
		rl.buf = append(rl.buf, r)
		rl.pos++
		fmt.Fprint(rl.out, string(r))
	} else {
		// Insert in middle — need to redraw rest of line.
		rl.buf = append(rl.buf, 0)
		copy(rl.buf[rl.pos+1:], rl.buf[rl.pos:])
		rl.buf[rl.pos] = r
		rl.pos++
		rl.redrawFromCursor()
	}
}

func (rl *Instance) backspace() {
	if rl.pos == 0 {
		return
	}
	copy(rl.buf[rl.pos-1:], rl.buf[rl.pos:])
	rl.buf = rl.buf[:len(rl.buf)-1]
	rl.pos--

	// Move cursor left, redraw from cursor, clear trailing char.
	fmt.Fprint(rl.out, "\b")
	rl.redrawFromCursor()
}

func (rl *Instance) deleteChar() {
	if rl.pos >= len(rl.buf) {
		return
	}
	copy(rl.buf[rl.pos:], rl.buf[rl.pos+1:])
	rl.buf = rl.buf[:len(rl.buf)-1]
	rl.redrawFromCursor()
}

func (rl *Instance) killToEnd() {
	if rl.pos >= len(rl.buf) {
		return
	}
	// Clear from cursor to end.
	n := len(rl.buf) - rl.pos
	rl.buf = rl.buf[:rl.pos]
	fmt.Fprintf(rl.out, "\033[%dP", n) // delete n chars
	// Simpler: just clear to end of line.
	fmt.Fprint(rl.out, "\033[K")
}

func (rl *Instance) killToStart() {
	if rl.pos == 0 {
		return
	}
	rl.buf = rl.buf[rl.pos:]
	rl.pos = 0
	rl.redraw()
}

func (rl *Instance) deleteWordBackward() {
	if rl.pos == 0 {
		return
	}
	// Skip trailing spaces.
	end := rl.pos
	for rl.pos > 0 && rl.buf[rl.pos-1] == ' ' {
		rl.pos--
	}
	// Delete until next space or start.
	for rl.pos > 0 && rl.buf[rl.pos-1] != ' ' {
		rl.pos--
	}
	rl.buf = append(rl.buf[:rl.pos], rl.buf[end:]...)
	rl.redraw()
}

// ── Cursor movement ──

func (rl *Instance) moveCursorLeft() {
	if rl.pos > 0 {
		rl.pos--
		fmt.Fprint(rl.out, "\033[D")
	}
}

func (rl *Instance) moveCursorRight() {
	if rl.pos < len(rl.buf) {
		rl.pos++
		fmt.Fprint(rl.out, "\033[C")
	}
}

func (rl *Instance) moveCursorHome() {
	if rl.pos > 0 {
		fmt.Fprintf(rl.out, "\033[%dD", rl.pos)
		rl.pos = 0
	}
}

func (rl *Instance) moveCursorEnd() {
	if rl.pos < len(rl.buf) {
		fmt.Fprintf(rl.out, "\033[%dC", len(rl.buf)-rl.pos)
		rl.pos = len(rl.buf)
	}
}

// ── History navigation ──

func (rl *Instance) historyUp() {
	if len(rl.history) == 0 || rl.histIdx <= 0 {
		return
	}

	// Save current input when first entering history.
	if rl.histIdx == len(rl.history) {
		rl.savedLine = make([]rune, len(rl.buf))
		copy(rl.savedLine, rl.buf)
	}

	rl.histIdx--
	rl.setLine([]rune(rl.history[rl.histIdx]))
}

func (rl *Instance) historyDown() {
	if rl.histIdx >= len(rl.history) {
		return
	}

	rl.histIdx++
	if rl.histIdx == len(rl.history) {
		// Restore saved input.
		rl.setLine(rl.savedLine)
	} else {
		rl.setLine([]rune(rl.history[rl.histIdx]))
	}
}

// setLine replaces the current line buffer and redraws.
func (rl *Instance) setLine(line []rune) {
	rl.buf = make([]rune, len(line))
	copy(rl.buf, line)
	rl.pos = len(rl.buf)
	rl.redraw()
}

// ── Tab completion ──

func (rl *Instance) handleTab() {
	if rl.complete == nil {
		return
	}

	line := string(rl.buf)
	completions := rl.complete(line, rl.pos)
	if len(completions) == 0 {
		return
	}

	if len(completions) == 1 {
		// Single match — complete inline.
		rl.applyCompletion(completions[0])
		return
	}

	// Multiple matches — find common prefix and show candidates.
	prefix := longestCommonPrefix(completions)
	if prefix != "" {
		rl.applyCompletion(prefix)
		// If we made progress, don't show the list yet.
		if prefix != partialWord(line, rl.pos) {
			return
		}
	}

	// Show completion candidates.
	fmt.Fprint(rl.out, "\r\n")
	rl.displayCompletions(completions)
	rl.redraw()
}

// applyCompletion replaces the partial word at cursor with the completion.
func (rl *Instance) applyCompletion(completion string) {
	line := string(rl.buf)
	partial := partialWord(line, rl.pos)

	// Find start of partial word.
	start := rl.pos - len([]rune(partial))

	// Build new buffer.
	newBuf := make([]rune, 0, start+len([]rune(completion))+len(rl.buf)-rl.pos)
	newBuf = append(newBuf, rl.buf[:start]...)
	newBuf = append(newBuf, []rune(completion)...)

	// Add trailing space if this is a complete match (not a prefix).
	newBuf = append(newBuf, rl.buf[rl.pos:]...)

	rl.buf = newBuf
	rl.pos = start + len([]rune(completion))
	rl.redraw()
}

// displayCompletions shows completion candidates in columns.
func (rl *Instance) displayCompletions(completions []string) {
	// Find max width.
	maxW := 0
	for _, c := range completions {
		if len(c) > maxW {
			maxW = len(c)
		}
	}

	// Get terminal width.
	tw := 80
	if rl.isTTY {
		if w, _, err := term.GetSize(int(rl.in.Fd())); err == nil && w > 0 {
			tw = w
		}
	}

	colWidth := maxW + 2
	cols := tw / colWidth
	if cols < 1 {
		cols = 1
	}

	for i, c := range completions {
		fmt.Fprintf(rl.out, "%-*s", colWidth, c)
		if (i+1)%cols == 0 || i == len(completions)-1 {
			fmt.Fprint(rl.out, "\r\n")
		}
	}
}

// ── Display helpers ──

// redraw clears the line and redraws prompt + buffer with cursor at rl.pos.
func (rl *Instance) redraw() {
	// Move to start of line, clear it.
	fmt.Fprint(rl.out, "\r\033[K")
	// Write prompt + buffer.
	fmt.Fprint(rl.out, rl.prompt)
	fmt.Fprint(rl.out, string(rl.buf))
	// Move cursor to correct position.
	if tail := len(rl.buf) - rl.pos; tail > 0 {
		fmt.Fprintf(rl.out, "\033[%dD", tail)
	}
}

// redrawFromCursor redraws from the cursor position to end of line and
// repositions the cursor.
func (rl *Instance) redrawFromCursor() {
	// Write from cursor to end.
	tail := string(rl.buf[rl.pos:])
	fmt.Fprint(rl.out, tail)
	// Clear any leftover characters.
	fmt.Fprint(rl.out, "\033[K")
	// Move cursor back.
	if n := len(rl.buf) - rl.pos; n > 0 {
		fmt.Fprintf(rl.out, "\033[%dD", n)
	}
}

// ── Utility functions ──

// partialWord extracts the word being typed at cursor position.
func partialWord(line string, pos int) string {
	runes := []rune(line)
	if pos > len(runes) {
		pos = len(runes)
	}
	start := pos
	for start > 0 && runes[start-1] != ' ' {
		start--
	}
	return string(runes[start:pos])
}

// longestCommonPrefix finds the longest common prefix among strings.
func longestCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	prefix := strs[0]
	for _, s := range strs[1:] {
		for !strings.HasPrefix(s, prefix) {
			prefix = prefix[:len(prefix)-1]
			if prefix == "" {
				return ""
			}
		}
	}
	return prefix
}

// visibleWidth returns the display width of a string, excluding ANSI escapes.
func visibleWidth(s string) int {
	width := 0
	inEscape := false
	for _, r := range s {
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '~' {
				inEscape = false
			}
			continue
		}
		if r == '\033' {
			inEscape = true
			continue
		}
		width++
	}
	return width
}

// ── Control characters ──

const (
	ctrlA     = 1
	ctrlB     = 2
	ctrlC     = 3
	ctrlD     = 4
	ctrlE     = 5
	ctrlF     = 6
	ctrlH     = 8
	tab       = 9
	ctrlJ     = 10
	ctrlK     = 11
	ctrlL     = 12
	enter     = 13
	ctrlU     = 21
	ctrlW     = 23
	escape    = 27
	backspace = 127
)
