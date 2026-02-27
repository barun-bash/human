package fixer

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/barun-bash/human/internal/analyzer"
	cerr "github.com/barun-bash/human/internal/errors"
	"github.com/barun-bash/human/internal/ir"
	"github.com/barun-bash/human/internal/parser"
)

// Fix describes a suggested change to a .human file.
type Fix struct {
	File        string // which .human file
	Line        int    // line number (0 = append to section)
	Code        string // warning/error code
	Description string // human-readable description
	Suggestion  string // the .human code to insert/replace
	Kind        string // "insert" or "append"
}

// Result holds analysis output: existing diagnostics plus auto-fixable suggestions.
type Result struct {
	Errors   []*cerr.CompilerError
	Warnings []*cerr.CompilerError
	Fixes    []Fix
}

var (
	fetchPattern   = regexp.MustCompile(`(?i)\b(fetch|load|get|show a list of)\b`)
	loadingPattern = regexp.MustCompile(`(?i)\bwhile loading\b`)
	listPattern    = regexp.MustCompile(`(?i)\bshow a list of\b`)
	emptyPattern   = regexp.MustCompile(`(?i)\bif no\b.*\bmatch\b`)
	formPattern    = regexp.MustCompile(`(?i)\bform to (create|edit|update)\b`)
	errorPattern   = regexp.MustCompile(`(?i)\bif there is an error\b`)
	crudModify     = regexp.MustCompile(`(?i)\b(create|update|delete)\b`)
)

