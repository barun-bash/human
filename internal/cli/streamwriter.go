package cli

import (
	"fmt"
	"io"
	"strings"
)

// StreamWriter wraps an io.Writer and prefixes each line with a left border
// character to visually distinguish LLM-streamed output from CLI output.
type StreamWriter struct {
	out       io.Writer
	atLineStart bool
	border    string
}

// NewStreamWriter creates a StreamWriter that prefixes lines with a colored border.
func NewStreamWriter(out io.Writer) *StreamWriter {
	border := "│ "
	if ColorEnabled {
		color := themeColor(RoleMuted, "\033[90m")
		border = color + "│" + reset + " "
	}
	return &StreamWriter{
		out:         out,
		atLineStart: true,
		border:      border,
	}
}

// Write implements io.Writer, adding the border prefix at the start of each line.
func (sw *StreamWriter) Write(p []byte) (n int, err error) {
	s := string(p)
	var written int

	for len(s) > 0 {
		if sw.atLineStart {
			if _, err := fmt.Fprint(sw.out, sw.border); err != nil {
				return written, err
			}
			sw.atLineStart = false
		}

		idx := strings.IndexByte(s, '\n')
		if idx >= 0 {
			// Write up to and including the newline.
			n, err := fmt.Fprint(sw.out, s[:idx+1])
			written += n
			if err != nil {
				return written, err
			}
			s = s[idx+1:]
			sw.atLineStart = true
		} else {
			// No newline — write remaining text.
			n, err := fmt.Fprint(sw.out, s)
			written += n
			if err != nil {
				return written, err
			}
			s = ""
		}
	}

	return len(p), nil
}

// Finish prints a final newline if needed, then shows completion.
func (sw *StreamWriter) Finish() {
	if !sw.atLineStart {
		fmt.Fprintln(sw.out)
	}
}
