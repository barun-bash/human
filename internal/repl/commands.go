package repl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/barun-bash/human/internal/cli"
	"github.com/barun-bash/human/internal/cmdutil"
)

// Command is a REPL command with metadata and a handler function.
type Command struct {
	Name        string
	Aliases     []string
	Description string
	Usage       string
	Handler     func(r *REPL, args []string)
}

// registerCommands sets up all built-in REPL commands.
func (r *REPL) registerCommands() {
	cmds := []*Command{
		{
			Name:        "/open",
			Description: "Load a .human file",
			Usage:       "/open <file.human>",
			Handler:     cmdOpen,
		},
		{
			Name:        "/new",
			Description: "Create a new Human project",
			Usage:       "/new [name]",
			Handler:     cmdNew,
		},
		{
			Name:        "/build",
			Aliases:     []string{"/b"},
			Description: "Compile the loaded project",
			Usage:       "/build",
			Handler:     cmdBuild,
		},
		{
			Name:        "/check",
			Description: "Validate the loaded project",
			Usage:       "/check",
			Handler:     cmdCheck,
		},
		{
			Name:        "/deploy",
			Description: "Deploy the application",
			Usage:       "/deploy [--dry-run]",
			Handler:     cmdDeploy,
		},
		{
			Name:        "/stop",
			Description: "Stop Docker containers",
			Usage:       "/stop",
			Handler:     cmdStop,
		},
		{
			Name:        "/status",
			Description: "Show project and Docker status",
			Usage:       "/status",
			Handler:     cmdStatus,
		},
		{
			Name:        "/run",
			Description: "Start the development server",
			Usage:       "/run",
			Handler:     cmdRun,
		},
		{
			Name:        "/test",
			Aliases:     []string{"/t"},
			Description: "Run generated tests",
			Usage:       "/test",
			Handler:     cmdTest,
		},
		{
			Name:        "/audit",
			Description: "Display security and quality report",
			Usage:       "/audit",
			Handler:     cmdAudit,
		},
		{
			Name:        "/review",
			Description: "Open the project file in your editor",
			Usage:       "/review",
			Handler:     cmdReview,
		},
		{
			Name:        "/examples",
			Description: "List available example projects",
			Usage:       "/examples",
			Handler:     cmdExamples,
		},
		{
			Name:        "/help",
			Aliases:     []string{"/?"},
			Description: "Show this help message",
			Usage:       "/help",
			Handler:     cmdHelp,
		},
		{
			Name:        "/clear",
			Description: "Clear the screen",
			Usage:       "/clear",
			Handler:     cmdClear,
		},
		{
			Name:        "/version",
			Aliases:     []string{"/v"},
			Description: "Print the compiler version",
			Usage:       "/version",
			Handler:     cmdVersion,
		},
		{
			Name:        "/quit",
			Aliases:     []string{"/exit", "/q"},
			Description: "Exit the REPL",
			Usage:       "/quit",
			Handler:     cmdQuit,
		},
	}

	for _, cmd := range cmds {
		r.commands[cmd.Name] = cmd
		for _, alias := range cmd.Aliases {
			r.aliases[alias] = cmd.Name
		}
	}
}

// ── Command Handlers ──

func cmdOpen(r *REPL, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(r.errOut, "Usage: /open <file.human>")
		return
	}
	file := args[0]
	if _, err := os.Stat(file); os.IsNotExist(err) {
		fmt.Fprintf(r.errOut, "File not found: %s\n", file)
		return
	}
	r.setProject(file)
	fmt.Fprintf(r.out, "Loaded %s\n", file)
}

func cmdNew(r *REPL, args []string) {
	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	outPath, err := cmdutil.InitProject(name, r.in, r.out)
	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error(err.Error()))
		return
	}
	fmt.Fprintln(r.out, cli.Success(fmt.Sprintf("Created %s", outPath)))
	r.setProject(outPath)
}

