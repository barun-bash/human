package react

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/barun-bash/human/internal/ir"
)

// Generator produces a React + TypeScript frontend from Intent IR.
type Generator struct{}

// Generate writes a complete React + TypeScript project to outputDir.
func (g Generator) Generate(app *ir.Application, outputDir string) error {
	// Create directory structure
	dirs := []string{
		filepath.Join(outputDir, "src", "types"),
		filepath.Join(outputDir, "src", "api"),
		filepath.Join(outputDir, "src", "pages"),
		filepath.Join(outputDir, "src", "components"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", d, err)
		}
	}

	// Generate and write each file
	files := map[string]string{
		filepath.Join(outputDir, "src", "types", "models.ts"):  generateTypes(app),
		filepath.Join(outputDir, "src", "api", "client.ts"):    generateAPIClient(app),
		filepath.Join(outputDir, "src", "App.tsx"):              generateApp(app),
	}

	// Generate page files
	for _, page := range app.Pages {
		name := page.Name + "Page"
		path := filepath.Join(outputDir, "src", "pages", name+".tsx")
		files[path] = generatePage(page, app)
	}

	// Generate component files
	for _, comp := range app.Components {
		path := filepath.Join(outputDir, "src", "components", comp.Name+".tsx")
		files[path] = generateComponent(comp, app)
	}

	for path, content := range files {
		if err := writeFile(path, content); err != nil {
			return err
		}
	}

	return nil
}

// writeFile writes content to a file, creating parent directories if needed.
func writeFile(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

// tsType maps an IR field type to a TypeScript type.
func tsType(irType string) string {
	switch strings.ToLower(irType) {
	case "text", "date", "datetime", "email", "url", "file", "image":
		return "string"
	case "number", "decimal":
		return "number"
	case "boolean":
		return "boolean"
	case "json":
		return "Record<string, unknown>"
	default:
		return "string"
	}
}

// tsEnumType produces a TypeScript union type from enum values.
// e.g. ["user", "admin"] → `"user" | "admin"`
func tsEnumType(values []string) string {
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = fmt.Sprintf("%q", v)
	}
	return strings.Join(parts, " | ")
}

// toCamelCase converts a PascalCase or space-separated string to camelCase.
// "GetTasks" → "getTasks", "Sign Up" → "signUp"
func toCamelCase(s string) string {
	if s == "" {
		return s
	}
	// Handle space-separated words
	if strings.Contains(s, " ") {
		words := strings.Fields(s)
		for i, w := range words {
			if i == 0 {
				words[i] = strings.ToLower(w)
			} else {
				words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
			}
		}
		return strings.Join(words, "")
	}
	// Handle PascalCase: find uppercase boundaries
	runes := []rune(s)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

// toKebabCase converts a PascalCase or camelCase string to kebab-case.
// "GetTasks" → "get-tasks", "Dashboard" → "dashboard"
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

// httpMethod infers the HTTP method from an API endpoint name.
func httpMethod(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.HasPrefix(lower, "get"):
		return "GET"
	case strings.HasPrefix(lower, "delete"):
		return "DELETE"
	case strings.HasPrefix(lower, "update"):
		return "PUT"
	default:
		return "POST"
	}
}

// apiPath infers the REST path from an API endpoint name.
// Strips CRUD prefixes and converts to kebab-case.
// "GetTasks" → "/api/tasks", "SignUp" → "/api/sign-up", "Login" → "/api/login"
func apiPath(name string) string {
	stripped := name
	for _, prefix := range []string{"Get", "Create", "Update", "Delete"} {
		if strings.HasPrefix(name, prefix) && len(name) > len(prefix) {
			stripped = name[len(prefix):]
			break
		}
	}
	return "/api/" + toKebabCase(stripped)
}
