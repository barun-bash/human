package gobackend

import (
	"fmt"
	"strings"

	"github.com/barun-bash/human/internal/ir"
)

func generateHandlers(moduleName string, app *ir.Application) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"%s/config"
	"%s/dto"
	"%s/models"
)
`, moduleName, moduleName, moduleName))

	sb.WriteString(fmt.Sprintf("\nimport auth \"%s/middleware\"\n\n", moduleName))

	for _, api := range app.APIs {
		sb.WriteString(fmt.Sprintf("func %s(db *gorm.DB, cfg *config.Config) gin.HandlerFunc {\n\treturn func(c *gin.Context) {\n", toPascalCase(api.Name)))

		if len(api.Params) > 0 {
			sb.WriteString(fmt.Sprintf("\t\tvar req dto.%sRequest\n", toPascalCase(api.Name)))
			sb.WriteString("\t\tif err := c.ShouldBindJSON(&req); err != nil {\n\t\t\tc.JSON(http.StatusBadRequest, gin.H{\"error\": err.Error()})\n\t\t\treturn\n\t\t}\n\n")
		}

		for _, val := range api.Validation {
			if val.Rule == "not_empty" {
				sb.WriteString(fmt.Sprintf("\t\tif req.%s == \"\" {\n\t\t\tc.JSON(http.StatusBadRequest, gin.H{\"error\": \"%s is required\"})\n\t\t\treturn\n\t\t}\n", toPascalCase(val.Field), val.Field))
			} else if val.Rule == "max_length" {
				sb.WriteString(fmt.Sprintf("\t\tif len(req.%s) > %s {\n\t\t\tc.JSON(http.StatusBadRequest, gin.H{\"error\": \"%s must be less than %s characters\"})\n\t\t\treturn\n\t\t}\n", toPascalCase(val.Field), val.Value, val.Field, val.Value))
			}
		}

		hasResponse := false
		for _, step := range api.Steps {
			sb.WriteString(fmt.Sprintf("\t\t// %s\n", step.Text))
			
			if step.Type == "create" {
				modelName := inferModelFromAction(step.Text)
				if modelName != "" {
					sb.WriteString(fmt.Sprintf("\t\tnewItem := models.%s{\n", toPascalCase(modelName)))
					if len(api.Params) > 0 {
						for _, p := range api.Params {
							sb.WriteString(fmt.Sprintf("\t\t\t%s: req.%s,\n", toPascalCase(p.Name), toPascalCase(p.Name)))
						}
					}
					sb.WriteString("\t\t}\n")
					sb.WriteString("\t\tif err := db.Create(&newItem).Error; err != nil {\n\t\t\tc.JSON(http.StatusInternalServerError, gin.H{\"error\": \"Failed to create\"})\n\t\t\treturn\n\t\t}\n")
				}
			} else if step.Type == "query" {
				modelName := inferModelFromAction(step.Text)
				if modelName != "" {
					sb.WriteString(fmt.Sprintf("\t\tvar items []models.%s\n", toPascalCase(modelName)))
					sb.WriteString("\t\tif err := db.Find(&items).Error; err != nil {\n\t\t\tc.JSON(http.StatusInternalServerError, gin.H{\"error\": \"Failed to fetch items\"})\n\t\t\treturn\n\t\t}\n")
				}
			} else if step.Type == "respond" {
				hasResponse = true
				if strings.Contains(step.Text, "created") {
					sb.WriteString("\t\tc.JSON(http.StatusCreated, gin.H{\"data\": newItem})\n")
				} else if strings.Contains(step.Text, "tasks") || strings.Contains(step.Text, "items") {
					sb.WriteString("\t\tc.JSON(http.StatusOK, gin.H{\"data\": items})\n")
				} else {
					sb.WriteString("\t\tc.JSON(http.StatusOK, gin.H{\"message\": \"Success\"})\n")
				}
			}
		}

		if !hasResponse {
			sb.WriteString("\t\tc.JSON(http.StatusOK, gin.H{\"message\": \"Not implemented\"})\n")
		}

		sb.WriteString("\t}\n}\n\n")
	}

	return strings.ReplaceAll(sb.String(), "auth \""+moduleName+"/middleware\"", "_ \""+moduleName+"/middleware\"")
}