func cmdBuild(r *REPL, args []string) {
	if !r.requireProject() {
		return
	}
	if _, _, _, err := cmdutil.FullBuild(r.projectFile); err != nil {
		fmt.Fprintln(r.errOut, cli.Error(err.Error()))
	}
}

func cmdCheck(r *REPL, args []string) {
	if !r.requireProject() {
		return
	}
	result, err := cmdutil.ParseAndAnalyze(r.projectFile)
	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error(err.Error()))
		return
	}
	if cmdutil.PrintDiagnostics(result.Errs) {
		fmt.Fprintf(r.errOut, "\n%s\n", cli.Error(fmt.Sprintf("%d error(s) found", len(result.Errs.Errors()))))
		return
	}
	fmt.Fprintln(r.out, cli.Success(cmdutil.CheckSummary(result.Prog, r.projectFile)))
}

func cmdDeploy(r *REPL, args []string) {
	if !r.requireProject() {
		return
	}
	dryRun := false
	for _, arg := range args {
		if arg == "--dry-run" {
			dryRun = true
		}
	}

	result, err := cmdutil.ParseAndAnalyze(r.projectFile)
	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error(err.Error()))
		return
	}

	deployTarget := ""
	if result.App.Config != nil {
		deployTarget = strings.ToLower(result.App.Config.Deploy)
	}
	if deployTarget == "" || !strings.Contains(deployTarget, "docker") {
		fmt.Fprintln(r.errOut, cli.Error("Only Docker deploy is supported from the REPL. Use the CLI for Terraform/AWS/GCP."))
		return
	}

	if _, _, _, err := cmdutil.FullBuild(r.projectFile); err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Build failed: %v", err)))
		return
	}

	outputDir := filepath.Join(".human", "output")
	if err := cmdutil.DeployDocker(result.App, outputDir, dryRun); err != nil {
		fmt.Fprintln(r.errOut, cli.Error(err.Error()))
	}
}

func cmdStop(r *REPL, args []string) {
	outputDir := filepath.Join(".human", "output")
	if err := cmdutil.StopDocker(outputDir); err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Stop failed: %v", err)))
	}
}

func cmdStatus(r *REPL, args []string) {
	fmt.Fprintln(r.out, cli.Info("Project Status"))
	fmt.Fprintln(r.out, strings.Repeat("─", 30))
	if r.projectFile != "" {
		fmt.Fprintf(r.out, "  Project:  %s\n", r.projectName)
		fmt.Fprintf(r.out, "  File:     %s\n", r.projectFile)
	} else {
		fmt.Fprintln(r.out, "  No project loaded.")
	}

	outputDir := filepath.Join(".human", "output")
	if _, err := os.Stat(outputDir); err == nil {
		fmt.Fprintf(r.out, "  Output:   %s/\n", outputDir)
	} else {
		fmt.Fprintln(r.out, "  Output:   (not built yet)")
	}
	fmt.Fprintln(r.out)

	// Try showing Docker status (non-fatal)
	composePath := filepath.Join(outputDir, "docker-compose.yml")
	if _, err := os.Stat(composePath); err == nil {
		fmt.Fprintln(r.out, cli.Info("Docker Status"))
		fmt.Fprintln(r.out, strings.Repeat("─", 30))
		if err := cmdutil.DockerStatus(outputDir); err != nil {
			fmt.Fprintln(r.out, "  Docker not running or not available.")
		}
	}
}

func cmdRun(r *REPL, args []string) {
	outputDir := filepath.Join(".human", "output")
	startSh := filepath.Join(outputDir, "start.sh")
	pkgJSON := filepath.Join(outputDir, "package.json")

	if _, err := os.Stat(startSh); err == nil {
		fmt.Fprintln(r.out, cli.Info("Starting application via start.sh..."))
		if err := cmdutil.RunCommand(outputDir, "bash", "start.sh"); err != nil {
			fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Run failed: %v", err)))
		}
		return
	}

	if _, err := os.Stat(pkgJSON); err == nil {
		fmt.Fprintln(r.out, cli.Info("Starting application via npm run dev..."))
		if err := cmdutil.RunCommand(outputDir, "npm", "run", "dev"); err != nil {
			fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Run failed: %v", err)))
		}
		return
	}

	fmt.Fprintln(r.errOut, cli.Error("No build found. Run /build first."))
}

