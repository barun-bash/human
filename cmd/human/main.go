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

	"github.com/barun-bash/human/internal/build"
	"github.com/barun-bash/human/internal/cli"
	"github.com/barun-bash/human/internal/cmdutil"
	"github.com/barun-bash/human/internal/config"
	"github.com/barun-bash/human/internal/fixer"
	"github.com/barun-bash/human/internal/ir"
	"github.com/barun-bash/human/internal/llm"
	_ "github.com/barun-bash/human/internal/llm/providers" // register providers
	"github.com/barun-bash/human/internal/repl"
	"github.com/barun-bash/human/internal/version"
)

func main() {
	// Parse global --no-color flag before command dispatch
	args := filterGlobalFlags(os.Args[1:])

	if len(args) < 1 {
		r := repl.New(version.Version)
		r.Run()
		return
	}

	switch args[0] {
	case "version", "--version", "-v":
		fmt.Printf("human v%s\n", version.Info())
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
	case "storybook":
		cmdStorybook()
	case "explain":
		cmdExplainCLI()
	case "syntax":
		cmdSyntaxCLI()
	case "fix":
		cmdFixCLI()
	case "doctor":
		cmdutil.RunDoctor(os.Stdout)
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

	result, err := cmdutil.ParseAndAnalyze(file)
	if err != nil {
		fmt.Fprintln(os.Stderr, cli.Error(err.Error()))
		os.Exit(1)
	}

	if cmdutil.PrintDiagnostics(result.Errs) {
		fmt.Fprintf(os.Stderr, "\n%s\n", cli.Error(fmt.Sprintf("%d error(s) found", len(result.Errs.Errors()))))
		os.Exit(1)
	}

	fmt.Println(cli.Success(cmdutil.CheckSummary(result.Prog, file)))
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

	if inspect {
		result, err := cmdutil.ParseAndAnalyze(file)
		if err != nil {
			fmt.Fprintln(os.Stderr, cli.Error(err.Error()))
			os.Exit(1)
		}
		if cmdutil.PrintDiagnostics(result.Errs) {
			fmt.Fprintf(os.Stderr, "\n%s\n", cli.Error(fmt.Sprintf("%d error(s) found — build aborted", len(result.Errs.Errors()))))
			os.Exit(1)
		}
		yaml, err := ir.ToYAML(result.App)
		if err != nil {
			fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Serialization error: %v", err)))
			os.Exit(1)
		}
		fmt.Print(yaml)
		return
	}

	if _, _, _, err := cmdutil.FullBuild(file); err != nil {
		fmt.Fprintln(os.Stderr, cli.Error(err.Error()))
		os.Exit(1)
	}
}

// ── init ──

func cmdInit() {
	name := ""
	if len(os.Args) >= 3 && !strings.HasPrefix(os.Args[2], "-") {
		name = os.Args[2]
	}

	outPath, err := cmdutil.InitProject(name, os.Stdin, os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, cli.Error(err.Error()))
		os.Exit(1)
	}

	fmt.Println(cli.Success(fmt.Sprintf("Created %s — run 'human check %s' to validate, 'human build %s' to compile", outPath, outPath, outPath)))
}

// ── run ──

func cmdRun() {
	outputDir := filepath.Join(".human", "output")

	startSh := filepath.Join(outputDir, "start.sh")
	pkgJSON := filepath.Join(outputDir, "package.json")

	if _, err := os.Stat(startSh); err == nil {
		fmt.Println(cli.Info("Starting application via start.sh..."))
		if err := cmdutil.RunCommand(outputDir, "bash", "start.sh"); err != nil {
			fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Run failed: %v", err)))
			os.Exit(1)
		}
		return
	}

	if _, err := os.Stat(pkgJSON); err == nil {
		fmt.Println(cli.Info("Starting application via npm run dev..."))
		if err := cmdutil.RunCommand(outputDir, "npm", "run", "dev"); err != nil {
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
	outputDir, err := cmdutil.RequireOutputDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, cli.Error(err.Error()))
		os.Exit(1)
	}

	fmt.Println(cli.Info("Running tests..."))
	if err := cmdutil.RunCommandSilent(outputDir, "npm", "test"); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Test failed: %v", err)))
		os.Exit(1)
	}
}

// ── audit ──

func cmdAudit() {
	outputDir, err := cmdutil.RequireOutputDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, cli.Error(err.Error()))
		os.Exit(1)
	}

	reportPath := filepath.Join(outputDir, "security-report.md")
	report, err := os.ReadFile(reportPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, cli.Error("No security report found. Run 'human build <file>' to generate one."))
		os.Exit(1)
	}

	cmdutil.PrintAuditReport(string(report))
}

// ── eject ──

