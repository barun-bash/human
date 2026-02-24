package repl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/barun-bash/human/internal/cli"
	"github.com/barun-bash/human/internal/cmdutil"
	"github.com/barun-bash/human/internal/config"
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
			Name:        "/ask",
			Description: "Generate a .human file from a description",
			Usage:       "/ask <description>",
			Handler:     cmdAsk,
		},
		{
			Name:        "/edit",
			Aliases:     []string{"/e"},
			Description: "AI-assisted editing of the loaded project",
			Usage:       "/edit [instruction]",
			Handler:     cmdEdit,
		},
		{
			Name:        "/undo",
			Description: "Revert the last /edit change",
			Usage:       "/undo",
			Handler:     cmdUndo,
		},
		{
			Name:        "/suggest",
			Description: "AI improvement suggestions for the loaded project",
			Usage:       "/suggest [apply <number|all>]",
			Handler:     cmdSuggest,
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
			Name:        "/connect",
			Description: "Set up LLM provider",
			Usage:       "/connect [provider|status]",
			Handler:     cmdConnect,
		},
		{
			Name:        "/theme",
			Description: "Show or change the color theme",
			Usage:       "/theme [name|list]",
			Handler:     cmdTheme,
		},
		{
			Name:        "/config",
			Description: "View or change settings",
			Usage:       "/config [set <key> <value>]",
			Handler:     cmdConfig,
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
	r.clearSuggestions()
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

	// Show build plan if plan_mode is "always" or "auto".
	mode := r.settings.EffectivePlanMode()
	if mode == "always" || mode == "auto" {
		result, err := cmdutil.ParseAndAnalyze(r.projectFile)
		if err != nil {
			fmt.Fprintln(r.errOut, cli.Error(err.Error()))
			return
		}

		// Build a summary plan.
		plan := &Plan{
			Title:    "Build Plan",
			Editable: false,
		}

		plan.Steps = append(plan.Steps, fmt.Sprintf("Source: %s", r.projectFile))

		if result.Prog != nil {
			if len(result.Prog.Pages) > 0 {
				names := make([]string, 0, 3)
				for i, p := range result.Prog.Pages {
					if i >= 3 {
						break
					}
					names = append(names, p.Name)
				}
				suffix := ""
				if len(result.Prog.Pages) > 3 {
					suffix = ", ..."
				}
				plan.Steps = append(plan.Steps, fmt.Sprintf("Pages: %d (%s%s)", len(result.Prog.Pages), strings.Join(names, ", "), suffix))
			}
			if len(result.Prog.APIs) > 0 {
				plan.Steps = append(plan.Steps, fmt.Sprintf("APIs: %d endpoints", len(result.Prog.APIs)))
			}
		}

		stackParts := []string{}
		if result.App != nil && result.App.Config != nil {
			if result.App.Config.Frontend != "" {
				stackParts = append(stackParts, result.App.Config.Frontend)
			}
			if result.App.Config.Backend != "" {
				stackParts = append(stackParts, result.App.Config.Backend)
			}
			if result.App.Config.Database != "" {
				stackParts = append(stackParts, result.App.Config.Database)
			}
		}
		if len(stackParts) > 0 {
			plan.Steps = append(plan.Steps, fmt.Sprintf("Stack: %s", strings.Join(stackParts, " \u2192 ")))
		}

		plan.Steps = append(plan.Steps, "Output: .human/output/")

		action := ShowPlan(r.out, r.in, plan)
		if action == PlanCancel {
			fmt.Fprintln(r.out, cli.Info("Build cancelled."))
			return
		}
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
	fmt.Fprintln(r.out, strings.Repeat("\u2500", 30))
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
		fmt.Fprintln(r.out, strings.Repeat("\u2500", 30))
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
	fmt.Fprintln(r.out, strings.Repeat("\u2500", 30))
	found := false
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		humanFile := filepath.Join(exDir, entry.Name(), "app.human")
		if _, err := os.Stat(humanFile); err == nil {
			fmt.Fprintf(r.out, "  %s  \u2192  /open %s\n", entry.Name(), humanFile)
			found = true
		}
	}
	if !found {
		fmt.Fprintln(r.out, "  No examples found.")
	}
}

// ── Theme Command ──

