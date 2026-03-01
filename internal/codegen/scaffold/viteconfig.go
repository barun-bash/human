package scaffold

import (
	"fmt"
	"strings"

	"github.com/barun-bash/human/internal/ir"
)

// generateViteConfig produces react/vite.config.ts with the React plugin
// and an API proxy to the backend dev server.
func generateViteConfig(app *ir.Application) string {
	port := 3001
	if app.Config != nil && app.Config.Ports.Backend > 0 {
		port = app.Config.Ports.Backend
	}

	var b strings.Builder

	b.WriteString("import { defineConfig } from 'vite'\n")
	b.WriteString("import react from '@vitejs/plugin-react'\n")
	b.WriteString("\n")
	b.WriteString("export default defineConfig({\n")
	b.WriteString("  plugins: [react()],\n")
	b.WriteString("  server: {\n")
	b.WriteString("    proxy: {\n")
	b.WriteString("      '/api': {\n")
	fmt.Fprintf(&b, "        target: 'http://localhost:%d',\n", port)
	b.WriteString("        changeOrigin: true,\n")
	b.WriteString("      },\n")
	b.WriteString("    },\n")
	b.WriteString("  },\n")
	b.WriteString("})\n")

	return b.String()
}