func cmdEject() {
	outputDir, err := cmdutil.RequireOutputDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, cli.Error(err.Error()))
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
	err = filepath.WalkDir(outputDir, func(path string, d fs.DirEntry, err error) error {
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
	result, err := cmdutil.ParseAndAnalyze(file)
	if err != nil {
		fmt.Fprintln(os.Stderr, cli.Error(err.Error()))
		os.Exit(1)
	}
	app := result.App

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
		if err := cmdutil.DeployDocker(app, outputDir, dryRun); err != nil {
			fmt.Fprintln(os.Stderr, cli.Error(err.Error()))
			os.Exit(1)
		}
	default:
		fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Unsupported deploy target: %s. Supported: Docker, AWS, GCP", app.Config.Deploy)))
		os.Exit(1)
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
		if err := cmdutil.RunCommandSilent(tfDir, "terraform", "init"); err != nil {
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
		if err := cmdutil.RunCommandSilent(tfDir, "terraform", planArgs...); err != nil {
			fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("terraform plan failed: %v", err)))
			os.Exit(1)
		}
	} else {
		fmt.Println(cli.Info("  (dry-run — showing plan only)"))
		_ = cmdutil.RunCommandSilent(tfDir, "terraform", planArgs...)
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
	if err := cmdutil.RunCommandSilent(tfDir, "terraform", applyArgs...); err != nil {
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

// runBuild executes the full build pipeline for watch mode and deploy,
// returning any error instead of calling os.Exit.
func runBuild(file string) error {
	result, err := cmdutil.ParseAndAnalyze(file)
	if err != nil {
		return err
	}

	if cmdutil.PrintDiagnostics(result.Errs) {
		return fmt.Errorf("%d error(s) found", len(result.Errs.Errors()))
	}

	yaml, err := ir.ToYAML(result.App)
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
	results, qResult, genErr := build.RunGenerators(result.App, outputDir)
	if genErr != nil {
		return genErr
	}
	// Suppress unused warnings — watch/deploy don't print summaries
	_ = results
	_ = qResult

	return nil
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
	choice := cmdutil.Prompt(scanner, os.Stdout, "LLM Provider", []string{"anthropic", "openai", "ollama"}, "anthropic")

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

// ── storybook ──

func cmdStorybook() {
	outputDir := filepath.Join(".human", "output")

	// Find the frontend directory that has a .storybook config.
	for _, fw := range []string{"react", "vue", "angular", "svelte"} {
		sbDir := filepath.Join(outputDir, fw, ".storybook")
		if _, err := os.Stat(sbDir); err == nil {
			fmt.Println(cli.Info(fmt.Sprintf("Starting Storybook in %s/%s...", outputDir, fw)))
			cmd := exec.Command("npx", "storybook", "dev", "-p", "6006")
			cmd.Dir = filepath.Join(outputDir, fw)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin
			if err := cmd.Run(); err != nil {
				fmt.Fprintln(os.Stderr, cli.Error(fmt.Sprintf("Storybook failed: %v", err)))
				os.Exit(1)
			}
			return
		}
	}

	fmt.Fprintln(os.Stderr, cli.Error("No Storybook found. Run 'human build <file>' first."))
	os.Exit(1)
}

// ── explain ──

func cmdExplainCLI() {
	topic := ""
	if len(os.Args) >= 3 {
		topic = strings.Join(os.Args[2:], " ")
	}
	cmdutil.RunExplain(os.Stdout, topic)
}

// ── syntax ──

func cmdSyntaxCLI() {
	section := ""
	search := ""
	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		if args[i] == "--search" && i+1 < len(args) {
			search = strings.Join(args[i+1:], " ")
			break
		}
		if !strings.HasPrefix(args[i], "-") && section == "" {
			section = args[i]
		}
	}
	cmdutil.RunSyntax(os.Stdout, section, search)
}

// ── fix ──

func cmdFixCLI() {
	dryRun := false
	var file string
	for _, arg := range os.Args[2:] {
		switch arg {
		case "--dry-run":
			dryRun = true
		default:
			if !strings.HasPrefix(arg, "-") {
				file = arg
			}
		}
	}

	if file == "" {
		// Auto-detect.
		matches, _ := filepath.Glob("*.human")
		var files []string
		for _, m := range matches {
			info, err := os.Stat(m)
			if err == nil && !info.IsDir() {
				files = append(files, m)
			}
		}
		if len(files) == 1 {
			file = files[0]
		} else {
			fmt.Fprintln(os.Stderr, "Usage: human fix [--dry-run] <file.human>")
			os.Exit(1)
		}
	}

	result, err := fixer.Analyze([]string{file})
	if err != nil {
		fmt.Fprintln(os.Stderr, cli.Error(err.Error()))
		os.Exit(1)
	}

	fixer.PrintResult(os.Stdout, result, file)

	if dryRun || len(result.Fixes) == 0 {
		return
	}

	// Ask user.
	fmt.Printf("\nAuto-fix %d issue(s)? [y/n] ", len(result.Fixes))
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if answer == "y" || answer == "yes" {
			if err := fixer.ApplyAll(result.Fixes); err != nil {
				fmt.Fprintln(os.Stderr, cli.Error(err.Error()))
				os.Exit(1)
			}
			fmt.Println(cli.Success(fmt.Sprintf("Applied %d fix(es). Backup saved as %s.bak", len(result.Fixes), file)))
		} else {
			fmt.Println(cli.Info("No changes made."))
		}
	}
}

// ── Helpers ──

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
  storybook                 Launch Storybook dev server from build output

Reference & Diagnostics:
  explain [topic]           Learn Human syntax by topic
  syntax [section]          Full syntax reference
  syntax --search <term>    Search syntax patterns
  fix [--dry-run] <file>    Find and auto-fix common issues
  doctor                    Check environment health

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
