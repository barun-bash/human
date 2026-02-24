package repl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/barun-bash/human/internal/cli"
)

// cmdCd handles the /cd command — changes the working directory.
func cmdCd(r *REPL, args []string) {
	var target string

	if len(args) == 0 {
		// No args: go to home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(r.errOut, cli.Error("Could not determine home directory."))
			return
		}
		target = home
	} else {
		target = args[0]

		// Handle /cd - (go to previous directory).
		if target == "-" {
			if r.lastDir == "" {
				fmt.Fprintln(r.errOut, cli.Error("No previous directory."))
				return
			}
			target = r.lastDir
		}

		// Expand ~ prefix.
		if strings.HasPrefix(target, "~/") || target == "~" {
			home, err := os.UserHomeDir()
			if err != nil {
				fmt.Fprintln(r.errOut, cli.Error("Could not determine home directory."))
				return
			}
			if target == "~" {
				target = home
			} else {
				target = filepath.Join(home, target[2:])
			}
		}
	}

	// Resolve to absolute path.
	absPath, err := filepath.Abs(target)
	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Invalid path: %v", err)))
		return
	}

	// Check that target exists and is a directory.
	info, err := os.Stat(absPath)
	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("No such directory: %s", absPath)))
		return
	}
	if !info.IsDir() {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Not a directory: %s", absPath)))
		return
	}

	// Save current directory before changing.
	prevDir, _ := os.Getwd()

	if err := os.Chdir(absPath); err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Could not change directory: %v", err)))
		return
	}
	r.lastDir = prevDir

	fmt.Fprintln(r.out, absPath)

	// Re-detect project in the new directory.
	r.projectFile = ""
	r.projectName = ""
	r.instructions = ""
	r.clearSuggestions()
	r.autoDetectProject()
}

// cmdPwd handles the /pwd command — prints the working directory.
func cmdPwd(r *REPL, args []string) {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Could not determine working directory: %v", err)))
		return
	}
	fmt.Fprintln(r.out, cwd)
}

// completeDirectories returns directories matching the partial path for /cd completion.
func completeDirectories(partial string) []string {
	dir := "."
	prefix := ""
	if partial != "" {
		// Expand ~ for completion.
		expanded := partial
		if strings.HasPrefix(expanded, "~/") {
			home, err := os.UserHomeDir()
			if err == nil {
				expanded = filepath.Join(home, expanded[2:])
			}
		}
		dir = filepath.Dir(expanded)
		prefix = filepath.Base(partial)
		// If partial ends with /, list contents of that dir.
		if strings.HasSuffix(partial, "/") {
			dir = expanded
			prefix = ""
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var matches []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if prefix == "" || strings.HasPrefix(name, prefix) {
			fullPath := filepath.Join(dir, name)
			if dir == "." {
				fullPath = name
			}
			matches = append(matches, fullPath+"/")
		}
	}
	return matches
}

// completeCd provides tab completion for /cd.
func completeCd(_ *REPL, args []string, partial string) []string {
	return completeDirectories(partial)
}
