package editor

import "strings"

// Buffer holds the text content with line-based editing operations.
type Buffer struct {
	lines     [][]rune
	cx, cy    int // cursor column (cx) and row (cy)
	undoStack []action
	redoStack []action
}

type actionKind int

const (
	actInsert     actionKind = iota // inserted rune(s) at position
	actDelete                       // deleted rune(s) at position
	actNewline  // split line at position
	actJoinLine // joined line with next (reverse of newline)
)

type action struct {
	kind    actionKind
	line    int
	col     int
	text    []rune // content for insert/delete
	oldCx   int    // cursor state before action
	oldCy   int
}

// NewBuffer creates a buffer from a string.
func NewBuffer(content string) *Buffer {
	b := &Buffer{}
	if content == "" {
		b.lines = [][]rune{{}}
		return b
	}
	raw := strings.Split(content, "\n")
	b.lines = make([][]rune, len(raw))
	for i, s := range raw {
		b.lines[i] = []rune(s)
	}
	// Remove trailing empty line from split if content ends with \n
	if len(b.lines) > 1 && len(b.lines[len(b.lines)-1]) == 0 && strings.HasSuffix(content, "\n") {
		b.lines = b.lines[:len(b.lines)-1]
	}
	return b
}

// Content serializes the buffer back to a string.
func (b *Buffer) Content() string {
	parts := make([]string, len(b.lines))
	for i, line := range b.lines {
		parts[i] = string(line)
	}
	return strings.Join(parts, "\n") + "\n"
}

// LineCount returns the number of lines.
func (b *Buffer) LineCount() int { return len(b.lines) }

// Line returns the content of line y (0-indexed).
func (b *Buffer) Line(y int) []rune {
	if y < 0 || y >= len(b.lines) {
		return nil
	}
	return b.lines[y]
}

// LineLen returns the length of line y.
func (b *Buffer) LineLen(y int) int {
	if y < 0 || y >= len(b.lines) {
		return 0
	}
	return len(b.lines[y])
}

// Cursor returns the current cursor position (col, row).
func (b *Buffer) Cursor() (int, int) { return b.cx, b.cy }

// SetCursor moves the cursor, clamping to valid bounds.
func (b *Buffer) SetCursor(x, y int) {
	if y < 0 {
		y = 0
	}
	if y >= len(b.lines) {
		y = len(b.lines) - 1
	}
	b.cy = y
	if x < 0 {
		x = 0
	}
	if x > len(b.lines[y]) {
		x = len(b.lines[y])
	}
	b.cx = x
}

// InsertChar inserts a rune at the cursor position.
func (b *Buffer) InsertChar(r rune) {
	b.pushUndo(action{kind: actInsert, line: b.cy, col: b.cx, text: []rune{r}, oldCx: b.cx, oldCy: b.cy})
	b.redoStack = nil

	line := b.lines[b.cy]
	newLine := make([]rune, len(line)+1)
	copy(newLine, line[:b.cx])
	newLine[b.cx] = r
	copy(newLine[b.cx+1:], line[b.cx:])
	b.lines[b.cy] = newLine
	b.cx++
}

// InsertTab inserts spaces to the next 2-space tab stop.
func (b *Buffer) InsertTab() {
	spaces := 2 - (b.cx % 2)
	for i := 0; i < spaces; i++ {
		b.InsertChar(' ')
	}
}

// Backspace deletes the character before the cursor.
func (b *Buffer) Backspace() {
	if b.cx > 0 {
		ch := b.lines[b.cy][b.cx-1]
		b.pushUndo(action{kind: actDelete, line: b.cy, col: b.cx - 1, text: []rune{ch}, oldCx: b.cx, oldCy: b.cy})
		b.redoStack = nil

		line := b.lines[b.cy]
		b.lines[b.cy] = append(line[:b.cx-1], line[b.cx:]...)
		b.cx--
	} else if b.cy > 0 {
		// Join with previous line.
		prevLen := len(b.lines[b.cy-1])
		b.pushUndo(action{kind: actJoinLine, line: b.cy - 1, col: prevLen, oldCx: b.cx, oldCy: b.cy})
		b.redoStack = nil

		b.lines[b.cy-1] = append(b.lines[b.cy-1], b.lines[b.cy]...)
		b.lines = append(b.lines[:b.cy], b.lines[b.cy+1:]...)
		b.cy--
		b.cx = prevLen
	}
}

// DeleteChar deletes the character at the cursor.
func (b *Buffer) DeleteChar() {
	if b.cx < len(b.lines[b.cy]) {
		ch := b.lines[b.cy][b.cx]
		b.pushUndo(action{kind: actDelete, line: b.cy, col: b.cx, text: []rune{ch}, oldCx: b.cx, oldCy: b.cy})
		b.redoStack = nil

		line := b.lines[b.cy]
		b.lines[b.cy] = append(line[:b.cx], line[b.cx+1:]...)
	} else if b.cy < len(b.lines)-1 {
		// Join with next line.
		b.pushUndo(action{kind: actJoinLine, line: b.cy, col: len(b.lines[b.cy]), oldCx: b.cx, oldCy: b.cy})
		b.redoStack = nil

		b.lines[b.cy] = append(b.lines[b.cy], b.lines[b.cy+1]...)
		b.lines = append(b.lines[:b.cy+1], b.lines[b.cy+2:]...)
	}
}

