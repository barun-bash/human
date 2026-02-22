package mcp

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/barun-bash/human/internal/analyzer"
	"github.com/barun-bash/human/internal/build"
	"github.com/barun-bash/human/internal/ir"
	"github.com/barun-bash/human/internal/parser"
)

// handleBuild compiles .human source through the full pipeline.
func (s *Server) handleBuild(args json.RawMessage) *CallToolResult {
	var params struct {
		Source    string `json:"source"`
		OutputDir string `json:"output_dir"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return toolError("Invalid arguments: " + err.Error())
	}
	if params.Source == "" {
		return toolError("'source' is required.")
	}

	// Parse
	prog, err := parser.Parse(params.Source)
	if err != nil {
		return toolError("Parse error: " + err.Error())
	}

	// Build IR
	app, err := ir.Build(prog)
	if err != nil {
		return toolError("IR build error: " + err.Error())
	}

	// Semantic analysis
	errs := analyzer.Analyze(app, "input.human")
	if errs.HasErrors() {
		var diags []string
		for _, e := range errs.All() {
			d := fmt.Sprintf("[%s] %s", e.Code, e.Message)
			if e.Suggestion != "" {
				d += " — suggestion: " + e.Suggestion
			}
			diags = append(diags, d)
		}
		return toolError("Validation errors:\n" + strings.Join(diags, "\n"))
	}

	// Determine output directory
	outputDir := params.OutputDir
	if outputDir == "" {
		tmp, err := os.MkdirTemp("", "human-mcp-build-*")
		if err != nil {
			return toolError("Failed to create temp directory: " + err.Error())
		}
		outputDir = tmp

		// Track for human_read_file and cleanup
		s.mu.Lock()
		oldDir := s.lastBuildDir
		s.lastBuildDir = outputDir
		s.mu.Unlock()

		// Clean up previous temp build
		if oldDir != "" {
			os.RemoveAll(oldDir)
		}
	} else {
		s.mu.Lock()
		s.lastBuildDir = outputDir
		s.mu.Unlock()
	}

	// Run generators (skips quality.PrintSummary — that writes to stdout)
	results, qResult, err := build.RunGenerators(app, outputDir)
	if err != nil {
		return toolError("Build failed: " + err.Error())
	}

	// Build response
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Build successful: %s (%s)\n", app.Name, app.Platform))
	sb.WriteString(fmt.Sprintf("Output: %s\n\n", outputDir))

	// File manifest
	totalFiles := 0
	sb.WriteString("File manifest:\n")
	for _, r := range results {
		sb.WriteString(fmt.Sprintf("  %-14s %d files\n", r.Name, r.Files))
		totalFiles += r.Files
	}
	sb.WriteString(fmt.Sprintf("  %-14s %d files\n", "Total", totalFiles))

	// Quality summary
	if qResult != nil {
		totalTests := qResult.TestCount + qResult.ComponentTestCount + qResult.EdgeTestCount + qResult.IntegrationTestCount
		sb.WriteString(fmt.Sprintf("\nQuality: %d tests generated", totalTests))
		criticals := 0
		for _, f := range qResult.SecurityFindings {
			if f.Severity == "critical" {
				criticals++
			}
		}
		if criticals > 0 {
			sb.WriteString(fmt.Sprintf(", %d security critical", criticals))
		}
		if len(qResult.LintWarnings) > 0 {
			sb.WriteString(fmt.Sprintf(", %d lint warnings", len(qResult.LintWarnings)))
		}
		sb.WriteString("\n")
	}

	// Warnings from analysis
	if errs.HasWarnings() {
		sb.WriteString("\nWarnings:\n")
		for _, w := range errs.Warnings() {
			sb.WriteString(fmt.Sprintf("  [%s] %s\n", w.Code, w.Message))
		}
	}

	// Read key files to include in response
	keyFiles := []string{
		"package.json",
		"docker-compose.yml",
		"start.sh",
	}
	for _, kf := range keyFiles {
		fullPath := filepath.Join(outputDir, kf)
		content, err := os.ReadFile(fullPath)
		if err == nil {
			sb.WriteString(fmt.Sprintf("\n--- %s ---\n%s\n", kf, string(content)))
		}
	}

	return toolText(sb.String())
}

// handleValidate validates .human source without code generation.
func (s *Server) handleValidate(args json.RawMessage) *CallToolResult {
	var params struct {
		Source string `json:"source"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return toolError("Invalid arguments: " + err.Error())
	}
	if params.Source == "" {
		return toolError("'source' is required.")
	}

	// Parse
	prog, err := parser.Parse(params.Source)
	if err != nil {
		return toolError("Parse error: " + err.Error())
	}

	// Build IR
	app, err := ir.Build(prog)
	if err != nil {
		return toolError("IR build error: " + err.Error())
	}

	// Semantic analysis
	errs := analyzer.Analyze(app, "input.human")

	var sb strings.Builder

	if !errs.HasErrors() && !errs.HasWarnings() {
		sb.WriteString("Valid .human source — no issues found.\n\n")
		// Summarize what was parsed
		if len(app.Data) > 0 {
			sb.WriteString(fmt.Sprintf("Data models: %d\n", len(app.Data)))
		}
		if len(app.Pages) > 0 {
			sb.WriteString(fmt.Sprintf("Pages: %d\n", len(app.Pages)))
		}
		if len(app.Components) > 0 {
			sb.WriteString(fmt.Sprintf("Components: %d\n", len(app.Components)))
		}
		if len(app.APIs) > 0 {
			sb.WriteString(fmt.Sprintf("APIs: %d\n", len(app.APIs)))
		}
		if len(app.Policies) > 0 {
			sb.WriteString(fmt.Sprintf("Policies: %d\n", len(app.Policies)))
		}
		if len(app.Workflows) > 0 {
			sb.WriteString(fmt.Sprintf("Workflows: %d\n", len(app.Workflows)))
		}
		return toolText(sb.String())
	}

	// Report diagnostics
	allErrs := errs.All()
	sb.WriteString(fmt.Sprintf("Found %d diagnostic(s):\n\n", len(allErrs)))
	for _, e := range allErrs {
		severity := "error"
		switch e.Severity {
		case 1:
			severity = "warning"
		case 2:
			severity = "hint"
		}
		sb.WriteString(fmt.Sprintf("[%s] %s: %s", e.Code, severity, e.Message))
		if e.Line > 0 {
			sb.WriteString(fmt.Sprintf(" (line %d)", e.Line))
		}
		if e.Suggestion != "" {
			sb.WriteString(fmt.Sprintf("\n  suggestion: %s", e.Suggestion))
		}
		sb.WriteString("\n")
	}

	if errs.HasErrors() {
		return &CallToolResult{
			Content: []ContentItem{{Type: "text", Text: sb.String()}},
			IsError: true,
		}
	}
	return toolText(sb.String())
}

