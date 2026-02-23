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
	backendDir := BackendDir(app)

	files := map[string]string{
		filepath.Join(outputDir, backendDir, "Dockerfile"):    generateBackendDockerfile(app),
		filepath.Join(outputDir, backendDir, ".dockerignore"): generateBackendDockerignore(app),
		filepath.Join(outputDir, "docker-compose.yml"):        generateDockerCompose(app),
		filepath.Join(outputDir, ".env.example"):              generateEnvExample(app),
		filepath.Join(outputDir, ".env"):                      generateEnvFile(app),
		filepath.Join(outputDir, "package.json"):              generatePackageJSON(app),
	}

	// Only generate frontend Dockerfile when a frontend framework is configured.
	if hasFrontend(app) {
		feDir := FrontendDir(app)
		files[filepath.Join(outputDir, feDir, "Dockerfile")] = generateFrontendDockerfile(app)
		files[filepath.Join(outputDir, feDir, ".dockerignore")] = generateFrontendDockerignore(app)
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

// CollectEnvVars gathers all required environment variables from the IR.
// Returns a sorted list of EnvVar entries.
func CollectEnvVars(app *ir.Application) []EnvVar {
	port := BackendPort(app)
	dbSuffix := "?schema=public" // Prisma (Node)
	if dir := BackendDir(app); dir == "go" || dir == "python" {
		dbSuffix = "?sslmode=disable"
	}
	vars := []EnvVar{
		{Name: "DATABASE_URL", Example: "postgresql://postgres:postgres@db:5432/" + DbName(app) + dbSuffix, Comment: "PostgreSQL connection string — use @localhost:5432 for local dev, @db:5432 for Docker"},
		{Name: "JWT_SECRET", Example: "change-me-to-a-random-secret", Comment: "Secret for signing JWT tokens"},
		{Name: "PORT", Example: port, Comment: "Backend server port"},
	}

	// Only include frontend API URL env var when a frontend framework is configured.
	if hasFrontend(app) {
		feEnvName := FrontendAPIEnvName(app)
		vars = append(vars, EnvVar{Name: feEnvName, Example: "http://localhost:" + port, Comment: "API URL for the frontend"})
	}

	// Integration credentials and config-derived env vars
	if len(app.Integrations) > 0 {
		seen := make(map[string]bool)
		for _, integ := range app.Integrations {
			for _, envVar := range integ.Credentials {
				if !seen[envVar] {
					seen[envVar] = true
					vars = append(vars, EnvVar{
						Name:    envVar,
						Example: "",
						Comment: integ.Service,
					})
				}
			}
			// Code generators reference well-known env vars per integration
			// type that are not stored in Credentials (they're hardcoded in
			// the generated TypeScript). Add those here so .env.example and
			// docker-compose.yml stay in sync with the generated code.
			for _, ev := range configEnvVars(integ) {
				if !seen[ev.Name] {
					seen[ev.Name] = true
					vars = append(vars, ev)
				}
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

// configEnvVars returns additional env vars that the code generators hardcode
// for a given integration type.  These aren't in integ.Credentials because the
// generators emit them directly (e.g. process.env.AWS_REGION in storage.ts).
func configEnvVars(integ *ir.Integration) []EnvVar {
	svc := strings.ToLower(integ.Service)
	switch {
	case integ.Type == "storage" && (strings.Contains(svc, "s3") || strings.Contains(svc, "aws")):
		region := "us-east-1"
		if v, ok := integ.Config["region"]; ok {
			region = v
		}
		return []EnvVar{
			{Name: "AWS_REGION", Example: region, Comment: integ.Service},
			{Name: "S3_BUCKET", Example: "", Comment: integ.Service},
		}
	default:
		return nil
	}
}

// hasFrontend returns true when the app has a frontend framework configured.
func hasFrontend(app *ir.Application) bool {
	if app.Config == nil || app.Config.Frontend == "" {
		return false
	}
	return strings.ToLower(app.Config.Frontend) != "none"
}

// DbName derives a database name from the application name.
func DbName(app *ir.Application) string {
	if app.Name != "" {
		return strings.ToLower(strings.ReplaceAll(app.Name, " ", "_"))
	}
	return "app"
}

// AppNameLower returns a lowercase, hyphenated version of the app name.
func AppNameLower(app *ir.Application) string {
	if app.Name != "" {
		return strings.ToLower(strings.ReplaceAll(app.Name, " ", "-"))
	}
	return "app"
}

// BackendDir returns the output subdirectory name for the backend
// based on the configured backend framework.
func BackendDir(app *ir.Application) string {
	if app.Config == nil {
		return "node"
	}
	lower := strings.ToLower(app.Config.Backend)
	switch {
	case strings.Contains(lower, "python") || strings.Contains(lower, "fastapi") || strings.Contains(lower, "django") || strings.Contains(lower, "flask"):
		return "python"
	case lower == "go" || strings.HasPrefix(lower, "go ") || strings.Contains(lower, "gin") || strings.Contains(lower, "fiber") || strings.Contains(lower, "golang"):
		return "go"
	default:
		return "node"
	}
}

// BackendPort returns the default port for the backend runtime.
func BackendPort(app *ir.Application) string {
	if app.Config == nil {
		return "3000"
	}
	lower := strings.ToLower(app.Config.Backend)
	switch {
	case strings.Contains(lower, "python") || strings.Contains(lower, "fastapi") || strings.Contains(lower, "django") || strings.Contains(lower, "flask"):
		return "8000"
	case lower == "go" || strings.HasPrefix(lower, "go ") || strings.Contains(lower, "gin") || strings.Contains(lower, "fiber") || strings.Contains(lower, "golang"):
		return "8080"
	default:
		return "3000"
	}
}

// FrontendDir returns the output subdirectory name for the frontend
// based on the configured frontend framework.
func FrontendDir(app *ir.Application) string {
	if app.Config == nil {
		return "react"
	}
	lower := strings.ToLower(app.Config.Frontend)
	switch {
	case strings.Contains(lower, "vue"):
		return "vue"
	case strings.Contains(lower, "angular"):
		return "angular"
	case strings.Contains(lower, "svelte"):
		return "svelte"
	default:
		return "react"
	}
}

// FrontendAPIEnvName returns the environment variable name used to pass the
// API URL to the frontend build. Vite-based frameworks (React, Vue, Svelte)
// use VITE_API_URL; Angular uses NG_APP_API_URL.
func FrontendAPIEnvName(app *ir.Application) string {
	if app.Config != nil && strings.Contains(strings.ToLower(app.Config.Frontend), "angular") {
		return "NG_APP_API_URL"
	}
	return "VITE_API_URL"
}

// frontendUsesVite returns true if the frontend framework uses Vite for bundling.
func frontendUsesVite(app *ir.Application) bool {
	if app.Config == nil {
		return true
	}
	lower := strings.ToLower(app.Config.Frontend)
	return !strings.Contains(lower, "angular")
}
