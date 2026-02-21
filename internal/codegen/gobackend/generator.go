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
		filepath.Join(outputDir, "setup.sh"):                  generateSetupScript(),
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

// goAcronyms are words that should be fully uppercased in Go identifiers.
var goAcronyms = map[string]string{
	"id": "ID", "url": "URL", "api": "API", "http": "HTTP",
	"ip": "IP", "json": "JSON", "sql": "SQL", "ssh": "SSH",
	"tcp": "TCP", "udp": "UDP", "uri": "URI", "uuid": "UUID",
}

func capitalizeWord(w string) string {
	if w == "" {
		return w
	}
	if upper, ok := goAcronyms[strings.ToLower(w)]; ok {
		return upper
	}
	return strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
}

func toPascalCase(s string) string {
	if s == "" {
		return s
	}
	if strings.Contains(s, " ") {
		words := strings.Fields(s)
		for i, w := range words {
			words[i] = capitalizeWord(w)
		}
		return strings.Join(words, "")
	}
	if strings.Contains(s, "_") {
		words := strings.Split(s, "_")
		for i, w := range words {
			if w != "" {
				words[i] = capitalizeWord(w)
			}
		}
		return strings.Join(words, "")
	}
	if strings.Contains(s, "-") {
		words := strings.Split(s, "-")
		for i, w := range words {
			if w != "" {
				words[i] = capitalizeWord(w)
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
	case strings.HasPrefix(lower, "get"), strings.HasPrefix(lower, "list"), strings.HasPrefix(lower, "search"):
		return "GET"
	case strings.HasPrefix(lower, "delete"):
		return "DELETE"
	case strings.HasPrefix(lower, "update"):
		return "PUT"
	default:
		return "POST"
	}
}

func isLoginEndpoint(name string) bool {
	return strings.ToLower(name) == "login"
}

func isSignUpEndpoint(name string) bool {
	lower := strings.ToLower(name)
	return lower == "signup" || lower == "sign_up" || lower == "signUp"
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

// findIDParam returns the PascalCase name of a likely ID parameter.
func findIDParam(api *ir.Endpoint) string {
	for _, p := range api.Params {
		lower := strings.ToLower(p.Name)
		if strings.HasSuffix(lower, "_id") || strings.HasSuffix(lower, "id") || lower == "slug" {
			return toPascalCase(p.Name)
		}
	}
	if len(api.Params) > 0 {
		return toPascalCase(api.Params[0].Name)
	}
	return "ID"
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
	// Common words that should not be treated as model names
	skip := map[string]bool{
		"current": true, "given": true, "same": true, "new": true,
		"user's": true, "author's": true, "their": true, "own": true,
	}
	words := strings.Fields(text)
	for i, w := range words {
		lower := strings.ToLower(w)
		if lower == "a" || lower == "an" || lower == "the" {
			if i+1 < len(words) {
				candidate := strings.ToLower(words[i+1])
				if skip[candidate] {
					continue
				}
				return toPascalCase(words[i+1])
			}
		} else if lower == "all" {
			if i+1 < len(words) {
				candidate := strings.ToLower(words[i+1])
				if skip[candidate] {
					continue
				}
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
	runes := []rune(s)
	for i, r := range runes {
		if unicode.IsUpper(r) {
			// Don't insert underscore if:
			// - first char
			// - previous char was uppercase AND (next char is also uppercase or we're at end)
			//   This handles "ID" → "id", "URL" → "url"
			if i > 0 && runes[i-1] != ' ' && runes[i-1] != '_' && runes[i-1] != '-' {
				prevUpper := unicode.IsUpper(runes[i-1])
				nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
				if !prevUpper || nextLower {
					result = append(result, '_')
				}
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
