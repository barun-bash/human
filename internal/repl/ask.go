package repl

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/barun-bash/human/internal/cli"
	"github.com/barun-bash/human/internal/cmdutil"
	"github.com/barun-bash/human/internal/config"
	"github.com/barun-bash/human/internal/llm"
)

// cmdAsk handles the /ask command — generates a .human file from a description.
func cmdAsk(r *REPL, args []string) {
	query := strings.Join(args, " ")

	// If no args, prompt for a description.
	if query == "" {
		fmt.Fprintln(r.out, "Describe the app you want to build:")
		line, ok := r.scanLine()
		if !ok || line == "" {
			fmt.Fprintln(r.errOut, cli.Error("No description provided."))
			return
		}
		query = line
	}

	// Load LLM connector (REPL-safe: returns errors, checks global config).
	connector, llmCfg, err := loadREPLConnector()
	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error(err.Error()))
		return
	}

	// Cost notice for non-local providers.
	if llmCfg.Provider != "ollama" {
		fmt.Fprintln(r.out, cli.Muted("  Note: This uses your API key and may incur costs."))
	}

	fmt.Fprintf(r.out, "%s  Generating with %s (%s)...\n",
		cli.Info(""), llmCfg.Provider, llmCfg.Model)
	fmt.Fprintln(r.out)

	// Pass project instructions to the connector.
	connector.Instructions = r.instructions

	// Use streaming for real-time output.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	ch, err := connector.AskStream(ctx, query)
	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("LLM request failed: %v", err)))
		return
	}

	// Show a spinner until the first token arrives.
	spinner := cli.NewSpinner(r.out, "Thinking...")
	spinner.Start()
	firstChunk := true

	var fullText strings.Builder
	var totalIn, totalOut int
	for chunk := range ch {
		if chunk.Err != nil {
			spinner.Stop()
			fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Stream error: %v", chunk.Err)))
			return
		}
		if firstChunk && chunk.Delta != "" {
			spinner.Stop()
			firstChunk = false
		}
		fmt.Fprint(r.out, chunk.Delta)
		fullText.WriteString(chunk.Delta)
		if chunk.Usage != nil {
			totalIn = chunk.Usage.InputTokens
			totalOut = chunk.Usage.OutputTokens
		}
	}
	if firstChunk {
		spinner.Stop() // no text chunks received
	}
	fmt.Fprintln(r.out)

	// Show token usage.
	if totalIn > 0 || totalOut > 0 {
		fmt.Fprintln(r.out, cli.Muted(fmt.Sprintf("  Tokens: %d in / %d out", totalIn, totalOut)))
	}

	// Extract and validate.
	code, valid, parseErr := llm.ExtractAndValidate(fullText.String())
	if strings.TrimSpace(code) == "" {
		fmt.Fprintln(r.errOut, cli.Error("LLM returned no code. Try rephrasing your description."))
		return
	}

	fmt.Fprintln(r.out)
	if valid {
		fmt.Fprintln(r.out, cli.Success("Generated code is valid .human syntax."))
	} else {
		fmt.Fprintln(r.out, cli.Warn(fmt.Sprintf("Syntax issue: %s", parseErr)))
		fmt.Fprintln(r.out, cli.Muted("  The file will be saved but may need manual adjustments."))
	}

	// Derive filename from the app name in the generated code.
	filename := deriveFilename(code)

	// Check for overwrite.
	if _, err := os.Stat(filename); err == nil {
		fmt.Fprintf(r.out, "%s already exists. Overwrite? (y/n): ", filename)
		answer, ok := r.scanLine()
		if !ok || !isYes(answer) {
			fmt.Fprintln(r.out, cli.Info("Save cancelled."))
			return
		}
	}

	// Write to disk.
	if err := os.WriteFile(filename, []byte(code+"\n"), 0644); err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Could not write file: %v", err)))
		return
	}
	fmt.Fprintln(r.out, cli.Success(fmt.Sprintf("Saved to %s", filename)))

	// Auto-load as current project.
	r.setProject(filename)
	r.clearSuggestions()

	// Prompt to build (auto-accept skips the prompt).
	if r.settings.AutoAcceptEnabled() {
		fmt.Fprintln(r.out, cli.Muted("  Auto-building..."))
	} else {
		fmt.Fprintf(r.out, "Build now? (y/n): ")
		answer, ok := r.scanLine()
		if !ok || !isYes(answer) {
			fmt.Fprintln(r.out, cli.Muted("  Run /build when you're ready."))
			return
		}
	}

	// Build directly, skipping plan mode (user already confirmed or auto-accepted).
	if _, _, _, err := cmdutil.FullBuild(r.projectFile); err != nil {
		fmt.Fprintln(r.errOut, cli.Error(err.Error()))
	}
}

