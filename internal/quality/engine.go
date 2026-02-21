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
	TestFiles            int
	TestCount            int
	SecurityFindings     []Finding
	LintWarnings         []Warning
	ComponentTestFiles   int
	ComponentTestCount   int
	EdgeTestFiles        int
	EdgeTestCount        int
	IntegrationTestCount int
	Coverage             *CoverageReport
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

	// 1. Generate API tests
	testDir := filepath.Join(outputDir, "node", "src", "__tests__")
	testFiles, testCount, err := generateTests(app, testDir)
	if err != nil {
		return nil, fmt.Errorf("test generation: %w", err)
	}
	result.TestFiles = testFiles
	result.TestCount = testCount

	// 2. Generate component tests
	compFiles, compCount, err := generateComponentTests(app, testDir)
	if err != nil {
		return nil, fmt.Errorf("component test generation: %w", err)
	}
	result.ComponentTestFiles = compFiles
	result.ComponentTestCount = compCount

	// 3. Generate edge case tests
	edgeFiles, edgeCount, err := generateEdgeTests(app, testDir)
	if err != nil {
		return nil, fmt.Errorf("edge test generation: %w", err)
	}
	result.EdgeTestFiles = edgeFiles
	result.EdgeTestCount = edgeCount

	// 4. Generate integration tests
	integCount, err := generateIntegrationTests(app, testDir)
	if err != nil {
		return nil, fmt.Errorf("integration test generation: %w", err)
	}
	result.IntegrationTestCount = integCount

	// 5. Security check
	result.SecurityFindings = checkSecurity(app)
	secReport := renderSecurityReport(app, result.SecurityFindings)
	if err := writeFile(filepath.Join(outputDir, "security-report.md"), secReport); err != nil {
		return nil, fmt.Errorf("security report: %w", err)
	}

	// 6. Lint check
	result.LintWarnings = checkLint(app)
	lintReport := renderLintReport(app, result.LintWarnings)
	if err := writeFile(filepath.Join(outputDir, "lint-report.md"), lintReport); err != nil {
		return nil, fmt.Errorf("lint report: %w", err)
	}

	// 7. Coverage
	result.Coverage = calculateCoverage(app, result)

	// 8. Build summary
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

	totalTests := result.TestCount + result.ComponentTestCount + result.EdgeTestCount + result.IntegrationTestCount

	parts := []string{
		fmt.Sprintf("%d tests generated", totalTests),
	}

	if result.Coverage != nil {
		parts = append(parts, fmt.Sprintf("%.0f%% coverage", result.Coverage.Overall))
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
