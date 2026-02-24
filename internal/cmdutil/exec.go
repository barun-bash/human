package cmdutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// RunCommand executes a command in the given directory with stdin, stdout,
// and stderr connected to the current process.
func RunCommand(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// RunCommandSilent executes a command in the given directory with stdout and
// stderr connected but no stdin.
func RunCommandSilent(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RequireOutputDir checks that .human/output/ exists and returns its path.
// Returns an error if the directory does not exist.
func RequireOutputDir() (string, error) {
	outputDir := filepath.Join(".human", "output")
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		return "", fmt.Errorf("no build found. Run 'human build <file>' first")
	}
	return outputDir, nil
}
