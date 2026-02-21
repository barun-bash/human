package main

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/barun-bash/human/internal/analyzer"
	"github.com/barun-bash/human/internal/cli"
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
	"github.com/barun-bash/human/internal/config"
	cerr "github.com/barun-bash/human/internal/errors"
	"github.com/barun-bash/human/internal/ir"
	"github.com/barun-bash/human/internal/llm"
	_ "github.com/barun-bash/human/internal/llm/providers" // register providers
	"github.com/barun-bash/human/internal/parser"
	"github.com/barun-bash/human/internal/quality"
)

var version = "0.4.0"

func main() {
	// Parse global --no-color flag before command dispatch
	args := filterGlobalFlags(os.Args[1:])

	if len(args) < 1 {
		printUsage()
		os.Exit(0)
	}

	switch args[0] {
	case "version", "--version", "-v":
		fmt.Printf("human v%s\n", version)
	case "help", "--help", "-h":
		printUsage()
	case "check":
		cmdCheck()
	case "build":
		cmdBuild()
	case "init":
		cmdInit()
	case "run":
		cmdRun()
	case "test":
		cmdTest()
	case "audit":
		cmdAudit()
	case "deploy":
		cmdDeploy()
	case "eject":
		cmdEject()
	case "ask":
		cmdAsk()
	case "suggest":
		cmdSuggest()
	case "edit":
		cmdEdit()
	case "convert":
		cmdConvert()
	default:
		fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Unknown command: %s", args[0])))
		fmt.Fprintln(os.Stderr)
		printUsage()
		os.Exit(1)
	}
}

// filterGlobalFlags strips --no-color from the args list and applies it.
func filterGlobalFlags(args []string) []string {
	var filtered []string
	for _, arg := range args {
		if arg == "--no-color" {
			cli.ColorEnabled = false
		} else {
			filtered = append(filtered, arg)
		}
	}
	return filtered
}

// ── check ──

func cmdCheck() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: human check <file.human>")
		os.Exit(1)
	}
	file := os.Args[2]

	source, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Error reading %s: %v", file, err)))
		os.Exit(1)
	}

	prog, err := parser.Parse(string(source))
	if err != nil {
		fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Error in %s: %v", file, err)))
		os.Exit(1)
	}

	// Semantic analysis
	app, irErr := ir.Build(prog)
	if irErr != nil {
		fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("IR build error: %v", irErr)))
		os.Exit(1)
	}

	errs := analyzer.Analyze(app, file)
	if errs.HasWarnings() {
		for _, w := range errs.Warnings() {
			printDiagnostic(w)
		}
	}
	if errs.HasErrors() {
		for _, e := range errs.Errors() {
			printDiagnostic(e)
		}
		fmt.Fprintf(os.Stderr, "\n%s\n", cli.Error(fmt.Sprintf("%d error(s) found", len(errs.Errors()))))
		os.Exit(1)
	}

	// Summarize what was found
	var parts []string
	if len(prog.Data) > 0 {
		parts = append(parts, fmt.Sprintf("%d data model%s", len(prog.Data), plural(len(prog.Data))))
	}
	if len(prog.Pages) > 0 {
		parts = append(parts, fmt.Sprintf("%d page%s", len(prog.Pages), plural(len(prog.Pages))))
	}
	if len(prog.Components) > 0 {
		parts = append(parts, fmt.Sprintf("%d component%s", len(prog.Components), plural(len(prog.Components))))
	}
	if len(prog.APIs) > 0 {
		parts = append(parts, fmt.Sprintf("%d API%s", len(prog.APIs), plural(len(prog.APIs))))
	}
	if len(prog.Policies) > 0 {
		parts = append(parts, fmt.Sprintf("%d polic%s", len(prog.Policies), pluralY(len(prog.Policies))))
	}
	if len(prog.Workflows) > 0 {
		parts = append(parts, fmt.Sprintf("%d workflow%s", len(prog.Workflows), plural(len(prog.Workflows))))
	}
	if len(prog.Integrations) > 0 {
		parts = append(parts, fmt.Sprintf("%d integration%s", len(prog.Integrations), plural(len(prog.Integrations))))
	}
	if len(prog.Environments) > 0 {
		parts = append(parts, fmt.Sprintf("%d environment%s", len(prog.Environments), plural(len(prog.Environments))))
	}
	if len(prog.ErrorHandlers) > 0 {
		parts = append(parts, fmt.Sprintf("%d error handler%s", len(prog.ErrorHandlers), plural(len(prog.ErrorHandlers))))
	}

	msg := fmt.Sprintf("%s is valid", file)
	if len(parts) > 0 {
		msg += " — " + strings.Join(parts, ", ")
	}
	fmt.Println(cli.Success(msg))
}

// ── build ──

