package editor

import (
	"os"
	"time"
)

// Key represents a keyboard event.
type Key int

const (
	KeyNone Key = iota
	KeyChar     // printable character (value in KeyEvent.Rune)
	KeyEnter
	KeyBackspace
	KeyDelete
	KeyTab
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyHome
	KeyEnd
	KeyPageUp
	KeyPageDown
	KeyEscape
	KeyCtrlA // home
	KeyCtrlB // left
	KeyCtrlD // delete / EOF
	KeyCtrlE // end
	KeyCtrlF // right
	KeyCtrlG // goto line
	KeyCtrlK // kill to EOL
	KeyCtrlQ // quit
	KeyCtrlS // save
	KeyCtrlU // kill to start
	KeyCtrlZ // undo
	KeyCtrlY // redo
)

// KeyEvent holds a parsed key press.
type KeyEvent struct {
	Key  Key
	Rune rune // valid when Key == KeyChar
}

// inputReader reads raw bytes from stdin in a background goroutine and
// provides them via a buffered channel. This ensures the editor main loop
// never blocks on I/O and escape sequence parsing can use real timeouts.
type inputReader struct {
	byteCh   chan byte
	fd       int
	pushback byte // single byte pushback for CR+LF handling
	hasPush  bool
}

func newInputReader(fd int) *inputReader {
	ir := &inputReader{
		byteCh: make(chan byte, 128), // buffer for rapid input
		fd:     fd,
	}
	go ir.readLoop()
	return ir
}

func (ir *inputReader) readLoop() {
	buf := make([]byte, 64) // read in small batches for efficiency
	f := os.NewFile(uintptr(ir.fd), "stdin")
	for {
		n, err := f.Read(buf)
		if err != nil {
			close(ir.byteCh)
			return
		}
		for i := 0; i < n; i++ {
			ir.byteCh <- buf[i]
		}
	}
}

// getByte blocks until a byte is available.
func (ir *inputReader) getByte() (byte, bool) {
	if ir.hasPush {
		ir.hasPush = false
		return ir.pushback, true
	}
	b, ok := <-ir.byteCh
	return b, ok
}

// getByteTimeout waits up to d for a byte. Returns (0, false) on timeout.
func (ir *inputReader) getByteTimeout(d time.Duration) (byte, bool) {
	if ir.hasPush {
		ir.hasPush = false
		return ir.pushback, true
	}
	select {
	case b, ok := <-ir.byteCh:
		return b, ok
	case <-time.After(d):
		return 0, false
	}
}

const escTimeout = 50 * time.Millisecond // standard escape sequence timeout

// ReadKey reads and parses a single key event.
func (ir *inputReader) ReadKey() KeyEvent {
	b, ok := ir.getByte()
	if !ok {
		return KeyEvent{Key: KeyNone}
	}

	// Control characters.
	switch b {
	case 1: // Ctrl+A
		return KeyEvent{Key: KeyCtrlA}
	case 2: // Ctrl+B
		return KeyEvent{Key: KeyCtrlB}
	case 4: // Ctrl+D
		return KeyEvent{Key: KeyCtrlD}
	case 5: // Ctrl+E
		return KeyEvent{Key: KeyCtrlE}
	case 6: // Ctrl+F
		return KeyEvent{Key: KeyCtrlF}
	case 7: // Ctrl+G
		return KeyEvent{Key: KeyCtrlG}
	case 8: // Ctrl+H / Backspace
		return KeyEvent{Key: KeyBackspace}
	case 9: // Tab
		return KeyEvent{Key: KeyTab}
	case 10: // LF — Enter
		return KeyEvent{Key: KeyEnter}
	case 13: // CR — Enter (consume trailing LF from CR+LF pair)
		if b2, ok := ir.getByteTimeout(5 * time.Millisecond); ok && b2 != 10 {
			ir.pushback = b2
			ir.hasPush = true
		}
		return KeyEvent{Key: KeyEnter}
	case 11: // Ctrl+K
		return KeyEvent{Key: KeyCtrlK}
	case 17: // Ctrl+Q
		return KeyEvent{Key: KeyCtrlQ}
	case 19: // Ctrl+S
		return KeyEvent{Key: KeyCtrlS}
	case 21: // Ctrl+U
		return KeyEvent{Key: KeyCtrlU}
	case 25: // Ctrl+Y
		return KeyEvent{Key: KeyCtrlY}
	case 26: // Ctrl+Z
		return KeyEvent{Key: KeyCtrlZ}
	case 27: // ESC — start of escape sequence or bare ESC
		return ir.readEscapeSeq()
	case 127: // Backspace (alternative)
		return KeyEvent{Key: KeyBackspace}
	}

	// Printable ASCII or start of UTF-8 multi-byte.
	if b >= 32 {
		r := ir.decodeUTF8(b)
		return KeyEvent{Key: KeyChar, Rune: r}
	}

	return KeyEvent{Key: KeyNone}
}

