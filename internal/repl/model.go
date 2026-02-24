package repl

import (
	"fmt"
	"strings"

	"github.com/barun-bash/human/internal/cli"
	"github.com/barun-bash/human/internal/config"
)

// cmdModel handles the /model command — show or switch the LLM model.
func cmdModel(r *REPL, args []string) {
	gc, err := config.LoadGlobalConfig()
	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Could not load config: %v", err)))
		return
	}

	if gc.LLM == nil {
		fmt.Fprintln(r.errOut, cli.Error("No LLM provider configured. Run /connect first."))
		return
	}

	if len(args) == 0 {
		fmt.Fprintf(r.out, "Current model: %s (%s)\n", gc.LLM.Model, gc.LLM.Provider)
		return
	}

	sub := strings.ToLower(args[0])

	if sub == "list" {
		models := knownModels(gc.LLM.Provider)
		if gc.LLM.Provider == "ollama" && gc.LLM.BaseURL != "" {
			if installed := fetchOllamaModels(gc.LLM.BaseURL); len(installed) > 0 {
				models = installed
			}
		}
		if len(models) == 0 {
			fmt.Fprintln(r.out, cli.Muted("No known models for this provider. You can set any model name."))
			return
		}
		fmt.Fprintln(r.out, cli.Muted(fmt.Sprintf("  Models for %s:", gc.LLM.Provider)))
		for _, m := range models {
			marker := "  "
			if m == gc.LLM.Model {
				marker = cli.Accent("* ")
			}
			fmt.Fprintf(r.out, "    %s%s\n", marker, m)
		}
		return
	}

	// Set model.
	model := args[0] // preserve original case
	gc.LLM.Model = model
	if err := config.SaveGlobalConfig(gc); err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Could not save config: %v", err)))
		return
	}
	fmt.Fprintln(r.out, cli.Success(fmt.Sprintf("Model set to %s", model)))
}

// completeModel provides tab completion for /model.
func completeModel(r *REPL, args []string, partial string) []string {
	if len(args) > 0 {
		return nil
	}

	choices := []string{"list"}

	gc, err := config.LoadGlobalConfig()
	if err != nil || gc.LLM == nil {
		return completeFromList(choices, partial)
	}

	models := knownModels(gc.LLM.Provider)
	choices = append(choices, models...)
	return completeFromList(choices, partial)
}