func cmdBuild() {
	// Parse flags
	inspect := false
	watch := false
	var file string
	for _, arg := range os.Args[2:] {
		switch arg {
		case "--inspect":
			inspect = true
		case "--watch", "-w":
			watch = true
		default:
			if !strings.HasPrefix(arg, "-") {
				file = arg
			}
		}
	}

	if file == "" {
		fmt.Fprintln(os.Stderr, "Usage: human build [--inspect] [--watch] <file.human>")
		os.Exit(1)
	}

	if watch {
		cmdBuildWatch(file)
		return
	}

	source, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Error reading %s: %v", file, err)))
		os.Exit(1)
	}

	prog, err := parser.Parse(string(source))
	if err != nil {
		fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Parse error in %s: %v", file, err)))
		os.Exit(1)
	}

	app, err := ir.Build(prog)
	if err != nil {
		fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("IR build error: %v", err)))
		os.Exit(1)
	}

	// Semantic analysis
	errs := analyzer.Analyze(app, file)
	if errs.HasWarnings() {
		for _, w := range errs.Warnings() {
			printDiagnostic(w)
		}
	}
	if errs.HasErrors() {
		for _, e := range errs.Errors() {
			printDiagnostic(e)
		}
		fmt.Fprintf(os.Stderr, "\n%s\n", cli.Error(fmt.Sprintf("%d error(s) found — build aborted", len(errs.Errors()))))
		os.Exit(1)
	}

	yaml, err := ir.ToYAML(app)
	if err != nil {
		fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Serialization error: %v", err)))
		os.Exit(1)
	}

	if inspect {
		fmt.Print(yaml)
		return
	}

	// Write IR to .human/intent/<name>.yaml
	outDir := filepath.Join(".human", "intent")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Error creating output directory: %v", err)))
		os.Exit(1)
	}

	base := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	outFile := filepath.Join(outDir, base+".yaml")

	if err := os.WriteFile(outFile, []byte(yaml), 0644); err != nil {
		fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Error writing %s: %v", outFile, err)))
		os.Exit(1)
	}

	// Print summary
	fmt.Printf("Built %s → %s\n", file, outFile)
	printIRSummary(app)

	// Run all code generators
	outputDir := filepath.Join(".human", "output")
	results, genErr := runGenerators(app, outputDir)
	if genErr != nil {
		fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Build failed: %v", genErr)))
		os.Exit(1)
	}

	printBuildSummary(results, outputDir)
}

func printIRSummary(app *ir.Application) {
	fmt.Println(cli.Info(fmt.Sprintf("  app:          %s (%s)", app.Name, app.Platform)))
	if app.Config != nil {
		fmt.Println(cli.Info(fmt.Sprintf("  config:       %s / %s / %s", app.Config.Frontend, app.Config.Backend, app.Config.Database)))
	}
	if len(app.Data) > 0 {
		fmt.Printf("  data models:  %d\n", len(app.Data))
	}
	if len(app.Pages) > 0 {
		fmt.Printf("  pages:        %d\n", len(app.Pages))
	}
	if len(app.Components) > 0 {
		fmt.Printf("  components:   %d\n", len(app.Components))
	}
	if len(app.APIs) > 0 {
		fmt.Printf("  APIs:         %d\n", len(app.APIs))
	}
	if len(app.Policies) > 0 {
		fmt.Printf("  policies:     %d\n", len(app.Policies))
	}
	if len(app.Workflows) > 0 {
		fmt.Printf("  workflows:    %d\n", len(app.Workflows))
	}
	if len(app.Pipelines) > 0 {
		fmt.Printf("  pipelines:    %d\n", len(app.Pipelines))
	}
	if app.Auth != nil && len(app.Auth.Methods) > 0 {
		fmt.Printf("  auth methods: %d\n", len(app.Auth.Methods))
	}
	if app.Database != nil {
		fmt.Printf("  database:     %s\n", app.Database.Engine)
	}
	if len(app.Integrations) > 0 {
		fmt.Printf("  integrations: %d\n", len(app.Integrations))
	}
	if len(app.Environments) > 0 {
		fmt.Printf("  environments: %d\n", len(app.Environments))
	}
	if app.Architecture != nil {
		fmt.Printf("  architecture: %s\n", app.Architecture.Style)
	}
	if len(app.Monitoring) > 0 {
		fmt.Printf("  monitoring:   %d rule(s)\n", len(app.Monitoring))
	}
}

// ── init ──

func cmdInit() {
	name := ""
	if len(os.Args) >= 3 && !strings.HasPrefix(os.Args[2], "-") {
		name = os.Args[2]
	}
	if name == "" {
		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, cli.Error("Could not determine current directory"))
			os.Exit(1)
		}
		name = filepath.Base(dir)
	}

	scanner := bufio.NewScanner(os.Stdin)

	platform := prompt(scanner, "Platform", []string{"web", "mobile", "api"}, "web")
	frontend := prompt(scanner, "Frontend", []string{"React", "Vue", "Angular", "Svelte", "None"}, "React")
	backend := prompt(scanner, "Backend", []string{"Node", "Python", "Go"}, "Node")
	database := prompt(scanner, "Database", []string{"PostgreSQL", "MySQL", "SQLite"}, "PostgreSQL")

	// Create project directory
	if err := os.MkdirAll(name, 0755); err != nil {
		fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Could not create directory %s: %v", name, err)))
		os.Exit(1)
	}

	// Generate starter app.human
	content := generateTemplate(name, platform, frontend, backend, database)
	outPath := filepath.Join(name, "app.human")
	if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
		fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Could not write %s: %v", outPath, err)))
		os.Exit(1)
	}

	fmt.Println(cli.Success(fmt.Sprintf("Created %s — run 'human check %s' to validate, 'human build %s' to compile", outPath, outPath, outPath)))
}

// prompt asks the user to choose from options with a default.
func prompt(scanner *bufio.Scanner, label string, options []string, defaultVal string) string {
	fmt.Printf("%s (%s) [%s]: ", label, strings.Join(options, "/"), defaultVal)
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input != "" {
			// Match case-insensitively against options
			for _, opt := range options {
				if strings.EqualFold(input, opt) {
					return opt
				}
			}
			return input
		}
	}
	return defaultVal
}

func generateTemplate(name, platform, frontend, backend, database string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "app %s is a %s application\n\n", name, platform)

	b.WriteString("# ── Data Models ──\n\n")
	b.WriteString("data User:\n")
	b.WriteString("  name is text, required\n")
	b.WriteString("  email is email, required, unique\n")
	b.WriteString("  password is text, required, encrypted\n\n")

	b.WriteString("# ── Pages ──\n\n")
	b.WriteString("page Home:\n")
	b.WriteString("  show heading \"Welcome to " + name + "\"\n\n")

	b.WriteString("# ── APIs ──\n\n")
	b.WriteString("api SignUp:\n")
	b.WriteString("  accepts name, email, password\n")
	b.WriteString("  create User with name, email, password\n")
	b.WriteString("  respond with success\n\n")

	b.WriteString("# ── Build ──\n\n")
	b.WriteString("build with:\n")
	fmt.Fprintf(&b, "  frontend using %s\n", frontend)
	fmt.Fprintf(&b, "  backend using %s\n", backend)
	fmt.Fprintf(&b, "  database using %s\n", database)
	b.WriteString("  deploy to Docker\n")

	return b.String()
}