// NewLine splits the current line at the cursor.
func (b *Buffer) NewLine() {
	line := b.lines[b.cy]

	// Capture indent from the ORIGINAL line before any modification.
	indent := b.leadingWhitespace(b.cy)

	// Determine auto-indent: only apply when cursor is at or past the
	// indentation. When cursor is within/before indent (e.g. Enter at col 0),
	// the 'after' part already has the correct indentation.
	var autoIndent []rune
	if b.cx >= len(indent) && len(indent) > 0 {
		autoIndent = make([]rune, len(indent))
		copy(autoIndent, indent)
	}

	// Store auto-indent in action.text so undo/redo can handle it correctly.
	b.pushUndo(action{kind: actNewline, line: b.cy, col: b.cx, text: autoIndent, oldCx: b.cx, oldCy: b.cy})
	b.redoStack = nil

	before := make([]rune, b.cx)
	copy(before, line[:b.cx])
	after := make([]rune, len(line)-b.cx)
	copy(after, line[b.cx:])

	b.lines[b.cy] = before
	// Insert new line after current.
	newLines := make([][]rune, len(b.lines)+1)
	copy(newLines, b.lines[:b.cy+1])
	newLines[b.cy+1] = after
	copy(newLines[b.cy+2:], b.lines[b.cy+1:])
	b.lines = newLines

	// Prepend auto-indent to the new line.
	if len(autoIndent) > 0 {
		newLines[b.cy+1] = append(autoIndent, newLines[b.cy+1]...)
	}

	b.cy++
	b.cx = len(autoIndent)
}

// KillToEnd deletes from cursor to end of line as a single undoable action.
func (b *Buffer) KillToEnd() {
	if b.cx >= len(b.lines[b.cy]) {
		return
	}
	deleted := make([]rune, len(b.lines[b.cy])-b.cx)
	copy(deleted, b.lines[b.cy][b.cx:])
	b.pushUndo(action{kind: actDelete, line: b.cy, col: b.cx, text: deleted, oldCx: b.cx, oldCy: b.cy})
	b.redoStack = nil
	b.lines[b.cy] = b.lines[b.cy][:b.cx]
}

// KillToStart deletes from start of line to cursor as a single undoable action.
func (b *Buffer) KillToStart() {
	if b.cx == 0 {
		return
	}
	deleted := make([]rune, b.cx)
	copy(deleted, b.lines[b.cy][:b.cx])
	b.pushUndo(action{kind: actDelete, line: b.cy, col: 0, text: deleted, oldCx: b.cx, oldCy: b.cy})
	b.redoStack = nil
	b.lines[b.cy] = b.lines[b.cy][b.cx:]
	b.cx = 0
}

// leadingWhitespace returns the leading spaces/tabs of line y.
func (b *Buffer) leadingWhitespace(y int) []rune {
	if y < 0 || y >= len(b.lines) {
		return nil
	}
	var ws []rune
	for _, r := range b.lines[y] {
		if r == ' ' || r == '\t' {
			ws = append(ws, r)
		} else {
			break
		}
	}
	return ws
}

// Undo reverts the last action.
func (b *Buffer) Undo() bool {
	if len(b.undoStack) == 0 {
		return false
	}
	act := b.undoStack[len(b.undoStack)-1]
	b.undoStack = b.undoStack[:len(b.undoStack)-1]
	b.redoStack = append(b.redoStack, act)
	b.applyReverse(act)
	return true
}

// Redo re-applies the last undone action.
func (b *Buffer) Redo() bool {
	if len(b.redoStack) == 0 {
		return false
	}
	act := b.redoStack[len(b.redoStack)-1]
	b.redoStack = b.redoStack[:len(b.redoStack)-1]
	b.undoStack = append(b.undoStack, act)
	b.applyForward(act)
	return true
}

func (b *Buffer) pushUndo(a action) {
	b.undoStack = append(b.undoStack, a)
	// Cap undo stack at 500.
	if len(b.undoStack) > 500 {
		b.undoStack = b.undoStack[len(b.undoStack)-500:]
	}
}

