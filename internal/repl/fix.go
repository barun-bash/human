package repl

import (
	"fmt"
	"strings"

	"github.com/barun-bash/human/internal/cli"
	"github.com/barun-bash/human/internal/fixer"
)

func cmdFix(r *REPL, args []string) {
	if !r.requireProject() {
		return
	}

	dryRun := false
	for _, arg := range args {
		if arg == "--dry-run" {
			dryRun = true
		}
	}

	result, err := fixer.Analyze([]string{r.projectFile})
	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error(err.Error()))
		return
	}

	fixer.PrintResult(r.out, result, r.projectFile)

	if dryRun || len(result.Fixes) == 0 {
		return
	}

	// Ask user.
	fmt.Fprintf(r.out, "\nAuto-fix %d issue(s)? [y/n] ", len(result.Fixes))
	line, ok := r.scanLine()
	if !ok {
		return
	}
	answer := strings.ToLower(line)
	if answer == "y" || answer == "yes" {
		if err := fixer.ApplyAll(result.Fixes); err != nil {
			fmt.Fprintln(r.errOut, cli.Error(err.Error()))
			return
		}
		fmt.Fprintln(r.out, cli.Success(fmt.Sprintf("Applied %d fix(es). Backup saved as %s.bak", len(result.Fixes), r.projectFile)))
	} else {
		fmt.Fprintln(r.out, cli.Info("No changes made."))
	}
}
