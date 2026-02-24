package cmdutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/barun-bash/human/internal/analyzer"
	"github.com/barun-bash/human/internal/build"
	"github.com/barun-bash/human/internal/cli"
	cerr "github.com/barun-bash/human/internal/errors"
	"github.com/barun-bash/human/internal/ir"
	"github.com/barun-bash/human/internal/parser"
	"github.com/barun-bash/human/internal/quality"
)

// ParseResult holds output from the parse-analyze pipeline.
type ParseResult struct {
	Prog *parser.Program
	App  *ir.Application
	Errs *cerr.CompilerErrors
}

// ParseAndAnalyze reads a .human file, parses it, builds the IR,
// and runs semantic analysis. Returns the combined result.
func ParseAndAnalyze(file string) (*ParseResult, error) {
	source, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", file, err)
	}

	prog, err := parser.Parse(string(source))
	if err != nil {
		return nil, fmt.Errorf("parse error in %s: %w", file, err)
	}

	app, err := ir.Build(prog)
	if err != nil {
		return nil, fmt.Errorf("IR build error: %w", err)
	}

	errs := analyzer.Analyze(app, file)
	return &ParseResult{Prog: prog, App: app, Errs: errs}, nil
}

// PrintDiagnostics prints all warnings and errors from a CompilerErrors
// collection. Returns true if errors exist.
func PrintDiagnostics(errs *cerr.CompilerErrors) bool {
	if errs.HasWarnings() {
		for _, w := range errs.Warnings() {
			PrintDiagnostic(w)
		}
	}
	if errs.HasErrors() {
		for _, e := range errs.Errors() {
			PrintDiagnostic(e)
		}
		return true
	}
	return false
}

// PrintDiagnostic prints a single CompilerError with its suggestion to stderr.
func PrintDiagnostic(e *cerr.CompilerError) {
	switch e.Severity {
	case cerr.SeverityWarning:
		fmt.Fprintln(os.Stderr, cli.Warn(e.Format()))
	default:
		fmt.Fprintln(os.Stderr, cli.Error(e.Format()))
	}
	if e.Suggestion != "" {
		fmt.Fprintf(os.Stderr, "  suggestion: %s\n", e.Suggestion)
	}
}

// FullBuild runs the complete build pipeline: parse, analyze, generate IR YAML,
// run code generators, and the quality engine. Returns the application IR,
// generator results, and quality result.
func FullBuild(file string) (*ir.Application, []build.Result, *quality.Result, error) {
	result, err := ParseAndAnalyze(file)
	if err != nil {
		return nil, nil, nil, err
	}

	if PrintDiagnostics(result.Errs) {
		return nil, nil, nil, fmt.Errorf("%d error(s) found", len(result.Errs.Errors()))
	}

	yaml, err := ir.ToYAML(result.App)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("serialization error: %w", err)
	}

	// Write IR to .human/intent/<name>.yaml
	outDir := filepath.Join(".human", "intent")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, nil, nil, fmt.Errorf("creating output directory: %w", err)
	}

	base := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	outFile := filepath.Join(outDir, base+".yaml")
	if err := os.WriteFile(outFile, []byte(yaml), 0644); err != nil {
		return nil, nil, nil, fmt.Errorf("writing %s: %w", outFile, err)
	}

	fmt.Printf("Built %s → %s\n", file, outFile)
	PrintIRSummary(result.App)

	// Run all code generators
	outputDir := filepath.Join(".human", "output")
	results, qResult, genErr := build.RunGenerators(result.App, outputDir)
	if genErr != nil {
		return nil, nil, nil, fmt.Errorf("build failed: %w", genErr)
	}

	quality.PrintSummary(qResult)
	PrintBuildSummary(results, outputDir)

	return result.App, results, qResult, nil
}
