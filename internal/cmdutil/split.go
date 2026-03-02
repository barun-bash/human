package cmdutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/barun-bash/human/internal/parser"
)

// SplitProgram splits a parsed Program into concern-based files.
// Returns filename → rendered .human content. Only includes files with content.
//
// File mapping:
//   - app.human:          App, Build
//   - frontend.human:     Pages, Components, Theme
//   - backend.human:      Data, APIs, Authentication, Database, Policies, ErrorHandlers
//   - devops.human:       Architecture, Environments, CI/CD Workflows, top-level Statements
//   - integrations.human: Integrations, business Workflows
func SplitProgram(prog *parser.Program) map[string]string {
	files := make(map[string]string)

	// Classify workflows into devops vs business.
	var devopsWorkflows, businessWorkflows []*parser.WorkflowDeclaration
	for _, w := range prog.Workflows {
		if isDevOpsWorkflow(w.Event) {
			devopsWorkflows = append(devopsWorkflows, w)
		} else {
			businessWorkflows = append(businessWorkflows, w)
		}
	}

	// app.human — App declaration + Build
	if prog.App != nil || prog.Build != nil {
		files["app.human"] = renderAppFile(prog)
	}

	// frontend.human — Pages, Components, Theme
	if len(prog.Pages) > 0 || len(prog.Components) > 0 || prog.Theme != nil {
		files["frontend.human"] = renderFrontendFile(prog)
	}

	// backend.human — Data, APIs, Authentication, Database, Policies, ErrorHandlers
	if len(prog.Data) > 0 || len(prog.APIs) > 0 || prog.Authentication != nil ||
		prog.Database != nil || len(prog.Policies) > 0 || len(prog.ErrorHandlers) > 0 {
		files["backend.human"] = renderBackendFile(prog)
	}

	// devops.human — Architecture, Environments, CI/CD Workflows, top-level Statements
	if prog.Architecture != nil || len(prog.Environments) > 0 ||
		len(devopsWorkflows) > 0 || len(prog.Statements) > 0 {
		files["devops.human"] = renderDevOpsFile(prog, devopsWorkflows)
	}

	// integrations.human — Integrations, business Workflows
	if len(prog.Integrations) > 0 || len(businessWorkflows) > 0 {
		files["integrations.human"] = renderIntegrationsFile(prog, businessWorkflows)
	}

	return files
}

// SplitToDir splits a program into concern-based files and writes them to outputDir.
// Creates the directory if it doesn't exist. Returns the list of created file paths.
func SplitToDir(prog *parser.Program, outputDir string) ([]string, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("creating directory %s: %w", outputDir, err)
	}

	files := SplitProgram(prog)

	var created []string
	for name, content := range files {
		path := filepath.Join(outputDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return created, fmt.Errorf("writing %s: %w", path, err)
		}
		created = append(created, path)
	}

	// Write .gitignore.
	gitignorePath := filepath.Join(outputDir, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		os.WriteFile(gitignorePath, []byte(generateGitignore()), 0644) // non-fatal
	}

	return created, nil
}

// isDevOpsWorkflow returns true if the workflow event is CI/CD-related.
func isDevOpsWorkflow(event string) bool {
	lower := strings.ToLower(event)
	return strings.Contains(lower, "code is pushed") ||
		strings.Contains(lower, "code is merged") ||
		strings.Contains(lower, "pull request") ||
		strings.Contains(lower, "branch")
}

// ── Renderers ──

func renderAppFile(prog *parser.Program) string {
	var b strings.Builder
	b.WriteString("# App declaration and build configuration\n\n")

	if prog.App != nil {
		fmt.Fprintf(&b, "app %s is a %s application\n", prog.App.Name, prog.App.Platform)
	}

	if prog.Build != nil {
		b.WriteString("\nbuild with:\n")
		for _, s := range prog.Build.Statements {
			fmt.Fprintf(&b, "  %s\n", renderStatement(s.Text))
		}
	}

	return b.String()
}

func renderFrontendFile(prog *parser.Program) string {
	var b strings.Builder
	b.WriteString("# Frontend: pages, components, and theme\n")

	if prog.Theme != nil {
		b.WriteString("\ntheme:\n")
		for _, s := range prog.Theme.Properties {
			fmt.Fprintf(&b, "  %s\n", renderStatement(s.Text))
		}
	}

	for _, page := range prog.Pages {
		fmt.Fprintf(&b, "\npage %s:\n", page.Name)
		for _, s := range page.Statements {
			fmt.Fprintf(&b, "  %s\n", renderStatement(s.Text))
		}
	}

	for _, comp := range prog.Components {
		fmt.Fprintf(&b, "\ncomponent %s:\n", comp.Name)
		for _, acc := range comp.Accepts {
			fmt.Fprintf(&b, "  accepts %s\n", acc)
		}
		for _, s := range comp.Statements {
			fmt.Fprintf(&b, "  %s\n", renderStatement(s.Text))
		}
	}

	return b.String()
}

