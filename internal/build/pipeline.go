package build

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/barun-bash/human/internal/codegen/angular"
	"github.com/barun-bash/human/internal/codegen/architecture"
	"github.com/barun-bash/human/internal/codegen/cicd"
	"github.com/barun-bash/human/internal/codegen/docker"
	"github.com/barun-bash/human/internal/codegen/gobackend"
	"github.com/barun-bash/human/internal/codegen/monitoring"
	"github.com/barun-bash/human/internal/codegen/node"
	"github.com/barun-bash/human/internal/codegen/postgres"
	"github.com/barun-bash/human/internal/codegen/python"
	"github.com/barun-bash/human/internal/codegen/react"
	"github.com/barun-bash/human/internal/codegen/scaffold"
	"github.com/barun-bash/human/internal/codegen/storybook"
	"github.com/barun-bash/human/internal/codegen/svelte"
	"github.com/barun-bash/human/internal/codegen/terraform"
	"github.com/barun-bash/human/internal/codegen/vue"
	"github.com/barun-bash/human/internal/ir"
	"github.com/barun-bash/human/internal/quality"
)

// Result tracks the output of a single generator.
type Result struct {
	Name  string
	Dir   string
	Files int
}

// MatchesGoBackend checks if the backend config indicates Go without
// false-matching strings like "django" or "mongodb".
func MatchesGoBackend(backend string) bool {
	lower := strings.ToLower(backend)
	if lower == "go" || strings.HasPrefix(lower, "go ") {
		return true
	}
	for _, kw := range []string{"gin", "fiber", "golang"} {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// CountFiles returns the number of regular files under dir.
func CountFiles(dir string) int {
	count := 0
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			count++
		}
		return nil
	})
	return count
}

