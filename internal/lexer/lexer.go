package lexer

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

// Lexer tokenizes Human language source code into a stream of tokens.
type Lexer struct {
	source  string  // the full source text
	tokens  []Token // accumulated tokens
	start   int     // byte offset of current token start
	current int     // byte offset of current position
	line    int     // current line number (1-based)
	column  int     // current column number (1-based)

	// Indentation tracking
	indentStack []int // stack of indentation levels
	atLineStart bool  // true when we're at the beginning of a new line
}

// New creates a new Lexer for the given source code.
func New(source string) *Lexer {
	return &Lexer{
		source:      source,
		tokens:      make([]Token, 0, 256),
		line:        1,
		column:      1,
		indentStack: []int{0},
		atLineStart: true,
	}
}

// Tokenize processes the entire source and returns all tokens.
// The token stream always ends with TOKEN_EOF.
func (l *Lexer) Tokenize() ([]Token, error) {
	for !l.isAtEnd() {
		if l.atLineStart {
			l.processLineStart()
			continue
		}

		l.start = l.current
		if err := l.scanToken(); err != nil {
			return nil, err
		}
	}

	// Emit DEDENT tokens for any remaining indentation levels
	for len(l.indentStack) > 1 {
		l.indentStack = l.indentStack[:len(l.indentStack)-1]
		l.emit(TOKEN_DEDENT, "")
	}

	l.emit(TOKEN_EOF, "")
	return l.tokens, nil
}

// processLineStart handles the beginning of a new line: blank line skipping,
// comment-only lines, section headers, and indentation changes.
func (l *Lexer) processLineStart() {
	// Measure leading whitespace
	indent := 0
	startPos := l.current
	for !l.isAtEnd() {
		r := l.peekRune()
		if r == ' ' {
			indent++
			l.advance()
		} else if r == '\t' {
			indent += 4 // tabs count as 4 spaces
			l.advance()
		} else {
			break
		}
	}

	// Check what follows the indentation
	if l.isAtEnd() {
		// End of file after whitespace
		l.atLineStart = false
		return
	}

	r := l.peekRune()

	// Blank line: skip entirely
	if r == '\n' {
		l.advance()
		l.line++
		l.column = 1
		// Stay in atLineStart mode
		return
	}
	if r == '\r' {
		l.advance()
		if !l.isAtEnd() && l.peekRune() == '\n' {
			l.advance()
		}
		l.line++
		l.column = 1
		return
	}

	// Comment-only line: emit comment, skip to next line
	if r == '#' {
		// Check if it's a color literal (unlikely at line start, but be safe)
		if !l.isColorLiteral() {
			l.start = l.current
			l.scanComment()
			l.atLineStart = true
			return
		}
	}

	// Section header: ── name ── (U+2500 BOX DRAWINGS LIGHT HORIZONTAL)
	if r == '\u2500' || (r == '-' && l.peekRuneAt(l.current+1) == '-') {
		l.start = startPos
		l.scanSectionHeader()
		l.atLineStart = true
		return
	}

	// Process indentation changes
	currentIndent := l.indentStack[len(l.indentStack)-1]

	if indent > currentIndent {
		l.indentStack = append(l.indentStack, indent)
		l.emit(TOKEN_INDENT, "")
	} else if indent < currentIndent {
		for len(l.indentStack) > 1 && l.indentStack[len(l.indentStack)-1] > indent {
			l.indentStack = l.indentStack[:len(l.indentStack)-1]
			l.emit(TOKEN_DEDENT, "")
		}
	}

	l.atLineStart = false
	l.start = l.current
}

// scanToken scans and emits the next token from the current position.
func (l *Lexer) scanToken() error {
	r := l.peekRune()

	switch {
	case r == '\n':
		l.advance()
		l.emit(TOKEN_NEWLINE, "")
		l.line++
		l.column = 1
		l.atLineStart = true
		return nil

	case r == '\r':
		l.advance()
		if !l.isAtEnd() && l.peekRune() == '\n' {
			l.advance()
		}
		l.emit(TOKEN_NEWLINE, "")
		l.line++
		l.column = 1
		l.atLineStart = true
		return nil

	case r == ' ' || r == '\t':
		l.skipWhitespace()
		l.start = l.current
		return nil

	case r == ':':
		l.advance()
		l.emit(TOKEN_COLON, ":")
		return nil

	case r == ',':
		l.advance()
		l.emit(TOKEN_COMMA, ",")
		return nil

	case r == '"':
		return l.scanString()

	case r == '#':
		if l.isColorLiteral() {
			l.scanColorLiteral()
		} else {
			l.scanComment()
		}
		return nil

	case r == '\'':
		// Check for possessive 's
		if l.peekRuneAt(l.current+1) == 's' {
			next2 := l.current + 2
			if next2 >= len(l.source) || !isAlphaNumeric(l.peekRuneAt(next2)) {
				l.advance() // consume '
				l.advance() // consume s
				l.emit(TOKEN_POSSESSIVE, "'s")
				return nil
			}
		}
		// Unknown character
		l.advance()
		return nil

	case isDigit(r):
		l.scanNumber()
		return nil

	case isAlpha(r) || r == '_':
		l.scanWord()
		return nil

	default:
		// Skip unrecognized characters
		l.advance()
		l.start = l.current
		return nil
	}
}

