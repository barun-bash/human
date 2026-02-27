package fixer

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/barun-bash/human/internal/cli"
	cerr "github.com/barun-bash/human/internal/errors"
)

// readFile reads a file as a string.
func readFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Apply writes a single fix to the file, creating a .bak backup first.
func Apply(fix Fix) error {
	// Read current content.
	data, err := os.ReadFile(fix.File)
	if err != nil {
		return fmt.Errorf("reading %s: %w", fix.File, err)
	}

	// Create backup.
	backupPath := fix.File + ".bak"
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("creating backup %s: %w", backupPath, err)
	}

	// Append fix to end of file.
	content := string(data)
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += fix.Suggestion + "\n"

	if err := os.WriteFile(fix.File, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", fix.File, err)
	}

	return nil
}

// ApplyAll applies a list of fixes to their files.
func ApplyAll(fixes []Fix) error {
	for _, fix := range fixes {
		if err := Apply(fix); err != nil {
			return err
		}
	}
	return nil
}

// PrintResult displays the analysis result to the writer.
func PrintResult(out io.Writer, result *Result, file string) {
	fmt.Fprintln(out)
	fmt.Fprintf(out, "Analyzing: %s\n", file)
	fmt.Fprintln(out)

	errorCount := len(result.Errors)
	warnCount := len(result.Warnings)
	fixCount := len(result.Fixes)

	header := fmt.Sprintf("── %d errors, %d warnings", errorCount, warnCount)
	if fixCount > 0 {
		header += fmt.Sprintf(" (%d auto-fixable)", fixCount)
	}
	header += " "
	pad := 50 - len([]rune(header))
	if pad < 0 {
		pad = 0
	}
	fmt.Fprintf(out, "%s%s\n\n", cli.Heading(header), strings.Repeat("─", pad))

	// Print errors.
	for _, e := range result.Errors {
		printDiag(out, e)
	}

	// Print warnings.
	for _, w := range result.Warnings {
		printDiag(out, w)
	}

	// Print fix suggestions.
	if fixCount > 0 {
		fmt.Fprintln(out)
		fmt.Fprintln(out, cli.Heading("Suggested Fixes:"))
		for i, fix := range result.Fixes {
			fmt.Fprintf(out, "  %d. [%s] %s\n", i+1, fix.Code, fix.Description)
			fmt.Fprintf(out, "     %s\n", cli.Muted(fix.Suggestion))
		}
	}
}

func printDiag(out io.Writer, e *cerr.CompilerError) {
	switch e.Severity {
	case cerr.SeverityWarning:
		fmt.Fprintf(out, "%s\n", cli.Warn(e.Format()))
	default:
		fmt.Fprintf(out, "%s\n", cli.Error(e.Format()))
	}
	if e.Suggestion != "" {
		fmt.Fprintf(out, "  %s\n", cli.Muted("suggestion: "+e.Suggestion))
	}
}