func renderBackendFile(prog *parser.Program) string {
	var b strings.Builder
	b.WriteString("# Backend: data models, APIs, auth, database, policies, error handlers\n")

	for _, data := range prog.Data {
		fmt.Fprintf(&b, "\ndata %s:\n", data.Name)
		for _, rel := range data.Relationships {
			if rel.Kind == "belongs_to" {
				fmt.Fprintf(&b, "  belongs to a %s\n", rel.Target)
			}
		}
		for _, f := range data.Fields {
			b.WriteString("  ")
			b.WriteString(renderField(f))
			b.WriteString("\n")
		}
		for _, rel := range data.Relationships {
			if rel.Kind == "has_many" {
				if rel.Through != "" {
					fmt.Fprintf(&b, "  has many %s through %s\n", rel.Target, rel.Through)
				} else {
					fmt.Fprintf(&b, "  has many %s\n", rel.Target)
				}
			}
		}
	}

	for _, api := range prog.APIs {
		fmt.Fprintf(&b, "\napi %s:\n", api.Name)
		if api.Auth {
			b.WriteString("  requires authentication\n")
		}
		if len(api.Accepts) > 0 {
			fmt.Fprintf(&b, "  accepts %s\n", renderAccepts(api.Accepts))
		}
		for _, s := range api.Statements {
			// Skip statements already covered by Auth/Accepts fields.
			lower := strings.ToLower(s.Text)
			if lower == "requires authentication" {
				continue
			}
			if strings.HasPrefix(lower, "accepts ") {
				continue
			}
			fmt.Fprintf(&b, "  %s\n", renderStatement(s.Text))
		}
	}

	if prog.Authentication != nil {
		b.WriteString("\nauthentication:\n")
		for _, s := range prog.Authentication.Statements {
			fmt.Fprintf(&b, "  %s\n", renderStatement(s.Text))
		}
	}

	if prog.Database != nil {
		b.WriteString("\ndatabase:\n")
		for _, s := range prog.Database.Statements {
			fmt.Fprintf(&b, "  %s\n", renderStatement(s.Text))
		}
	}

	for _, pol := range prog.Policies {
		fmt.Fprintf(&b, "\npolicy %s:\n", pol.Name)
		for _, r := range pol.Rules {
			if r.Allowed {
				fmt.Fprintf(&b, "  can %s\n", r.Text)
			} else {
				fmt.Fprintf(&b, "  cannot %s\n", r.Text)
			}
		}
	}

	for _, eh := range prog.ErrorHandlers {
		fmt.Fprintf(&b, "\nif %s:\n", eh.Condition)
		for _, s := range eh.Statements {
			fmt.Fprintf(&b, "  %s\n", renderStatement(s.Text))
		}
	}

	return b.String()
}

func renderDevOpsFile(prog *parser.Program, devopsWorkflows []*parser.WorkflowDeclaration) string {
	var b strings.Builder
	b.WriteString("# DevOps: architecture, CI/CD, environments, monitoring\n")

	if prog.Architecture != nil {
		fmt.Fprintf(&b, "\narchitecture: %s\n", prog.Architecture.Style)
		for _, s := range prog.Architecture.Statements {
			fmt.Fprintf(&b, "  %s\n", renderStatement(s.Text))
		}
	}

	// Top-level statements (monitoring, source control, etc.) come before workflows.
	if len(prog.Statements) > 0 {
		b.WriteString("\n")
		for _, s := range prog.Statements {
			fmt.Fprintf(&b, "%s\n", renderStatement(s.Text))
		}
	}

	for _, w := range devopsWorkflows {
		fmt.Fprintf(&b, "\nwhen %s:\n", w.Event)
		for _, s := range w.Statements {
			fmt.Fprintf(&b, "  %s\n", renderStatement(s.Text))
		}
	}

	for _, env := range prog.Environments {
		fmt.Fprintf(&b, "\nenvironment %s:\n", env.Name)
		for _, s := range env.Statements {
			fmt.Fprintf(&b, "  %s\n", renderStatement(s.Text))
		}
	}

	return b.String()
}

func renderIntegrationsFile(prog *parser.Program, businessWorkflows []*parser.WorkflowDeclaration) string {
	var b strings.Builder
	b.WriteString("# Integrations and business workflows\n")

	for _, integ := range prog.Integrations {
		fmt.Fprintf(&b, "\nintegrate with %s:\n", integ.Service)
		for _, s := range integ.Statements {
			fmt.Fprintf(&b, "  %s\n", renderStatement(s.Text))
		}
	}

	for _, w := range businessWorkflows {
		fmt.Fprintf(&b, "\nwhen %s:\n", w.Event)
		for _, s := range w.Statements {
			fmt.Fprintf(&b, "  %s\n", renderStatement(s.Text))
		}
	}

	return b.String()
}

