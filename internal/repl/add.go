package repl

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/barun-bash/human/internal/cli"
)

func cmdAdd(r *REPL, args []string) {
	if !r.requireProject() {
		return
	}

	description := strings.Join(args, " ")
	if description == "" {
		fmt.Fprintln(r.out, cli.Error("Usage: /add <what to add>"))
		fmt.Fprintln(r.out, cli.Muted("  Example: /add a Settings page with profile editing"))
		fmt.Fprintln(r.out, cli.Muted("  Example: /add a Comment data model with author and body"))
		fmt.Fprintln(r.out, cli.Muted("  Example: /add an API for searching products"))
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

	fmt.Fprintln(r.out, cli.Info(fmt.Sprintf("Adding with %s (%s)...", llmCfg.Provider, llmCfg.Model)))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	spinner := cli.NewSpinner(r.out, "Generating...")
	spinner.Start()

	result, err := connector.Add(ctx, string(source), description)
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

	showDiff(r, string(source), result.Code)

	_, yesFlag := extractYesFlag(args)
	if r.shouldAutoAccept(yesFlag) {
		fmt.Fprintln(r.out, cli.Muted("  Auto-applying changes..."))
	} else {
		fmt.Fprintf(r.out, "Apply changes? (y/n): ")
		answer, ok := r.scanLine()
		if !ok || !isYes(answer) {
			fmt.Fprintln(r.out, cli.Info("No changes made."))
			return
		}
	}

	backupFile(r.projectFile)

	if err := os.WriteFile(r.projectFile, []byte(result.Code), 0644); err != nil {
		fmt.Fprintln(r.errOut, cli.Error(err.Error()))
		return
	}

	r.clearSuggestions()
	fmt.Fprintln(r.out, cli.Success(fmt.Sprintf("Updated: %s", r.projectFile)))
}
