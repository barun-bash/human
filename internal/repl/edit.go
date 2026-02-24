package repl

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/barun-bash/human/internal/cli"
	"github.com/barun-bash/human/internal/config"
	"github.com/barun-bash/human/internal/llm"
)

// cmdEdit handles the /edit command — AI-assisted modification of the loaded .human file.
// Two modes:
//   - /edit <instruction>  — single-shot edit, show diff, accept/decline
//   - /edit                — interactive editing session (sub-REPL loop)
func cmdEdit(r *REPL, args []string) {
	if !r.requireProject() {
		return
	}

	// Load source.
	source, err := os.ReadFile(r.projectFile)
	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Could not read %s: %v", r.projectFile, err)))
		return
	}

	// Load LLM connector.
	connector, llmCfg, err := loadREPLConnector()
	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error(err.Error()))
		return
	}

	// Cost notice for non-local providers.
	if llmCfg.Provider != "ollama" {
		fmt.Fprintln(r.out, cli.Muted("  Note: This uses your API key and may incur costs."))
	}

	currentSource := string(source)

	if len(args) > 0 {
		// Single-instruction mode.
		instruction := strings.Join(args, " ")
		editOnce(r, connector, llmCfg, instruction, currentSource, nil)
		return
	}

	// Interactive editing session (sub-REPL loop).
	editInteractive(r, connector, llmCfg, currentSource)
}

// editOnce performs a single edit: call LLM, show diff, accept/decline.
// Returns the new source if accepted (or original if declined), whether
// the change was accepted, and the updated conversation history.
func editOnce(r *REPL, connector *llm.Connector, llmCfg *config.LLMConfig, instruction, currentSource string, history []llm.Message) (string, bool, []llm.Message) {
	fmt.Fprintf(r.out, "%s  Editing with %s (%s)...\n",
		cli.Info(""), llmCfg.Provider, llmCfg.Model)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	result, err := connector.Edit(ctx, currentSource, instruction, history)
	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Edit failed: %v", err)))
		return currentSource, false, history
	}

	// Show token usage.
	if result.Usage.InputTokens > 0 || result.Usage.OutputTokens > 0 {
		fmt.Fprintln(r.out, cli.Muted(fmt.Sprintf("  Tokens: %d in / %d out", result.Usage.InputTokens, result.Usage.OutputTokens)))
	}

	if strings.TrimSpace(result.Code) == "" {
		fmt.Fprintln(r.errOut, cli.Error("LLM returned no code. Try rephrasing your instruction."))
		return currentSource, false, history
	}

	fmt.Fprintln(r.out)

	// Show validation status.
	if result.Valid {
		fmt.Fprintln(r.out, cli.Success("Valid .human syntax."))
	} else {
		fmt.Fprintln(r.out, cli.Warn(fmt.Sprintf("Syntax issue: %s", result.ParseError)))
	}

	// Show diff.
	fmt.Fprintln(r.out)
	showDiff(r, currentSource, result.Code)

	// Accept?
	fmt.Fprintf(r.out, "Apply changes? (y/n): ")
	answer, ok := r.scanLine()
	if !ok || !isYes(answer) {
		fmt.Fprintln(r.out, cli.Info("Changes discarded."))
		return currentSource, false, history
	}

	// Create backup before applying.
	backupFile(r.projectFile)

	// Write to disk.
	if err := os.WriteFile(r.projectFile, []byte(result.Code), 0644); err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Could not write file: %v", err)))
		return currentSource, false, history
	}

	fmt.Fprintln(r.out, cli.Success(fmt.Sprintf("Applied to %s", r.projectFile)))

	// Update history for multi-turn context.
	newHistory := append(history,
		llm.Message{Role: llm.RoleUser, Content: instruction},
		llm.Message{Role: llm.RoleAssistant, Content: result.RawResponse},
	)

	return result.Code, true, newHistory
}

// editInteractive runs a blocking sub-REPL loop for multi-turn editing.
// The user types instructions at the "edit>" prompt until "done" or "quit".
// All input goes through r.scanLine() (the shared scanner).
func editInteractive(r *REPL, connector *llm.Connector, llmCfg *config.LLMConfig, currentSource string) {
	fmt.Fprintln(r.out, cli.Info(fmt.Sprintf("Editing %s with %s (%s)", r.projectFile, llmCfg.Provider, llmCfg.Model)))
	fmt.Fprintln(r.out, cli.Muted("  Type edit instructions. Commands: show, done, quit"))
	fmt.Fprintln(r.out)

	var history []llm.Message

	for {
		fmt.Fprintf(r.out, "edit> ")
		line, ok := r.scanLine()
		if !ok {
			// EOF
			break
		}
		if line == "" {
			continue
		}

		switch strings.ToLower(line) {
		case "done", "quit", "exit", "q":
			fmt.Fprintln(r.out, cli.Info("Edit session ended."))
			return
		case "show":
			fmt.Fprintln(r.out)
			fmt.Fprintln(r.out, currentSource)
			fmt.Fprintln(r.out)
			continue
		}

		var accepted bool
		currentSource, accepted, history = editOnce(r, connector, llmCfg, line, currentSource, history)
		_ = accepted
		fmt.Fprintln(r.out)
	}
}

