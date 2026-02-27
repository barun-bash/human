package editor

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/term"
)

// MenuAction represents the user's choice from the ESC menu.
type MenuAction int

const (
	MenuNone    MenuAction = iota
	MenuSave               // save file
	MenuBuild              // save + build
	MenuExit               // exit editor
	MenuDiscard            // exit without saving
)

// Editor is a terminal-based text editor for .human files.
type Editor struct {
	filepath string // full file path
	filename string // display name (basename)
	buf      *Buffer
	renderer *Renderer
	comp     *Completer
	val      *Validator
	input    *inputReader   // channel-based non-blocking key reader
	keyCh    chan KeyEvent   // single key channel — all reads MUST go through this

	viewY int // first visible line (vertical scroll)
	viewX int // first visible column (horizontal scroll)

	dirty    bool   // unsaved changes
	validMu  sync.Mutex
	validErr string // last validation error (protected by validMu)
	statusMsg string // transient status message (protected by validMu)

	width  int // terminal width
	height int // terminal height

	out     io.Writer
	stdinFd int
	oldTerm *term.State // saved terminal state

	running   bool
	needBuild bool // set when user chooses "Save & Build"
	redrawCh  chan struct{}
}

// Open creates an editor for the given file.
func Open(filepath string) (*Editor, error) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("cannot open %s: %w", filepath, err)
	}

	// Extract basename for display.
	name := filepath
	if idx := strings.LastIndexByte(filepath, '/'); idx >= 0 {
		name = filepath[idx+1:]
	}

	e := &Editor{
		filepath: filepath,
		filename: name,
		buf:      NewBuffer(string(content)),
		comp:     NewCompleter(),
		out:      os.Stdout,
		stdinFd:  int(os.Stdin.Fd()),
		redrawCh: make(chan struct{}, 1),
	}

	// Set up validator with 300ms debounce.
	e.val = NewValidator(300*time.Millisecond, func() {
		e.setValidErr(e.val.Error())
		// Signal redraw needed.
		select {
		case e.redrawCh <- struct{}{}:
		default:
		}
	})

	return e, nil
}

// NeedBuild returns true if the user chose "Save & Build" from the menu.
func (e *Editor) NeedBuild() bool { return e.needBuild }

// Run enters the editor, takes over the terminal, and blocks until exit.
func (e *Editor) Run() error {
	if !term.IsTerminal(e.stdinFd) {
		return fmt.Errorf("editor requires an interactive terminal")
	}

	// Enter alternate screen buffer + raw mode.
	if err := e.setupTerminal(); err != nil {
		return err
	}
	defer e.restoreTerminal()

	// Start channel-based input reader (background goroutine reads stdin bytes).
	e.input = newInputReader(e.stdinFd)

	// Parse keys in a goroutine and send to the single key channel.
	// ALL key reads (main loop, menu, goto-line) MUST use e.keyCh to avoid races.
	e.keyCh = make(chan KeyEvent, 16)
	go func() {
		for {
			key := e.input.ReadKey()
			e.keyCh <- key
		}
	}()

	// Handle terminal resize.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	defer signal.Stop(sigCh)

	// Initial render.
	e.updateSize()
	e.renderer = NewRenderer(e.out, e.width, e.height)

	// Initial validation.
	e.val.Schedule(e.buf.Content())

	e.running = true
	e.renderer.RenderFull(e)

	for e.running {
		// Event-driven loop: wait on any channel.
		select {
		case <-sigCh:
			e.updateSize()
			e.renderer.Resize(e.width, e.height)
			e.renderer.RenderFull(e)
			continue

		case <-e.redrawCh:
			e.renderer.RenderFull(e)
			continue

		case key := <-e.keyCh:
			if key.Key == KeyNone {
				continue
			}

			if e.comp.Active() {
				if e.handleCompletionKey(key) {
					e.renderer.RenderFull(e)
					if e.comp.Active() {
						e.renderer.RenderComplete(e, e.comp.Items(), e.comp.Selected())
					}
					continue
				}
			}

			e.handleKey(key)

			// Scroll viewport to keep cursor visible.
			e.scrollToCursor()

			e.renderer.RenderFull(e)

			if e.comp.Active() {
				e.renderer.RenderComplete(e, e.comp.Items(), e.comp.Selected())
			}
		}
	}

	e.val.Stop()
	return nil
}

