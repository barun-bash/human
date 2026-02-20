package node

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/barun-bash/human/internal/ir"
)

// Generator produces a Node + Express + TypeScript backend from Intent IR.
type Generator struct{}

// Generate writes a complete Express backend project to outputDir.
func (g Generator) Generate(app *ir.Application, outputDir string) error {
	dirs := []string{
		filepath.Join(outputDir, "prisma"),
		filepath.Join(outputDir, "src", "routes"),
		filepath.Join(outputDir, "src", "middleware"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", d, err)
		}
	}

	files := map[string]string{
		filepath.Join(outputDir, "prisma", "schema.prisma"):        generatePrismaSchema(app),
		filepath.Join(outputDir, "src", "middleware", "auth.ts"):    generateAuthMiddleware(app),
		filepath.Join(outputDir, "src", "middleware", "errors.ts"):  generateErrorHandler(app),
		filepath.Join(outputDir, "src", "routes", "index.ts"):      generateRouteIndex(app),
		filepath.Join(outputDir, "src", "server.ts"):                generateServer(app),
	}

	// One route file per endpoint
	for _, ep := range app.APIs {
		filename := toKebabCase(ep.Name) + ".ts"
		path := filepath.Join(outputDir, "src", "routes", filename)
		files[path] = generateRoute(ep, app)
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

// toCamelCase converts PascalCase or space-separated to camelCase.
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

// toKebabCase converts PascalCase/camelCase to kebab-case.
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
		return "get"
	case strings.HasPrefix(lower, "delete"):
		return "delete"
	case strings.HasPrefix(lower, "update"):
		return "put"
	default:
		return "post"
	}
}

// routePath infers the REST path from an endpoint name.
func routePath(name string) string {
	stripped := name
	for _, prefix := range []string{"Get", "Create", "Update", "Delete"} {
		if strings.HasPrefix(name, prefix) && len(name) > len(prefix) {
			stripped = name[len(prefix):]
			break
		}
	}
	return "/" + toKebabCase(stripped)
}

// prismaType maps an IR field type to a Prisma scalar type.
func prismaType(irType string) string {
	switch strings.ToLower(irType) {
	case "text", "email", "url", "file", "image":
		return "String"
	case "number":
		return "Int"
	case "decimal":
		return "Float"
	case "boolean":
		return "Boolean"
	case "date", "datetime":
		return "DateTime"
	case "json":
		return "Json"
	default:
		return "String"
	}
}

// sanitizeParamName ensures a param name is a valid TS identifier.
func sanitizeParamName(name string) string {
	if !strings.Contains(name, " ") {
		return name
	}
	return toCamelCase(name)
}
