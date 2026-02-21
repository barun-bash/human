package gobackend

import (
	"fmt"
	"strings"

	"github.com/barun-bash/human/internal/ir"
)

// modelFieldInfo holds type information for a model field.
type modelFieldInfo struct {
	exists   bool
	required bool
}

// modelFieldSet builds a map of field names (lowercase) → info for a given model.
func modelFieldSet(app *ir.Application, modelName string) map[string]modelFieldInfo {
	fields := map[string]modelFieldInfo{}
	for _, m := range app.Data {
		if strings.EqualFold(m.Name, modelName) {
			for _, f := range m.Fields {
				fields[strings.ToLower(f.Name)] = modelFieldInfo{exists: true, required: f.Required}
			}
			// Add foreign key fields from belongs_to relations
			for _, r := range m.Relations {
				fields[strings.ToLower(r.Target)+"_id"] = modelFieldInfo{exists: true, required: true}
			}
			break
		}
	}
	return fields
}

func generateHandlers(moduleName string, app *ir.Application) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"%s/config"
	"%s/dto"
	"%s/middleware"
	"%s/models"
)

`, moduleName, moduleName, moduleName, moduleName))

	for _, api := range app.APIs {
		isLogin := isLoginEndpoint(api.Name)
		isSignUp := isSignUpEndpoint(api.Name)

		sb.WriteString(fmt.Sprintf("func %s(db *gorm.DB, cfg *config.Config) gin.HandlerFunc {\n\treturn func(c *gin.Context) {\n", toPascalCase(api.Name)))

		// Bind request body if params exist
		if len(api.Params) > 0 {
			sb.WriteString(fmt.Sprintf("\t\tvar req dto.%sRequest\n", toPascalCase(api.Name)))
			sb.WriteString("\t\tif err := c.ShouldBindJSON(&req); err != nil {\n\t\t\tc.JSON(http.StatusBadRequest, gin.H{\"error\": err.Error()})\n\t\t\treturn\n\t\t}\n\n")
		}

		// Validation
		for _, val := range api.Validation {
			if val.Rule == "not_empty" {
				sb.WriteString(fmt.Sprintf("\t\tif req.%s == \"\" {\n\t\t\tc.JSON(http.StatusBadRequest, gin.H{\"error\": \"%s is required\"})\n\t\t\treturn\n\t\t}\n", toPascalCase(val.Field), val.Field))
			} else if val.Rule == "max_length" {
				sb.WriteString(fmt.Sprintf("\t\tif len(req.%s) > %s {\n\t\t\tc.JSON(http.StatusBadRequest, gin.H{\"error\": \"%s must be less than %s characters\"})\n\t\t\treturn\n\t\t}\n", toPascalCase(val.Field), val.Value, val.Field, val.Value))
			}
		}

		// Track state
		queryModelName := ""
		queryUsedItems := false // true if we queried a list (items), false if single (item)
		hasCreate := false
		hasReturn := false

		// Generate code for each step
		for _, step := range api.Steps {
			sb.WriteString(fmt.Sprintf("\t\t// %s\n", step.Text))

			switch step.Type {
			case "create":
				modelName := inferModelFromAction(step.Text)
				if modelName == "" {
					continue
				}

				if hasCreate {
					// Avoid duplicate variable declarations
					continue
				}
				hasCreate = true

				fields := modelFieldSet(app, modelName)

				if isSignUp {
					sb.WriteString("\t\thashedPassword, err := middleware.HashPassword(req.Password)\n")
					sb.WriteString("\t\tif err != nil {\n\t\t\tc.JSON(http.StatusInternalServerError, gin.H{\"error\": \"Failed to hash password\"})\n\t\t\treturn\n\t\t}\n")
					sb.WriteString(fmt.Sprintf("\t\tnewItem := models.%s{\n", toPascalCase(modelName)))
					for _, p := range api.Params {
						pLower := strings.ToLower(p.Name)
						if pLower == "password" {
							sb.WriteString("\t\t\tPassword: hashedPassword,\n")
						} else if fi, ok := fields[pLower]; ok && fi.exists {
							pName := toPascalCase(p.Name)
							if !fi.required {
								sb.WriteString(fmt.Sprintf("\t\t\t%s: &req.%s,\n", pName, pName))
							} else {
								sb.WriteString(fmt.Sprintf("\t\t\t%s: req.%s,\n", pName, pName))
							}
						}
					}
					sb.WriteString("\t\t}\n")
				} else {
					sb.WriteString(fmt.Sprintf("\t\tnewItem := models.%s{\n", toPascalCase(modelName)))
					for _, p := range api.Params {
						pLower := strings.ToLower(p.Name)
						if strings.HasPrefix(pLower, "optional") {
							continue
						}
						// Only assign if this field exists on the model
						if fi, ok := fields[pLower]; ok && fi.exists {
							pName := toPascalCase(p.Name)
							if !fi.required {
								sb.WriteString(fmt.Sprintf("\t\t\t%s: &req.%s,\n", pName, pName))
							} else {
								sb.WriteString(fmt.Sprintf("\t\t\t%s: req.%s,\n", pName, pName))
							}
						}
					}
					if api.Auth {
						sb.WriteString("\t\t\tUserID: c.GetString(\"userID\"),\n")
					}
					sb.WriteString("\t\t}\n")
				}
				sb.WriteString("\t\tif err := db.Create(&newItem).Error; err != nil {\n\t\t\tc.JSON(http.StatusInternalServerError, gin.H{\"error\": \"Failed to create\"})\n\t\t\treturn\n\t\t}\n")

			case "query":
				modelName := inferModelFromAction(step.Text)
				lowerText := strings.ToLower(step.Text)

				// Skip if we can't determine the model or already queried
				if modelName == "" || queryModelName != "" {
					continue
				}
				queryModelName = modelName

				if strings.Contains(lowerText, " by ") {
					parts := strings.SplitN(lowerText, " by ", 2)
					fieldParts := strings.Fields(parts[1])
					queryField := fieldParts[0]
					// Map <model>_id params to the model's id column
					dbCol := toSnakeCase(queryField)
					if strings.HasSuffix(queryField, "_id") {
						dbCol = "id"
					}
					reqField := toPascalCase(queryField)
					sb.WriteString(fmt.Sprintf("\t\tvar item models.%s\n", toPascalCase(modelName)))
					sb.WriteString(fmt.Sprintf("\t\tif err := db.Where(\"%s = ?\", req.%s).First(&item).Error; err != nil {\n",
						dbCol, reqField))
					if isLogin {
						sb.WriteString("\t\t\tc.JSON(http.StatusUnauthorized, gin.H{\"error\": \"Invalid credentials\"})\n")
					} else {
						sb.WriteString(fmt.Sprintf("\t\t\tc.JSON(http.StatusNotFound, gin.H{\"error\": \"%s not found\"})\n", modelName))
					}
					sb.WriteString("\t\t\treturn\n\t\t}\n")
				} else if strings.Contains(lowerText, "all") || strings.Contains(lowerText, "where") {
					queryUsedItems = true
					sb.WriteString(fmt.Sprintf("\t\tvar items []models.%s\n", toPascalCase(modelName)))
					sb.WriteString("\t\tif err := db.Find(&items).Error; err != nil {\n\t\t\tc.JSON(http.StatusInternalServerError, gin.H{\"error\": \"Failed to fetch items\"})\n\t\t\treturn\n\t\t}\n")
				} else {
					idParam := findIDParam(api)
					sb.WriteString(fmt.Sprintf("\t\tvar item models.%s\n", toPascalCase(modelName)))
					sb.WriteString(fmt.Sprintf("\t\tif err := db.Where(\"id = ?\", req.%s).First(&item).Error; err != nil {\n", idParam))
					sb.WriteString(fmt.Sprintf("\t\t\tc.JSON(http.StatusNotFound, gin.H{\"error\": \"%s not found\"})\n", modelName))
					sb.WriteString("\t\t\treturn\n\t\t}\n")
				}

			case "condition":
				lowerText := strings.ToLower(step.Text)
				if isLogin && (strings.Contains(lowerText, "password") || strings.Contains(lowerText, "does not match")) {
					sb.WriteString("\t\tif !middleware.CheckPasswordHash(req.Password, item.Password) {\n")
					sb.WriteString("\t\t\tc.JSON(http.StatusUnauthorized, gin.H{\"error\": \"Invalid credentials\"})\n")
					sb.WriteString("\t\t\treturn\n\t\t}\n")
				}

			case "update":
				lowerText := strings.ToLower(step.Text)
				if strings.Contains(lowerText, "update") && strings.Contains(lowerText, "with") {
					sb.WriteString("\t\tif err := db.Model(&item).Updates(req).Error; err != nil {\n\t\t\tc.JSON(http.StatusInternalServerError, gin.H{\"error\": \"Failed to update\"})\n\t\t\treturn\n\t\t}\n")
				} else if strings.Contains(lowerText, "update") && strings.Contains(lowerText, "status") {
					sb.WriteString("\t\tif err := db.Model(&item).Update(\"status\", req.Status).Error; err != nil {\n\t\t\tc.JSON(http.StatusInternalServerError, gin.H{\"error\": \"Failed to update\"})\n\t\t\treturn\n\t\t}\n")
				}

			case "delete":
				sb.WriteString("\t\tif err := db.Delete(&item).Error; err != nil {\n\t\t\tc.JSON(http.StatusInternalServerError, gin.H{\"error\": \"Failed to delete\"})\n\t\t\treturn\n\t\t}\n")

			case "respond":
				hasReturn = true
				lowerText := strings.ToLower(step.Text)
				if (isLogin || isSignUp) && strings.Contains(lowerText, "token") {
					if isLogin {
						sb.WriteString("\t\ttoken, err := middleware.GenerateToken(item.ID, cfg)\n")
					} else {
						sb.WriteString("\t\ttoken, err := middleware.GenerateToken(newItem.ID, cfg)\n")
					}
					sb.WriteString("\t\tif err != nil {\n\t\t\tc.JSON(http.StatusInternalServerError, gin.H{\"error\": \"Failed to generate token\"})\n\t\t\treturn\n\t\t}\n")
					if isLogin {
						sb.WriteString("\t\tc.JSON(http.StatusOK, gin.H{\"data\": item, \"token\": token})\n")
					} else {
						sb.WriteString("\t\tc.JSON(http.StatusCreated, gin.H{\"data\": newItem, \"token\": token})\n")
					}
				} else if strings.Contains(lowerText, "created") {
					sb.WriteString("\t\tc.JSON(http.StatusCreated, gin.H{\"data\": newItem})\n")
				} else if strings.Contains(lowerText, "updated") {
					sb.WriteString("\t\tc.JSON(http.StatusOK, gin.H{\"data\": item})\n")
				} else if strings.Contains(lowerText, "deleted") {
					sb.WriteString("\t\tc.JSON(http.StatusOK, gin.H{\"message\": \"Deleted successfully\"})\n")
				} else if queryUsedItems {
					sb.WriteString("\t\tc.JSON(http.StatusOK, gin.H{\"data\": items})\n")
				} else if hasCreate {
					sb.WriteString("\t\tc.JSON(http.StatusCreated, gin.H{\"data\": newItem})\n")
				} else if queryModelName != "" {
					sb.WriteString("\t\tc.JSON(http.StatusOK, gin.H{\"data\": item})\n")
				} else {
					sb.WriteString("\t\tc.JSON(http.StatusOK, gin.H{\"message\": \"Success\"})\n")
				}
			}
		}

		if !hasReturn {
			sb.WriteString("\t\tc.JSON(http.StatusOK, gin.H{\"message\": \"Not implemented\"})\n")
		}

		sb.WriteString("\t}\n}\n\n")
	}

	return sb.String()
}