// scanString scans a double-quoted string literal.
func (l *Lexer) scanString() error {
	l.advance() // consume opening "

	for !l.isAtEnd() {
		r := l.peekRune()
		if r == '"' {
			// Capture the string content (without quotes)
			content := l.source[l.start+1 : l.current]
			l.advance() // consume closing "
			l.emit(TOKEN_STRING_LIT, content)
			return nil
		}
		if r == '\\' {
			l.advance() // skip backslash
			if !l.isAtEnd() {
				l.advance() // skip escaped character
			}
			continue
		}
		if r == '\n' {
			return l.errorf("unterminated string starting at line %d, column %d", l.line, l.column)
		}
		l.advance()
	}

	return l.errorf("unterminated string starting at line %d", l.line)
}

// scanNumber scans an integer or decimal number.
func (l *Lexer) scanNumber() {
	for !l.isAtEnd() && isDigit(l.peekRune()) {
		l.advance()
	}

	// Check for decimal point
	if !l.isAtEnd() && l.peekRune() == '.' {
		next := l.current + 1
		if next < len(l.source) && isDigit(l.peekRuneAt(next)) {
			l.advance() // consume .
			for !l.isAtEnd() && isDigit(l.peekRune()) {
				l.advance()
			}
		}
	}

	// Check for units attached to number (e.g., 3am, 500ms)
	// These stay as separate tokens, so stop here

	l.emit(TOKEN_NUMBER_LIT, l.source[l.start:l.current])
}

// scanWord scans a keyword or identifier.
func (l *Lexer) scanWord() {
	for !l.isAtEnd() {
		r := l.peekRune()
		if isAlphaNumeric(r) || r == '_' {
			l.advance()
			continue
		}
		// Handle hyphens within words (e.g., "getting-started", "in_progress")
		if r == '-' {
			nextPos := l.current + 1
			if nextPos < len(l.source) && isAlpha(l.peekRuneAt(nextPos)) {
				l.advance() // consume -
				continue
			}
		}
		// Handle contractions (e.g., "don't", "doesn't")
		if r == '\'' {
			nextPos := l.current + 1
			if nextPos < len(l.source) {
				nextR := l.peekRuneAt(nextPos)
				// Check for possessive 's — don't consume, let it be a separate token
				if nextR == 's' {
					afterS := nextPos + 1
					if afterS >= len(l.source) || !isAlphaNumeric(l.peekRuneAt(afterS)) {
						break // stop word here, 's will be scanned as POSSESSIVE
					}
				}
				// Contraction (don't, can't, etc.) — include in the word
				if isAlpha(nextR) {
					l.advance() // consume '
					continue
				}
			}
		}
		break
	}

	word := l.source[l.start:l.current]
	tokenType := LookupKeyword(word)
	l.emit(tokenType, word)
}

// scanComment scans a comment from # to end of line.
func (l *Lexer) scanComment() {
	l.start = l.current
	l.advance() // consume #

	for !l.isAtEnd() && l.peekRune() != '\n' && l.peekRune() != '\r' {
		l.advance()
	}

	text := l.source[l.start:l.current]
	l.emit(TOKEN_COMMENT, text)

	// Consume the newline
	if !l.isAtEnd() {
		r := l.peekRune()
		if r == '\r' {
			l.advance()
			if !l.isAtEnd() && l.peekRune() == '\n' {
				l.advance()
			}
		} else if r == '\n' {
			l.advance()
		}
		l.line++
		l.column = 1
	}
}

