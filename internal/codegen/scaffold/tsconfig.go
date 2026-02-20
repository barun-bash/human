package scaffold

import "strings"

// generateNodeTSConfig produces node/tsconfig.json for the Express backend.
func generateNodeTSConfig() string {
	var b strings.Builder

	b.WriteString("{\n")
	b.WriteString("  \"compilerOptions\": {\n")
	b.WriteString("    \"target\": \"ES2022\",\n")
	b.WriteString("    \"module\": \"commonjs\",\n")
	b.WriteString("    \"lib\": [\"ES2022\"],\n")
	b.WriteString("    \"outDir\": \"./dist\",\n")
	b.WriteString("    \"rootDir\": \"./src\",\n")
	b.WriteString("    \"strict\": true,\n")
	b.WriteString("    \"esModuleInterop\": true,\n")
	b.WriteString("    \"skipLibCheck\": true,\n")
	b.WriteString("    \"forceConsistentCasingInFileNames\": true,\n")
	b.WriteString("    \"resolveJsonModule\": true,\n")
	b.WriteString("    \"declaration\": true,\n")
	b.WriteString("    \"declarationMap\": true,\n")
	b.WriteString("    \"sourceMap\": true\n")
	b.WriteString("  },\n")
	b.WriteString("  \"include\": [\"src/**/*\"],\n")
	b.WriteString("  \"exclude\": [\"node_modules\", \"dist\"]\n")
	b.WriteString("}\n")

	return b.String()
}

// generateReactTSConfig produces react/tsconfig.json for the Vite+React frontend.
func generateReactTSConfig() string {
	var b strings.Builder

	b.WriteString("{\n")
	b.WriteString("  \"compilerOptions\": {\n")
	b.WriteString("    \"target\": \"ES2020\",\n")
	b.WriteString("    \"useDefineForClassFields\": true,\n")
	b.WriteString("    \"lib\": [\"ES2020\", \"DOM\", \"DOM.Iterable\"],\n")
	b.WriteString("    \"module\": \"ESNext\",\n")
	b.WriteString("    \"skipLibCheck\": true,\n")
	b.WriteString("    \"moduleResolution\": \"bundler\",\n")
	b.WriteString("    \"allowImportingTsExtensions\": true,\n")
	b.WriteString("    \"isolatedModules\": true,\n")
	b.WriteString("    \"moduleDetection\": \"force\",\n")
	b.WriteString("    \"noEmit\": true,\n")
	b.WriteString("    \"jsx\": \"react-jsx\",\n")
	b.WriteString("    \"strict\": true,\n")
	b.WriteString("    \"noUnusedLocals\": true,\n")
	b.WriteString("    \"noUnusedParameters\": true,\n")
	b.WriteString("    \"noFallthroughCasesInSwitch\": true,\n")
	b.WriteString("    \"forceConsistentCasingInFileNames\": true\n")
	b.WriteString("  },\n")
	b.WriteString("  \"include\": [\"src\"]\n")
	b.WriteString("}\n")

	return b.String()
}