// handleIR parses .human source and returns the Intent IR as YAML.
func (s *Server) handleIR(args json.RawMessage) *CallToolResult {
	var params struct {
		Source string `json:"source"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return toolError("Invalid arguments: " + err.Error())
	}
	if params.Source == "" {
		return toolError("'source' is required.")
	}

	prog, err := parser.Parse(params.Source)
	if err != nil {
		return toolError("Parse error: " + err.Error())
	}

	app, err := ir.Build(prog)
	if err != nil {
		return toolError("IR build error: " + err.Error())
	}

	yaml, err := ir.ToYAML(app)
	if err != nil {
		return toolError("YAML serialization error: " + err.Error())
	}

	return toolText(yaml)
}

// handleExamples lists examples or returns a specific example's source.
func (s *Server) handleExamples(args json.RawMessage) *CallToolResult {
	var params struct {
		Name string `json:"name"`
	}
	if args != nil {
		json.Unmarshal(args, &params)
	}

	if params.Name != "" {
		source, ok := s.examples[params.Name]
		if !ok {
			// Try with/without common suffixes
			for name, src := range s.examples {
				if strings.EqualFold(name, params.Name) {
					return toolText(fmt.Sprintf("# Example: %s\n\n%s", name, src))
				}
			}
			var names []string
			for name := range s.examples {
				names = append(names, name)
			}
			sort.Strings(names)
			return toolError(fmt.Sprintf("Example %q not found. Available: %s", params.Name, strings.Join(names, ", ")))
		}
		return toolText(fmt.Sprintf("# Example: %s\n\n%s", params.Name, source))
	}

	// List all examples
	var names []string
	for name := range s.examples {
		names = append(names, name)
	}
	sort.Strings(names)

	var sb strings.Builder
	sb.WriteString("Available examples:\n\n")
	for _, name := range names {
		source := s.examples[name]
		// Extract first line as description
		desc := name
		lines := strings.SplitN(source, "\n", 2)
		if len(lines) > 0 {
			desc = strings.TrimSpace(lines[0])
		}
		sb.WriteString(fmt.Sprintf("  %-20s %s\n", name, desc))
	}
	sb.WriteString("\nUse human_examples with name parameter to view source.")
	return toolText(sb.String())
}

// handleSpec returns the embedded language specification.
func (s *Server) handleSpec(args json.RawMessage) *CallToolResult {
	if s.spec == "" {
		return toolError("Language specification not available.")
	}
	return toolText(s.spec)
}

// handleReadFile reads a file from the last build output.
func (s *Server) handleReadFile(args json.RawMessage) *CallToolResult {
	var params struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return toolError("Invalid arguments: " + err.Error())
	}
	if params.Path == "" {
		return toolError("'path' is required.")
	}

	s.mu.Lock()
	buildDir := s.lastBuildDir
	s.mu.Unlock()

	if buildDir == "" {
		return toolError("No build output available. Run human_build first.")
	}

	// Path traversal protection
	cleaned := filepath.Clean(params.Path)
	if filepath.IsAbs(cleaned) || strings.HasPrefix(cleaned, "..") {
		return toolError("Invalid path: must be a relative path within the build output.")
	}

	fullPath := filepath.Join(buildDir, cleaned)
	// Verify it's still under buildDir after resolution
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return toolError("Invalid path: " + err.Error())
	}
	absBuildDir, _ := filepath.Abs(buildDir)
	if !strings.HasPrefix(absPath, absBuildDir+string(filepath.Separator)) && absPath != absBuildDir {
		return toolError("Invalid path: must be within the build output directory.")
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		// If file not found, list available files
		if os.IsNotExist(err) {
			var available []string
			filepath.WalkDir(buildDir, func(path string, d fs.DirEntry, err error) error {
				if err == nil && !d.IsDir() {
					rel, _ := filepath.Rel(buildDir, path)
					available = append(available, rel)
				}
				return nil
			})
			sort.Strings(available)
			msg := fmt.Sprintf("File not found: %s\n\nAvailable files:\n", params.Path)
			limit := 50
			for i, f := range available {
				if i >= limit {
					msg += fmt.Sprintf("  ... and %d more\n", len(available)-limit)
					break
				}
				msg += "  " + f + "\n"
			}
			return toolError(msg)
		}
		return toolError("Error reading file: " + err.Error())
	}

	if info.IsDir() {
		// List directory contents
		var entries []string
		filepath.WalkDir(fullPath, func(path string, d fs.DirEntry, err error) error {
			if err == nil && path != fullPath {
				rel, _ := filepath.Rel(fullPath, path)
				if d.IsDir() {
					entries = append(entries, rel+"/")
				} else {
					entries = append(entries, rel)
				}
			}
			return nil
		})
		sort.Strings(entries)
		return toolText(fmt.Sprintf("Directory: %s\n\n%s", params.Path, strings.Join(entries, "\n")))
	}

	// Cap at 100KB
	const maxSize = 100 * 1024
	if info.Size() > maxSize {
		content := make([]byte, maxSize)
		f, err := os.Open(fullPath)
		if err != nil {
			return toolError("Error reading file: " + err.Error())
		}
		defer f.Close()
		n, _ := f.Read(content)
		return toolText(string(content[:n]) + fmt.Sprintf("\n\n[Truncated at 100KB — file is %d bytes]", info.Size()))
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return toolError("Error reading file: " + err.Error())
	}
	return toolText(string(content))
}

// toolText creates a successful text result.
func toolText(text string) *CallToolResult {
	return &CallToolResult{
		Content: []ContentItem{{Type: "text", Text: text}},
	}
}

// toolError creates an error text result.
func toolError(text string) *CallToolResult {
	return &CallToolResult{
		Content: []ContentItem{{Type: "text", Text: text}},
		IsError: true,
	}
}
