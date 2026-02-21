package gobackend

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/barun-bash/human/internal/ir"
)

type Generator struct{}

func (g Generator) Generate(app *ir.Application, outputDir string) error {
	dirs := []string{
		filepath.Join(outputDir, "config"),
		filepath.Join(outputDir, "database"),
		filepath.Join(outputDir, "models"),
		filepath.Join(outputDir, "dto"),
		filepath.Join(outputDir, "middleware"),
		filepath.Join(outputDir, "handlers"),
		filepath.Join(outputDir, "routes"),
		filepath.Join(outputDir, "migrations"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", d, err)
		}
	}

	moduleName := appNameLower(app)
	if moduleName == "" {
		moduleName = "app"
	}

	files := map[string]string{
		filepath.Join(outputDir, "go.mod"):                    generateGoMod(moduleName),
		filepath.Join(outputDir, "main.go"):                   generateMain(moduleName, app),
		filepath.Join(outputDir, "config", "config.go"):       generateConfig(moduleName),
		filepath.Join(outputDir, "database", "database.go"):   generateDatabase(moduleName, app),
		filepath.Join(outputDir, "models", "models.go"):       generateModels(moduleName, app),
		filepath.Join(outputDir, "dto", "dto.go"):             generateDTOs(moduleName, app),
		filepath.Join(outputDir, "middleware", "auth.go"):     generateAuth(moduleName, app),
		filepath.Join(outputDir, "handlers", "handlers.go"):   generateHandlers(moduleName, app),
		filepath.Join(outputDir, "routes", "routes.go"):       generateRoutes(moduleName, app),
		filepath.Join(outputDir, "migrations", "initial.sql"): generateMigration(app),
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

func appNameLower(app *ir.Application) string {
	if app.Name != "" {
		return strings.ToLower(strings.ReplaceAll(app.Name, " ", "-"))
	}
	return "app"
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
	if strings.Contains(s, "_") {
		words := strings.Split(s, "_")
		for i, w := range words {
			if w != "" {
				words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
			}
		}
		return strings.Join(words, "")
	}
	if strings.Contains(s, "-") {
		words := strings.Split(s, "-")
		for i, w := range words {
			if w != "" {
				words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
			}
		}
		return strings.Join(words, "")
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
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
	if strings.Contains(s, "_") {
		words := strings.Split(s, "_")
		for i, w := range words {
			if i == 0 {
				words[i] = strings.ToLower(w)
			} else {
				words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
			}
		}
		return strings.Join(words, "")
	}
	if strings.Contains(s, "-") {
		words := strings.Split(s, "-")
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

func routePath(name string) string {
	stripped := name
	for _, prefix := range []string{"Get", "Create", "Update", "Delete"} {
		if strings.HasPrefix(name, prefix) && len(name) > len(prefix) {
			stripped = name[len(prefix):]
			break
		}
	}
	var result []rune
	for i, r := range stripped {
		if unicode.IsUpper(r) && i > 0 {
			result = append(result, '-')
		}
		result = append(result, unicode.ToLower(r))
	}
	return "/" + string(result)
}

func goType(irType string, required bool) string {
	base := ""
	switch strings.ToLower(irType) {
	case "text", "email", "url", "file", "image", "enum":
		base = "string"
	case "number":
		base = "int"
	case "decimal":
		base = "float64"
	case "boolean":
		base = "bool"
	case "date", "datetime":
		base = "time.Time"
	case "json":
		base = "map[string]any"
	default:
		base = "string"
	}
	if !required {
		return "*" + base
	}
	return base
}

func inferModelFromAction(text string) string {
	words := strings.Fields(text)
	for i, w := range words {
		lower := strings.ToLower(w)
		if lower == "a" || lower == "an" || lower == "the" {
			if i+1 < len(words) {
				return toPascalCase(words[i+1])
			}
		} else if lower == "all" {
			if i+1 < len(words) {
				name := words[i+1]
				if strings.HasSuffix(name, "s") {
					name = name[:len(name)-1]
				}
				return toPascalCase(name)
			}
		}
	}
	return ""
}

func toSnakeCase(s string) string {
	if s == "" {
		return ""
	}
	var result []rune
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 && s[i-1] != ' ' && s[i-1] != '_' && s[i-1] != '-' {
				result = append(result, '_')
			}
			result = append(result, unicode.ToLower(r))
		} else if r == ' ' || r == '-' {
			result = append(result, '_')
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}