// ── run ──

func cmdRun() {
	outputDir := filepath.Join(".human", "output")

	startSh := filepath.Join(outputDir, "start.sh")
	pkgJSON := filepath.Join(outputDir, "package.json")

	if _, err := os.Stat(startSh); err == nil {
		fmt.Println(cli.Info("Starting application via start.sh..."))
		cmd := exec.Command("bash", "start.sh")
		cmd.Dir = outputDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Run failed: %v", err)))
			os.Exit(1)
		}
		return
	}

	if _, err := os.Stat(pkgJSON); err == nil {
		fmt.Println(cli.Info("Starting application via npm run dev..."))
		cmd := exec.Command("npm", "run", "dev")
		cmd.Dir = outputDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Run failed: %v", err)))
			os.Exit(1)
		}
		return
	}

	fmt.Fprintln(os.Stderr, cli.Error("No build found. Run 'human build <file>' first."))
	os.Exit(1)
}

// ── test ──

func cmdTest() {
	outputDir := filepath.Join(".human", "output")

	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, cli.Error("No build found. Run 'human build <file>' first."))
		os.Exit(1)
	}

	fmt.Println(cli.Info("Running tests..."))
	cmd := exec.Command("npm", "test")
	cmd.Dir = outputDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Test failed: %v", err)))
		os.Exit(1)
	}
}

// ── audit ──

func cmdAudit() {
	outputDir := filepath.Join(".human", "output")

	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, cli.Error("No build found. Run 'human build <file>' first."))
		os.Exit(1)
	}

	reportPath := filepath.Join(outputDir, "security-report.md")
	report, err := os.ReadFile(reportPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, cli.Error("No security report found. Run 'human build <file>' to generate one."))
		os.Exit(1)
	}

	fmt.Println(cli.Info("Security & Quality Audit"))
	fmt.Println(cli.Info(strings.Repeat("─", 40)))
	fmt.Println()

	// Parse and colorize the report
	for _, line := range strings.Split(string(report), "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.Contains(trimmed, "CRITICAL") || strings.Contains(trimmed, "❌"):
			fmt.Println(cli.Error(line))
		case strings.Contains(trimmed, "WARNING") || strings.Contains(trimmed, "⚠"):
			fmt.Println(cli.Warn(line))
		case strings.Contains(trimmed, "✅") || strings.Contains(trimmed, "PASS"):
			fmt.Println(cli.Success(line))
		case strings.HasPrefix(trimmed, "#"):
			fmt.Println(cli.Info(line))
		default:
			fmt.Println(line)
		}
	}
}

// ── eject ──

func cmdEject() {
	outputDir := filepath.Join(".human", "output")

	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, cli.Error("No build found. Run 'human build <file>' first."))
		os.Exit(1)
	}

	// Determine target directory
	target := "output"
	if len(os.Args) >= 3 && !strings.HasPrefix(os.Args[2], "-") {
		target = os.Args[2]
	}

	if _, err := os.Stat(target); err == nil {
		fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Directory %q already exists. Choose a different path or remove it first.", target)))
		os.Exit(1)
	}

	// Copy all files from .human/output/ to target
	err := filepath.WalkDir(outputDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(outputDir, path)
		destPath := filepath.Join(target, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Strip "Generated by Human compiler" comments
		cleaned := stripGeneratedComments(string(content))

		return os.WriteFile(destPath, []byte(cleaned), 0644)
	})

	if err != nil {
		fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Eject failed: %v", err)))
		os.Exit(1)
	}

	fmt.Println(cli.Success(fmt.Sprintf("Ejected to %s/ — this is now a standalone project. No Human dependency required.", target)))
}

// stripGeneratedComments removes "Generated by Human compiler" lines from file content.
func stripGeneratedComments(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "Generated by Human compiler") {
			continue
		}
		result = append(result, line)
	}
	return strings.Join(result, "\n")
}

// ── deploy ──

func cmdDeploy() {
	// Parse flags
	dryRun := false
	envName := ""
	var file string
	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dry-run":
			dryRun = true
		case "--env", "-e":
			if i+1 < len(args) {
				i++
				envName = args[i]
			} else {
				fmt.Fprintln(os.Stderr, cli.Error("--env requires a value (e.g. --env staging)"))
				os.Exit(1)
			}
		default:
			if !strings.HasPrefix(args[i], "-") {
				file = args[i]
			}
		}
	}

	// Auto-detect .human file if not provided
	if file == "" {
		matches, _ := filepath.Glob("*.human")
		if len(matches) == 1 {
			file = matches[0]
		} else if len(matches) > 1 {
			fmt.Fprintln(os.Stderr, cli.Error("Multiple .human files found. Specify which one to deploy."))
			fmt.Fprintln(os.Stderr, "Usage: human deploy [--dry-run] [--env <name>] <file.human>")
			os.Exit(1)
		} else {
			fmt.Fprintln(os.Stderr, cli.Error("No .human file found. Specify a file to deploy."))
			fmt.Fprintln(os.Stderr, "Usage: human deploy [--dry-run] [--env <name>] <file.human>")
			os.Exit(1)
		}
	}

	outputDir := filepath.Join(".human", "output")

	// Build the project
	fmt.Println(cli.Info("Building before deploy..."))
	if err := runBuild(file); err != nil {
		fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Build failed: %v", err)))
		os.Exit(1)
	}

	// Load the IR to read config
	source, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Error reading %s: %v", file, err)))
		os.Exit(1)
	}
	prog, err := parser.Parse(string(source))
	if err != nil {
		fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Parse error: %v", err)))
		os.Exit(1)
	}
	app, err := ir.Build(prog)
	if err != nil {
		fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("IR error: %v", err)))
		os.Exit(1)
	}

	// Determine deploy target
	deployTarget := ""
	if app.Config != nil {
		deployTarget = strings.ToLower(app.Config.Deploy)
	}
	if deployTarget == "" {
		fmt.Fprintln(os.Stderr, cli.Error("No deployment target configured. Add 'deploy to Docker' in your build block."))
		os.Exit(1)
	}

	// Print environment info if --env is used
	if envName != "" {
		found := false
		for _, env := range app.Environments {
			if strings.EqualFold(env.Name, envName) {
				found = true
				fmt.Println(cli.Info(fmt.Sprintf("Environment: %s", env.Name)))
				for k, v := range env.Config {
					fmt.Printf("  %s: %s\n", k, v)
				}
				break
			}
		}
		if !found {
			var available []string
			for _, env := range app.Environments {
				available = append(available, env.Name)
			}
			msg := fmt.Sprintf("Environment %q not found.", envName)
			if len(available) > 0 {
				msg += fmt.Sprintf(" Available: %s", strings.Join(available, ", "))
			}
			fmt.Fprintln(os.Stderr, cli.Error(msg))
			os.Exit(1)
		}
	}

	// Deploy based on target
	switch {
	case strings.Contains(deployTarget, "aws"), strings.Contains(deployTarget, "gcp"), strings.Contains(deployTarget, "terraform"):
		deployTerraform(app, outputDir, envName, dryRun)
	case strings.Contains(deployTarget, "docker"):
		deployDocker(app, outputDir, dryRun)
	default:
		fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Unsupported deploy target: %s. Supported: Docker, AWS, GCP", app.Config.Deploy)))
		os.Exit(1)
	}
}

