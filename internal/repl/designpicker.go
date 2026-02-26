package repl

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/barun-bash/human/internal/cli"
)

// designSystemOption describes one design system choice.
type designSystemOption struct {
	ID   string
	Name string
	Desc string
}

// designSystemOptions lists the available design systems.
var designSystemOptions = []designSystemOption{
	{"material", "Material UI", "Google's design system"},
	{"shadcn", "Shadcn/ui", "Modern, minimal components"},
	{"ant", "Ant Design", "Enterprise-grade UI"},
	{"chakra", "Chakra UI", "Accessible, simple"},
	{"bootstrap", "Bootstrap", "Classic, widely supported"},
	{"tailwind", "Tailwind CSS", "Utility-first, no components"},
	{"untitled", "Untitled UI", "Clean, modern, React Aria"},
}

// frameworkCompat maps design system IDs to the set of frameworks they natively support.
// If a framework is not listed, the build pipeline falls back to Tailwind CSS.
var frameworkCompat = map[string]map[string]bool{
	"material":  {"react": true, "vue": true, "angular": true},
	"shadcn":    {"react": true, "vue": true, "svelte": true},
	"ant":       {"react": true, "vue": true, "angular": true},
	"chakra":    {"react": true},
	"bootstrap": {"react": true, "vue": true, "angular": true, "svelte": true},
	"tailwind":  {"react": true, "vue": true, "angular": true, "svelte": true},
	"untitled":  {"react": true, "vue": true, "angular": true, "svelte": true},
}

// promptDesignSystem displays an interactive picker and returns the selected
// design system ID. Returns empty string if user cancels or enters invalid input.
// If framework is non-empty, a compatibility warning is shown for mismatches.
func (r *REPL) promptDesignSystem(framework string) string {
	fmt.Fprintln(r.out)
	fmt.Fprintln(r.out, cli.Heading("Design System"))
	fmt.Fprintln(r.out, strings.Repeat("\u2500", 50))
	fmt.Fprintln(r.out)

	for i, ds := range designSystemOptions {
		fmt.Fprintf(r.out, "  %d. %-16s %s\n", i+1, ds.Name, cli.Muted(ds.Desc))
	}
	fmt.Fprintf(r.out, "  %d. %-16s %s\n", len(designSystemOptions)+1, "Custom", cli.Muted("Specify colors and fonts"))
	fmt.Fprintln(r.out)

	fmt.Fprintf(r.out, "Selection [1-%d]: ", len(designSystemOptions)+1)
	answer, ok := r.scanLine()
	if !ok || answer == "" {
		return ""
	}

	n, err := strconv.Atoi(answer)
	if err != nil || n < 1 || n > len(designSystemOptions)+1 {
		fmt.Fprintln(r.errOut, cli.Error("Invalid selection."))
		return ""
	}

	// Custom option
	if n == len(designSystemOptions)+1 {
		return r.promptCustomDesignSystem()
	}

	selected := designSystemOptions[n-1]

	// Framework compatibility check
	if framework != "" {
		fw := strings.ToLower(framework)
		if compat, exists := frameworkCompat[selected.ID]; exists {
			if !compat[fw] {
				fmt.Fprintf(r.out, "\n%s\n",
					cli.Warn(fmt.Sprintf("%s doesn't natively support %s. Tailwind CSS will be used with %s's color palette.",
						selected.Name, framework, selected.Name)))
			}
		}
	}

	fmt.Fprintln(r.out, cli.Success(fmt.Sprintf("Selected: %s", selected.Name)))
	return selected.ID
}

// promptCustomDesignSystem asks for custom theme settings and returns "tailwind"
// as the base system (custom themes build on Tailwind utilities).
func (r *REPL) promptCustomDesignSystem() string {
	fmt.Fprintln(r.out)
	fmt.Fprintln(r.out, cli.Info("Custom theme settings:"))

	fmt.Fprintf(r.out, "  Primary color [#7F56D9]: ")
	color, _ := r.scanLine()
	if color == "" {
		color = "#7F56D9"
	}

	fmt.Fprintf(r.out, "  Font family [Inter]: ")
	font, _ := r.scanLine()
	if font == "" {
		font = "Inter"
	}

	fmt.Fprintf(r.out, "  Border radius (sharp/smooth/rounded/pill) [rounded]: ")
	radius, _ := r.scanLine()
	if radius == "" {
		radius = "rounded"
	}

	fmt.Fprintf(r.out, "  Spacing (compact/comfortable/spacious) [comfortable]: ")
	spacing, _ := r.scanLine()
	if spacing == "" {
		spacing = "comfortable"
	}

	fmt.Fprintln(r.out)
	fmt.Fprintln(r.out, cli.Success(fmt.Sprintf("Custom theme: %s, %s, %s, %s", color, font, radius, spacing)))
	fmt.Fprintln(r.out, cli.Muted("  Add these to your .human file's theme: block for full control."))

	// Return tailwind as the base; the custom values should be set in the .human theme block.
	return "tailwind"
}

// containsDesignSystem checks if a description already mentions a design system.
func containsDesignSystem(text string) bool {
	lower := strings.ToLower(text)
	keywords := []string{
		"design system", "material", "shadcn", "ant design", "chakra",
		"bootstrap", "tailwind", "untitled ui",
	}
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}
