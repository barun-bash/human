package cmdutil

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/barun-bash/human/internal/build"
	"github.com/barun-bash/human/internal/cli"
	"github.com/barun-bash/human/internal/ir"
	"github.com/barun-bash/human/internal/parser"
)

// PrintIRSummary displays a summary of the IR application to stdout.
func PrintIRSummary(app *ir.Application) {
	fmt.Println(cli.Info(fmt.Sprintf("  app:          %s (%s)", app.Name, app.Platform)))
	if app.Config != nil {
		fmt.Println(cli.Info(fmt.Sprintf("  config:       %s / %s / %s", app.Config.Frontend, app.Config.Backend, app.Config.Database)))
	}
	if len(app.Data) > 0 {
		fmt.Printf("  data models:  %d\n", len(app.Data))
	}
	if len(app.Pages) > 0 {
		fmt.Printf("  pages:        %d\n", len(app.Pages))
	}
	if len(app.Components) > 0 {
		fmt.Printf("  components:   %d\n", len(app.Components))
	}
	if len(app.APIs) > 0 {
		fmt.Printf("  APIs:         %d\n", len(app.APIs))
	}
	if len(app.Policies) > 0 {
		fmt.Printf("  policies:     %d\n", len(app.Policies))
	}
	if len(app.Workflows) > 0 {
		fmt.Printf("  workflows:    %d\n", len(app.Workflows))
	}
	if len(app.Pipelines) > 0 {
		fmt.Printf("  pipelines:    %d\n", len(app.Pipelines))
	}
	if app.Auth != nil && len(app.Auth.Methods) > 0 {
		fmt.Printf("  auth methods: %d\n", len(app.Auth.Methods))
	}
	if app.Database != nil {
		fmt.Printf("  database:     %s\n", app.Database.Engine)
	}
	if len(app.Integrations) > 0 {
		fmt.Printf("  integrations: %d\n", len(app.Integrations))
	}
	if len(app.Environments) > 0 {
		fmt.Printf("  environments: %d\n", len(app.Environments))
	}
	if app.Architecture != nil {
		fmt.Printf("  architecture: %s\n", app.Architecture.Style)
	}
	if len(app.Monitoring) > 0 {
		fmt.Printf("  monitoring:   %d rule(s)\n", len(app.Monitoring))
	}
}

// PrintBuildSummary displays a table of generator results.
func PrintBuildSummary(results []build.Result, outputDir string, timing *build.BuildTiming) {
	total := 0
	for _, r := range results {
		total += r.Files
	}

	fmt.Println()
	fmt.Println("  " + cli.Info("Build Summary"))
	fmt.Println("  " + strings.Repeat("─", 50))
	fmt.Printf("  %-14s %-8s %s\n", "Generator", "Files", "Output")
	fmt.Println("  " + strings.Repeat("─", 50))
	for _, r := range results {
		relDir := r.Dir
		if rel, err := filepath.Rel(".", r.Dir); err == nil {
			relDir = rel
		}
		fmt.Printf("  %-14s %-8d %s/\n", r.Name, r.Files, relDir)
	}
	fmt.Println("  " + strings.Repeat("─", 50))
	fmt.Printf("  %-14s %-8d\n", "Total", total)
	fmt.Println()
	if timing != nil {
		fmt.Println(cli.Success(fmt.Sprintf("Build complete — %d files in %s/ (%s)", total, outputDir, formatDuration(timing.Total))))
	} else {
		fmt.Println(cli.Success(fmt.Sprintf("Build complete — %d files in %s/", total, outputDir)))
	}
}

// PrintBuildSummaryTiming displays a detailed per-stage timing breakdown.
func PrintBuildSummaryTiming(results []build.Result, outputDir string, timing *build.BuildTiming) {
	total := 0
	for _, r := range results {
		total += r.Files
	}

	fmt.Println()
	fmt.Println("  " + cli.Info("Build Timing"))
	fmt.Println("  " + strings.Repeat("─", 40))
	for _, r := range results {
		fmt.Printf("  %-14s %3d files  %6s\n", r.Name, r.Files, formatDuration(r.Duration))
	}
	fmt.Println("  " + strings.Repeat("─", 40))
	if timing != nil {
		fmt.Printf("  %-14s %3d files  %6s\n", "Total", total, formatDuration(timing.Total))
	}
	fmt.Println()
	if timing != nil {
		fmt.Println(cli.Success(fmt.Sprintf("Build complete — %d files in %s/ (%s)", total, outputDir, formatDuration(timing.Total))))
	}
}

// formatDuration formats a duration as a human-readable string (e.g. "42ms", "1.2s").
func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

// PrintAuditReport reads and displays the security report with colorized output.
func PrintAuditReport(report string) {
	fmt.Println(cli.Info("Security & Quality Audit"))
	fmt.Println(cli.Info(strings.Repeat("─", 40)))
	fmt.Println()

	for _, line := range strings.Split(report, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.Contains(trimmed, "CRITICAL") || strings.Contains(trimmed, "\u274c"):
			fmt.Println(cli.Error(line))
		case strings.Contains(trimmed, "WARNING") || strings.Contains(trimmed, "\u26a0"):
			fmt.Println(cli.Warn(line))
		case strings.Contains(trimmed, "\u2705") || strings.Contains(trimmed, "PASS"):
			fmt.Println(cli.Success(line))
		case strings.HasPrefix(trimmed, "#"):
			fmt.Println(cli.Info(line))
		default:
			fmt.Println(line)
		}
	}
}

// CheckSummary returns a formatted summary of what was found in a parsed program.
func CheckSummary(prog *parser.Program, file string) string {
	var parts []string
	if len(prog.Data) > 0 {
		parts = append(parts, fmt.Sprintf("%d data model%s", len(prog.Data), Plural(len(prog.Data))))
	}
	if len(prog.Pages) > 0 {
		parts = append(parts, fmt.Sprintf("%d page%s", len(prog.Pages), Plural(len(prog.Pages))))
	}
	if len(prog.Components) > 0 {
		parts = append(parts, fmt.Sprintf("%d component%s", len(prog.Components), Plural(len(prog.Components))))
	}
	if len(prog.APIs) > 0 {
		parts = append(parts, fmt.Sprintf("%d API%s", len(prog.APIs), Plural(len(prog.APIs))))
	}
	if len(prog.Policies) > 0 {
		parts = append(parts, fmt.Sprintf("%d polic%s", len(prog.Policies), PluralY(len(prog.Policies))))
	}
	if len(prog.Workflows) > 0 {
		parts = append(parts, fmt.Sprintf("%d workflow%s", len(prog.Workflows), Plural(len(prog.Workflows))))
	}
	if len(prog.Integrations) > 0 {
		parts = append(parts, fmt.Sprintf("%d integration%s", len(prog.Integrations), Plural(len(prog.Integrations))))
	}
	if len(prog.Environments) > 0 {
		parts = append(parts, fmt.Sprintf("%d environment%s", len(prog.Environments), Plural(len(prog.Environments))))
	}
	if len(prog.ErrorHandlers) > 0 {
		parts = append(parts, fmt.Sprintf("%d error handler%s", len(prog.ErrorHandlers), Plural(len(prog.ErrorHandlers))))
	}

	msg := fmt.Sprintf("%s is valid", file)
	if len(parts) > 0 {
		msg += " — " + strings.Join(parts, ", ")
	}
	return msg
}

// Plural returns "s" for n != 1, empty string for n == 1.
func Plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// PluralY returns "y" for n == 1, "ies" for n != 1.
func PluralY(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}