// RunGenerators dispatches all code generators based on the app's build config,
// then runs the quality engine and scaffolder. Returns build results for each
// generator and the quality result.
func RunGenerators(app *ir.Application, outputDir string) ([]Result, *quality.Result, error) {
	var results []Result

	frontendLower := ""
	backendLower := ""
	deployLower := ""
	databaseLower := ""
	if app.Config != nil {
		frontendLower = strings.ToLower(app.Config.Frontend)
		backendLower = strings.ToLower(app.Config.Backend)
		deployLower = strings.ToLower(app.Config.Deploy)
		databaseLower = strings.ToLower(app.Config.Database)
	}

	// Frontend generators
	if strings.Contains(frontendLower, "react") {
		dir := filepath.Join(outputDir, "react")
		g := react.Generator{}
		if err := g.Generate(app, dir); err != nil {
			return nil, nil, fmt.Errorf("React codegen: %w", err)
		}
		results = append(results, Result{"react", dir, CountFiles(dir)})
	}
	if strings.Contains(frontendLower, "vue") {
		dir := filepath.Join(outputDir, "vue")
		g := vue.Generator{}
		if err := g.Generate(app, dir); err != nil {
			return nil, nil, fmt.Errorf("Vue codegen: %w", err)
		}
		results = append(results, Result{"vue", dir, CountFiles(dir)})
	}
	if strings.Contains(frontendLower, "angular") {
		dir := filepath.Join(outputDir, "angular")
		g := angular.Generator{}
		if err := g.Generate(app, dir); err != nil {
			return nil, nil, fmt.Errorf("Angular codegen: %w", err)
		}
		results = append(results, Result{"angular", dir, CountFiles(dir)})
	}
	if strings.Contains(frontendLower, "svelte") {
		dir := filepath.Join(outputDir, "svelte")
		g := svelte.Generator{}
		if err := g.Generate(app, dir); err != nil {
			return nil, nil, fmt.Errorf("Svelte codegen: %w", err)
		}
		results = append(results, Result{"svelte", dir, CountFiles(dir)})
	}

	// Storybook — generates into the frontend directory that was just created
	if frontendLower != "" {
		fw := storybook.GetFramework(app)
		// Determine the frontend output directory
		frontendDir := ""
		switch {
		case strings.Contains(frontendLower, "react"):
			frontendDir = filepath.Join(outputDir, "react")
		case strings.Contains(frontendLower, "vue"):
			frontendDir = filepath.Join(outputDir, "vue")
		case strings.Contains(frontendLower, "angular"):
			frontendDir = filepath.Join(outputDir, "angular")
		case strings.Contains(frontendLower, "svelte"):
			frontendDir = filepath.Join(outputDir, "svelte")
		}
		if frontendDir != "" {
			sg := storybook.Generator{}
			if err := sg.Generate(app, frontendDir); err != nil {
				return nil, nil, fmt.Errorf("Storybook codegen: %w", err)
			}
			results = append(results, Result{"storybook", frontendDir, CountFiles(filepath.Join(frontendDir, ".storybook")) + CountFiles(filepath.Join(frontendDir, "src", "stories"))})
			_ = fw // used by scaffold for dependency injection
		}
	}

	// Backend generators
	if strings.Contains(backendLower, "node") {
		dir := filepath.Join(outputDir, "node")
		g := node.Generator{}
		if err := g.Generate(app, dir); err != nil {
			return nil, nil, fmt.Errorf("Node codegen: %w", err)
		}
		results = append(results, Result{"node", dir, CountFiles(dir)})
	}
	if strings.Contains(backendLower, "python") {
		dir := filepath.Join(outputDir, "python")
		g := python.Generator{}
		if err := g.Generate(app, dir); err != nil {
			return nil, nil, fmt.Errorf("Python codegen: %w", err)
		}
		results = append(results, Result{"python", dir, CountFiles(dir)})
	}
	if MatchesGoBackend(backendLower) {
		dir := filepath.Join(outputDir, "go")
		g := gobackend.Generator{}
		if err := g.Generate(app, dir); err != nil {
			return nil, nil, fmt.Errorf("Go codegen: %w", err)
		}
		results = append(results, Result{"go", dir, CountFiles(dir)})
	}

	// Database generator
	if strings.Contains(databaseLower, "postgres") {
		dir := filepath.Join(outputDir, "postgres")
		g := postgres.Generator{}
		if err := g.Generate(app, dir); err != nil {
			return nil, nil, fmt.Errorf("PostgreSQL codegen: %w", err)
		}
		results = append(results, Result{"postgres", dir, CountFiles(dir)})
	}

	// Docker — conditional on deploy config
	if strings.Contains(deployLower, "docker") {
		before := CountFiles(outputDir)
		g := docker.Generator{}
		if err := g.Generate(app, outputDir); err != nil {
			return nil, nil, fmt.Errorf("Docker codegen: %w", err)
		}
		after := CountFiles(outputDir)
		results = append(results, Result{"docker", outputDir, after - before})
	}

	// CI/CD — always runs
	{
		cicdDir := filepath.Join(outputDir, ".github")
		g := cicd.Generator{}
		if err := g.Generate(app, outputDir); err != nil {
			return nil, nil, fmt.Errorf("CI/CD codegen: %w", err)
		}
		results = append(results, Result{"cicd", outputDir, CountFiles(cicdDir)})
	}

	// Terraform — conditional on deploy config (aws, gcp, or terraform keyword)
	if strings.Contains(deployLower, "aws") || strings.Contains(deployLower, "gcp") || strings.Contains(deployLower, "terraform") {
		dir := filepath.Join(outputDir, "terraform")
		g := terraform.Generator{}
		if err := g.Generate(app, dir); err != nil {
			return nil, nil, fmt.Errorf("Terraform codegen: %w", err)
		}
		results = append(results, Result{"terraform", dir, CountFiles(dir)})
	}

	// Architecture — conditional on architecture style
	if app.Architecture != nil && app.Architecture.Style != "" {
		g := architecture.Generator{}
		if err := g.Generate(app, outputDir); err != nil {
			return nil, nil, fmt.Errorf("Architecture codegen: %w", err)
		}
		archDir := filepath.Join(outputDir, "services")
		fnDir := filepath.Join(outputDir, "functions")
		archFiles := CountFiles(archDir) + CountFiles(fnDir) + CountFiles(filepath.Join(outputDir, "gateway"))
		if archFiles > 0 {
			results = append(results, Result{"architecture", outputDir, archFiles})
		}
	}

	// Monitoring — conditional on monitoring rules
	if len(app.Monitoring) > 0 {
		dir := filepath.Join(outputDir, "monitoring")
		g := monitoring.Generator{}
		if err := g.Generate(app, dir); err != nil {
			return nil, nil, fmt.Errorf("Monitoring codegen: %w", err)
		}
		results = append(results, Result{"monitoring", dir, CountFiles(dir)})
	}

	// Quality engine — always runs after code generators
	qResult, err := quality.Run(app, outputDir)
	if err != nil {
		return nil, nil, fmt.Errorf("quality engine: %w", err)
	}
	qualityFiles := qResult.TestFiles + qResult.ComponentTestFiles + qResult.EdgeTestFiles + 3
	results = append(results, Result{"quality", outputDir, qualityFiles})

	// Scaffolder — always runs last
	{
		sg := scaffold.Generator{}
		if err := sg.Generate(app, outputDir); err != nil {
			return nil, nil, fmt.Errorf("scaffold: %w", err)
		}
		scaffoldFiles := 0
		for _, name := range []string{"package.json", "README.md", ".env.example", "start.sh"} {
			if _, err := os.Stat(filepath.Join(outputDir, name)); err == nil {
				scaffoldFiles++
			}
		}
		for _, sub := range []string{"node", "react", "vue"} {
			for _, name := range []string{"package.json", "tsconfig.json", "vite.config.ts"} {
				if _, err := os.Stat(filepath.Join(outputDir, sub, name)); err == nil {
					scaffoldFiles++
				}
			}
		}
		results = append(results, Result{"scaffold", outputDir, scaffoldFiles})
	}

	return results, qResult, nil
}