// readEscapeSeq reads an escape sequence after ESC byte.
func (ir *inputReader) readEscapeSeq() KeyEvent {
	b, ok := ir.getByteTimeout(escTimeout)
	if !ok {
		return KeyEvent{Key: KeyEscape} // bare ESC (timeout)
	}

	switch b {
	case '[': // CSI sequence
		return ir.readCSI()
	case 'O': // SS3 sequence
		return ir.readSS3()
	}
	return KeyEvent{Key: KeyEscape}
}

// readCSI reads a CSI escape sequence (ESC [ ...).
func (ir *inputReader) readCSI() KeyEvent {
	b, ok := ir.getByteTimeout(escTimeout)
	if !ok {
		return KeyEvent{Key: KeyNone}
	}

	switch b {
	case 'A':
		return KeyEvent{Key: KeyUp}
	case 'B':
		return KeyEvent{Key: KeyDown}
	case 'C':
		return KeyEvent{Key: KeyRight}
	case 'D':
		return KeyEvent{Key: KeyLeft}
	case 'H':
		return KeyEvent{Key: KeyHome}
	case 'F':
		return KeyEvent{Key: KeyEnd}
	case '1': // Extended sequences: ESC[1~, ESC[1;...
		return ir.readExtCSI()
	case '3': // ESC[3~ = Delete
		ir.getByteTimeout(escTimeout) // consume ~
		return KeyEvent{Key: KeyDelete}
	case '4': // ESC[4~ = End
		ir.getByteTimeout(escTimeout) // consume ~
		return KeyEvent{Key: KeyEnd}
	case '5': // ESC[5~ = Page Up
		ir.getByteTimeout(escTimeout) // consume ~
		return KeyEvent{Key: KeyPageUp}
	case '6': // ESC[6~ = Page Down
		ir.getByteTimeout(escTimeout) // consume ~
		return KeyEvent{Key: KeyPageDown}
	}

	// Consume remaining bytes of unknown CSI sequence.
	for {
		cb, ok := ir.getByteTimeout(escTimeout)
		if !ok || (cb >= 0x40 && cb <= 0x7E) {
			break
		}
	}
	return KeyEvent{Key: KeyNone}
}

// readExtCSI handles ESC[1... sequences.
func (ir *inputReader) readExtCSI() KeyEvent {
	b, ok := ir.getByteTimeout(escTimeout)
	if !ok {
		return KeyEvent{Key: KeyNone}
	}
	if b == '~' {
		return KeyEvent{Key: KeyHome} // ESC[1~
	}
	// Consume rest of sequence.
	for {
		cb, ok := ir.getByteTimeout(escTimeout)
		if !ok || (cb >= 0x40 && cb <= 0x7E) {
			break
		}
	}
	return KeyEvent{Key: KeyNone}
}

// readSS3 reads an SS3 escape sequence (ESC O ...).
func (ir *inputReader) readSS3() KeyEvent {
	b, ok := ir.getByteTimeout(escTimeout)
	if !ok {
		return KeyEvent{Key: KeyNone}
	}
	switch b {
	case 'A':
		return KeyEvent{Key: KeyUp}
	case 'B':
		return KeyEvent{Key: KeyDown}
	case 'C':
		return KeyEvent{Key: KeyRight}
	case 'D':
		return KeyEvent{Key: KeyLeft}
	case 'H':
		return KeyEvent{Key: KeyHome}
	case 'F':
		return KeyEvent{Key: KeyEnd}
	}
	return KeyEvent{Key: KeyNone}
}

// decodeUTF8 decodes a UTF-8 character starting with the given byte.
func (ir *inputReader) decodeUTF8(first byte) rune {
	if first < 0x80 {
		return rune(first)
	}

	var n int
	switch {
	case first&0xE0 == 0xC0:
		n = 1
	case first&0xF0 == 0xE0:
		n = 2
	case first&0xF8 == 0xF0:
		n = 3
	default:
		return rune(first)
	}

	bytes := make([]byte, n+1)
	bytes[0] = first
	for i := 1; i <= n; i++ {
		b, ok := ir.getByteTimeout(escTimeout)
		if !ok {
			return rune(first)
		}
		bytes[i] = b
	}
	r := []rune(string(bytes))
	if len(r) > 0 {
		return r[0]
	}
	return rune(first)
}
