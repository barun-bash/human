package scaffold

import "strings"

// generateViteConfig produces react/vite.config.ts with the React plugin
// and an API proxy to the backend dev server.
func generateViteConfig() string {
	var b strings.Builder

	b.WriteString("import { defineConfig } from 'vite'\n")
	b.WriteString("import react from '@vitejs/plugin-react'\n")
	b.WriteString("\n")
	b.WriteString("export default defineConfig({\n")
	b.WriteString("  plugins: [react()],\n")
	b.WriteString("  server: {\n")
	b.WriteString("    proxy: {\n")
	b.WriteString("      '/api': {\n")
	b.WriteString("        target: 'http://localhost:3000',\n")
	b.WriteString("        changeOrigin: true,\n")
	b.WriteString("      },\n")
	b.WriteString("    },\n")
	b.WriteString("  },\n")
	b.WriteString("})\n")

	return b.String()
}