func deployDocker(app *ir.Application, outputDir string, dryRun bool) {
	// Check docker-compose.yml exists
	composePath := filepath.Join(outputDir, "docker-compose.yml")
	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, cli.Error("docker-compose.yml not found. Run 'human build <file>' first."))
		os.Exit(1)
	}

	// Check that docker is available
	if _, err := exec.LookPath("docker"); err != nil {
		fmt.Fprintln(os.Stderr, cli.Error("docker not found in PATH. Install Docker to deploy."))
		os.Exit(1)
	}

	// Determine compose command (docker compose v2 or docker-compose v1)
	composeCmd := []string{"docker", "compose"}
	if err := exec.Command("docker", "compose", "version").Run(); err != nil {
		// Try v1
		if _, err := exec.LookPath("docker-compose"); err != nil {
			fmt.Fprintln(os.Stderr, cli.Error("Neither 'docker compose' nor 'docker-compose' found. Install Docker Compose."))
			os.Exit(1)
		}
		composeCmd = []string{"docker-compose"}
	}

	// Check .env file
	envPath := filepath.Join(outputDir, ".env")
	envExamplePath := filepath.Join(outputDir, ".env.example")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		if _, err := os.Stat(envExamplePath); err == nil {
			fmt.Println(cli.Warn("No .env file found. Copying .env.example → .env"))
			fmt.Println(cli.Warn("Review and update .env with production values before deploying to production."))
			if !dryRun {
				content, readErr := os.ReadFile(envExamplePath)
				if readErr != nil {
					fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Error reading .env.example: %v", readErr)))
					os.Exit(1)
				}
				if writeErr := os.WriteFile(envPath, content, 0644); writeErr != nil {
					fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Error creating .env: %v", writeErr)))
					os.Exit(1)
				}
			}
		}
	}

	// Build step
	buildArgs := append(composeCmd, "build")
	fmt.Println(cli.Info(fmt.Sprintf("Step 1/2: %s", strings.Join(buildArgs, " "))))
	if dryRun {
		fmt.Println(cli.Info("  (dry-run — skipped)"))
	} else {
		cmd := exec.Command(buildArgs[0], buildArgs[1:]...)
		cmd.Dir = outputDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Docker build failed: %v", err)))
			os.Exit(1)
		}
	}

	// Up step
	upArgs := append(composeCmd, "up", "-d")
	fmt.Println(cli.Info(fmt.Sprintf("Step 2/2: %s", strings.Join(upArgs, " "))))
	if dryRun {
		fmt.Println(cli.Info("  (dry-run — skipped)"))
	} else {
		cmd := exec.Command(upArgs[0], upArgs[1:]...)
		cmd.Dir = outputDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Docker deploy failed: %v", err)))
			os.Exit(1)
		}
	}

	if dryRun {
		fmt.Println(cli.Success("Dry run complete — no changes were made."))
	} else {
		fmt.Println(cli.Success(fmt.Sprintf("Deployed %s via Docker.", app.Name)))
		fmt.Println(cli.Info("  Run 'docker compose ps' in .human/output/ to check status."))
		fmt.Println(cli.Info("  Run 'docker compose logs -f' to view logs."))
		fmt.Println(cli.Info("  Run 'docker compose down' to stop."))
	}
}

