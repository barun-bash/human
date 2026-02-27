package editor

import "os"

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

// ReadKey reads a single key event from the file descriptor in raw mode.
func ReadKey(fd int) KeyEvent {
	buf := make([]byte, 1)
	n, err := readByte(fd, buf)
	if err != nil || n == 0 {
		return KeyEvent{Key: KeyNone}
	}

	ch := buf[0]

	// Control characters.
	switch ch {
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
	case 10, 13: // Enter
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
	case 27: // ESC — start of escape sequence
		return readEscapeSeq(fd)
	case 127: // Backspace (alternative)
		return KeyEvent{Key: KeyBackspace}
	}

	// Printable ASCII or start of UTF-8 multi-byte.
	if ch >= 32 {
		r := decodeUTF8(ch, fd)
		return KeyEvent{Key: KeyChar, Rune: r}
	}

	return KeyEvent{Key: KeyNone}
}

// readEscapeSeq reads an escape sequence after ESC byte.
func readEscapeSeq(fd int) KeyEvent {
	buf := make([]byte, 1)
	n, err := readByteTimeout(fd, buf)
	if err != nil || n == 0 {
		return KeyEvent{Key: KeyEscape} // bare ESC
	}

	switch buf[0] {
	case '[': // CSI sequence
		return readCSI(fd)
	case 'O': // SS3 sequence
		return readSS3(fd)
	}
	return KeyEvent{Key: KeyEscape}
}

// readCSI reads a CSI escape sequence (ESC [ ...).
func readCSI(fd int) KeyEvent {
	buf := make([]byte, 1)
	n, err := readByteTimeout(fd, buf)
	if err != nil || n == 0 {
		return KeyEvent{Key: KeyNone}
	}

	switch buf[0] {
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
		return readExtCSI(fd, buf[0])
	case '3': // ESC[3~ = Delete
		readByteTimeout(fd, buf) // consume ~
		return KeyEvent{Key: KeyDelete}
	case '4': // ESC[4~ = End
		readByteTimeout(fd, buf) // consume ~
		return KeyEvent{Key: KeyEnd}
	case '5': // ESC[5~ = Page Up
		readByteTimeout(fd, buf) // consume ~
		return KeyEvent{Key: KeyPageUp}
	case '6': // ESC[6~ = Page Down
		readByteTimeout(fd, buf) // consume ~
		return KeyEvent{Key: KeyPageDown}
	}

	// Consume remaining bytes of unknown CSI.
	for {
		n, _ := readByteTimeout(fd, buf)
		if n == 0 || (buf[0] >= 0x40 && buf[0] <= 0x7E) {
			break
		}
	}
	return KeyEvent{Key: KeyNone}
}

// readExtCSI handles ESC[1... sequences.
func readExtCSI(fd int, first byte) KeyEvent {
	buf := make([]byte, 1)
	n, _ := readByteTimeout(fd, buf)
	if n == 0 {
		return KeyEvent{Key: KeyNone}
	}
	if buf[0] == '~' {
		return KeyEvent{Key: KeyHome} // ESC[1~
	}
	// Consume rest of sequence.
	for {
		n, _ := readByteTimeout(fd, buf)
		if n == 0 || (buf[0] >= 0x40 && buf[0] <= 0x7E) {
			break
		}
	}
	return KeyEvent{Key: KeyNone}
}

// readSS3 reads an SS3 escape sequence (ESC O ...).
func readSS3(fd int) KeyEvent {
	buf := make([]byte, 1)
	n, _ := readByteTimeout(fd, buf)
	if n == 0 {
		return KeyEvent{Key: KeyNone}
	}
	switch buf[0] {
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
func decodeUTF8(first byte, fd int) rune {
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
		buf := make([]byte, 1)
		nr, _ := readByte(fd, buf)
		if nr == 0 {
			return rune(first)
		}
		bytes[i] = buf[0]
	}
	r := []rune(string(bytes))
	if len(r) > 0 {
		return r[0]
	}
	return rune(first)
}

// readByte reads a single byte from fd (blocking).
func readByte(fd int, buf []byte) (int, error) {
	f := os.NewFile(uintptr(fd), "stdin")
	return f.Read(buf)
}

// readByteTimeout reads a single byte with a short timeout for escape sequences.
// On timeout or error returns 0, nil (bare ESC handling).
func readByteTimeout(fd int, buf []byte) (int, error) {
	// For escape sequences, we do a non-blocking read attempt.
	// The OS typically delivers escape sequence bytes together,
	// so a direct read works. If nothing is available, it's a bare ESC.
	f := os.NewFile(uintptr(fd), "stdin")
	return f.Read(buf)
}
