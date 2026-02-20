package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/barun-bash/human/internal/codegen/docker"
	"github.com/barun-bash/human/internal/codegen/node"
	"github.com/barun-bash/human/internal/codegen/postgres"
	"github.com/barun-bash/human/internal/codegen/react"
	"github.com/barun-bash/human/internal/codegen/scaffold"
	"github.com/barun-bash/human/internal/ir"
	"github.com/barun-bash/human/internal/parser"
	"github.com/barun-bash/human/internal/quality"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	switch os.Args[1] {
	case "version", "--version", "-v":
		fmt.Printf("human v%s\n", version)
	case "help", "--help", "-h":
		printUsage()
	case "check":
		cmdCheck()
	case "build":
		cmdBuild()
	case "init":
		fmt.Println("human init — coming soon.")
	case "run":
		fmt.Println("human run — coming soon.")
	case "test":
		fmt.Println("human test — coming soon.")
	case "audit":
		fmt.Println("human audit — coming soon.")
	case "deploy":
		fmt.Println("human deploy — coming soon.")
	case "eject":
		fmt.Println("human eject — coming soon.")
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
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
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", file, err)
		os.Exit(1)
	}

	prog, err := parser.Parse(string(source))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error in %s: %v\n", file, err)
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

	fmt.Printf("\u2713 %s is valid", file)
	if len(parts) > 0 {
		fmt.Printf(" — %s", strings.Join(parts, ", "))
	}
	fmt.Println()
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
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", file, err)
		os.Exit(1)
	}

	prog, err := parser.Parse(string(source))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Parse error in %s: %v\n", file, err)
		os.Exit(1)
	}

	app, err := ir.Build(prog)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IR build error: %v\n", err)
		os.Exit(1)
	}

	yaml, err := ir.ToYAML(app)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Serialization error: %v\n", err)
		os.Exit(1)
	}

	if inspect {
		fmt.Print(yaml)
		return
	}

	// Write IR to .human/intent/<name>.yaml
	outDir := filepath.Join(".human", "intent")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	base := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	outFile := filepath.Join(outDir, base+".yaml")

	if err := os.WriteFile(outFile, []byte(yaml), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", outFile, err)
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
			fmt.Fprintf(os.Stderr, "React codegen error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("  react:        %s/\n", reactDir)
	}

	if app.Config != nil && strings.Contains(strings.ToLower(app.Config.Backend), "node") {
		nodeDir := filepath.Join(".human", "output", "node")
		g := node.Generator{}
		if err := g.Generate(app, nodeDir); err != nil {
			fmt.Fprintf(os.Stderr, "Node codegen error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("  node:         %s/\n", nodeDir)
	}

	if app.Config != nil && strings.Contains(strings.ToLower(app.Config.Deploy), "docker") {
		outputDir := filepath.Join(".human", "output")
		g := docker.Generator{}
		if err := g.Generate(app, outputDir); err != nil {
			fmt.Fprintf(os.Stderr, "Docker codegen error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("  docker:       %s/\n", outputDir)
	}

	if app.Config != nil && strings.Contains(strings.ToLower(app.Config.Database), "postgres") {
		pgDir := filepath.Join(".human", "output", "postgres")
		g := postgres.Generator{}
		if err := g.Generate(app, pgDir); err != nil {
			fmt.Fprintf(os.Stderr, "PostgreSQL codegen error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("  postgres:     %s/\n", pgDir)
	}

	// Quality engine — always runs after code generators
	outputDir := filepath.Join(".human", "output")
	qResult, err := quality.Run(app, outputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Quality engine error: %v\n", err)
		os.Exit(1)
	}
	quality.PrintSummary(qResult)

	// Scaffolder — always runs last, produces project files
	sg := scaffold.Generator{}
	if err := sg.Generate(app, outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "Scaffold error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  scaffold:     %s/ (package.json, tsconfig, README, start.sh)\n", outputDir)
}

func printIRSummary(app *ir.Application) {
	fmt.Printf("  app:          %s (%s)\n", app.Name, app.Platform)
	if app.Config != nil {
		fmt.Printf("  config:       %s / %s / %s\n", app.Config.Frontend, app.Config.Backend, app.Config.Database)
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
  init                      Create a new Human project
  run                       Start the development server
  test                      Run generated tests
  audit                     Security and quality audit
  deploy                    Deploy the application
  eject                     Export as standalone code

Flags:
  --version, -v   Print the compiler version
  --help, -h      Show this help message

Documentation:
  https://github.com/barun-bash/human
`)
}
