package postgres

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/barun-bash/human/internal/ir"
)

// Generator produces PostgreSQL migration files from Intent IR.
type Generator struct{}

// Generate writes SQL migration and seed files to outputDir.
func (g Generator) Generate(app *ir.Application, outputDir string) error {
	migrationsDir := filepath.Join(outputDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", migrationsDir, err)
	}

	files := map[string]string{
		filepath.Join(migrationsDir, "001_initial.sql"): generateMigration(app),
		filepath.Join(outputDir, "seed.sql"):            generateSeed(app),
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

// pgType maps an IR field type to a PostgreSQL column type.
func pgType(irType string) string {
	switch strings.ToLower(irType) {
	case "text", "email", "url", "file", "image":
		return "TEXT"
	case "number":
		return "INTEGER"
	case "decimal":
		return "NUMERIC"
	case "boolean":
		return "BOOLEAN"
	case "date":
		return "DATE"
	case "datetime":
		return "TIMESTAMPTZ"
	case "json":
		return "JSONB"
	default:
		return "TEXT"
	}
}

// toSnakeCase converts PascalCase/camelCase to snake_case.
func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			result = append(result, '_')
		}
		result = append(result, unicode.ToLower(r))
	}
	return string(result)
}

// toTableName converts a model name to a plural snake_case table name.
// "User" → "users", "TaskTag" → "task_tags"
func toTableName(modelName string) string {
	snake := toSnakeCase(modelName)
	if strings.HasSuffix(snake, "s") {
		return snake
	}
	return snake + "s"
}

// enumTypeName returns the PostgreSQL enum type name for a model field.
// "User", "role" → "user_role"
func enumTypeName(modelName, fieldName string) string {
	return toSnakeCase(modelName) + "_" + toSnakeCase(fieldName)
}

// sanitizeIdentifier ensures a SQL identifier has no spaces.
// "due date" → "due_date"
func sanitizeIdentifier(name string) string {
	return strings.ReplaceAll(name, " ", "_")
}

// isJoinTable checks if a model only has belongs_to relations and no fields
// (i.e. it's purely a join table for many-to-many).
func isJoinTable(model *ir.DataModel) bool {
	if len(model.Fields) > 0 {
		return false
	}
	for _, rel := range model.Relations {
		if rel.Kind != "belongs_to" {
			return false
		}
	}
	return len(model.Relations) >= 2
}

// sortModelsForCreation returns models in dependency order:
// models with no belongs_to first, then models that depend on them.
func sortModelsForCreation(models []*ir.DataModel) []*ir.DataModel {
	// Simple topological sort: non-dependent first, then dependent.
	var independent, dependent []*ir.DataModel
	for _, m := range models {
		hasDep := false
		for _, rel := range m.Relations {
			if rel.Kind == "belongs_to" {
				hasDep = true
				break
			}
		}
		if hasDep {
			dependent = append(dependent, m)
		} else {
			independent = append(independent, m)
		}
	}
	return append(independent, dependent...)
}
