package editor

import (
	"fmt"
	"io"
	"strings"
)

// ANSI escape helpers.
const (
	escClearScreen = "\033[2J"
	escCursorHome  = "\033[H"
	escClearLine   = "\033[K"
	escHideCursor  = "\033[?25l"
	escShowCursor  = "\033[?25h"
	escAltScreen   = "\033[?1049h"
	escMainScreen  = "\033[?1049l"
	escBold        = "\033[1m"
	escDim         = "\033[2m"
	escReverse     = "\033[7m"
)

// Theme colors for UI chrome.
var (
	colorTitleBar  = "\033[48;2;40;40;50m\033[38;2;220;220;230m" // dark bg, light text
	colorStatusBar = "\033[48;2;40;40;50m\033[38;2;180;180;190m"
	colorLineNum   = "\033[38;2;100;100;120m"                     // dim line numbers
	colorAnnot     = "\033[38;2;120;100;160m"                     // dim purple annotations
	colorGutter    = "\033[38;2;60;60;70m"                        // gutter separator
	colorValid     = "\033[38;2;45;140;90m"                       // green
	colorInvalid   = "\033[38;2;196;48;48m"                       // red
	colorModified  = "\033[38;2;212;148;10m"                      // yellow
	colorMenuBg    = "\033[48;2;50;50;65m\033[38;2;220;220;230m"
	colorMenuSel   = "\033[48;2;80;70;120m\033[38;2;255;255;255m"
	colorMenuDim   = "\033[38;2;140;140;160m"
)

// Renderer handles all screen drawing for the editor.
type Renderer struct {
	out         io.Writer
	width       int
	height      int
	digitWidth  int // digits for line numbers (min 4)
	gutterWidth int // line number column width (digitWidth + 2)
	annotWidth  int // right annotation column width
	codeWidth   int // available width for code
}

// NewRenderer creates a renderer for the given terminal dimensions.
func NewRenderer(out io.Writer, width, height int) *Renderer {
	r := &Renderer{
		out:        out,
		width:      width,
		height:     height,
		annotWidth: 10,
	}
	r.recalc()
	return r
}

func (r *Renderer) recalc() {
	if r.digitWidth < 4 {
		r.digitWidth = 4
	}
	r.gutterWidth = r.digitWidth + 2 // digits + space + pipe
	r.codeWidth = r.width - r.gutterWidth - r.annotWidth
	if r.codeWidth < 20 {
		r.annotWidth = 0
		r.codeWidth = r.width - r.gutterWidth
	}
}

// updateGutter adjusts gutter width for the current file line count.
func (r *Renderer) updateGutter(lineCount int) {
	digits := 1
	for n := lineCount; n >= 10; n /= 10 {
		digits++
	}
	if digits < 4 {
		digits = 4
	}
	if digits != r.digitWidth {
		r.digitWidth = digits
		r.recalc()
	}
}

// Resize updates dimensions.
func (r *Renderer) Resize(width, height int) {
	r.width = width
	r.height = height
	r.recalc()
}

// codeRows returns usable rows for code (minus title + status bar).
func (r *Renderer) codeRows() int {
	return r.height - 2 // title bar + status bar
}

// RenderFull draws the entire screen.
func (r *Renderer) RenderFull(e *Editor) {
	r.updateGutter(e.buf.LineCount())

	var b strings.Builder

	b.WriteString(escHideCursor)
	b.WriteString(escCursorHome)

	// Title bar.
	r.renderTitleBar(&b, e)

	// Code lines.
	rows := r.codeRows()
	blockCtx := r.resolveBlockContext(e, e.viewY)

	for i := 0; i < rows; i++ {
		lineIdx := e.viewY + i
		b.WriteString(moveTo(i+2, 1)) // row i+2 (1-indexed, after title bar)
		b.WriteString(escClearLine)

		if lineIdx < e.buf.LineCount() {
			// Detect block context for annotations.
			cat := LineCategory(e.buf.Line(lineIdx), blockCtx)
			if cat != "" && cat != blockCtx {
				blockCtx = cat
			}

			r.renderCodeLine(&b, e, lineIdx, blockCtx)
		} else {
			// Empty line (after file end).
			b.WriteString(colorLineNum)
			b.WriteString(fmt.Sprintf("%*s ", r.digitWidth, "~"))
			b.WriteString(ansiReset)
			b.WriteString(colorGutter)
			b.WriteString("|")
			b.WriteString(ansiReset)
		}
	}

	// Status bar.
	r.renderStatusBar(&b, e)

	// Position cursor.
	screenRow := e.buf.cy - e.viewY + 2 // +1 for title bar, +1 for 1-indexing
	screenCol := r.gutterWidth + 1 + (e.buf.cx - e.viewX)
	if screenCol < r.gutterWidth+1 {
		screenCol = r.gutterWidth + 1
	}
	b.WriteString(moveTo(screenRow, screenCol))
	b.WriteString(escShowCursor)

	fmt.Fprint(r.out, b.String())
}

