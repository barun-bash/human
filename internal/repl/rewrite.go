package repl

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/barun-bash/human/internal/cli"
)

func cmdRewrite(r *REPL, args []string) {
	if !r.requireProject() {
		return
	}

	approach := strings.Join(args, " ")
	if approach == "" {
		fmt.Fprintln(r.out, cli.Error("Usage: /rewrite <approach description>"))
		fmt.Fprintln(r.out, cli.Muted("  Example: /rewrite use microservices instead of monolith"))
		fmt.Fprintln(r.out, cli.Muted("  Example: /rewrite add real-time features with WebSockets"))
		return
	}

	source, err := os.ReadFile(r.projectFile)
	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error(err.Error()))
		return
	}

	connector, llmCfg, err := loadREPLConnector()
	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error(err.Error()))
		return
	}
	connector.Instructions = r.instructions

	if llmCfg.Provider != "ollama" {
		fmt.Fprintln(r.out, cli.Muted("  Note: This uses your API key and may incur costs."))
	}

	fmt.Fprintln(r.out, cli.Info(fmt.Sprintf("Rewriting with %s (%s)...", llmCfg.Provider, llmCfg.Model)))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	spinner := cli.NewSpinner(r.out, "Rewriting...")
	spinner.Start()

	result, err := connector.Rewrite(ctx, string(source), approach)
	spinner.Stop()

	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error(err.Error()))
		return
	}

	fmt.Fprintln(r.out, cli.Muted(fmt.Sprintf("  Tokens: %d in / %d out", result.Usage.InputTokens, result.Usage.OutputTokens)))

	if result.Code == "" {
		fmt.Fprintln(r.errOut, cli.Error("LLM returned empty code."))
		return
	}

	if result.Valid {
		fmt.Fprintln(r.out, cli.Success("Valid .human syntax."))
	} else {
		fmt.Fprintln(r.out, cli.Warn(fmt.Sprintf("Syntax issue: %s", result.ParseError)))
	}

	// Show diff.
	showDiff(r, string(source), result.Code)

	// Accept/reject.
	_, yesFlag := extractYesFlag(args)
	if r.shouldAutoAccept(yesFlag) {
		fmt.Fprintln(r.out, cli.Muted("  Auto-applying rewrite..."))
	} else {
		fmt.Fprintf(r.out, "Apply rewrite? (y/n): ")
		answer, ok := r.scanLine()
		if !ok || !isYes(answer) {
			fmt.Fprintln(r.out, cli.Info("Rewrite discarded."))
			return
		}
	}

	backupFile(r.projectFile)

	if err := os.WriteFile(r.projectFile, []byte(result.Code), 0644); err != nil {
		fmt.Fprintln(r.errOut, cli.Error(err.Error()))
		return
	}

	r.clearSuggestions()
	fmt.Fprintln(r.out, cli.Success(fmt.Sprintf("Rewritten: %s", r.projectFile)))
}