// handleKey processes a single key event.
func (e *Editor) handleKey(key KeyEvent) {
	switch key.Key {
	// Navigation.
	case KeyUp:
		e.buf.MoveUp()
	case KeyDown:
		e.buf.MoveDown()
	case KeyLeft, KeyCtrlB:
		e.buf.MoveLeft()
	case KeyRight, KeyCtrlF:
		e.buf.MoveRight()
	case KeyHome, KeyCtrlA:
		e.buf.Home()
	case KeyEnd, KeyCtrlE:
		e.buf.End()
	case KeyPageUp:
		e.buf.PageUp(e.renderer.codeRows())
	case KeyPageDown:
		e.buf.PageDown(e.renderer.codeRows())

	// Editing.
	case KeyChar:
		e.buf.InsertChar(key.Rune)
		e.markDirty()
	case KeyEnter:
		e.buf.NewLine()
		e.markDirty()
	case KeyBackspace:
		e.buf.Backspace()
		e.markDirty()
	case KeyDelete, KeyCtrlD:
		e.buf.DeleteChar()
		e.markDirty()
	case KeyTab:
		e.comp.TriggerComplete(e.buf.Line(e.buf.cy), e.buf.cx)
		if !e.comp.Active() {
			e.buf.InsertTab()
			e.markDirty()
		}
	case KeyCtrlK:
		// Kill to end of line (single undo action).
		e.buf.KillToEnd()
		e.markDirty()
	case KeyCtrlU:
		// Kill to start of line (single undo action).
		e.buf.KillToStart()
		e.markDirty()

	// Commands.
	case KeyCtrlS:
		e.save()
	case KeyCtrlQ:
		e.quit()
	case KeyCtrlZ:
		if e.buf.Undo() {
			e.markDirty()
		}
	case KeyCtrlY:
		if e.buf.Redo() {
			e.markDirty()
		}
	case KeyCtrlG:
		e.gotoLine()
	case KeyEscape:
		e.showMenu()
	}
}

// handleCompletionKey handles keys while autocomplete is active.
// Returns true if the key was consumed.
func (e *Editor) handleCompletionKey(key KeyEvent) bool {
	switch key.Key {
	case KeyUp:
		e.comp.Prev()
		return true
	case KeyDown:
		e.comp.Next()
		return true
	case KeyTab, KeyEnter:
		e.comp.Accept(e.buf)
		e.markDirty()
		return true
	case KeyEscape:
		e.comp.Dismiss()
		return true
	}
	// Any other key dismisses completion and falls through.
	e.comp.Dismiss()
	return false
}

func (e *Editor) markDirty() {
	e.dirty = true
	e.val.Schedule(e.buf.Content())
}

func (e *Editor) setValidErr(msg string) {
	e.validMu.Lock()
	e.validErr = msg
	e.validMu.Unlock()
}

func (e *Editor) getValidErr() string {
	e.validMu.Lock()
	defer e.validMu.Unlock()
	return e.validErr
}

func (e *Editor) setStatusMsg(msg string) {
	e.validMu.Lock()
	e.statusMsg = msg
	e.validMu.Unlock()

	// Auto-clear after 2 seconds and trigger redraw.
	time.AfterFunc(2*time.Second, func() {
		e.validMu.Lock()
		if e.statusMsg == msg {
			e.statusMsg = ""
		}
		e.validMu.Unlock()
		select {
		case e.redrawCh <- struct{}{}:
		default:
		}
	})
}

func (e *Editor) getStatusMsg() string {
	e.validMu.Lock()
	defer e.validMu.Unlock()
	return e.statusMsg
}

// scrollToCursor adjusts viewport to keep cursor on screen.
func (e *Editor) scrollToCursor() {
	rows := e.renderer.codeRows()

	// Vertical.
	if e.buf.cy < e.viewY {
		e.viewY = e.buf.cy
	}
	if e.buf.cy >= e.viewY+rows {
		e.viewY = e.buf.cy - rows + 1
	}

	// Horizontal.
	codeW := e.renderer.codeWidth
	if e.buf.cx < e.viewX {
		e.viewX = e.buf.cx
	}
	if e.buf.cx >= e.viewX+codeW {
		e.viewX = e.buf.cx - codeW + 1
	}
}