// loadREPLConnector resolves the LLM provider for use in the REPL.
// Resolution order: project config → global config → env vars → error.
// Unlike the CLI's loadLLMConnector(), this returns errors (no os.Exit)
// and checks the global config from /connect.
func loadREPLConnector() (*llm.Connector, *config.LLMConfig, error) {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	// 1. Try project config.
	cfg, err := config.Load(cwd)
	if err != nil {
		return nil, nil, fmt.Errorf("config error: %w", err)
	}

	// 2. If no project LLM config, try global config from /connect.
	if cfg.LLM == nil {
		gc, err := config.LoadGlobalConfig()
		if err == nil && gc.LLM != nil {
			cfg.LLM = &config.LLMConfig{
				Provider: gc.LLM.Provider,
				Model:    gc.LLM.Model,
				APIKey:   gc.LLM.APIKey,
				BaseURL:  gc.LLM.BaseURL,
			}
		}
	}

	// 3. If still nothing, try env vars.
	if cfg.LLM == nil {
		cfg.LLM = detectProviderFromEnv()
	}

	// 4. No provider found — helpful error.
	if cfg.LLM == nil {
		return nil, nil, fmt.Errorf("no LLM provider configured. Run /connect to set one up")
	}

	// Apply defaults.
	if cfg.LLM.Model == "" {
		defaults := config.DefaultLLMConfig(cfg.LLM.Provider)
		cfg.LLM.Model = defaults.Model
	}
	if cfg.LLM.MaxTokens == 0 {
		cfg.LLM.MaxTokens = 4096
	}

	provider, err := llm.NewProvider(cfg.LLM)
	if err != nil {
		return nil, nil, err
	}

	return llm.NewConnector(provider, cfg.LLM), cfg.LLM, nil
}

// detectProviderFromEnv checks for API keys in environment variables.
func detectProviderFromEnv() *config.LLMConfig {
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		cfg := config.DefaultLLMConfig("anthropic")
		cfg.APIKey = key
		return cfg
	}
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		cfg := config.DefaultLLMConfig("openai")
		cfg.APIKey = key
		return cfg
	}
	return nil
}

// appNamePattern matches: app <Name> is a ...
var appNamePattern = regexp.MustCompile(`(?im)^\s*app\s+(\S+)\s+is\s+`)

// deriveFilename extracts the app name from generated .human code and returns
// a suitable filename. Falls back to "app.human" if no app name is found.
func deriveFilename(code string) string {
	matches := appNamePattern.FindStringSubmatch(code)
	if len(matches) >= 2 {
		name := matches[1]
		// Convert CamelCase to lowercase with hyphens.
		name = camelToKebab(name)
		// Sanitize: only allow lowercase alphanumeric and hyphens.
		name = sanitizeFilename(name)
		if name != "" {
			return name + ".human"
		}
	}
	return "app.human"
}

// camelToKebab converts "KanbanFlow" to "kanban-flow".
func camelToKebab(s string) string {
	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			prev := rune(s[i-1])
			if unicode.IsLower(prev) || unicode.IsDigit(prev) {
				result.WriteByte('-')
			}
		}
		result.WriteRune(unicode.ToLower(r))
	}
	return result.String()
}

// sanitizeFilename keeps only lowercase letters, digits, and hyphens.
func sanitizeFilename(s string) string {
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	// Trim leading/trailing hyphens.
	return strings.Trim(result.String(), "-")
}

// isYes returns true for common affirmative inputs.
func isYes(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "y", "yes":
		return true
	}
	return false
}
