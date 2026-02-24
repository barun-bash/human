package repl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/barun-bash/human/internal/cli"
)

// Plan represents an execution plan shown to the user before an action.
type Plan struct {
	Title    string
	Steps    []string
	Editable bool
}

// PlanAction is what the user chose to do with a plan.
type PlanAction int

const (
	PlanGo     PlanAction = iota // proceed with execution
	PlanCancel                   // abort
	PlanEdit                     // open in editor and re-show
)

// ShowPlan renders a bordered plan box and prompts for an action.
// It reads one line from in. For non-interactive input (tests), it reads
// a full line rather than a single raw keypress.
func ShowPlan(w io.Writer, in io.Reader, plan *Plan) PlanAction {
	renderPlanBox(w, plan)
	fmt.Fprintln(w)

	if plan.Editable {
		fmt.Fprintf(w, "  %s  %s  %s\n\n",
			cli.Accent("[e]dit plan"),
			cli.Accent("[g]o"),
			cli.Accent("[c]ancel"))
	} else {
		fmt.Fprintf(w, "  %s  %s\n\n",
			cli.Accent("[g]o"),
			cli.Accent("[c]ancel"))
	}

	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		return PlanCancel
	}
	choice := strings.TrimSpace(strings.ToLower(scanner.Text()))

	switch {
	case choice == "g" || choice == "go":
		return PlanGo
	case (choice == "e" || choice == "edit") && plan.Editable:
		return PlanEdit
	default:
		return PlanCancel
	}
}

// renderPlanBox draws a Unicode-bordered box around the plan steps.
func renderPlanBox(w io.Writer, plan *Plan) {
	// Determine content width.
	titleLen := utf8.RuneCountInString(plan.Title)
	maxContent := titleLen + 2 // title padding
	for i, step := range plan.Steps {
		lineLen := numWidth(i+1) + 2 + utf8.RuneCountInString(step) // "N. step"
		if lineLen > maxContent {
			maxContent = lineLen
		}
	}

	// Clamp width.
	boxInner := maxContent + 2 // 1 space padding on each side
	if boxInner < 40 {
		boxInner = 40
	}
	if boxInner > 60 {
		boxInner = 60
	}

	// Top border with embedded title.
	titleBar := fmt.Sprintf(" %s ", plan.Title)
	dashesAfter := boxInner - utf8.RuneCountInString(titleBar) - 1
	if dashesAfter < 1 {
		dashesAfter = 1
	}
	fmt.Fprintf(w, "  \u250c\u2500%s%s\u2510\n", titleBar, strings.Repeat("\u2500", dashesAfter))

	// Steps.
	for i, step := range plan.Steps {
		line := fmt.Sprintf("%d. %s", i+1, step)
		runeLen := utf8.RuneCountInString(line)
		pad := boxInner - 2 - runeLen // subtract 2 for side padding
		if pad < 0 {
			// Truncate long lines.
			runes := []rune(line)
			if boxInner-5 > 0 {
				line = string(runes[:boxInner-5]) + "..."
			}
			pad = 0
		}
		fmt.Fprintf(w, "  \u2502 %s%s \u2502\n", line, strings.Repeat(" ", pad))
	}

	// Bottom border.
	fmt.Fprintf(w, "  \u2514%s\u2518\n", strings.Repeat("\u2500", boxInner))
}

// numWidth returns the number of digits in n.
func numWidth(n int) int {
	if n < 10 {
		return 1
	}
	if n < 100 {
		return 2
	}
	return 3
}

// EditPlan opens the plan steps in the user's $EDITOR and returns modified steps.
func EditPlan(plan *Plan) ([]string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}

	tmpFile := filepath.Join(os.TempDir(), "human-plan.md")

	var content strings.Builder
	content.WriteString(fmt.Sprintf("# %s\n\n", plan.Title))
	content.WriteString("# Edit the plan below. Each numbered line is a step.\n")
	content.WriteString("# Lines starting with # are ignored.\n")
	content.WriteString("# Delete lines to remove steps. Add lines to add steps.\n\n")
	for i, step := range plan.Steps {
		content.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
	}

	if err := os.WriteFile(tmpFile, []byte(content.String()), 0600); err != nil {
		return nil, fmt.Errorf("could not create temp file: %w", err)
	}
	defer os.Remove(tmpFile)

	cmd := exec.Command(editor, tmpFile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("editor exited with error: %w", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		return nil, fmt.Errorf("could not read edited plan: %w", err)
	}

	return parseSteps(string(data)), nil
}

// parseSteps extracts numbered or plain steps from edited plan text,
// ignoring blank lines and comments.
func parseSteps(text string) []string {
	var steps []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Strip leading "N. " if present.
		step := stripNumberPrefix(line)
		if step != "" {
			steps = append(steps, step)
		}
	}
	return steps
}

// stripNumberPrefix removes a leading "123. " from a line.
// If the line doesn't have a number prefix, returns the line as-is.
func stripNumberPrefix(line string) string {
	for i, c := range line {
		if c >= '0' && c <= '9' {
			continue
		}
		if c == '.' && i > 0 {
			return strings.TrimSpace(line[i+1:])
		}
		// Not a numbered line — return as-is.
		return line
	}
	return line
}
