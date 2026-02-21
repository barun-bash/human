package vue

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/barun-bash/human/internal/codegen/themes"
	"github.com/barun-bash/human/internal/ir"
)

// Generator produces a Vue 3 + TypeScript frontend from Intent IR.
type Generator struct{}

// Generate writes a complete Vue 3 + TypeScript project to outputDir.
func (g Generator) Generate(app *ir.Application, outputDir string) error {
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

	files := map[string]string{
		filepath.Join(outputDir, "src", "types", "models.ts"): generateTypes(app),
		filepath.Join(outputDir, "src", "api", "client.ts"):   generateAPIClient(app),
		filepath.Join(outputDir, "src", "router.ts"):          generateRouter(app),
		filepath.Join(outputDir, "src", "App.vue"):            generateApp(app),
	}

	for _, page := range app.Pages {
		name := page.Name + "Page"
		path := filepath.Join(outputDir, "src", "pages", name+".vue")
		files[path] = generatePage(page, app)
	}

	for _, comp := range app.Components {
		path := filepath.Join(outputDir, "src", "components", comp.Name+".vue")
		files[path] = generateComponent(comp, app)
	}

	// Generate theme files
	if app.Theme != nil {
		themeFiles := themes.GenerateVueTheme(app.Theme)
		for relPath, content := range themeFiles {
			files[filepath.Join(outputDir, relPath)] = content
		}
	}

	for path, content := range files {
		if err := writeFile(path, content); err != nil {
			return err
		}
	}

	return nil
}

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

func tsEnumType(values []string) string {
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = fmt.Sprintf("%q", v)
	}
	return strings.Join(parts, " | ")
}

func toCamelCase(s string) string {
	if s == "" {
		return s
	}
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
	runes := []rune(s)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

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
