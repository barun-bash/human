package cmdutil

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/barun-bash/human/internal/cli"
	"github.com/barun-bash/human/internal/syntax"
)

// RunSyntax displays syntax reference.
// If section is empty, shows full reference (via pager if available).
// If search is non-empty, searches patterns.
func RunSyntax(out io.Writer, section, search string) {
	if search != "" {
		runSyntaxSearch(out, search)
		return
	}

	if section != "" {
		runSyntaxSection(out, section)
		return
	}

	// Full reference — try to pipe through a pager.
	runSyntaxFull(out)
}

func runSyntaxSearch(out io.Writer, query string) {
	results := syntax.Search(query)
	if len(results) == 0 {
		fmt.Fprintf(out, "No patterns matching %q found.\n", query)
		return
	}

	fmt.Fprintln(out)
	fmt.Fprintf(out, "%s\n\n",
		cli.Heading(fmt.Sprintf("Found %d patterns matching %q:", len(results), query)))

	max := 15
	if len(results) > max {
		results = results[:max]
	}
	for _, p := range results {
		template := highlightPlaceholders(p.Template)
		fmt.Fprintf(out, "  %-42s %s\n", template, cli.Muted(fmt.Sprintf("(%s)", p.Category)))
	}
	fmt.Fprintln(out)

	// Tip line.
	if len(results) > 0 {
		cat := results[0].Category
		fmt.Fprintf(out, "%s\n", cli.Muted(fmt.Sprintf("Tip: Run 'human explain %s' for full %s reference.", cat, syntax.CategoryLabel(cat))))
	}
	fmt.Fprintln(out)
}

func runSyntaxSection(out io.Writer, section string) {
	section = strings.ToLower(section)

	// Try as category.
	for _, cat := range syntax.AllCategories() {
		if string(cat) == section {
			patterns := syntax.ByCategory(cat)
			fmt.Fprintln(out)
			printCategorySection(out, cat, patterns)
			return
		}
	}

	// Try as alias.
	match, ok := topicAliases[section]
	if ok {
		fmt.Fprintln(out)
		for _, cat := range match.cats {
			patterns := syntax.ByCategory(cat)
			if match.filter != "" {
				patterns = filterPatterns(patterns, match.filter)
			}
			if len(patterns) == 0 {
				continue
			}
			printCategorySection(out, cat, patterns)
		}
		return
	}

	// Not found.
	fmt.Fprintf(out, "Unknown section: %s\n", section)
	fmt.Fprintln(out, "Available sections:")
	for _, cat := range syntax.AllCategories() {
		fmt.Fprintf(out, "  %s\n", cat)
	}
}

func runSyntaxFull(out io.Writer) {
	// Build the full output.
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(cli.Heading("Human Language — Syntax Reference"))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("═", 50))
	sb.WriteString("\n\n")

	for _, cat := range syntax.AllCategories() {
		patterns := syntax.ByCategory(cat)
		if len(patterns) == 0 {
			continue
		}

		header := fmt.Sprintf("── %s ", syntax.CategoryLabel(cat))
		pad := 50 - len([]rune(header))
		if pad < 0 {
			pad = 0
		}
		sb.WriteString(cli.Heading(header) + strings.Repeat("─", pad) + "\n\n")

		for _, p := range patterns {
			template := highlightPlaceholders(p.Template)
			sb.WriteString(fmt.Sprintf("  %-42s %s\n", template, cli.Muted(p.Description)))
		}
		sb.WriteString("\n")
	}

	content := sb.String()

	// Try pager.
	if pagerCmd := findPager(); pagerCmd != "" {
		if err := runPager(pagerCmd, content); err == nil {
			return
		}
	}

	// Fallback: print directly.
	fmt.Fprint(out, content)
}

func findPager() string {
	if pager := os.Getenv("PAGER"); pager != "" {
		return pager
	}
	if _, err := exec.LookPath("less"); err == nil {
		return "less -R"
	}
	if _, err := exec.LookPath("more"); err == nil {
		return "more"
	}
	return ""
}

func runPager(pagerCmd, content string) error {
	parts := strings.Fields(pagerCmd)
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdin = strings.NewReader(content)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
