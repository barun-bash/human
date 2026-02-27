package angular

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/barun-bash/human/internal/codegen/themes"
	"github.com/barun-bash/human/internal/ir"
)

type Generator struct{}

func (g Generator) Generate(app *ir.Application, outputDir string) error {
	dirs := []string{
		filepath.Join(outputDir, "src", "app", "models"),
		filepath.Join(outputDir, "src", "app", "services"),
		filepath.Join(outputDir, "src", "app", "pages"),
		filepath.Join(outputDir, "src", "app", "components"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", d, err)
		}
	}

	files := map[string]string{
		filepath.Join(outputDir, "package.json"):                     generatePackageJson(app),
		filepath.Join(outputDir, "angular.json"):                     generateAngularJson(app),
		filepath.Join(outputDir, "tsconfig.json"):                    generateTsConfig(app),
		filepath.Join(outputDir, "src", "index.html"):                generateIndexHtml(app),
		filepath.Join(outputDir, "src", "main.ts"):                   generateMainTs(app),
		filepath.Join(outputDir, "src", "app", "app.config.ts"):      generateAppConfig(app),
		filepath.Join(outputDir, "src", "app", "app.routes.ts"):      generateRoutes(app),
		filepath.Join(outputDir, "src", "app", "app.component.ts"):   generateAppComponent(app),
		filepath.Join(outputDir, "src", "app", "models", "types.ts"): generateTypes(app),
		filepath.Join(outputDir, "src", "app", "services", "api.service.ts"): generateApiService(app),
	}

	for _, page := range app.Pages {
		name := toKebabCase(page.Name)
		path := filepath.Join(outputDir, "src", "app", "pages", name, name+".component.ts")
		files[path] = generatePage(page, app)
	}

	for _, comp := range app.Components {
		name := toKebabCase(comp.Name)
		path := filepath.Join(outputDir, "src", "app", "components", name, name+".component.ts")
		files[path] = generateComponent(comp, app)
	}

	// 404 not-found page
	files[filepath.Join(outputDir, "src", "app", "pages", "not-found", "not-found.component.ts")] = generateNotFoundComponent()

	// Generate theme files
	if app.Theme != nil {
		themeFiles := themes.GenerateAngularTheme(app.Theme)
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

func toPascalCase(s string) string {
	if s == "" {
		return s
	}
	if strings.Contains(s, " ") {
		words := strings.Fields(s)
		for i, w := range words {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
		return strings.Join(words, "")
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
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

func httpMethod(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.HasPrefix(lower, "get"),
		strings.HasPrefix(lower, "list"),
		strings.HasPrefix(lower, "search"),
		strings.HasPrefix(lower, "fetch"):
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
	for _, prefix := range []string{"Get", "List", "Search", "Fetch", "Create", "Update", "Delete"} {
		if strings.HasPrefix(name, prefix) && len(name) > len(prefix) {
			stripped = name[len(prefix):]
			break
		}
	}
	return "/api/" + toKebabCase(stripped)
}

func pluralize(s string) string {
	if s == "" {
		return s
	}
	lower := strings.ToLower(s)
	if strings.HasSuffix(lower, "s") || strings.HasSuffix(lower, "sh") || strings.HasSuffix(lower, "ch") || strings.HasSuffix(lower, "x") || strings.HasSuffix(lower, "z") {
		return s + "es"
	}
	if strings.HasSuffix(lower, "y") && len(lower) > 1 {
		prev := lower[len(lower)-2]
		if prev != 'a' && prev != 'e' && prev != 'i' && prev != 'o' && prev != 'u' {
			return s[:len(s)-1] + "ies"
		}
	}
	return s + "s"
}
