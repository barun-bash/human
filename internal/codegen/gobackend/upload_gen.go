package gobackend

import (
	"fmt"
	"strings"

	"github.com/barun-bash/human/internal/ir"
)

// hasStorageIntegration returns true if the app has a storage integration.
func hasStorageIntegration(app *ir.Application) bool {
	for _, integ := range app.Integrations {
		if strings.ToLower(integ.Type) == "storage" {
			return true
		}
	}
	return false
}

// generateUploadHandler produces a Go Gin handler for file uploads.
func generateUploadHandler(moduleName string, app *ir.Application) string {
	var b strings.Builder

	fmt.Fprintf(&b, "package handlers\n\n")
	fmt.Fprintf(&b, "import (\n")
	fmt.Fprintf(&b, "\t\"fmt\"\n")
	fmt.Fprintf(&b, "\t\"net/http\"\n")
	fmt.Fprintf(&b, "\t\"io\"\n")
	fmt.Fprintf(&b, "\t\"time\"\n\n")
	fmt.Fprintf(&b, "\t\"github.com/gin-gonic/gin\"\n")
	fmt.Fprintf(&b, "\t\"%s/services\"\n", moduleName)
	fmt.Fprintf(&b, ")\n\n")

	b.WriteString("// UploadFile handles multipart file uploads.\n")
	b.WriteString("func UploadFile(c *gin.Context) {\n")
	b.WriteString("\tfile, header, err := c.Request.FormFile(\"file\")\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\tc.JSON(http.StatusBadRequest, gin.H{\"error\": \"No file provided\"})\n")
	b.WriteString("\t\treturn\n")
	b.WriteString("\t}\n")
	b.WriteString("\tdefer file.Close()\n\n")

	b.WriteString("\tdata, err := io.ReadAll(file)\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\tc.JSON(http.StatusInternalServerError, gin.H{\"error\": \"Failed to read file\"})\n")
	b.WriteString("\t\treturn\n")
	b.WriteString("\t}\n\n")

	b.WriteString("\tkey := fmt.Sprintf(\"uploads/%d-%s\", time.Now().Unix(), header.Filename)\n")
	b.WriteString("\tresult, err := services.UploadFile(c.Request.Context(), key, data, header.Header.Get(\"Content-Type\"))\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\tc.JSON(http.StatusInternalServerError, gin.H{\"error\": \"Upload failed\"})\n")
	b.WriteString("\t\treturn\n")
	b.WriteString("\t}\n\n")

	b.WriteString("\tc.JSON(http.StatusOK, gin.H{\"key\": result, \"filename\": header.Filename, \"size\": header.Size})\n")
	b.WriteString("}\n")

	return b.String()
}
