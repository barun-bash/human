package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/barun-bash/human/internal/cli"
	"github.com/barun-bash/human/internal/codegen/docker"
	"github.com/barun-bash/human/internal/codegen/node"
	"github.com/barun-bash/human/internal/codegen/postgres"
	"github.com/barun-bash/human/internal/codegen/react"
	"github.com/barun-bash/human/internal/codegen/scaffold"
	"github.com/barun-bash/human/internal/ir"
	"github.com/barun-bash/human/internal/parser"
	"github.com/barun-bash/human/internal/quality"
)

const version = "0.1.1"

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
		fmt.Println("human audit — coming soon.")
	case "deploy":
		fmt.Println("human deploy — coming soon.")
	case "eject":
		fmt.Println("human eject — coming soon.")
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
	var file string
	for _, arg := range os.Args[2:] {
		if arg == "--inspect" {
			inspect = true
		} else if !strings.HasPrefix(arg, "-") {
			file = arg
		}
	}

	if file == "" {
		fmt.Fprintln(os.Stderr, "Usage: human build [--inspect] <file.human>")
		os.Exit(1)
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

	// Run code generators based on build config
	if app.Config != nil && strings.Contains(strings.ToLower(app.Config.Frontend), "react") {
		reactDir := filepath.Join(".human", "output", "react")
		g := react.Generator{}
		if err := g.Generate(app, reactDir); err != nil {
			fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("React codegen error: %v", err)))
			os.Exit(1)
		}
		fmt.Println(cli.Info(fmt.Sprintf("  react:        %s/", reactDir)))
	}

	if app.Config != nil && strings.Contains(strings.ToLower(app.Config.Backend), "node") {
		nodeDir := filepath.Join(".human", "output", "node")
		g := node.Generator{}
		if err := g.Generate(app, nodeDir); err != nil {
			fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Node codegen error: %v", err)))
			os.Exit(1)
		}
		fmt.Println(cli.Info(fmt.Sprintf("  node:         %s/", nodeDir)))
	}

	if app.Config != nil && strings.Contains(strings.ToLower(app.Config.Deploy), "docker") {
		outputDir := filepath.Join(".human", "output")
		g := docker.Generator{}
		if err := g.Generate(app, outputDir); err != nil {
			fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Docker codegen error: %v", err)))
			os.Exit(1)
		}
		fmt.Println(cli.Info(fmt.Sprintf("  docker:       %s/", outputDir)))
	}

	if app.Config != nil && strings.Contains(strings.ToLower(app.Config.Database), "postgres") {
		pgDir := filepath.Join(".human", "output", "postgres")
		g := postgres.Generator{}
		if err := g.Generate(app, pgDir); err != nil {
			fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("PostgreSQL codegen error: %v", err)))
			os.Exit(1)
		}
		fmt.Println(cli.Info(fmt.Sprintf("  postgres:     %s/", pgDir)))
	}

	// Quality engine — always runs after code generators
	outputDir := filepath.Join(".human", "output")
	qResult, err := quality.Run(app, outputDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Quality engine error: %v", err)))
		os.Exit(1)
	}
	quality.PrintSummary(qResult)

	// Scaffolder — always runs last, produces project files
	sg := scaffold.Generator{}
	if err := sg.Generate(app, outputDir); err != nil {
		fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Scaffold error: %v", err)))
		os.Exit(1)
	}
	fmt.Println(cli.Info(fmt.Sprintf("  scaffold:     %s/ (package.json, tsconfig, README, start.sh)", outputDir)))

	fmt.Println(cli.Success(fmt.Sprintf("Build complete — output in %s/", outputDir)))
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

	fmt.Fprintf(&b, "app %s\n", name)
	fmt.Fprintf(&b, "  platform is %s\n\n", platform)

	b.WriteString("# ── Data Models ──\n\n")
	b.WriteString("data User\n")
	b.WriteString("  name is text, required\n")
	b.WriteString("  email is email, required, unique\n")
	b.WriteString("  password is text, required, encrypted\n\n")

	b.WriteString("# ── Pages ──\n\n")
	b.WriteString("page Home\n")
	b.WriteString("  show heading \"Welcome to " + name + "\"\n\n")

	b.WriteString("# ── APIs ──\n\n")
	b.WriteString("api SignUp\n")
	b.WriteString("  accepts name, email, password\n")
	b.WriteString("  create User with name, email, password\n")
	b.WriteString("  respond with success\n\n")

	b.WriteString("# ── Build ──\n\n")
	b.WriteString("build\n")
	fmt.Fprintf(&b, "  frontend is %s\n", frontend)
	fmt.Fprintf(&b, "  backend is %s\n", backend)
	fmt.Fprintf(&b, "  database is %s\n", database)
	b.WriteString("  deploy with Docker\n")

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

// ── Helpers ──

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
  build <file>              Compile to IR (writes .human/intent/<name>.yaml)
  build --inspect <file>    Parse and print IR as YAML to stdout
  init [name]               Create a new Human project
  run                       Start the development server
  test                      Run generated tests
  audit                     Security and quality audit
  deploy                    Deploy the application
  eject                     Export as standalone code

Flags:
  --no-color        Disable colored output
  --version, -v     Print the compiler version
  --help, -h        Show this help message

Documentation:
  https://github.com/barun-bash/human
`)
}
