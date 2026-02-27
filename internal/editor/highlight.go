package editor

import (
	"strings"
	"unicode"
)

// Token types for syntax highlighting.
type tokenType int

const (
	tokNormal tokenType = iota
	tokKeyword
	tokType
	tokString
	tokComment
	tokSection  // top-level section header (data X:, page Y:)
	tokOperator // is, has, to, etc.
	tokNumber
)

// token represents a colored span within a line.
type token struct {
	Start, End int
	Type       tokenType
}

// ANSI color codes for syntax highlighting (theme-aware).
var tokenColors = map[tokenType]string{
	tokNormal:   "",
	tokKeyword:  "\033[38;2;232;93;58m",   // coral/accent
	tokType:     "\033[38;2;86;182;194m",   // teal
	tokString:   "\033[38;2;152;195;121m",  // green
	tokComment:  "\033[38;2;128;128;128m",  // gray
	tokSection:  "\033[38;2;198;120;221m",  // purple
	tokOperator: "\033[38;2;212;148;10m",   // yellow
	tokNumber:   "\033[38;2;209;154;102m",  // orange
}

const ansiReset = "\033[0m"

// Top-level keywords that start a block.
var sectionKeywords = map[string]bool{
	"app": true, "data": true, "page": true, "component": true,
	"api": true, "build": true, "deploy": true, "theme": true,
	"integration": true, "workflow": true, "policy": true,
}

// Keywords that appear inside blocks.
var keywords = map[string]bool{
	"show": true, "has": true, "is": true, "a": true, "an": true,
	"the": true, "with": true, "for": true, "each": true,
	"clicking": true, "navigates": true, "to": true,
	"if": true, "when": true, "while": true, "loading": true,
	"there": true, "require": true, "requires": true,
	"required": true, "optional": true, "unique": true,
	"encrypted": true, "authenticated": true, "authentication": true,
	"accepts": true, "returns": true, "validates": true,
	"sends": true, "triggers": true, "stores": true,
	"deletes": true, "updates": true, "creates": true,
	"belongs": true, "many": true, "index": true,
	"frontend": true, "backend": true, "database": true,
	"using": true, "on": true, "from": true, "into": true,
	"where": true, "that": true, "of": true, "by": true,
	"and": true, "or": true, "not": true, "no": true,
	"restrict": true, "allow": true, "deny": true,
	"only": true, "every": true, "all": true,
	"import": true, "export": true, "include": true,
}

// Type names in the Human language.
var typeKeywords = map[string]bool{
	"text": true, "number": true, "decimal": true, "boolean": true,
	"date": true, "datetime": true, "email": true, "url": true,
	"file": true, "image": true, "json": true, "integer": true,
}

// HighlightLine tokenizes a single line for syntax highlighting.
func HighlightLine(line []rune) []token {
	s := string(line)
	trimmed := strings.TrimSpace(s)

	// Comment line.
	if strings.HasPrefix(trimmed, "#") {
		return []token{{Start: 0, End: len(line), Type: tokComment}}
	}

	var tokens []token
	i := 0
	for i < len(line) {
		// Skip whitespace — no token needed.
		if unicode.IsSpace(line[i]) {
			i++
			continue
		}

		// String literal.
		if line[i] == '"' {
			start := i
			i++
			for i < len(line) && line[i] != '"' {
				i++
			}
			if i < len(line) {
				i++ // closing quote
			}
			tokens = append(tokens, token{Start: start, End: i, Type: tokString})
			continue
		}

		// Inline comment.
		if line[i] == '#' {
			tokens = append(tokens, token{Start: i, End: len(line), Type: tokComment})
			break
		}

		// Number.
		if unicode.IsDigit(line[i]) {
			start := i
			for i < len(line) && (unicode.IsDigit(line[i]) || line[i] == '.') {
				i++
			}
			tokens = append(tokens, token{Start: start, End: i, Type: tokNumber})
			continue
		}

		// Word.
		if unicode.IsLetter(line[i]) || line[i] == '_' {
			start := i
			for i < len(line) && (unicode.IsLetter(line[i]) || unicode.IsDigit(line[i]) || line[i] == '_' || line[i] == '-') {
				i++
			}
			word := strings.ToLower(string(line[start:i]))

			// Check if this is a section header (e.g., "data Task:")
			if start == 0 || isLineStart(line, start) {
				if sectionKeywords[word] {
					tokens = append(tokens, token{Start: start, End: i, Type: tokSection})
					continue
				}
			}

			if typeKeywords[word] {
				tokens = append(tokens, token{Start: start, End: i, Type: tokType})
			} else if keywords[word] {
				tokens = append(tokens, token{Start: start, End: i, Type: tokKeyword})
			}
			// Normal words get no token (rendered in default color).
			continue
		}

		// Colon after section name — highlight as section.
		if line[i] == ':' {
			tokens = append(tokens, token{Start: i, End: i + 1, Type: tokSection})
			i++
			continue
		}

		i++
	}

	return tokens
}

// isLineStart checks if position is at the effective start of the line (only whitespace before).
func isLineStart(line []rune, pos int) bool {
	for i := 0; i < pos; i++ {
		if !unicode.IsSpace(line[i]) {
			return false
		}
	}
	return true
}

// RenderHighlighted returns an ANSI-colored string for a line.
func RenderHighlighted(line []rune) string {
	tokens := HighlightLine(line)
	if len(tokens) == 0 {
		return string(line)
	}

	var b strings.Builder
	pos := 0
	for _, tok := range tokens {
		// Write normal text before this token.
		if tok.Start > pos {
			b.WriteString(string(line[pos:tok.Start]))
		}
		// Write colored token.
		color := tokenColors[tok.Type]
		if color != "" {
			b.WriteString(color)
			b.WriteString(string(line[tok.Start:tok.End]))
			b.WriteString(ansiReset)
		} else {
			b.WriteString(string(line[tok.Start:tok.End]))
		}
		pos = tok.End
	}
	// Write remaining text.
	if pos < len(line) {
		b.WriteString(string(line[pos:]))
	}
	return b.String()
}

// LineCategory returns the category annotation for a line based on context.
func LineCategory(line []rune, blockCtx string) string {
	trimmed := strings.TrimSpace(string(line))
	lower := strings.ToLower(trimmed)

	// Detect block-starting lines.
	for keyword := range sectionKeywords {
		if strings.HasPrefix(lower, keyword+" ") || strings.HasPrefix(lower, keyword+":") {
			return keyword
		}
	}

	// Build sub-sections.
	if strings.HasPrefix(lower, "frontend") || strings.HasPrefix(lower, "backend") ||
		strings.HasPrefix(lower, "database") || strings.HasPrefix(lower, "deploy") {
		return "build"
	}

	return blockCtx // inherit from parent block
}