func cmdTheme(r *REPL, args []string) {
	if len(args) == 0 {
		// Show current theme.
		fmt.Fprintf(r.out, "Current theme: %s\n", cli.Accent(cli.CurrentThemeName()))
		return
	}

	if strings.ToLower(args[0]) == "list" {
		fmt.Fprintln(r.out)
		fmt.Fprintln(r.out, cli.Heading("Available Themes"))
		fmt.Fprintln(r.out, strings.Repeat("\u2500", 40))
		for _, name := range cli.ThemeNames() {
			marker := "  "
			if name == cli.CurrentThemeName() {
				marker = cli.Accent("\u2713 ")
			}
			preview := cli.ThemePreview(name)
			fmt.Fprintf(r.out, "  %s%-10s%s\n", marker, name, preview)
		}
		fmt.Fprintln(r.out)
		return
	}

	name := strings.ToLower(args[0])
	if err := cli.SetTheme(name); err != nil {
		fmt.Fprintln(r.errOut, cli.Error(err.Error()))
		return
	}

	// Persist the choice.
	r.settings.Theme = name
	if err := config.SaveGlobal(r.settings); err != nil {
		fmt.Fprintln(r.errOut, cli.Warn(fmt.Sprintf("Theme set but could not save: %v", err)))
	} else {
		fmt.Fprintf(r.out, "%s\n", cli.Success(fmt.Sprintf("Theme set to %s", name)))
	}
}

// ── Config Command ──

func cmdConfig(r *REPL, args []string) {
	if len(args) == 0 {
		// Show current settings.
		fmt.Fprintln(r.out)
		fmt.Fprintln(r.out, cli.Heading("Settings"))
		fmt.Fprintln(r.out, strings.Repeat("\u2500", 30))
		fmt.Fprintf(r.out, "  theme:     %s\n", cli.CurrentThemeName())
		fmt.Fprintf(r.out, "  animate:   %v\n", r.settings.AnimateEnabled())
		fmt.Fprintf(r.out, "  plan_mode: %s\n", r.settings.EffectivePlanMode())
		fmt.Fprintln(r.out)
		fmt.Fprintln(r.out, cli.Muted("  Use /config set <key> <value> to change."))
		fmt.Fprintln(r.out)
		return
	}

	if strings.ToLower(args[0]) != "set" || len(args) < 3 {
		fmt.Fprintln(r.errOut, "Usage: /config set <key> <value>")
		fmt.Fprintln(r.errOut, "  Keys: animate (on/off), plan_mode (always/auto/off), theme (<name>)")
		return
	}

	key := strings.ToLower(args[1])
	value := strings.ToLower(args[2])

	switch key {
	case "animate":
		switch value {
		case "on", "true", "yes":
			r.settings.SetAnimate(true)
		case "off", "false", "no":
			r.settings.SetAnimate(false)
		default:
			fmt.Fprintln(r.errOut, cli.Error("animate must be on or off"))
			return
		}
	case "plan_mode":
		switch value {
		case "always", "auto", "off":
			r.settings.PlanMode = value
		default:
			fmt.Fprintln(r.errOut, cli.Error("plan_mode must be always, auto, or off"))
			return
		}
	case "theme":
		if err := cli.SetTheme(value); err != nil {
			fmt.Fprintln(r.errOut, cli.Error(err.Error()))
			return
		}
		r.settings.Theme = value
	default:
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Unknown setting: %s", key)))
		fmt.Fprintln(r.errOut, "  Available: animate, plan_mode, theme")
		return
	}

	if err := config.SaveGlobal(r.settings); err != nil {
		fmt.Fprintln(r.errOut, cli.Warn(fmt.Sprintf("Setting applied but could not save: %v", err)))
	} else {
		fmt.Fprintf(r.out, "%s\n", cli.Success(fmt.Sprintf("Set %s = %s", key, value)))
	}
}

// ── Standard Commands ──

func cmdHelp(r *REPL, args []string) {
	fmt.Fprintln(r.out)
	fmt.Fprintln(r.out, cli.Heading("Available Commands"))
	fmt.Fprintln(r.out, strings.Repeat("\u2500", 50))

	// Ordered list of command names for display
	order := []string{
		"/open", "/new", "/ask", "/edit", "/undo", "/suggest", "/check", "/build", "/deploy", "/stop",
		"/status", "/run", "/test", "/audit", "/review", "/examples",
		"/connect", "/theme", "/config",
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
		fmt.Fprintf(r.out, "  %-24s %s%s\n", cmd.Usage, cmd.Description, aliases)
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
