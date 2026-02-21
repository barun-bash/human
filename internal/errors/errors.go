package errors

import (
	"fmt"
	"strings"
)

// Severity indicates how serious a compiler diagnostic is.
type Severity int

const (
	SeverityError Severity = iota
	SeverityWarning
	SeverityHint
)

// CompilerError is a single diagnostic from the compiler.
type CompilerError struct {
	Message    string   // human-readable description
	Severity   Severity // error, warning, or hint
	File       string   // source file path (empty if unknown)
	Line       int      // 0 if unknown
	Column     int      // 0 if unknown
	Suggestion string   // e.g. "Did you mean 'User'?" (optional)
	Code       string   // "E101" style error code
}

// Format returns a single-line representation of this error
// suitable for terminal output (without ANSI — the caller wraps with cli colors).
func (e *CompilerError) Format() string {
	var b strings.Builder

	if e.File != "" {
		b.WriteString(e.File)
		b.WriteString(" — ")
	}

	b.WriteString(e.Message)

	if e.Code != "" {
		b.WriteString(" [")
		b.WriteString(e.Code)
		b.WriteString("]")
	}

	return b.String()
}

// CompilerErrors collects diagnostics produced during compilation.
type CompilerErrors struct {
	errors []*CompilerError
	file   string // default file context
}

// New creates a CompilerErrors collection scoped to a file.
func New(file string) *CompilerErrors {
	return &CompilerErrors{file: file}
}

// Add appends an error to the collection.
func (ce *CompilerErrors) Add(err *CompilerError) {
	if err.File == "" {
		err.File = ce.file
	}
	ce.errors = append(ce.errors, err)
}

// AddError is a shorthand for adding a SeverityError diagnostic.
func (ce *CompilerErrors) AddError(code, message string) {
	ce.Add(&CompilerError{
		Code:     code,
		Message:  message,
		Severity: SeverityError,
	})
}

// AddWarning is a shorthand for adding a SeverityWarning diagnostic.
func (ce *CompilerErrors) AddWarning(code, message string) {
	ce.Add(&CompilerError{
		Code:     code,
		Message:  message,
		Severity: SeverityWarning,
	})
}

// AddWarningWithSuggestion adds a warning with a "did you mean" suggestion.
func (ce *CompilerErrors) AddWarningWithSuggestion(code, message, suggestion string) {
	ce.Add(&CompilerError{
		Code:       code,
		Message:    message,
		Severity:   SeverityWarning,
		Suggestion: suggestion,
	})
}

// AddErrorWithSuggestion adds an error with a "did you mean" suggestion.
func (ce *CompilerErrors) AddErrorWithSuggestion(code, message, suggestion string) {
	ce.Add(&CompilerError{
		Code:       code,
		Message:    message,
		Severity:   SeverityError,
		Suggestion: suggestion,
	})
}

// HasErrors returns true if the collection contains any SeverityError entries.
func (ce *CompilerErrors) HasErrors() bool {
	for _, e := range ce.errors {
		if e.Severity == SeverityError {
			return true
		}
	}
	return false
}

// HasWarnings returns true if the collection contains any SeverityWarning entries.
func (ce *CompilerErrors) HasWarnings() bool {
	for _, e := range ce.errors {
		if e.Severity == SeverityWarning {
			return true
		}
	}
	return false
}

// Errors returns only the SeverityError entries.
func (ce *CompilerErrors) Errors() []*CompilerError {
	var result []*CompilerError
	for _, e := range ce.errors {
		if e.Severity == SeverityError {
			result = append(result, e)
		}
	}
	return result
}

// Warnings returns only the SeverityWarning entries.
func (ce *CompilerErrors) Warnings() []*CompilerError {
	var result []*CompilerError
	for _, e := range ce.errors {
		if e.Severity == SeverityWarning {
			result = append(result, e)
		}
	}
	return result
}

// All returns every diagnostic in the collection.
func (ce *CompilerErrors) All() []*CompilerError {
	return ce.errors
}

// Format returns a human-friendly multiline string of all diagnostics.
func (ce *CompilerErrors) Format() string {
	var b strings.Builder
	for i, e := range ce.errors {
		if i > 0 {
			b.WriteString("\n")
		}

		switch e.Severity {
		case SeverityError:
			fmt.Fprintf(&b, "✗ %s", e.Format())
		case SeverityWarning:
			fmt.Fprintf(&b, "⚠ %s", e.Format())
		case SeverityHint:
			fmt.Fprintf(&b, "· %s", e.Format())
		}

		if e.Suggestion != "" {
			fmt.Fprintf(&b, "\n  suggestion: %s", e.Suggestion)
		}
	}
	return b.String()
}
