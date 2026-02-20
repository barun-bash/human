package quality

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/barun-bash/human/internal/ir"
)

// Result holds the output of the quality engine.
type Result struct {
	TestFiles       int
	TestCount       int
	SecurityFindings []Finding
	LintWarnings     []Warning
}

// Finding is a security audit finding.
type Finding struct {
	Severity string // "critical", "warning", "info"
	Category string // "auth", "validation", "secrets", "rate-limiting"
	Message  string
	Target   string // endpoint or model name
}

// Warning is a lint warning.
type Warning struct {
	Category string // "unused", "empty", "missing-validation", "empty-workflow"
	Message  string
	Target   string
}

// Run executes the full quality engine against the IR and writes output files.
func Run(app *ir.Application, outputDir string) (*Result, error) {
	result := &Result{}

	// 1. Generate tests
	testDir := filepath.Join(outputDir, "node", "src", "__tests__")
	testFiles, testCount, err := generateTests(app, testDir)
	if err != nil {
		return nil, fmt.Errorf("test generation: %w", err)
	}
	result.TestFiles = testFiles
	result.TestCount = testCount

	// 2. Security check
	result.SecurityFindings = checkSecurity(app)
	secReport := renderSecurityReport(app, result.SecurityFindings)
	if err := writeFile(filepath.Join(outputDir, "security-report.md"), secReport); err != nil {
		return nil, fmt.Errorf("security report: %w", err)
	}

	// 3. Lint check
	result.LintWarnings = checkLint(app)
	lintReport := renderLintReport(app, result.LintWarnings)
	if err := writeFile(filepath.Join(outputDir, "lint-report.md"), lintReport); err != nil {
		return nil, fmt.Errorf("lint report: %w", err)
	}

	// 4. Build summary
	summary := renderBuildSummary(app, outputDir, result)
	if err := writeFile(filepath.Join(outputDir, "build-report.md"), summary); err != nil {
		return nil, fmt.Errorf("build summary: %w", err)
	}

	return result, nil
}

// PrintSummary prints a one-line quality summary to stdout.
func PrintSummary(result *Result) {
	criticals := 0
	warnings := 0
	for _, f := range result.SecurityFindings {
		if f.Severity == "critical" {
			criticals++
		} else if f.Severity == "warning" {
			warnings++
		}
	}

	parts := []string{
		fmt.Sprintf("%d tests generated", result.TestCount),
	}

	if criticals > 0 {
		parts = append(parts, fmt.Sprintf("%d security critical", criticals))
	}
	if warnings > 0 {
		parts = append(parts, fmt.Sprintf("%d security warnings", warnings))
	}
	if len(result.LintWarnings) > 0 {
		parts = append(parts, fmt.Sprintf("%d lint warnings", len(result.LintWarnings)))
	}

	if criticals == 0 && warnings == 0 && len(result.LintWarnings) == 0 {
		parts = append(parts, "no issues")
	}

	fmt.Printf("  quality:      %s\n", strings.Join(parts, ", "))
}

func writeFile(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// toKebabCase converts PascalCase to kebab-case.
func toKebabCase(s string) string {
	var result []rune
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			result = append(result, '-')
		}
		result = append(result, unicode.ToLower(r))
	}
	return string(result)
}

// toCamelCase converts PascalCase to camelCase.
func toCamelCase(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}