// isColorLiteral checks if the current position starts a color literal (#RGB or #RRGGBB).
func (l *Lexer) isColorLiteral() bool {
	pos := l.current + 1 // skip the #
	hexCount := 0
	for i := pos; i < len(l.source); i++ {
		ch := l.source[i]
		if isHexDigit(ch) {
			hexCount++
		} else {
			break
		}
	}
	if hexCount != 3 && hexCount != 6 {
		return false
	}
	endPos := pos + hexCount
	if endPos >= len(l.source) {
		return true
	}
	// Next char must not be alphanumeric
	r, _ := utf8.DecodeRune([]byte{l.source[endPos]})
	return !isAlphaNumeric(r)
}

// scanColorLiteral scans a color literal (#RGB or #RRGGBB).
func (l *Lexer) scanColorLiteral() {
	l.advance() // consume #
	for !l.isAtEnd() && isHexDigit(byte(l.peekRune())) {
		l.advance()
	}
	l.emit(TOKEN_COLOR_LIT, l.source[l.start:l.current])
}

// scanSectionHeader scans a section header like ── name ──
func (l *Lexer) scanSectionHeader() {
	// Consume leading dash characters (─ or -)
	for !l.isAtEnd() {
		r := l.peekRune()
		if r == '\u2500' || r == '-' {
			l.advance()
		} else {
			break
		}
	}

	// Skip whitespace
	l.skipInlineWhitespace()

	// Read the section name
	nameStart := l.current
	for !l.isAtEnd() {
		r := l.peekRune()
		if r == '\u2500' || r == '-' || r == '\n' || r == '\r' {
			break
		}
		l.advance()
	}
	name := trimSpaces(l.source[nameStart:l.current])

	// Consume trailing dash characters
	for !l.isAtEnd() {
		r := l.peekRune()
		if r == '\u2500' || r == '-' {
			l.advance()
		} else {
			break
		}
	}

	l.emit(TOKEN_SECTION_HEADER, name)

	// Consume trailing whitespace and newline
	l.skipInlineWhitespace()
	if !l.isAtEnd() {
		r := l.peekRune()
		if r == '\r' {
			l.advance()
			if !l.isAtEnd() && l.peekRune() == '\n' {
				l.advance()
			}
			l.line++
			l.column = 1
		} else if r == '\n' {
			l.advance()
			l.line++
			l.column = 1
		}
	}
}

// ── Character scanning helpers ──

// isAtEnd returns true if the lexer has reached the end of the source.
func (l *Lexer) isAtEnd() bool {
	return l.current >= len(l.source)
}

// peekRune returns the rune at the current position without advancing.
func (l *Lexer) peekRune() rune {
	if l.isAtEnd() {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.source[l.current:])
	return r
}

// peekRuneAt returns the rune at the given byte offset.
func (l *Lexer) peekRuneAt(offset int) rune {
	if offset >= len(l.source) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.source[offset:])
	return r
}

// advance consumes the current rune and moves forward.
func (l *Lexer) advance() rune {
	if l.isAtEnd() {
		return 0
	}
	r, size := utf8.DecodeRuneInString(l.source[l.current:])
	l.current += size
	l.column++
	return r
}

// skipWhitespace consumes spaces and tabs.
func (l *Lexer) skipWhitespace() {
	for !l.isAtEnd() {
		r := l.peekRune()
		if r == ' ' || r == '\t' {
			l.advance()
		} else {
			break
		}
	}
}

// skipInlineWhitespace consumes spaces and tabs (not newlines).
func (l *Lexer) skipInlineWhitespace() {
	for !l.isAtEnd() {
		r := l.peekRune()
		if r == ' ' || r == '\t' {
			l.advance()
		} else {
			break
		}
	}
}

// emit adds a token to the output stream.
func (l *Lexer) emit(tokenType TokenType, literal string) {
	l.tokens = append(l.tokens, Token{
		Type:    tokenType,
		Literal: literal,
		Line:    l.line,
		Column:  l.column,
	})
	l.start = l.current
}

// errorf returns a formatted error with the current position.
func (l *Lexer) errorf(format string, args ...interface{}) error {
	return fmt.Errorf("lexer error at line %d, column %d: %s",
		l.line, l.column, fmt.Sprintf(format, args...))
}

// ── Character classification helpers ──

func isAlpha(r rune) bool {
	return unicode.IsLetter(r)
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isAlphaNumeric(r rune) bool {
	return isAlpha(r) || isDigit(r) || r == '_'
}

func isHexDigit(ch byte) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

// trimSpaces trims leading and trailing spaces from a string
// without importing strings (to keep imports minimal in this file).
func trimSpaces(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
