package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/barun-bash/human/internal/ir"
)

// Generator produces Docker infrastructure files from Intent IR.
type Generator struct{}

// Generate writes Dockerfiles, docker-compose.yml, .env.example, and
// a root package.json to outputDir.
func (g Generator) Generate(app *ir.Application, outputDir string) error {
	files := map[string]string{
		filepath.Join(outputDir, "node", "Dockerfile"):  generateBackendDockerfile(app),
		filepath.Join(outputDir, "react", "Dockerfile"): generateFrontendDockerfile(app),
		filepath.Join(outputDir, "docker-compose.yml"):  generateDockerCompose(app),
		filepath.Join(outputDir, ".env.example"):        generateEnvExample(app),
		filepath.Join(outputDir, "package.json"):        generatePackageJSON(app),
	}

	for path, content := range files {
		if err := writeFile(path, content); err != nil {
			return err
		}
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

// collectEnvVars gathers all required environment variables from the IR.
// Returns a sorted list of EnvVar entries.
func collectEnvVars(app *ir.Application) []EnvVar {
	vars := []EnvVar{
		{Name: "DATABASE_URL", Example: "postgresql://postgres:postgres@db:5432/" + dbName(app) + "?schema=public", Comment: "PostgreSQL connection string"},
		{Name: "JWT_SECRET", Example: "change-me-to-a-random-secret", Comment: "Secret for signing JWT tokens"},
		{Name: "PORT", Example: "3000", Comment: "Backend server port"},
		{Name: "VITE_API_URL", Example: "http://localhost:3000", Comment: "API URL for the React frontend"},
	}

	// Integration credentials
	if len(app.Integrations) > 0 {
		for _, integ := range app.Integrations {
			for _, envVar := range integ.Credentials {
				vars = append(vars, EnvVar{
					Name:    envVar,
					Example: "",
					Comment: integ.Service,
				})
			}
		}
	}

	// Sort by name for stable output
	sort.Slice(vars, func(i, j int) bool {
		return vars[i].Name < vars[j].Name
	})

	return vars
}

// EnvVar represents an environment variable entry.
type EnvVar struct {
	Name    string
	Example string
	Comment string
}

// dbName derives a database name from the application name.
func dbName(app *ir.Application) string {
	if app.Name != "" {
		return strings.ToLower(strings.ReplaceAll(app.Name, " ", "_"))
	}
	return "app"
}

// appNameLower returns a lowercase, hyphenated version of the app name.
func appNameLower(app *ir.Application) string {
	if app.Name != "" {
		return strings.ToLower(strings.ReplaceAll(app.Name, " ", "-"))
	}
	return "app"
}