// renderTitleBar draws the top bar with filename and status.
func (r *Renderer) renderTitleBar(b *strings.Builder, e *Editor) {
	b.WriteString(moveTo(1, 1))
	b.WriteString(colorTitleBar)

	left := fmt.Sprintf(" %s ", e.filename)
	right := ""
	if e.dirty {
		right = colorModified + " Modified " + colorTitleBar
	}

	b.WriteString(left)
	// Pad middle.
	padLen := r.width - visLen(left) - 10
	if e.dirty {
		padLen -= 1
	}
	if padLen > 0 {
		b.WriteString(strings.Repeat(" ", padLen))
	}
	b.WriteString(right)

	// Fill remaining.
	remaining := r.width - visLen(left) - padLen
	if remaining > 0 && !e.dirty {
		b.WriteString(strings.Repeat(" ", remaining))
	}

	b.WriteString(ansiReset)
}

// renderCodeLine draws a single code line with line number and annotation.
func (r *Renderer) renderCodeLine(b *strings.Builder, e *Editor, lineIdx int, blockCtx string) {
	line := e.buf.Line(lineIdx)

	// Line number.
	b.WriteString(colorLineNum)
	b.WriteString(fmt.Sprintf("%*d ", r.digitWidth, lineIdx+1))
	b.WriteString(ansiReset)
	b.WriteString(colorGutter)
	b.WriteString("|")
	b.WriteString(ansiReset)

	// Code content with syntax highlighting.
	visibleLine := line
	if e.viewX > 0 && e.viewX < len(line) {
		visibleLine = line[e.viewX:]
	} else if e.viewX >= len(line) {
		visibleLine = nil
	}

	highlighted := RenderHighlighted(visibleLine)
	// Truncate to code width (approximate — ANSI codes make exact truncation complex).
	codeStr := truncateVisible(highlighted, r.codeWidth)
	b.WriteString(codeStr)

	// Pad to annotation column.
	visCodeLen := visibleRuneCount(visibleLine)
	if visCodeLen > r.codeWidth {
		visCodeLen = r.codeWidth
	}
	pad := r.codeWidth - visCodeLen
	if pad > 0 {
		b.WriteString(strings.Repeat(" ", pad))
	}

	// Right annotation.
	if r.annotWidth > 0 && blockCtx != "" {
		trimmed := strings.TrimSpace(string(line))
		lower := strings.ToLower(trimmed)
		// Only show annotation on block-starting lines.
		isBlockStart := false
		for kw := range sectionKeywords {
			if strings.HasPrefix(lower, kw+" ") || strings.HasPrefix(lower, kw+":") || lower == kw {
				isBlockStart = true
				break
			}
		}
		if isBlockStart {
			label := fmt.Sprintf("[%s]", blockCtx)
			b.WriteString(colorAnnot)
			b.WriteString(fmt.Sprintf(" %-8s", label))
			b.WriteString(ansiReset)
		}
	}
}

// renderStatusBar draws the bottom status bar.
func (r *Renderer) renderStatusBar(b *strings.Builder, e *Editor) {
	b.WriteString(moveTo(r.height, 1))
	b.WriteString(colorStatusBar)

	cx, cy := e.buf.Cursor()
	left := fmt.Sprintf(" Ln %d, Col %d", cy+1, cx+1)

	// Validation status (read once, mutex-protected).
	validErrMsg := e.getValidErr()
	var validStr string
	switch {
	case validErrMsg == "":
		validStr = colorValid + " Valid" + colorStatusBar
	default:
		errMsg := validErrMsg
		if len(errMsg) > 30 {
			errMsg = errMsg[:30] + "..."
		}
		validStr = colorInvalid + " " + errMsg + colorStatusBar
	}

	hints := " ESC:menu  Ctrl+S:save  Ctrl+Q:quit"

	b.WriteString(left)
	b.WriteString(" ")
	b.WriteString(validStr)

	padLen := r.width - visLen(left) - visLen(validErrMsg) - visLen(hints) - 4
	if padLen < 1 {
		padLen = 1
	}
	b.WriteString(strings.Repeat(" ", padLen))
	b.WriteString(hints)

	// Fill to edge.
	b.WriteString(strings.Repeat(" ", 2))
	b.WriteString(ansiReset)
}