// Analyze reads a .human file, runs the standard analyzer plus additional
// fixable checks, and returns all diagnostics and suggested fixes.
func Analyze(files []string) (*Result, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("no files specified")
	}

	result := &Result{}

	for _, file := range files {
		if err := analyzeFile(file, result); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func analyzeFile(file string, result *Result) error {
	source, err := readFile(file)
	if err != nil {
		return fmt.Errorf("reading %s: %w", file, err)
	}

	prog, err := parser.Parse(source)
	if err != nil {
		return fmt.Errorf("parse error in %s: %w", file, err)
	}

	app, err := ir.Build(prog)
	if err != nil {
		return fmt.Errorf("IR build error in %s: %w", file, err)
	}

	// Run standard analyzer.
	errs := analyzer.Analyze(app, file)
	result.Errors = append(result.Errors, errs.Errors()...)
	result.Warnings = append(result.Warnings, errs.Warnings()...)

	// Run fixable checks.
	checkLoadingStates(app, file, result)
	checkEmptyStates(app, file, result)
	checkFormErrorStates(app, file, source, result)
	checkAPIAuth(app, file, result)
	checkMissingIndexes(app, file, result)

	return nil
}

// W601: Page fetches data but no "while loading".
func checkLoadingStates(app *ir.Application, file string, result *Result) {
	for _, page := range app.Pages {
		hasFetch := false
		hasLoading := false
		for _, action := range page.Content {
			if fetchPattern.MatchString(action.Text) {
				hasFetch = true
			}
			if loadingPattern.MatchString(action.Text) {
				hasLoading = true
			}
		}
		if hasFetch && !hasLoading {
			result.Warnings = append(result.Warnings,
				&cerr.CompilerError{
					Code:     "W601",
					Message:  fmt.Sprintf("Page %q fetches data but has no loading state", page.Name),
					Severity: cerr.SeverityWarning,
				})
			result.Fixes = append(result.Fixes, Fix{
				File:        file,
				Code:        "W601",
				Description: fmt.Sprintf("Add loading state to page %s", page.Name),
				Suggestion:  fmt.Sprintf("  while loading, show a spinner"),
				Kind:        "append",
			})
		}
	}
}

// W602: Page shows list but no empty state.
func checkEmptyStates(app *ir.Application, file string, result *Result) {
	for _, page := range app.Pages {
		hasList := false
		hasEmpty := false
		for _, action := range page.Content {
			if listPattern.MatchString(action.Text) {
				hasList = true
			}
			if emptyPattern.MatchString(action.Text) {
				hasEmpty = true
			}
		}
		if hasList && !hasEmpty {
			result.Warnings = append(result.Warnings,
				&cerr.CompilerError{
					Code:     "W602",
					Message:  fmt.Sprintf("Page %q shows a list but has no empty state", page.Name),
					Severity: cerr.SeverityWarning,
				})
			result.Fixes = append(result.Fixes, Fix{
				File:        file,
				Code:        "W602",
				Description: fmt.Sprintf("Add empty state to page %s", page.Name),
				Suggestion:  fmt.Sprintf(`  if no items match, show "No items found"`),
				Kind:        "append",
			})
		}
	}
}

// W603: Form without error display.
func checkFormErrorStates(app *ir.Application, file string, source string, result *Result) {
	for _, page := range app.Pages {
		hasForm := false
		hasError := false
		for _, action := range page.Content {
			if formPattern.MatchString(action.Text) {
				hasForm = true
			}
			if errorPattern.MatchString(action.Text) {
				hasError = true
			}
		}
		if hasForm && !hasError {
			result.Warnings = append(result.Warnings,
				&cerr.CompilerError{
					Code:     "W603",
					Message:  fmt.Sprintf("Page %q has a form but no error display", page.Name),
					Severity: cerr.SeverityWarning,
				})
			result.Fixes = append(result.Fixes, Fix{
				File:        file,
				Code:        "W603",
				Description: fmt.Sprintf("Add error display to page %s", page.Name),
				Suggestion:  "  if there is an error, show the error message",
				Kind:        "append",
			})
		}
	}
}

// W604: API modifies data without auth.
func checkAPIAuth(app *ir.Application, file string, result *Result) {
	for _, api := range app.APIs {
		if api.Auth {
			continue
		}
		modifiesData := false
		for _, step := range api.Steps {
			if crudModify.MatchString(step.Text) {
				modifiesData = true
				break
			}
		}
		if modifiesData {
			result.Warnings = append(result.Warnings,
				&cerr.CompilerError{
					Code:     "W604",
					Message:  fmt.Sprintf("API %q modifies data but does not require authentication", api.Name),
					Severity: cerr.SeverityWarning,
				})
			result.Fixes = append(result.Fixes, Fix{
				File:        file,
				Code:        "W604",
				Description: fmt.Sprintf("Add authentication to API %s", api.Name),
				Suggestion:  "  requires authentication",
				Kind:        "insert",
			})
		}
	}
}

// W605: Queried data with no index.
func checkMissingIndexes(app *ir.Application, file string, result *Result) {
	if app.Database == nil {
		return
	}

	// Build set of indexed fields per model.
	indexed := make(map[string]map[string]bool)
	for _, idx := range app.Database.Indexes {
		key := strings.ToLower(idx.Entity)
		if indexed[key] == nil {
			indexed[key] = make(map[string]bool)
		}
		for _, f := range idx.Fields {
			indexed[key][strings.ToLower(f)] = true
		}
	}

	// Check APIs for fetch-by-field patterns.
	fetchByField := regexp.MustCompile(`(?i)fetch\s+(?:the\s+)?(\w+)\s+by\s+(\w+)`)
	for _, api := range app.APIs {
		for _, step := range api.Steps {
			matches := fetchByField.FindStringSubmatch(step.Text)
			if len(matches) < 3 {
				continue
			}
			entity := strings.ToLower(matches[1])
			field := strings.ToLower(matches[2])

			if m, ok := indexed[entity]; ok && m[field] {
				continue
			}

			result.Warnings = append(result.Warnings,
				&cerr.CompilerError{
					Code:     "W605",
					Message:  fmt.Sprintf("API %q fetches %s by %s but no index exists", api.Name, matches[1], matches[2]),
					Severity: cerr.SeverityWarning,
				})
			result.Fixes = append(result.Fixes, Fix{
				File:        file,
				Code:        "W605",
				Description: fmt.Sprintf("Add index on %s.%s", matches[1], matches[2]),
				Suggestion:  fmt.Sprintf("  index %s by %s", matches[1], matches[2]),
				Kind:        "append",
			})
		}
	}
}
