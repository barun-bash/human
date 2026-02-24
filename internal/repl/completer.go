package repl

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/barun-bash/human/internal/cli"
	"github.com/barun-bash/human/internal/llm"
)

// buildCompleter returns a readline CompleteFunc that dispatches tab
// completion to the appropriate command completer.
func (r *REPL) buildCompleter() func(line string, pos int) []string {
	return func(line string, pos int) []string {
		runes := []rune(line)
		if pos > len(runes) {
			pos = len(runes)
		}
		before := string(runes[:pos])

		parts := strings.Fields(before)
		trailingSpace := len(before) > 0 && before[len(before)-1] == ' '

		// No input yet — show all commands.
		if len(parts) == 0 {
			return r.commandNames()
		}

		cmdName := strings.ToLower(parts[0])

		// Still typing the command name (no space yet).
		if len(parts) == 1 && !trailingSpace {
			return r.completeCommandName(cmdName)
		}

		// Resolve alias to canonical name.
		if target, ok := r.aliases[cmdName]; ok {
			cmdName = target
		}

		// Delegate to command-specific completer.
		cmd, ok := r.commands[cmdName]
		if !ok || cmd.Complete == nil {
			return nil
		}

		// Build args for the completer.
		var args []string
		if len(parts) > 1 {
			args = parts[1:]
		}
		partial := ""
		if !trailingSpace && len(args) > 0 {
			partial = args[len(args)-1]
			args = args[:len(args)-1]
		}

		return cmd.Complete(r, args, partial)
	}
}

// completeCommandName returns commands matching the partial prefix.
func (r *REPL) completeCommandName(partial string) []string {
	var matches []string
	for name := range r.commands {
		if strings.HasPrefix(name, partial) {
			matches = append(matches, name)
		}
	}
	// Also check aliases.
	for alias := range r.aliases {
		if strings.HasPrefix(alias, partial) {
			matches = append(matches, alias)
		}
	}
	return matches
}

// commandNames returns all command names (not aliases).
func (r *REPL) commandNames() []string {
	names := make([]string, 0, len(r.commands))
	for name := range r.commands {
		names = append(names, name)
	}
	return names
}

// ── Reusable completion helpers ──

// completeFromList completes partial against a fixed list of choices.
func completeFromList(choices []string, partial string) []string {
	if partial == "" {
		return choices
	}
	var matches []string
	p := strings.ToLower(partial)
	for _, c := range choices {
		if strings.HasPrefix(strings.ToLower(c), p) {
			matches = append(matches, c)
		}
	}
	return matches
}

// completeFiles returns .human files matching the partial path.
func completeFiles(partial string) []string {
	dir := "."
	prefix := ""
	if partial != "" {
		dir = filepath.Dir(partial)
		prefix = filepath.Base(partial)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var matches []string
	for _, e := range entries {
		name := e.Name()
		// Skip hidden files/dirs.
		if strings.HasPrefix(name, ".") {
			continue
		}

		fullPath := filepath.Join(dir, name)
		if dir == "." {
			fullPath = name
		}

		if e.IsDir() {
			// Show directories for navigation.
			candidate := fullPath + "/"
			if prefix == "" || strings.HasPrefix(name, prefix) {
				matches = append(matches, candidate)
			}
		} else if strings.HasSuffix(name, ".human") {
			if prefix == "" || strings.HasPrefix(name, prefix) {
				matches = append(matches, fullPath)
			}
		}
	}
	return matches
}

// ── Command-specific completers ──

func completeOpen(r *REPL, args []string, partial string) []string {
	return completeFiles(partial)
}

func completeInstructions(_ *REPL, args []string, partial string) []string {
	return completeFromList([]string{"edit", "init"}, partial)
}

func completeConnect(_ *REPL, args []string, partial string) []string {
	choices := append([]string{"status"}, llm.SupportedProviders...)
	return completeFromList(choices, partial)
}

func completeMCP(_ *REPL, args []string, partial string) []string {
	if len(args) == 0 {
		return completeFromList([]string{"list", "add", "remove", "status"}, partial)
	}
	// /mcp add <server> — complete known server names.
	sub := strings.ToLower(args[0])
	if sub == "add" || sub == "remove" {
		return completeFromList(knownServerNames(), partial)
	}
	return nil
}

func completeTheme(_ *REPL, args []string, partial string) []string {
	choices := append([]string{"list"}, cli.ThemeNames()...)
	return completeFromList(choices, partial)
}

func completeConfig(_ *REPL, args []string, partial string) []string {
	if len(args) == 0 {
		return completeFromList([]string{"set"}, partial)
	}
	if strings.ToLower(args[0]) == "set" {
		if len(args) == 1 {
			return completeFromList([]string{"animate", "auto_accept", "plan_mode", "theme"}, partial)
		}
		key := strings.ToLower(args[1])
		switch key {
		case "animate", "auto_accept":
			return completeFromList([]string{"on", "off"}, partial)
		case "plan_mode":
			return completeFromList([]string{"always", "auto", "off"}, partial)
		case "theme":
			return completeFromList(cli.ThemeNames(), partial)
		}
	}
	return nil
}

func completeSuggest(r *REPL, args []string, partial string) []string {
	if len(args) == 0 {
		return completeFromList([]string{"apply"}, partial)
	}
	if strings.ToLower(args[0]) == "apply" {
		choices := []string{"all"}
		for i := range r.lastSuggestions {
			choices = append(choices, strconv.Itoa(i+1))
		}
		return completeFromList(choices, partial)
	}
	return nil
}

func completeDeploy(_ *REPL, args []string, partial string) []string {
	return completeFromList([]string{"--dry-run"}, partial)
}

func completeBuild(_ *REPL, args []string, partial string) []string {
	return completeFromList([]string{"--dry-run"}, partial)
}

func completeHistory(_ *REPL, args []string, partial string) []string {
	return completeFromList([]string{"clear"}, partial)
}