func cmdTest(r *REPL, args []string) {
	outputDir, err := cmdutil.RequireOutputDir()
	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error(err.Error()))
		return
	}
	fmt.Fprintln(r.out, cli.Info("Running tests..."))
	if err := cmdutil.RunCommandSilent(outputDir, "npm", "test"); err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Tests failed: %v", err)))
	}
}

func cmdAudit(r *REPL, args []string) {
	outputDir, err := cmdutil.RequireOutputDir()
	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error(err.Error()))
		return
	}
	reportPath := filepath.Join(outputDir, "security-report.md")
	report, err := os.ReadFile(reportPath)
	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error("No security report found. Run /build to generate one."))
		return
	}
	cmdutil.PrintAuditReport(string(report))
}

func cmdReview(r *REPL, args []string) {
	if !r.requireProject() {
		return
	}
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}
	if err := cmdutil.RunCommand(".", editor, r.projectFile); err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Could not open editor: %v", err)))
	}
}

func cmdExamples(r *REPL, args []string) {
	// Find the examples directory relative to the executable
	exDir := "examples"
	if _, err := os.Stat(exDir); os.IsNotExist(err) {
		// Try relative to the binary
		exe, _ := os.Executable()
		if exe != "" {
			exDir = filepath.Join(filepath.Dir(exe), "..", "examples")
		}
		if _, err := os.Stat(exDir); os.IsNotExist(err) {
			fmt.Fprintln(r.errOut, "No examples directory found.")
			return
		}
	}

	entries, err := os.ReadDir(exDir)
	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Could not read examples: %v", err)))
		return
	}

	fmt.Fprintln(r.out, cli.Info("Available Examples"))
	fmt.Fprintln(r.out, strings.Repeat("─", 30))
	found := false
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		humanFile := filepath.Join(exDir, entry.Name(), "app.human")
		if _, err := os.Stat(humanFile); err == nil {
			fmt.Fprintf(r.out, "  %s  →  /open %s\n", entry.Name(), humanFile)
			found = true
		}
	}
	if !found {
		fmt.Fprintln(r.out, "  No examples found.")
	}
}

func cmdHelp(r *REPL, args []string) {
	fmt.Fprintln(r.out)
	fmt.Fprintln(r.out, "\033[1mAvailable Commands\033[0m")
	fmt.Fprintln(r.out, strings.Repeat("─", 50))

	// Ordered list of command names for display
	order := []string{
		"/open", "/new", "/check", "/build", "/deploy", "/stop",
		"/status", "/run", "/test", "/audit", "/review", "/examples",
		"/clear", "/version", "/help", "/quit",
	}

	for _, name := range order {
		cmd := r.commands[name]
		if cmd == nil {
			continue
		}
		aliases := ""
		if len(cmd.Aliases) > 0 {
			aliases = " (" + strings.Join(cmd.Aliases, ", ") + ")"
		}
		fmt.Fprintf(r.out, "  %-18s %s%s\n", cmd.Usage, cmd.Description, aliases)
	}
	fmt.Fprintln(r.out)
}

func cmdClear(r *REPL, args []string) {
	fmt.Fprint(r.out, "\033[2J\033[H")
}

func cmdVersion(r *REPL, args []string) {
	fmt.Fprintf(r.out, "human v%s\n", r.version)
}

func cmdQuit(r *REPL, args []string) {
	fmt.Fprintln(r.out, "Goodbye.")
	r.running = false
}