func deployTerraform(app *ir.Application, outputDir, envName string, dryRun bool) {
	tfDir := filepath.Join(outputDir, "terraform")
	if _, err := os.Stat(tfDir); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, cli.Error("Terraform files not found. Run 'human build <file>' first."))
		os.Exit(1)
	}

	// Check terraform is installed
	if _, err := exec.LookPath("terraform"); err != nil {
		fmt.Fprintln(os.Stderr, cli.Error("terraform not found in PATH. Install Terraform to deploy."))
		os.Exit(1)
	}

	// Init
	fmt.Println(cli.Info("Step 1/3: terraform init"))
	if !dryRun {
		cmd := exec.Command("terraform", "init")
		cmd.Dir = tfDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("terraform init failed: %v", err)))
			os.Exit(1)
		}
	} else {
		fmt.Println(cli.Info("  (dry-run — skipped)"))
	}

	// Plan
	planArgs := []string{"plan"}
	if envName != "" {
		tfvars := filepath.Join("envs", strings.ToLower(envName)+".tfvars")
		if _, err := os.Stat(filepath.Join(tfDir, tfvars)); err == nil {
			planArgs = append(planArgs, "-var-file="+tfvars)
		}
	}
	fmt.Println(cli.Info(fmt.Sprintf("Step 2/3: terraform %s", strings.Join(planArgs, " "))))
	if !dryRun {
		cmd := exec.Command("terraform", planArgs...)
		cmd.Dir = tfDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("terraform plan failed: %v", err)))
			os.Exit(1)
		}
	} else {
		fmt.Println(cli.Info("  (dry-run — showing plan only)"))
		cmd := exec.Command("terraform", planArgs...)
		cmd.Dir = tfDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()
	}

	// Apply (only if not dry-run)
	if dryRun {
		fmt.Println(cli.Success("Dry run complete — run without --dry-run to apply changes."))
		return
	}

	applyArgs := []string{"apply", "-auto-approve"}
	if envName != "" {
		tfvars := filepath.Join("envs", strings.ToLower(envName)+".tfvars")
		if _, err := os.Stat(filepath.Join(tfDir, tfvars)); err == nil {
			applyArgs = append(applyArgs, "-var-file="+tfvars)
		}
	}
	fmt.Println(cli.Info(fmt.Sprintf("Step 3/3: terraform %s", strings.Join(applyArgs, " "))))
	cmd := exec.Command("terraform", applyArgs...)
	cmd.Dir = tfDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("terraform apply failed: %v", err)))
		os.Exit(1)
	}

	target := "cloud"
	if app.Config != nil {
		target = app.Config.Deploy
	}
	fmt.Println(cli.Success(fmt.Sprintf("Deployed %s via Terraform to %s.", app.Name, target)))
}

// ── build --watch ──

func cmdBuildWatch(file string) {
	fmt.Println(cli.Info(fmt.Sprintf("Watching %s for changes... (Ctrl+C to stop)", file)))

	// Catch interrupt to exit cleanly
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	lastMod := time.Time{}

	for {
		select {
		case <-sigCh:
			fmt.Println("\n" + cli.Info("Watch stopped."))
			return
		default:
		}

		info, err := os.Stat(file)
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		if info.ModTime().After(lastMod) {
			lastMod = info.ModTime()

			// Small debounce — editors often write multiple times
			time.Sleep(100 * time.Millisecond)

			now := time.Now().Format("15:04:05")
			fmt.Printf("\n%s %s\n", cli.Info(now), cli.Info("Building..."))

			if err := runBuild(file); err != nil {
				fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Build failed: %v", err)))
			} else {
				fmt.Println(cli.Success(fmt.Sprintf("%s Rebuilt successfully", now)))
			}
		}

		time.Sleep(500 * time.Millisecond)
	}
}

// runBuild executes the full build pipeline for watch mode, returning any error
// instead of calling os.Exit.
func runBuild(file string) error {
	source, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("reading %s: %w", file, err)
	}

	prog, err := parser.Parse(string(source))
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	app, err := ir.Build(prog)
	if err != nil {
		return fmt.Errorf("IR build error: %w", err)
	}

	errs := analyzer.Analyze(app, file)
	if errs.HasWarnings() {
		for _, w := range errs.Warnings() {
			printDiagnostic(w)
		}
	}
	if errs.HasErrors() {
		for _, e := range errs.Errors() {
			printDiagnostic(e)
		}
		return fmt.Errorf("%d error(s) found", len(errs.Errors()))
	}

	yaml, err := ir.ToYAML(app)
	if err != nil {
		return fmt.Errorf("serialization error: %w", err)
	}

	outDir := filepath.Join(".human", "intent")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	base := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	outFile := filepath.Join(outDir, base+".yaml")
	if err := os.WriteFile(outFile, []byte(yaml), 0644); err != nil {
		return err
	}

	// Run all code generators
	outputDir := filepath.Join(".human", "output")
	if _, err := runGenerators(app, outputDir); err != nil {
		return err
	}

	return nil
}

// ── Generator Dispatch ──

// buildResult tracks the output of a single generator.
type buildResult struct {
	name  string
	dir   string
	files int
}

// matchesGoBackend checks if the backend config indicates Go without
// false-matching strings like "django" or "mongodb".
func matchesGoBackend(backend string) bool {
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

// countFiles returns the number of regular files under dir.
func countFiles(dir string) int {
	count := 0
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			count++
		}
		return nil
	})
	return count
}

