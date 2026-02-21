package gobackend

import (
	"fmt"
	"strings"

	"github.com/barun-bash/human/internal/ir"
)

func generateRoutes(moduleName string, app *ir.Application) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"%s/config"
	"%s/handlers"
	"%s/middleware"
)

func Setup(r *gin.Engine, db *gorm.DB) {
	cfg := config.Load()
	api := r.Group("/api")

`, moduleName, moduleName, moduleName))

	for _, api := range app.APIs {
		method := httpMethod(api.Name)
		path := routePath(api.Name)

		if api.Auth {
			sb.WriteString(fmt.Sprintf("\tapi.%s(\"%s\", middleware.RequireAuth(db, cfg), handlers.%s(db, cfg))\n", method, path, toPascalCase(api.Name)))
		} else {
			sb.WriteString(fmt.Sprintf("\tapi.%s(\"%s\", handlers.%s(db, cfg))\n", method, path, toPascalCase(api.Name)))
		}
	}

	sb.WriteString("}\n")
	return sb.String()
}

func generateMigration(app *ir.Application) string {
	return "-- Initial Migration\n-- Schema auto-generated via GORM AutoMigrate is recommended for early dev, but here is a placeholder.\n\nSELECT 1;\n"
}