func (b *Buffer) applyReverse(a action) {
	switch a.kind {
	case actInsert:
		// Reverse of insert = delete the inserted text.
		line := b.lines[a.line]
		b.lines[a.line] = append(line[:a.col], line[a.col+len(a.text):]...)
	case actDelete:
		// Reverse of delete = insert the deleted text.
		line := b.lines[a.line]
		newLine := make([]rune, len(line)+len(a.text))
		copy(newLine, line[:a.col])
		copy(newLine[a.col:], a.text)
		copy(newLine[a.col+len(a.text):], line[a.col:])
		b.lines[a.line] = newLine
	case actNewline:
		// Reverse of newline = join lines, stripping auto-indent from new line.
		if a.line+1 < len(b.lines) {
			newLine := b.lines[a.line+1]
			// Strip the auto-indent that was prepended during NewLine.
			if len(a.text) > 0 && len(newLine) >= len(a.text) {
				newLine = newLine[len(a.text):]
			}
			b.lines[a.line] = append(b.lines[a.line][:a.col], newLine...)
			b.lines = append(b.lines[:a.line+1], b.lines[a.line+2:]...)
		}
	case actJoinLine:
		// Reverse of join = split at col.
		line := b.lines[a.line]
		before := make([]rune, a.col)
		copy(before, line[:a.col])
		after := make([]rune, len(line)-a.col)
		copy(after, line[a.col:])
		b.lines[a.line] = before
		newLines := make([][]rune, len(b.lines)+1)
		copy(newLines, b.lines[:a.line+1])
		newLines[a.line+1] = after
		copy(newLines[a.line+2:], b.lines[a.line+1:])
		b.lines = newLines
	}
	b.cx = a.oldCx
	b.cy = a.oldCy
}

func (b *Buffer) applyForward(a action) {
	switch a.kind {
	case actInsert:
		line := b.lines[a.line]
		newLine := make([]rune, len(line)+len(a.text))
		copy(newLine, line[:a.col])
		copy(newLine[a.col:], a.text)
		copy(newLine[a.col+len(a.text):], line[a.col:])
		b.lines[a.line] = newLine
		b.cx = a.col + len(a.text)
		b.cy = a.line
	case actDelete:
		line := b.lines[a.line]
		b.lines[a.line] = append(line[:a.col], line[a.col+len(a.text):]...)
		b.cx = a.col
		b.cy = a.line
	case actNewline:
		line := b.lines[a.line]
		before := make([]rune, a.col)
		copy(before, line[:a.col])
		after := make([]rune, len(line)-a.col)
		copy(after, line[a.col:])
		b.lines[a.line] = before
		newLines := make([][]rune, len(b.lines)+1)
		copy(newLines, b.lines[:a.line+1])
		newLines[a.line+1] = after
		copy(newLines[a.line+2:], b.lines[a.line+1:])
		b.lines = newLines
		// Re-apply auto-indent stored in action.text.
		if len(a.text) > 0 {
			newLines[a.line+1] = append(a.text, newLines[a.line+1]...)
		}
		b.cy = a.line + 1
		b.cx = len(a.text)
	case actJoinLine:
		if a.line+1 < len(b.lines) {
			b.lines[a.line] = append(b.lines[a.line], b.lines[a.line+1]...)
			b.lines = append(b.lines[:a.line+1], b.lines[a.line+2:]...)
		}
		b.cx = a.col
		b.cy = a.line
	}
}

// ── Navigation helpers ──

// MoveLeft moves cursor one position left, wrapping to previous line.
func (b *Buffer) MoveLeft() {
	if b.cx > 0 {
		b.cx--
	} else if b.cy > 0 {
		b.cy--
		b.cx = len(b.lines[b.cy])
	}
}

// MoveRight moves cursor one position right, wrapping to next line.
func (b *Buffer) MoveRight() {
	if b.cx < len(b.lines[b.cy]) {
		b.cx++
	} else if b.cy < len(b.lines)-1 {
		b.cy++
		b.cx = 0
	}
}

// MoveUp moves cursor one line up, clamping column.
func (b *Buffer) MoveUp() {
	if b.cy > 0 {
		b.cy--
		if b.cx > len(b.lines[b.cy]) {
			b.cx = len(b.lines[b.cy])
		}
	}
}

// MoveDown moves cursor one line down, clamping column.
func (b *Buffer) MoveDown() {
	if b.cy < len(b.lines)-1 {
		b.cy++
		if b.cx > len(b.lines[b.cy]) {
			b.cx = len(b.lines[b.cy])
		}
	}
}

// Home moves cursor to start of line.
func (b *Buffer) Home() { b.cx = 0 }

// End moves cursor to end of line.
func (b *Buffer) End() { b.cx = len(b.lines[b.cy]) }

// PageUp moves cursor up by n lines.
func (b *Buffer) PageUp(n int) {
	b.cy -= n
	if b.cy < 0 {
		b.cy = 0
	}
	if b.cx > len(b.lines[b.cy]) {
		b.cx = len(b.lines[b.cy])
	}
}

// PageDown moves cursor down by n lines.
func (b *Buffer) PageDown(n int) {
	b.cy += n
	if b.cy >= len(b.lines) {
		b.cy = len(b.lines) - 1
	}
	if b.cx > len(b.lines[b.cy]) {
		b.cx = len(b.lines[b.cy])
	}
}