// RenderMenu draws the ESC menu overlay centered on screen.
func (r *Renderer) RenderMenu(e *Editor, items []string, selected int) {
	menuW := 30
	menuH := len(items) + 4
	startRow := (r.height-menuH)/2 + 1
	startCol := (r.width-menuW)/2 + 1
	if startRow < 1 {
		startRow = 1
	}
	if startCol < 1 {
		startCol = 1
	}

	var b strings.Builder
	b.WriteString(escHideCursor)

	// Top border.
	b.WriteString(moveTo(startRow, startCol))
	b.WriteString(colorMenuBg)
	b.WriteString(" Menu")
	b.WriteString(strings.Repeat(" ", menuW-5))
	b.WriteString(ansiReset)

	// Separator.
	b.WriteString(moveTo(startRow+1, startCol))
	b.WriteString(colorMenuBg)
	b.WriteString(strings.Repeat("─", menuW))
	b.WriteString(ansiReset)

	// Items.
	for i, item := range items {
		b.WriteString(moveTo(startRow+2+i, startCol))
		if i == selected {
			b.WriteString(colorMenuSel)
		} else {
			b.WriteString(colorMenuBg)
		}
		label := fmt.Sprintf("  %-*s", menuW-2, item)
		b.WriteString(label)
		b.WriteString(ansiReset)
	}

	// Bottom.
	b.WriteString(moveTo(startRow+2+len(items), startCol))
	b.WriteString(colorMenuBg)
	b.WriteString(strings.Repeat("─", menuW))
	b.WriteString(ansiReset)
	b.WriteString(moveTo(startRow+3+len(items), startCol))
	b.WriteString(colorMenuDim)
	b.WriteString(fmt.Sprintf(" %-*s", menuW-1, "Arrow keys + Enter"))
	b.WriteString(ansiReset)

	fmt.Fprint(r.out, b.String())
}

// RenderComplete draws the autocomplete popup.
func (r *Renderer) RenderComplete(e *Editor, items []string, selected int) {
	if len(items) == 0 {
		return
	}

	maxItems := 8
	if len(items) < maxItems {
		maxItems = len(items)
	}

	// Position popup below cursor.
	screenRow := e.buf.cy - e.viewY + 3 // +2 for title, +1 for below cursor
	screenCol := r.gutterWidth + 1 + e.buf.cx - e.viewX

	if screenRow+maxItems >= r.height {
		screenRow = e.buf.cy - e.viewY + 1 // above cursor
	}

	maxW := 0
	for _, item := range items[:maxItems] {
		if len(item) > maxW {
			maxW = len(item)
		}
	}
	maxW += 4
	if maxW > 40 {
		maxW = 40
	}

	var b strings.Builder
	for i := 0; i < maxItems; i++ {
		b.WriteString(moveTo(screenRow+i, screenCol))
		if i == selected {
			b.WriteString(colorMenuSel)
		} else {
			b.WriteString(colorMenuBg)
		}
		label := items[i]
		if len(label) > maxW-2 {
			label = label[:maxW-3] + "."
		}
		b.WriteString(fmt.Sprintf(" %-*s", maxW-1, label))
		b.WriteString(ansiReset)
	}

	fmt.Fprint(r.out, b.String())
}

// ── Helpers ──

func moveTo(row, col int) string {
	return fmt.Sprintf("\033[%d;%dH", row, col)
}

// visLen returns visible length, stripping ANSI sequences.
func visLen(s string) int {
	n := 0
	inEsc := false
	for _, r := range s {
		if inEsc {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '~' {
				inEsc = false
			}
			continue
		}
		if r == '\033' {
			inEsc = true
			continue
		}
		n++
	}
	return n
}

// visibleRuneCount counts runes in a line (no ANSI in raw runes).
func visibleRuneCount(line []rune) int {
	return len(line)
}

// truncateVisible truncates an ANSI-colored string to maxVisible visible characters.
func truncateVisible(s string, maxVisible int) string {
	if maxVisible <= 0 {
		return ""
	}
	var b strings.Builder
	vis := 0
	inEsc := false
	for _, r := range s {
		if inEsc {
			b.WriteRune(r)
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '~' {
				inEsc = false
			}
			continue
		}
		if r == '\033' {
			inEsc = true
			b.WriteRune(r)
			continue
		}
		if vis >= maxVisible {
			break
		}
		b.WriteRune(r)
		vis++
	}
	// Always reset at end to prevent color bleed.
	b.WriteString(ansiReset)
	return b.String()
}

// resolveBlockContext determines what block the viewport starts in.
func (r *Renderer) resolveBlockContext(e *Editor, startLine int) string {
	ctx := ""
	for i := 0; i < startLine && i < e.buf.LineCount(); i++ {
		cat := LineCategory(e.buf.Line(i), ctx)
		if cat != "" {
			ctx = cat
		}
	}
	return ctx
}
