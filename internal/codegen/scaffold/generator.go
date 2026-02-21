package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/barun-bash/human/internal/ir"
)

// Generator produces project scaffolding files (package.json, tsconfig,
// README, start script, etc.) that make the generated output a runnable project.
type Generator struct{}

// Generate writes all scaffolding files to outputDir.
// It runs after all code generators and the quality engine, and overwrites
// the Docker generator's root package.json and .env.example with enhanced versions.
// Files are generated conditionally based on the app's build config (frontend/backend).
func (g Generator) Generate(app *ir.Application, outputDir string) error {
	frontend := ""
	backend := ""
	if app.Config != nil {
		frontend = strings.ToLower(app.Config.Frontend)
		backend = strings.ToLower(app.Config.Backend)
	}

	files := map[string]string{
		filepath.Join(outputDir, "package.json"):   generateRootPackageJSON(app),
		filepath.Join(outputDir, "README.md"):      generateReadme(app),
		filepath.Join(outputDir, ".env.example"):   generateEnvExample(app),
	}

	// React scaffold files (Vue/Angular/Svelte generators write their own)
	if strings.Contains(frontend, "react") {
		files[filepath.Join(outputDir, "react", "package.json")] = generateReactPackageJSON(app)
		files[filepath.Join(outputDir, "react", "tsconfig.json")] = generateReactTSConfig()
		files[filepath.Join(outputDir, "react", "vite.config.ts")] = generateViteConfig()
	}

	// Vue scaffold files (generator doesn't write package.json/tsconfig)
	if strings.Contains(frontend, "vue") {
		files[filepath.Join(outputDir, "vue", "package.json")] = generateVuePackageJSON(app)
		files[filepath.Join(outputDir, "vue", "tsconfig.json")] = generateVueTSConfig()
	}

	// Angular and Svelte generators already write their own project config files

	// Node backend scaffold files
	if strings.Contains(backend, "node") {
		files[filepath.Join(outputDir, "node", "package.json")] = generateNodePackageJSON(app)
		files[filepath.Join(outputDir, "node", "tsconfig.json")] = generateNodeTSConfig()
	}

	// Python and Go backends don't need scaffold package.json/tsconfig

	for path, content := range files {
		if err := writeFile(path, content); err != nil {
			return err
		}
	}

	// start.sh needs executable permissions
	startPath := filepath.Join(outputDir, "start.sh")
	if err := writeExecutable(startPath, generateStartScript(app)); err != nil {
		return err
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

func writeExecutable(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

// appNameLower returns a lowercase, hyphenated version of the app name.
func appNameLower(app *ir.Application) string {
	if app.Name != "" {
		return strings.ToLower(strings.ReplaceAll(app.Name, " ", "-"))
	}
	return "app"
}
