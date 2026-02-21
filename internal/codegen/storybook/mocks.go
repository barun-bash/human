package storybook

import (
	"fmt"
	"strings"

	"github.com/barun-bash/human/internal/ir"
)

func generateMockData(app *ir.Application) string {
	var b strings.Builder

	b.WriteString("// Auto-generated mock data factories based on Human IR Data Models\n\n")

	for _, model := range app.Data {
		b.WriteString(fmt.Sprintf("export const mock%s = (overrides?: Partial<any>) => ({\n", model.Name))
		b.WriteString("  id: 'mock-id-123',\n")

		for _, field := range model.Fields {
			val := generateMockValue(field)
			b.WriteString(fmt.Sprintf("  %s: %s,\n", field.Name, val))
		}

		b.WriteString("  ...overrides,\n")
		b.WriteString("});\n\n")

		b.WriteString(fmt.Sprintf("export const mock%sList = (count: number = 3) => \n", model.Name))
		b.WriteString(fmt.Sprintf("  Array.from({ length: count }).map((_, i) => mock%s({ id: `mock-id-${i}` }));\n\n", model.Name))
	}

	return b.String()
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
