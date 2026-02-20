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
func (g Generator) Generate(app *ir.Application, outputDir string) error {
	files := map[string]string{
		filepath.Join(outputDir, "package.json"):        generateRootPackageJSON(app),
		filepath.Join(outputDir, "node", "package.json"):  generateNodePackageJSON(app),
		filepath.Join(outputDir, "react", "package.json"): generateReactPackageJSON(app),
		filepath.Join(outputDir, "node", "tsconfig.json"):  generateNodeTSConfig(),
		filepath.Join(outputDir, "react", "tsconfig.json"): generateReactTSConfig(),
		filepath.Join(outputDir, "react", "vite.config.ts"): generateViteConfig(),
		filepath.Join(outputDir, "README.md"):             generateReadme(app),
		filepath.Join(outputDir, ".env.example"):          generateEnvExample(app),
	}

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
