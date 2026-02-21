package gobackend

import (
	"fmt"
	"strings"

	"github.com/barun-bash/human/internal/ir"
)

func generateDatabase(moduleName string, app *ir.Application) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`package database

import (
	"fmt"
	"time"

	"%s/config"
	"%s/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(cfg *config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %%w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %%w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// AutoMigrate models
	err = db.AutoMigrate(
`, moduleName, moduleName))

	for _, model := range app.Data {
		sb.WriteString(fmt.Sprintf("\t\t&models.%s{},\n", toPascalCase(model.Name)))
	}

	sb.WriteString("\t)\n")
	sb.WriteString("\tif err != nil {\n\t\treturn nil, fmt.Errorf(\"migration failed: %w\", err)\n\t}\n\n")
	sb.WriteString("\treturn db, nil\n}\n")
	return sb.String()
}

func generateModels(moduleName string, app *ir.Application) string {
	var sb strings.Builder
	sb.WriteString("package models\n\nimport (\n\t\"time\"\n)\n\n")

	for _, model := range app.Data {
		sb.WriteString(fmt.Sprintf("type %s struct {\n", toPascalCase(model.Name)))
		// ID, CreatedAt, UpdatedAt
		sb.WriteString("\tID        string    `gorm:\"primaryKey;type:uuid;default:gen_random_uuid()\" json:\"id\"`\n")

		for _, field := range model.Fields {
			goT := goType(field.Type, field.Required)
			tags := []string{}
			
			if field.Unique {
				tags = append(tags, "uniqueIndex")
			}
			if field.Required {
				tags = append(tags, "not null")
			}

			gormTag := ""
			if len(tags) > 0 {
				gormTag = fmt.Sprintf(` gorm:"%s"`, strings.Join(tags, ";"))
			}

			jsonTag := fmt.Sprintf(` json:"%s"`, toCamelCase(field.Name))
			
			// Optional pointer handling for time/bools when required
			if strings.Contains(goT, "time.Time") && !strings.Contains(sb.String(), "\"time\"") {
				sb.WriteString("\t\"time\"\n") // basic check
			}

			tagString := strings.TrimSpace(gormTag + jsonTag)
			sb.WriteString(fmt.Sprintf("\t%s %s `%s`\n", toPascalCase(field.Name), goT, tagString))
		}

		// Relations
		for _, rel := range model.Relations {
			if rel.Kind == "belongs_to" {
				sb.WriteString(fmt.Sprintf("\t%sID string `json:\"%sId\"`\n", toPascalCase(rel.Target), toCamelCase(rel.Target)))
				sb.WriteString(fmt.Sprintf("\t%s *%s `gorm:\"foreignKey:%sID\" json:\"%s,omitempty\"`\n", toPascalCase(rel.Target), toPascalCase(rel.Target), toPascalCase(rel.Target), toCamelCase(rel.Target)))
			} else if rel.Kind == "has_many" {
				plural := pluralize(toPascalCase(rel.Target))
				pluralCamel := pluralize(toCamelCase(rel.Target))
				sb.WriteString(fmt.Sprintf("\t%s []%s `gorm:\"foreignKey:%sID\" json:\"%s,omitempty\"`\n", plural, toPascalCase(rel.Target), toPascalCase(model.Name), pluralCamel))
			} else if rel.Kind == "has_many_through" {
				plural := pluralize(toPascalCase(rel.Target))
				pluralCamel := pluralize(toCamelCase(rel.Target))
				sb.WriteString(fmt.Sprintf("\t%s []%s `gorm:\"many2many:%s;\" json:\"%s,omitempty\"`\n", plural, toPascalCase(rel.Target), toSnakeCase(rel.Through), pluralCamel))
			}
		}

		sb.WriteString("\tCreatedAt time.Time `json:\"createdAt\"`\n")
		sb.WriteString("\tUpdatedAt time.Time `json:\"updatedAt\"`\n")
		sb.WriteString("}\n\n")
	}

	return strings.ReplaceAll(sb.String(), "`gorm:\"\" ", "`")
}

func generateDTOs(moduleName string, app *ir.Application) string {
	// Build a map of model fields for type lookups
	fieldTypes := map[string]map[string]string{} // modelNameLower -> fieldNameLower -> irType
	for _, model := range app.Data {
		m := map[string]string{}
		for _, f := range model.Fields {
			m[strings.ToLower(f.Name)] = f.Type
		}
		fieldTypes[strings.ToLower(model.Name)] = m
	}

	var sb strings.Builder
	sb.WriteString("package dto\n\n")

	for _, api := range app.APIs {
		if len(api.Params) > 0 {
			// Determine the target model for this endpoint
			targetModel := inferTargetModel(api)

			sb.WriteString(fmt.Sprintf("type %sRequest struct {\n", toPascalCase(api.Name)))
			for _, p := range api.Params {
				goT := "string"
				pLower := strings.ToLower(p.Name)

				// First try the target model's fields
				if targetModel != "" {
					if fields, ok := fieldTypes[strings.ToLower(targetModel)]; ok {
						if irType, ok := fields[pLower]; ok {
							goT = goType(irType, true)
						}
					}
				}
				// Fall back to searching all models only if we didn't find it
				if goT == "string" && targetModel == "" {
					for _, fields := range fieldTypes {
						if irType, ok := fields[pLower]; ok {
							goT = goType(irType, true)
							break
						}
					}
				}

				binding := "required"
				if strings.HasPrefix(pLower, "optional") {
					binding = ""
				}
				bindTag := ""
				if binding != "" {
					bindTag = fmt.Sprintf(" binding:\"%s\"", binding)
				}
				sb.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"%s`\n", toPascalCase(p.Name), goT, toCamelCase(p.Name), bindTag))
			}
			sb.WriteString("}\n\n")
		}
	}

	return sb.String()
}

// inferTargetModel extracts the model name from an endpoint name or its steps.
func inferTargetModel(api *ir.Endpoint) string {
	// Try to extract from endpoint name: CreateReview → Review, GetProduct → Product
	lower := strings.ToLower(api.Name)
	for _, prefix := range []string{"create", "update", "delete", "get", "list", "search"} {
		if strings.HasPrefix(lower, prefix) && len(lower) > len(prefix) {
			return api.Name[len(prefix):]
		}
	}
	// Try to extract from create/query steps
	for _, step := range api.Steps {
		if step.Type == "create" || step.Type == "query" {
			if m := inferModelFromAction(step.Text); m != "" {
				return m
			}
		}
	}
	return ""
}