// save writes the buffer to disk.
func (e *Editor) save() {
	content := e.buf.Content()
	err := os.WriteFile(e.filepath, []byte(content), 0644)
	if err != nil {
		e.setValidErr("Save failed: " + err.Error())
		return
	}
	e.dirty = false
	e.setStatusMsg("Saved.")
	// Re-validate after save.
	e.val.Schedule(content)
}

// quit exits the editor, warning about unsaved changes.
func (e *Editor) quit() {
	if !e.dirty {
		e.running = false
		return
	}
	// Show discard confirmation via menu.
	e.showMenu()
}

// gotoLine prompts for a line number in the status bar area.
func (e *Editor) gotoLine() {
	// Simple: read digits until Enter.
	var digits []rune

	// Show prompt in status bar.
	fmt.Fprint(e.out, moveTo(e.height, 1))
	fmt.Fprint(e.out, colorStatusBar)
	fmt.Fprint(e.out, fmt.Sprintf(" Go to line: %-*s", e.width-14, ""))
	fmt.Fprint(e.out, ansiReset)
	fmt.Fprint(e.out, moveTo(e.height, 14))
	fmt.Fprint(e.out, escShowCursor)

	for {
		key := <-e.keyCh // read from the single key channel (no race)
		switch key.Key {
		case KeyEnter:
			lineNum := 0
			for _, d := range digits {
				lineNum = lineNum*10 + int(d-'0')
			}
			if lineNum > 0 {
				e.buf.SetCursor(0, lineNum-1)
				e.scrollToCursor()
			}
			return
		case KeyEscape:
			return
		case KeyBackspace:
			if len(digits) > 0 {
				digits = digits[:len(digits)-1]
			}
		case KeyChar:
			if key.Rune >= '0' && key.Rune <= '9' {
				digits = append(digits, key.Rune)
			}
		default:
			continue
		}

		// Update display.
		fmt.Fprint(e.out, moveTo(e.height, 14))
		fmt.Fprint(e.out, escClearLine)
		fmt.Fprint(e.out, string(digits))
	}
}

// showMenu displays the ESC menu overlay.
func (e *Editor) showMenu() {
	items := []string{"Save", "Save & Build", "Exit"}
	if e.dirty {
		items = append(items, "Discard Changes & Exit")
	}
	selected := 0

	for {
		e.renderer.RenderMenu(e, items, selected)

		key := <-e.keyCh // read from the single key channel (no race)
		switch key.Key {
		case KeyUp:
			selected--
			if selected < 0 {
				selected = len(items) - 1
			}
		case KeyDown:
			selected++
			if selected >= len(items) {
				selected = 0
			}
		case KeyEnter:
			switch items[selected] {
			case "Save":
				e.save()
			case "Save & Build":
				e.save()
				e.needBuild = true
				e.running = false
			case "Exit":
				if !e.dirty {
					e.running = false
				}
				// If dirty, do nothing (force save or discard).
			case "Discard Changes & Exit":
				e.running = false
			}
			return
		case KeyEscape:
			return // close menu
		case KeyCtrlQ:
			if !e.dirty {
				e.running = false
			}
			return
		}
	}
}

// ── Terminal setup/teardown ──

func (e *Editor) setupTerminal() error {
	// Save terminal state and enter raw mode.
	old, err := term.MakeRaw(e.stdinFd)
	if err != nil {
		return fmt.Errorf("cannot enter raw mode: %w", err)
	}
	e.oldTerm = old

	// Enter alternate screen buffer.
	fmt.Fprint(e.out, escAltScreen)
	fmt.Fprint(e.out, escClearScreen)

	return nil
}

func (e *Editor) restoreTerminal() {
	// Show cursor, leave alt screen, restore terminal.
	fmt.Fprint(e.out, escShowCursor)
	fmt.Fprint(e.out, escMainScreen)
	if e.oldTerm != nil {
		term.Restore(e.stdinFd, e.oldTerm)
	}
}

func (e *Editor) updateSize() {
	w, h, err := term.GetSize(e.stdinFd)
	if err != nil || w == 0 || h == 0 {
		e.width = 80
		e.height = 24
		return
	}
	e.width = w
	e.height = h
}

// RunExternal runs an external command (e.g., build) after restoring terminal.
// Used for "Save & Build" — caller should handle this after Run() returns.
func RunExternal(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