// cmdUndo handles the /undo command — reverts the last /edit change.
// Single-level: only one undo is available (the most recent edit).
func cmdUndo(r *REPL, args []string) {
	if !r.requireProject() {
		return
	}

	bp := backupPath(r.projectFile)
	data, err := os.ReadFile(bp)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintln(r.errOut, cli.Error("Nothing to undo."))
		} else {
			fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Could not read backup: %v", err)))
		}
		return
	}

	// Check if backup differs from current.
	current, readErr := os.ReadFile(r.projectFile)
	if readErr != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Could not read %s: %v", r.projectFile, readErr)))
		return
	}

	if string(current) == string(data) {
		fmt.Fprintln(r.out, cli.Info("Backup is identical to current file. Nothing to undo."))
		return
	}

	// Restore from backup.
	if err := os.WriteFile(r.projectFile, data, 0644); err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Could not restore file: %v", err)))
		return
	}

	// Remove backup — single-level undo, no undo-of-undo.
	os.Remove(bp)

	fmt.Fprintln(r.out, cli.Success(fmt.Sprintf("Reverted %s to previous version.", r.projectFile)))
}

// ── Backup helpers ──

const backupDir = ".human/backup"

// backupPath returns the backup file path for a given project file.
func backupPath(projectFile string) string {
	base := filepath.Base(projectFile)
	return filepath.Join(backupDir, base+".bak")
}

// backupFile saves a copy of the project file before an edit is applied.
func backupFile(projectFile string) {
	data, err := os.ReadFile(projectFile)
	if err != nil {
		return // best-effort
	}
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return
	}
	_ = os.WriteFile(backupPath(projectFile), data, 0644)
}

// ── Diff display ──

// showDiff displays the differences between old and new source.
// Uses system `diff -u` for a unified diff, falls back to a line-count summary.
func showDiff(r *REPL, oldSrc, newSrc string) {
	if oldSrc == newSrc {
		fmt.Fprintln(r.out, cli.Muted("  No changes."))
		return
	}

	// Try system diff.
	diffOutput, err := systemDiff(oldSrc, newSrc)
	if err == nil && strings.TrimSpace(diffOutput) != "" {
		fmt.Fprintln(r.out, cli.Heading("Changes"))
		fmt.Fprintln(r.out, diffOutput)
		return
	}

	// Fallback: line-count summary.
	oldLines := strings.Split(oldSrc, "\n")
	newLines := strings.Split(newSrc, "\n")
	added, removed := diffSummary(oldLines, newLines)
	fmt.Fprintf(r.out, "  %s lines, %s lines (total: %d → %d)\n",
		cli.Success(fmt.Sprintf("+%d", added)),
		cli.Error(fmt.Sprintf("-%d", removed)),
		len(oldLines), len(newLines))
}

// systemDiff runs `diff -u` on two strings via temp files.
func systemDiff(oldSrc, newSrc string) (string, error) {
	tmpOld, err := os.CreateTemp("", "human-edit-old-*.human")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpOld.Name())

	tmpNew, err := os.CreateTemp("", "human-edit-new-*.human")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpNew.Name())

	if _, err := tmpOld.WriteString(oldSrc); err != nil {
		return "", err
	}
	tmpOld.Close()

	if _, err := tmpNew.WriteString(newSrc); err != nil {
		return "", err
	}
	tmpNew.Close()

	cmd := exec.Command("diff", "-u", "--label", "before", "--label", "after", tmpOld.Name(), tmpNew.Name())
	out, err := cmd.CombinedOutput()

	// diff exits with code 1 when files differ — that's not an error for us.
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return string(out), nil
		}
		return "", err
	}

	return string(out), nil
}

// diffSummary counts added and removed lines using bag-of-lines comparison.
func diffSummary(oldLines, newLines []string) (added, removed int) {
	oldBag := make(map[string]int, len(oldLines))
	for _, line := range oldLines {
		oldBag[line]++
	}

	newBag := make(map[string]int, len(newLines))
	for _, line := range newLines {
		newBag[line]++
	}

	for line, count := range newBag {
		if oldCount, ok := oldBag[line]; ok {
			if count > oldCount {
				added += count - oldCount
			}
		} else {
			added += count
		}
	}

	for line, count := range oldBag {
		if newCount, ok := newBag[line]; ok {
			if count > newCount {
				removed += count - newCount
			}
		} else {
			removed += count
		}
	}

	return added, removed
}
