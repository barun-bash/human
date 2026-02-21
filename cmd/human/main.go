package main

import (
	"bufio"
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
	"github.com/barun-bash/human/internal/codegen/cicd"
	"github.com/barun-bash/human/internal/codegen/docker"
	"github.com/barun-bash/human/internal/codegen/gobackend"
	"github.com/barun-bash/human/internal/codegen/node"
	"github.com/barun-bash/human/internal/codegen/postgres"
	"github.com/barun-bash/human/internal/codegen/python"
	"github.com/barun-bash/human/internal/codegen/react"
	"github.com/barun-bash/human/internal/codegen/scaffold"
	"github.com/barun-bash/human/internal/codegen/svelte"
	"github.com/barun-bash/human/internal/codegen/vue"
	cerr "github.com/barun-bash/human/internal/errors"
	"github.com/barun-bash/human/internal/ir"
	"github.com/barun-bash/human/internal/parser"
	"github.com/barun-bash/human/internal/quality"
)

const version = "0.3.0"

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
	case strings.Contains(deployTarget, "docker"):
		deployDocker(app, outputDir, dryRun)
	default:
		fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Unsupported deploy target: %s. Currently supported: Docker", app.Config.Deploy)))
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
  deploy [file]             Deploy the application (Docker)
  deploy --dry-run [file]   Show deploy steps without executing
  deploy --env <name> [file]  Deploy with a specific environment
  eject [path]              Export as standalone code (default: ./output/)

Flags:
  --no-color        Disable colored output
  --version, -v     Print the compiler version
  --help, -h        Show this help message

Documentation:
  https://github.com/barun-bash/human
`)
}
