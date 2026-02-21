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
		filepath.Join(outputDir, "node", "Dockerfile"): generateBackendDockerfile(app),
		filepath.Join(outputDir, "docker-compose.yml"): generateDockerCompose(app),
		filepath.Join(outputDir, ".env.example"):       generateEnvExample(app),
		filepath.Join(outputDir, "package.json"):       generatePackageJSON(app),
	}

	// Only generate frontend Dockerfile when a frontend framework is configured.
	if hasFrontend(app) {
		files[filepath.Join(outputDir, "react", "Dockerfile")] = generateFrontendDockerfile(app)
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
	vars := []EnvVar{
		{Name: "DATABASE_URL", Example: "postgresql://postgres:postgres@db:5432/" + DbName(app) + "?schema=public", Comment: "PostgreSQL connection string — use @localhost:5432 for local dev, @db:5432 for Docker"},
		{Name: "JWT_SECRET", Example: "change-me-to-a-random-secret", Comment: "Secret for signing JWT tokens"},
		{Name: "PORT", Example: "3000", Comment: "Backend server port"},
	}

	// Only include VITE_API_URL when a frontend framework is configured.
	if hasFrontend(app) {
		vars = append(vars, EnvVar{Name: "VITE_API_URL", Example: "http://localhost:3000", Comment: "API URL for the React frontend"})
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