// runGenerators dispatches all code generators based on the app's build config,
// then runs the quality engine and scaffolder. Returns build results for each generator.
func runGenerators(app *ir.Application, outputDir string) ([]buildResult, error) {
	var results []buildResult

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
			return nil, fmt.Errorf("React codegen: %w", err)
		}
		results = append(results, buildResult{"react", dir, countFiles(dir)})
	}
	if strings.Contains(frontendLower, "vue") {
		dir := filepath.Join(outputDir, "vue")
		g := vue.Generator{}
		if err := g.Generate(app, dir); err != nil {
			return nil, fmt.Errorf("Vue codegen: %w", err)
		}
		results = append(results, buildResult{"vue", dir, countFiles(dir)})
	}
	if strings.Contains(frontendLower, "angular") {
		dir := filepath.Join(outputDir, "angular")
		g := angular.Generator{}
		if err := g.Generate(app, dir); err != nil {
			return nil, fmt.Errorf("Angular codegen: %w", err)
		}
		results = append(results, buildResult{"angular", dir, countFiles(dir)})
	}
	if strings.Contains(frontendLower, "svelte") {
		dir := filepath.Join(outputDir, "svelte")
		g := svelte.Generator{}
		if err := g.Generate(app, dir); err != nil {
			return nil, fmt.Errorf("Svelte codegen: %w", err)
		}
		results = append(results, buildResult{"svelte", dir, countFiles(dir)})
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
				return nil, fmt.Errorf("Storybook codegen: %w", err)
			}
			results = append(results, buildResult{"storybook", frontendDir, countFiles(filepath.Join(frontendDir, ".storybook")) + countFiles(filepath.Join(frontendDir, "src", "stories"))})
			_ = fw // used by scaffold for dependency injection
		}
	}

	// Backend generators
	if strings.Contains(backendLower, "node") {
		dir := filepath.Join(outputDir, "node")
		g := node.Generator{}
		if err := g.Generate(app, dir); err != nil {
			return nil, fmt.Errorf("Node codegen: %w", err)
		}
		results = append(results, buildResult{"node", dir, countFiles(dir)})
	}
	if strings.Contains(backendLower, "python") {
		dir := filepath.Join(outputDir, "python")
		g := python.Generator{}
		if err := g.Generate(app, dir); err != nil {
			return nil, fmt.Errorf("Python codegen: %w", err)
		}
		results = append(results, buildResult{"python", dir, countFiles(dir)})
	}
	if matchesGoBackend(backendLower) {
		dir := filepath.Join(outputDir, "go")
		g := gobackend.Generator{}
		if err := g.Generate(app, dir); err != nil {
			return nil, fmt.Errorf("Go codegen: %w", err)
		}
		results = append(results, buildResult{"go", dir, countFiles(dir)})
	}

	// Database generator
	if strings.Contains(databaseLower, "postgres") {
		dir := filepath.Join(outputDir, "postgres")
		g := postgres.Generator{}
		if err := g.Generate(app, dir); err != nil {
			return nil, fmt.Errorf("PostgreSQL codegen: %w", err)
		}
		results = append(results, buildResult{"postgres", dir, countFiles(dir)})
	}

	// Docker — conditional on deploy config
	if strings.Contains(deployLower, "docker") {
		before := countFiles(outputDir)
		g := docker.Generator{}
		if err := g.Generate(app, outputDir); err != nil {
			return nil, fmt.Errorf("Docker codegen: %w", err)
		}
		after := countFiles(outputDir)
		results = append(results, buildResult{"docker", outputDir, after - before})
	}

	// CI/CD — always runs
	{
		cicdDir := filepath.Join(outputDir, ".github")
		g := cicd.Generator{}
		if err := g.Generate(app, outputDir); err != nil {
			return nil, fmt.Errorf("CI/CD codegen: %w", err)
		}
		results = append(results, buildResult{"cicd", outputDir, countFiles(cicdDir)})
	}

	// Terraform — conditional on deploy config (aws, gcp, or terraform keyword)
	if strings.Contains(deployLower, "aws") || strings.Contains(deployLower, "gcp") || strings.Contains(deployLower, "terraform") {
		dir := filepath.Join(outputDir, "terraform")
		g := terraform.Generator{}
		if err := g.Generate(app, dir); err != nil {
			return nil, fmt.Errorf("Terraform codegen: %w", err)
		}
		results = append(results, buildResult{"terraform", dir, countFiles(dir)})
	}

	// Architecture — conditional on architecture style
	if app.Architecture != nil && app.Architecture.Style != "" {
		g := architecture.Generator{}
		if err := g.Generate(app, outputDir); err != nil {
			return nil, fmt.Errorf("Architecture codegen: %w", err)
		}
		// Count files only if microservices or serverless generated something
		archDir := filepath.Join(outputDir, "services")
		fnDir := filepath.Join(outputDir, "functions")
		archFiles := countFiles(archDir) + countFiles(fnDir) + countFiles(filepath.Join(outputDir, "gateway"))
		if archFiles > 0 {
			results = append(results, buildResult{"architecture", outputDir, archFiles})
		}
	}

	// Monitoring — conditional on monitoring rules
	if len(app.Monitoring) > 0 {
		dir := filepath.Join(outputDir, "monitoring")
		g := monitoring.Generator{}
		if err := g.Generate(app, dir); err != nil {
			return nil, fmt.Errorf("Monitoring codegen: %w", err)
		}
		results = append(results, buildResult{"monitoring", dir, countFiles(dir)})
	}

	// Quality engine — always runs after code generators
	qResult, err := quality.Run(app, outputDir)
	if err != nil {
		return nil, fmt.Errorf("quality engine: %w", err)
	}
	quality.PrintSummary(qResult)
	// Count quality files: test files + 3 reports (security, lint, build)
	qualityFiles := qResult.TestFiles + qResult.ComponentTestFiles + qResult.EdgeTestFiles + 3
	results = append(results, buildResult{"quality", outputDir, qualityFiles})

	// Scaffolder — always runs last
	{
		sg := scaffold.Generator{}
		if err := sg.Generate(app, outputDir); err != nil {
			return nil, fmt.Errorf("scaffold: %w", err)
		}
		// Count scaffold-specific files (package.json, tsconfig, README, start.sh, .env.example)
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
		results = append(results, buildResult{"scaffold", outputDir, scaffoldFiles})
	}

	return results, nil
}

// printBuildSummary displays a table of generator results.
func printBuildSummary(results []buildResult, outputDir string) {
	total := 0
	for _, r := range results {
		total += r.files
	}

	fmt.Println()
	fmt.Println("  " + cli.Info("Build Summary"))
	fmt.Println("  " + strings.Repeat("─", 50))
	fmt.Printf("  %-14s %-8s %s\n", "Generator", "Files", "Output")
	fmt.Println("  " + strings.Repeat("─", 50))
	for _, r := range results {
		relDir := r.dir
		if rel, err := filepath.Rel(".", r.dir); err == nil {
			relDir = rel
		}
		fmt.Printf("  %-14s %-8d %s/\n", r.name, r.files, relDir)
	}
	fmt.Println("  " + strings.Repeat("─", 50))
	fmt.Printf("  %-14s %-8d\n", "Total", total)
	fmt.Println()
	fmt.Println(cli.Success(fmt.Sprintf("Build complete — %d files in %s/", total, outputDir)))
}

// ── LLM Commands ──

