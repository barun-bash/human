package repl

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/barun-bash/human/internal/cli"
)

// cmdSuggest handles the /suggest command — AI analysis of the loaded .human file.
//
// Subcommands:
//   - /suggest           — analyze and display numbered suggestions
//   - /suggest apply 1   — apply suggestion #1 via /edit logic
//   - /suggest apply all — apply all suggestions sequentially
func cmdSuggest(r *REPL, args []string) {
	if !r.requireProject() {
		return
	}

	// Dispatch subcommands.
	if len(args) > 0 && strings.ToLower(args[0]) == "apply" {
		suggestApply(r, args[1:])
		return
	}

	// Main flow: analyze the file.
	suggestAnalyze(r)
}

// suggestAnalyze runs the LLM suggestion analysis and displays results.
func suggestAnalyze(r *REPL) {
	source, err := os.ReadFile(r.projectFile)
	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Could not read %s: %v", r.projectFile, err)))
		return
	}

	connector, llmCfg, err := loadREPLConnector()
	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error(err.Error()))
		return
	}
	connector.Instructions = r.instructions

	// Cost notice.
	if llmCfg.Provider != "ollama" {
		fmt.Fprintln(r.out, cli.Muted("  Note: This uses your API key and may incur costs."))
	}

	fmt.Fprintf(r.out, "%s  Analyzing %s with %s (%s)...\n",
		cli.Info(""), r.projectFile, llmCfg.Provider, llmCfg.Model)
	fmt.Fprintln(r.out)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	result, err := connector.Suggest(ctx, string(source))
	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Analysis failed: %v", err)))
		return
	}

	// Token usage.
	if result.Usage.InputTokens > 0 || result.Usage.OutputTokens > 0 {
		fmt.Fprintln(r.out, cli.Muted(fmt.Sprintf("  Tokens: %d in / %d out", result.Usage.InputTokens, result.Usage.OutputTokens)))
		fmt.Fprintln(r.out)
	}

	// No structured suggestions — fall back to raw response.
	if len(result.Suggestions) == 0 {
		r.lastSuggestions = nil
		fmt.Fprintln(r.out, result.RawResponse)
		fmt.Fprintln(r.out)
		fmt.Fprintln(r.out, cli.Muted("  Could not parse structured suggestions. Showing raw response."))
		return
	}

	// Cache for /suggest apply.
	r.lastSuggestions = result.Suggestions

	// Display as numbered list.
	fmt.Fprintln(r.out, cli.Heading("Suggestions"))
	fmt.Fprintln(r.out, strings.Repeat("\u2500", 50))
	for i, s := range result.Suggestions {
		fmt.Fprintf(r.out, "  %d. [%s] %s\n", i+1, s.Category, s.Text)
	}
	fmt.Fprintln(r.out)
	fmt.Fprintln(r.out, cli.Muted(fmt.Sprintf("  %d suggestion(s). Use /suggest apply <number> or /suggest apply all.", len(result.Suggestions))))
	fmt.Fprintln(r.out)
}

// suggestApply applies one or all cached suggestions using /edit logic.
func suggestApply(r *REPL, args []string) {
	if len(r.lastSuggestions) == 0 {
		fmt.Fprintln(r.errOut, cli.Error("No suggestions available. Run /suggest first."))
		return
	}

	if len(args) == 0 {
		fmt.Fprintln(r.errOut, "Usage: /suggest apply <number|all>")
		return
	}

	target := strings.ToLower(args[0])

	if target == "all" {
		suggestApplyAll(r)
		return
	}

	// Parse number.
	num, err := strconv.Atoi(target)
	if err != nil || num < 1 || num > len(r.lastSuggestions) {
		fmt.Fprintf(r.errOut, "Invalid suggestion number: %s (valid: 1-%d)\n", target, len(r.lastSuggestions))
		return
	}

	suggestApplyOne(r, num-1) // 0-indexed internally
}

// suggestApplyOne applies a single suggestion by index.
func suggestApplyOne(r *REPL, idx int) {
	s := r.lastSuggestions[idx]

	fmt.Fprintf(r.out, "Applying suggestion %d: [%s] %s\n", idx+1, s.Category, s.Text)
	fmt.Fprintln(r.out)

	connector, llmCfg, err := loadREPLConnector()
	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error(err.Error()))
		return
	}
	connector.Instructions = r.instructions

	source, err := os.ReadFile(r.projectFile)
	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Could not read %s: %v", r.projectFile, err)))
		return
	}

	// Use the suggestion text as the edit instruction.
	_, accepted, _ := editOnce(r, connector, llmCfg, s.Text, string(source), nil)
	if accepted {
		// Source changed — clear suggestions since they're now stale.
		r.clearSuggestions()
	}
}

// suggestApplyAll applies all suggestions sequentially with per-suggestion y/n.
func suggestApplyAll(r *REPL) {
	connector, llmCfg, err := loadREPLConnector()
	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error(err.Error()))
		return
	}
	connector.Instructions = r.instructions

	total := len(r.lastSuggestions)
	applied := 0
	skipped := 0
	failed := 0

	// Take a snapshot — suggestions get cleared after first accepted edit.
	suggestions := make([]struct{ category, text string }, total)
	for i, s := range r.lastSuggestions {
		suggestions[i] = struct{ category, text string }{s.Category, s.Text}
	}

	fmt.Fprintf(r.out, "Applying %d suggestion(s) sequentially...\n", total)
	fmt.Fprintln(r.out)

	for i, s := range suggestions {
		fmt.Fprintf(r.out, "── Suggestion %d/%d: [%s] %s ──\n", i+1, total, s.category, s.text)
		fmt.Fprintln(r.out)

		// Re-read source from disk (previous edit may have changed it).
		source, err := os.ReadFile(r.projectFile)
		if err != nil {
			fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Could not read %s: %v", r.projectFile, err)))
			failed++
			continue
		}

		newSource, accepted, _ := editOnce(r, connector, llmCfg, s.text, string(source), nil)
		_ = newSource

		if accepted {
			applied++
		} else {
			skipped++
		}
		fmt.Fprintln(r.out)
	}

	// Clear suggestions — source has changed.
	r.clearSuggestions()

	// Summary.
	fmt.Fprintln(r.out, cli.Heading("Summary"))
	fmt.Fprintf(r.out, "  Applied: %d, Skipped: %d, Failed: %d (of %d)\n", applied, skipped, failed, total)
	fmt.Fprintln(r.out)
}

