package quality

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
// Test generation, security, and lint stages run in parallel where possible.
func Run(app *ir.Application, outputDir string) (*Result, error) {
	result := &Result{}
	testDir := filepath.Join(outputDir, "node", "src", "__tests__")

	// Group 1: Generate all test types in parallel (they write to separate files).
	var mu sync.Mutex
	var firstErr error
	var wg sync.WaitGroup

	setErr := func(err error) {
		mu.Lock()
		if firstErr == nil {
			firstErr = err
		}
		mu.Unlock()
	}

	wg.Add(4)
	go func() {
		defer wg.Done()
		testFiles, testCount, err := generateTests(app, testDir)
		if err != nil {
			setErr(fmt.Errorf("test generation: %w", err))
			return
		}
		mu.Lock()
		result.TestFiles = testFiles
		result.TestCount = testCount
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		compFiles, compCount, err := generateComponentTests(app, testDir)
		if err != nil {
			setErr(fmt.Errorf("component test generation: %w", err))
			return
		}
		mu.Lock()
		result.ComponentTestFiles = compFiles
		result.ComponentTestCount = compCount
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		edgeFiles, edgeCount, err := generateEdgeTests(app, testDir)
		if err != nil {
			setErr(fmt.Errorf("edge test generation: %w", err))
			return
		}
		mu.Lock()
		result.EdgeTestFiles = edgeFiles
		result.EdgeTestCount = edgeCount
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		integCount, err := generateIntegrationTests(app, testDir)
		if err != nil {
			setErr(fmt.Errorf("integration test generation: %w", err))
			return
		}
		mu.Lock()
		result.IntegrationTestCount = integCount
		mu.Unlock()
	}()
	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}

	// Group 2: Security and lint checks in parallel (read-only on app, write to separate files).
	wg.Add(2)
	go func() {
		defer wg.Done()
		findings := checkSecurity(app)
		secReport := renderSecurityReport(app, findings)
		if err := writeFile(filepath.Join(outputDir, "security-report.md"), secReport); err != nil {
			setErr(fmt.Errorf("security report: %w", err))
			return
		}
		mu.Lock()
		result.SecurityFindings = findings
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		warnings := checkLint(app)
		lintReport := renderLintReport(app, warnings)
		if err := writeFile(filepath.Join(outputDir, "lint-report.md"), lintReport); err != nil {
			setErr(fmt.Errorf("lint report: %w", err))
			return
		}
		mu.Lock()
		result.LintWarnings = warnings
		mu.Unlock()
	}()
	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}

	// Group 3: Sequential — coverage and summary depend on all prior results.
	result.Coverage = calculateCoverage(app, result)

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