// loadLLMConnector loads config, resolves the provider, and returns a ready Connector.
// If no config exists and no env vars are set, it auto-prompts the user to choose a provider.
func loadLLMConnector() (*llm.Connector, *config.LLMConfig) {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	cfg, err := config.Load(cwd)
	if err != nil {
		fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Config error: %v", err)))
		os.Exit(1)
	}

	// If no LLM config, try to auto-detect from environment variables.
	if cfg.LLM == nil {
		cfg.LLM = detectProviderFromEnv()
	}

	// If still no config, prompt the user.
	if cfg.LLM == nil {
		cfg.LLM = promptProviderSetup(cwd)
	}

	provider, err := llm.NewProvider(cfg.LLM)
	if err != nil {
		fmt.Fprintln(os.Stderr, cli.Error(err.Error()))
		os.Exit(1)
	}

	// One-time cost notice.
	if cfg.LLM.Provider != "ollama" {
		fmt.Fprintln(os.Stderr, cli.Info("Note: LLM calls use your API key and may incur costs."))
	}

	return llm.NewConnector(provider, cfg.LLM), cfg.LLM
}

// detectProviderFromEnv checks for API keys in environment variables and
// returns a config if found, or nil.
func detectProviderFromEnv() *config.LLMConfig {
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		cfg := config.DefaultLLMConfig("anthropic")
		cfg.APIKey = os.Getenv("ANTHROPIC_API_KEY")
		return cfg
	}
	if os.Getenv("OPENAI_API_KEY") != "" {
		cfg := config.DefaultLLMConfig("openai")
		cfg.APIKey = os.Getenv("OPENAI_API_KEY")
		return cfg
	}
	return nil
}

// promptProviderSetup interactively asks the user which LLM provider to use
// and saves the choice to .human/config.json.
func promptProviderSetup(projectDir string) *config.LLMConfig {
	fmt.Println(cli.Info("No LLM provider configured."))
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	choice := prompt(scanner, "LLM Provider", []string{"anthropic", "openai", "ollama"}, "anthropic")

	llmCfg := config.DefaultLLMConfig(choice)

	// Resolve API key.
	key, err := config.ResolveAPIKey(choice)
	if err != nil && choice != "ollama" {
		fmt.Fprintln(os.Stderr, cli.Error(err.Error()))
		os.Exit(1)
	}
	llmCfg.APIKey = key

	// Save the choice.
	cfg := &config.Config{LLM: llmCfg}
	if err := config.Save(projectDir, cfg); err != nil {
		fmt.Fprintln(os.Stderr, cli.Warn(fmt.Sprintf("Could not save config: %v", err)))
	} else {
		fmt.Println(cli.Success(fmt.Sprintf("Saved LLM config to .human/config.json (provider: %s, model: %s)", llmCfg.Provider, llmCfg.Model)))
	}

	return llmCfg
}

func cmdAsk() {
	// Collect query from args.
	args := os.Args[2:]
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: human ask \"<description>\"")
		fmt.Fprintln(os.Stderr, "  Example: human ask \"describe a blog application with users and posts\"")
		os.Exit(1)
	}
	query := strings.Join(args, " ")

	connector, _ := loadLLMConnector()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	fmt.Println(cli.Info("Generating .human code..."))
	fmt.Println()

	// Stream the response.
	ch, err := connector.AskStream(ctx, query)
	if err != nil {
		fmt.Fprintln(os.Stderr, cli.Error(err.Error()))
		os.Exit(1)
	}

	var fullText strings.Builder
	for chunk := range ch {
		if chunk.Err != nil {
			fmt.Fprintln(os.Stderr, cli.Error(chunk.Err.Error()))
			os.Exit(1)
		}
		fmt.Print(chunk.Delta)
		fullText.WriteString(chunk.Delta)
		if chunk.Usage != nil {
			fmt.Fprintf(os.Stderr, "\n\n%s\n",
				cli.Info(fmt.Sprintf("Tokens: %d in / %d out", chunk.Usage.InputTokens, chunk.Usage.OutputTokens)))
		}
	}
	fmt.Println()

	// Post-stream validation: extract code from fences, then validate.
	fmt.Println()
	code, valid, parseErr := llm.ExtractAndValidate(fullText.String())
	_ = code // code is displayed via streaming already
	if valid {
		fmt.Println(cli.Success("Generated code is valid .human syntax."))
	} else {
		fmt.Println(cli.Warn(fmt.Sprintf("Generated code has syntax issues: %s", parseErr)))
		fmt.Println(cli.Info("The code may need manual adjustments."))
	}
}

func cmdSuggest() {
	args := os.Args[2:]
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: human suggest <file.human>")
		os.Exit(1)
	}
	file := args[0]

	source, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Error reading %s: %v", file, err)))
		os.Exit(1)
	}

	connector, _ := loadLLMConnector()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	fmt.Println(cli.Info(fmt.Sprintf("Analyzing %s...", file)))
	fmt.Println()

	result, err := connector.Suggest(ctx, string(source))
	if err != nil {
		fmt.Fprintln(os.Stderr, cli.Error(err.Error()))
		os.Exit(1)
	}

	if len(result.Suggestions) == 0 {
		// No structured suggestions — print the raw response.
		fmt.Println(result.RawResponse)
	} else {
		// Group by category.
		categories := map[string][]string{}
		order := []string{}
		for _, s := range result.Suggestions {
			if _, exists := categories[s.Category]; !exists {
				order = append(order, s.Category)
			}
			categories[s.Category] = append(categories[s.Category], s.Text)
		}

		for _, cat := range order {
			fmt.Printf("\n%s\n", cli.Info(strings.ToUpper(cat)))
			for _, text := range categories[cat] {
				fmt.Printf("  • %s\n", text)
			}
		}
	}

	fmt.Fprintf(os.Stderr, "\n%s\n",
		cli.Info(fmt.Sprintf("Tokens: %d in / %d out", result.Usage.InputTokens, result.Usage.OutputTokens)))
}