// ── Field/Accept Rendering ──

func renderField(f *parser.Field) string {
	// Enum field: has a <name> which is either "val1" or "val2"
	if len(f.EnumValues) > 0 {
		article := fieldArticle(f.Name)
		quoted := make([]string, len(f.EnumValues))
		for i, v := range f.EnumValues {
			quoted[i] = fmt.Sprintf("%q", v)
		}
		return fmt.Sprintf("has %s %s which is either %s", article, f.Name, strings.Join(quoted, " or "))
	}

	// Shorthand: has a created datetime (no modifiers, type-like name pattern)
	if f.Type != "" && len(f.Modifiers) == 0 && isShorthandField(f.Name, f.Type) {
		article := fieldArticle(f.Name)
		return fmt.Sprintf("has %s %s %s", article, f.Name, f.Type)
	}

	// Optional field: has an optional <name> which is <type>
	isOptional := false
	var otherMods []string
	for _, m := range f.Modifiers {
		if m == "optional" {
			isOptional = true
		} else {
			otherMods = append(otherMods, m)
		}
	}

	if isOptional {
		modStr := ""
		if len(otherMods) > 0 {
			modStr = strings.Join(otherMods, " ") + " "
		}
		return fmt.Sprintf("has an optional %s which is %s%s", f.Name, modStr, f.Type)
	}

	// Standard field: has a <name> which is [unique|encrypted] <type>
	article := fieldArticle(f.Name)
	modStr := ""
	if len(f.Modifiers) > 0 {
		modStr = strings.Join(f.Modifiers, " ") + " "
	}
	if f.Type != "" {
		return fmt.Sprintf("has %s %s which is %s%s", article, f.Name, modStr, f.Type)
	}
	return fmt.Sprintf("has %s %s", article, f.Name)
}

// fieldArticle returns "a" or "an" based on the first letter of the name.
func fieldArticle(name string) string {
	if name == "" {
		return "a"
	}
	switch strings.ToLower(name[:1]) {
	case "a", "e", "i", "o", "u":
		return "an"
	default:
		return "a"
	}
}

// isShorthandField returns true if this field was likely parsed from shorthand syntax
// like "has a created datetime" or "has a due date" (no "which is").
func isShorthandField(name, typ string) bool {
	shorthandNames := map[string]bool{
		"created": true, "updated": true, "due": true,
		"start": true, "end": true, "last": true,
	}
	return shorthandNames[strings.ToLower(name)]
}

// preserveText re-quotes portions of Statement.Text that contain characters the
// lexer would silently drop (periods, exclamation marks, slashes, etc.).
// This ensures split output can be re-parsed faithfully.
func preserveText(text string) string {
	if !containsLossyChar(text) {
		return text
	}

	// Find the range of words containing lossy characters.
	words := strings.Fields(text)
	firstLossy := -1
	lastLossy := -1
	for i, w := range words {
		if containsLossyChar(w) {
			if firstLossy == -1 {
				firstLossy = i
			}
			lastLossy = i
		}
	}

	if firstLossy == -1 {
		return text
	}

	// Build the result: prefix + quoted segment + suffix.
	var parts []string
	if firstLossy > 0 {
		parts = append(parts, strings.Join(words[:firstLossy], " "))
	}
	quoted := strings.Join(words[firstLossy:lastLossy+1], " ")
	parts = append(parts, fmt.Sprintf("%q", quoted))
	if lastLossy+1 < len(words) {
		parts = append(parts, strings.Join(words[lastLossy+1:], " "))
	}

	return strings.Join(parts, " ")
}

// containsLossyChar returns true if text has characters the lexer would silently skip.
func containsLossyChar(text string) bool {
	for _, r := range text {
		switch r {
		case '.', '!', '?', '/', '(', ')', '[', ']', '{', '}',
			'<', '>', '=', '+', '*', '@', '$', '%', '^', '&', '~', '|', ';':
			return true
		}
	}
	return false
}

// renderStatement outputs a statement's text, re-quoting if needed for faithful re-parsing.
func renderStatement(text string) string {
	return preserveText(text)
}

// renderAccepts formats a list of parameter names with proper English conjunctions.
func renderAccepts(params []string) string {
	if len(params) <= 1 {
		return strings.Join(params, "")
	}
	if len(params) == 2 {
		return params[0] + " and " + params[1]
	}
	return strings.Join(params[:len(params)-1], ", ") + ", and " + params[len(params)-1]
}
