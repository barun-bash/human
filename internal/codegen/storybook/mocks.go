package storybook

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/barun-bash/human/internal/ir"
)

func generateMockData(app *ir.Application, fw string) string {
	var b strings.Builder

	b.WriteString("// Auto-generated mock data factories based on Human IR Data Models\n\n")

	// Import model types — path varies by framework
	if len(app.Data) > 0 {
		names := make([]string, len(app.Data))
		for i, m := range app.Data {
			names[i] = m.Name
		}
		typesImportPath := "../types/models"
		if fw == "angular" {
			typesImportPath = "../app/models/types"
		} else if fw == "svelte" {
			typesImportPath = "$lib/types"
		}
		fmt.Fprintf(&b, "import type { %s } from '%s';\n\n", strings.Join(names, ", "), typesImportPath)
	}

	for _, model := range app.Data {
		fmt.Fprintf(&b, "export const mock%s = (overrides?: Partial<%s>) => ({\n", model.Name, model.Name)
		b.WriteString("  id: 'mock-id-123',\n")

		for _, field := range model.Fields {
			val := generateMockValue(field)
			b.WriteString(fmt.Sprintf("  %s: %s,\n", field.Name, val))
		}

		for _, rel := range model.Relations {
			if rel.Kind == "belongs_to" {
				fmt.Fprintf(&b, "  %s: undefined, // Replace with mock%s() if needed\n", toCamelCase(rel.Target), rel.Target)
			} else if rel.Kind == "has_many" || rel.Kind == "has_many_through" {
				fmt.Fprintf(&b, "  %s: [], // Replace with mock%sList() if needed\n", pluralize(toCamelCase(rel.Target)), rel.Target)
			}
		}

		b.WriteString("  ...overrides,\n")
		b.WriteString("});\n\n")

		fmt.Fprintf(&b, "export const mock%sList = (count: number = 3) => \n", model.Name)
		fmt.Fprintf(&b, "  Array.from({ length: count }).map((_, i) => mock%s({ id: `mock-id-${i}` }));\n\n", model.Name)
	}

	return b.String()
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

func generateMockValue(field *ir.DataField) string {
	switch strings.ToLower(field.Type) {
	case "text":
		lowerName := strings.ToLower(field.Name)
		if strings.Contains(lowerName, "name") {
			return "'Jane Doe'"
		}
		if strings.Contains(lowerName, "title") {
			return "'Sample Title'"
		}
		return "'Lorem ipsum dolor sit amet'"
	case "email":
		return "'jane.doe@example.com'"
	case "url":
		return "'https://example.com'"
	case "number", "decimal":
		lowerName := strings.ToLower(field.Name)
		if strings.Contains(lowerName, "age") {
			return "28"
		}
		if strings.Contains(lowerName, "price") {
			return "99.99"
		}
		return "42"
	case "boolean":
		return "true"
	case "date", "datetime":
		return "'2025-01-01T12:00:00Z'"
	case "enum":
		if len(field.EnumValues) > 0 {
			return fmt.Sprintf("'%s'", field.EnumValues[0])
		}
		return "'unknown'"
	default:
		return "'mock-data'"
	}
}