func cmdEdit() {
	// Parse flags.
	var file string
	for _, arg := range os.Args[2:] {
		if !strings.HasPrefix(arg, "-") {
			file = arg
		}
	}

	if file == "" {
		fmt.Fprintln(os.Stderr, "Usage: human edit <file.human>")
		fmt.Fprintln(os.Stderr, "  Interactive editing session with LLM assistance.")
		os.Exit(1)
	}

	source, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Error reading %s: %v", file, err)))
		os.Exit(1)
	}

	connector, llmCfg := loadLLMConnector()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	currentSource := string(source)
	var history []llm.Message
	var totalInput, totalOutput int

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println(cli.Info(fmt.Sprintf("Editing %s with %s (%s)", file, llmCfg.Provider, llmCfg.Model)))
	fmt.Println(cli.Info("Type your edit instructions, 'save' to write changes, 'quit' to exit."))
	fmt.Println()

	for {
		fmt.Print("edit> ")
		if !scanner.Scan() {
			break
		}
		instruction := strings.TrimSpace(scanner.Text())

		if instruction == "" {
			continue
		}

		switch strings.ToLower(instruction) {
		case "quit", "exit", "q":
			fmt.Printf("\n%s\n", cli.Info(fmt.Sprintf("Session tokens: %d in / %d out", totalInput, totalOutput)))
			return
		case "save", "s":
			if err := os.WriteFile(file, []byte(currentSource), 0644); err != nil {
				fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Error writing %s: %v", file, err)))
			} else {
				fmt.Println(cli.Success(fmt.Sprintf("Saved %s", file)))
			}
			continue
		case "show":
			fmt.Println()
			fmt.Println(currentSource)
			fmt.Println()
			continue
		}

		fmt.Println(cli.Info("Editing..."))

		result, err := connector.Edit(ctx, currentSource, instruction, history)
		if err != nil {
			fmt.Fprintln(os.Stderr, cli.Error(err.Error()))
			continue
		}

		totalInput += result.Usage.InputTokens
		totalOutput += result.Usage.OutputTokens

		fmt.Println()
		fmt.Println(result.Code)
		fmt.Println()

		if result.Valid {
			fmt.Println(cli.Success("Valid .human syntax."))
		} else {
			fmt.Println(cli.Warn(fmt.Sprintf("Syntax issue: %s", result.ParseError)))
		}

		// Ask to accept.
		fmt.Print("Accept? (y/n): ")
		if scanner.Scan() {
			answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
			if answer == "y" || answer == "yes" {
				currentSource = result.Code
				fmt.Println(cli.Success("Change applied."))

				// Add to history.
				history = append(history,
					llm.Message{Role: llm.RoleUser, Content: instruction},
					llm.Message{Role: llm.RoleAssistant, Content: result.RawResponse},
				)
			} else {
				fmt.Println(cli.Info("Change discarded."))
			}
		}
		fmt.Println()
	}
}

func cmdConvert() {
	args := os.Args[2:]
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: human convert \"<description>\"")
		fmt.Fprintln(os.Stderr, "  Converts a natural language description to .human code.")
		fmt.Fprintln(os.Stderr, "  Future: will also support design file import (Figma, images).")
		os.Exit(1)
	}

	// For now, convert uses the same pipeline as ask.
	query := strings.Join(args, " ")

	connector, _ := loadLLMConnector()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	fmt.Println(cli.Info("Converting to .human code..."))
	fmt.Println(cli.Info("(Design file import is planned for a future release.)"))
	fmt.Println()

	result, err := connector.Ask(ctx, query)
	if err != nil {
		fmt.Fprintln(os.Stderr, cli.Error(err.Error()))
		os.Exit(1)
	}

	fmt.Println(result.Code)
	fmt.Println()

	if result.Valid {
		fmt.Println(cli.Success("Generated code is valid .human syntax."))
	} else {
		fmt.Println(cli.Warn(fmt.Sprintf("Syntax issue: %s", result.ParseError)))
	}

	fmt.Fprintf(os.Stderr, "%s\n",
		cli.Info(fmt.Sprintf("Tokens: %d in / %d out", result.Usage.InputTokens, result.Usage.OutputTokens)))
}

// ── Helpers ──

// printDiagnostic prints a CompilerError with its suggestion (if any) to stderr.
func printDiagnostic(e *cerr.CompilerError) {
	switch e.Severity {
	case cerr.SeverityWarning:
		fmt.Fprintln(os.Stderr, cli.Warn(e.Format()))
	default:
		fmt.Fprintln(os.Stderr, cli.Error(e.Format()))
	}
	if e.Suggestion != "" {
		fmt.Fprintf(os.Stderr, "  suggestion: %s\n", e.Suggestion)
	}
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func pluralY(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}

func printUsage() {
	fmt.Print(`Human — English in, production-ready code out.

Usage:
  human <command> [options] [file]

Commands:
  check <file>              Validate a .human file
  build <file>              Compile to IR and generate code
  build --inspect <file>    Parse and print IR as YAML to stdout
  build --watch <file>      Rebuild automatically on file changes
  init [name]               Create a new Human project
  run                       Start the development server
  test                      Run generated tests
  audit                     Display security and quality report
  deploy [file]             Deploy the application (Docker/AWS/GCP)
  deploy --dry-run [file]   Show deploy steps without executing
  deploy --env <name> [file]  Deploy with a specific environment
  eject [path]              Export as standalone code (default: ./output/)

AI-Assisted (optional, requires API key or Ollama):
  ask "<description>"       Generate .human code from English
  suggest <file.human>      Get improvement suggestions for a file
  edit <file.human>         Interactive AI-assisted editing session
  convert "<description>"   Convert description to .human (design import planned)

Flags:
  --no-color        Disable colored output
  --version, -v     Print the compiler version
  --help, -h        Show this help message

Documentation:
  https://github.com/barun-bash/human
`)
}
