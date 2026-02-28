package build

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	Name     string
	Dir      string
	Files    int
	Duration time.Duration
}

// BuildTiming holds the total build duration.
type BuildTiming struct {
	Total time.Duration
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

// PlanStages returns the list of stage names that will run for the given app.
// Use this to pre-populate a progress display.
func PlanStages(app *ir.Application) []string {
	var stages []string

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

	if strings.Contains(frontendLower, "react") {
		stages = append(stages, "Generating React frontend")
	}
	if strings.Contains(frontendLower, "vue") {
		stages = append(stages, "Generating Vue frontend")
	}
	if strings.Contains(frontendLower, "angular") {
		stages = append(stages, "Generating Angular frontend")
	}
	if strings.Contains(frontendLower, "svelte") {
		stages = append(stages, "Generating Svelte frontend")
	}
	if frontendLower != "" {
		stages = append(stages, "Generating Storybook stories")
	}
	if strings.Contains(backendLower, "node") {
		stages = append(stages, "Generating Node.js backend")
	}
	if strings.Contains(backendLower, "python") {
		stages = append(stages, "Generating Python backend")
	}
	if MatchesGoBackend(backendLower) {
		stages = append(stages, "Generating Go backend")
	}
	if strings.Contains(databaseLower, "postgres") {
		stages = append(stages, "Generating PostgreSQL schema")
	}
	if strings.Contains(deployLower, "docker") {
		stages = append(stages, "Generating Docker configuration")
	}
	stages = append(stages, "Generating CI/CD pipelines")
	if strings.Contains(deployLower, "aws") || strings.Contains(deployLower, "gcp") || strings.Contains(deployLower, "terraform") {
		stages = append(stages, "Generating Terraform infrastructure")
	}
	if app.Architecture != nil && app.Architecture.Style != "" {
		stages = append(stages, "Generating architecture layout")
	}
	if len(app.Monitoring) > 0 {
		stages = append(stages, "Generating monitoring configuration")
	}
	stages = append(stages, "Running quality checks")
	stages = append(stages, "Scaffolding project files")

	return stages
}

// ProgressFunc is called before each build stage with the stage name.
type ProgressFunc func(stage string)

// RunGenerators dispatches all code generators based on the app's build config,
// then runs the quality engine and scaffolder. Returns build results for each
// generator, the quality result, and build timing.
func RunGenerators(app *ir.Application, outputDir string) ([]Result, *quality.Result, *BuildTiming, error) {
	return RunGeneratorsWithProgress(app, outputDir, nil)
}

// RunGeneratorsWithProgress is like RunGenerators but calls progress before each stage.
func RunGeneratorsWithProgress(app *ir.Application, outputDir string, progress ProgressFunc) ([]Result, *quality.Result, *BuildTiming, error) {
	buildStart := time.Now()
	var results []Result

	report := func(stage string) {
		if progress != nil {
			progress(stage)
		}
	}

	timeGen := func(name, dir string, files int, start time.Time) Result {
		return Result{Name: name, Dir: dir, Files: files, Duration: time.Since(start)}
	}

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
		report("Generating React frontend")
		start := time.Now()
		dir := filepath.Join(outputDir, "react")
		g := react.Generator{}
		if err := g.Generate(app, dir); err != nil {
			return nil, nil, nil, fmt.Errorf("React codegen: %w", err)
		}
		results = append(results, timeGen("react", dir, CountFiles(dir), start))
	}
	if strings.Contains(frontendLower, "vue") {
		report("Generating Vue frontend")
		start := time.Now()
		dir := filepath.Join(outputDir, "vue")
		g := vue.Generator{}
		if err := g.Generate(app, dir); err != nil {
			return nil, nil, nil, fmt.Errorf("Vue codegen: %w", err)
		}
		results = append(results, timeGen("vue", dir, CountFiles(dir), start))
	}
	if strings.Contains(frontendLower, "angular") {
		report("Generating Angular frontend")
		start := time.Now()
		dir := filepath.Join(outputDir, "angular")
		g := angular.Generator{}
		if err := g.Generate(app, dir); err != nil {
			return nil, nil, nil, fmt.Errorf("Angular codegen: %w", err)
		}
		results = append(results, timeGen("angular", dir, CountFiles(dir), start))
	}
	if strings.Contains(frontendLower, "svelte") {
		report("Generating Svelte frontend")
		start := time.Now()
		dir := filepath.Join(outputDir, "svelte")
		g := svelte.Generator{}
		if err := g.Generate(app, dir); err != nil {
			return nil, nil, nil, fmt.Errorf("Svelte codegen: %w", err)
		}
		results = append(results, timeGen("svelte", dir, CountFiles(dir), start))
	}

	// Storybook — generates into the frontend directory that was just created
	if frontendLower != "" {
		report("Generating Storybook stories")
		start := time.Now()
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
				return nil, nil, nil, fmt.Errorf("Storybook codegen: %w", err)
			}
			sbFiles := CountFiles(filepath.Join(frontendDir, ".storybook")) + CountFiles(filepath.Join(frontendDir, "src", "stories"))
			results = append(results, timeGen("storybook", frontendDir, sbFiles, start))
			_ = fw // used by scaffold for dependency injection
		}
	}

	// Backend generators
	if strings.Contains(backendLower, "node") {
		report("Generating Node.js backend")
		start := time.Now()
		dir := filepath.Join(outputDir, "node")
		g := node.Generator{}
		if err := g.Generate(app, dir); err != nil {
			return nil, nil, nil, fmt.Errorf("Node codegen: %w", err)
		}
		results = append(results, timeGen("node", dir, CountFiles(dir), start))
	}
	if strings.Contains(backendLower, "python") {
		report("Generating Python backend")
		start := time.Now()
		dir := filepath.Join(outputDir, "python")
		g := python.Generator{}
		if err := g.Generate(app, dir); err != nil {
			return nil, nil, nil, fmt.Errorf("Python codegen: %w", err)
		}
		results = append(results, timeGen("python", dir, CountFiles(dir), start))
	}
	if MatchesGoBackend(backendLower) {
		report("Generating Go backend")
		start := time.Now()
		dir := filepath.Join(outputDir, "go")
		g := gobackend.Generator{}
		if err := g.Generate(app, dir); err != nil {
			return nil, nil, nil, fmt.Errorf("Go codegen: %w", err)
		}
		results = append(results, timeGen("go", dir, CountFiles(dir), start))
	}

	// Database generator
	if strings.Contains(databaseLower, "postgres") {
		report("Generating PostgreSQL schema")
		start := time.Now()
		dir := filepath.Join(outputDir, "postgres")
		g := postgres.Generator{}
		if err := g.Generate(app, dir); err != nil {
			return nil, nil, nil, fmt.Errorf("PostgreSQL codegen: %w", err)
		}
		results = append(results, timeGen("postgres", dir, CountFiles(dir), start))
	}

	// Docker — conditional on deploy config
	if strings.Contains(deployLower, "docker") {
		report("Generating Docker configuration")
		start := time.Now()
		before := CountFiles(outputDir)
		g := docker.Generator{}
		if err := g.Generate(app, outputDir); err != nil {
			return nil, nil, nil, fmt.Errorf("Docker codegen: %w", err)
		}
		after := CountFiles(outputDir)
		results = append(results, timeGen("docker", outputDir, after-before, start))
	}

	// CI/CD — always runs
	report("Generating CI/CD pipelines")
	{
		start := time.Now()
		cicdDir := filepath.Join(outputDir, ".github")
		g := cicd.Generator{}
		if err := g.Generate(app, outputDir); err != nil {
			return nil, nil, nil, fmt.Errorf("CI/CD codegen: %w", err)
		}
		results = append(results, timeGen("cicd", outputDir, CountFiles(cicdDir), start))
	}

	// Terraform — conditional on deploy config (aws, gcp, or terraform keyword)
	if strings.Contains(deployLower, "aws") || strings.Contains(deployLower, "gcp") || strings.Contains(deployLower, "terraform") {
		report("Generating Terraform infrastructure")
		start := time.Now()
		dir := filepath.Join(outputDir, "terraform")
		g := terraform.Generator{}
		if err := g.Generate(app, dir); err != nil {
			return nil, nil, nil, fmt.Errorf("Terraform codegen: %w", err)
		}
		results = append(results, timeGen("terraform", dir, CountFiles(dir), start))
	}

	// Architecture — conditional on architecture style
	if app.Architecture != nil && app.Architecture.Style != "" {
		report("Generating architecture layout")
		start := time.Now()
		g := architecture.Generator{}
		if err := g.Generate(app, outputDir); err != nil {
			return nil, nil, nil, fmt.Errorf("Architecture codegen: %w", err)
		}
		archDir := filepath.Join(outputDir, "services")
		fnDir := filepath.Join(outputDir, "functions")
		archFiles := CountFiles(archDir) + CountFiles(fnDir) + CountFiles(filepath.Join(outputDir, "gateway"))
		if archFiles > 0 {
			results = append(results, timeGen("architecture", outputDir, archFiles, start))
		}
	}

	// Monitoring — conditional on monitoring rules
	if len(app.Monitoring) > 0 {
		report("Generating monitoring configuration")
		start := time.Now()
		dir := filepath.Join(outputDir, "monitoring")
		g := monitoring.Generator{}
		if err := g.Generate(app, dir); err != nil {
			return nil, nil, nil, fmt.Errorf("Monitoring codegen: %w", err)
		}
		results = append(results, timeGen("monitoring", dir, CountFiles(dir), start))
	}

	// Quality engine — always runs after code generators
	report("Running quality checks")
	{
		start := time.Now()
		qResult, err := quality.Run(app, outputDir)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("quality engine: %w", err)
		}
		qualityFiles := qResult.TestFiles + qResult.ComponentTestFiles + qResult.EdgeTestFiles + 3
		results = append(results, timeGen("quality", outputDir, qualityFiles, start))

		// Scaffolder — always runs last
		report("Scaffolding project files")
		scaffoldStart := time.Now()
		sg := scaffold.Generator{}
		if err := sg.Generate(app, outputDir); err != nil {
			return nil, nil, nil, fmt.Errorf("scaffold: %w", err)
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
		results = append(results, timeGen("scaffold", outputDir, scaffoldFiles, scaffoldStart))

		timing := &BuildTiming{Total: time.Since(buildStart)}
		return results, qResult, timing, nil
	}
}
