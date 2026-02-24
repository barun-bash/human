package cmdutil

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/barun-bash/human/internal/ir"
)

// InitProject scaffolds a new Human project. It prompts for platform, frontend,
// backend, and database choices, creates the project directory, and writes a
// starter app.human file. Returns the path to the generated file.
func InitProject(name string, in io.Reader, out io.Writer) (string, error) {
	if name == "" {
		dir, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("could not determine current directory")
		}
		name = filepath.Base(dir)
	}

	scanner := bufio.NewScanner(in)

	platform := Prompt(scanner, out, "Platform", []string{"web", "mobile", "api"}, "web")
	frontend := Prompt(scanner, out, "Frontend", []string{"React", "Vue", "Angular", "Svelte", "None"}, "React")
	backend := Prompt(scanner, out, "Backend", []string{"Node", "Python", "Go"}, "Node")
	database := Prompt(scanner, out, "Database", []string{"PostgreSQL", "MySQL", "SQLite"}, "PostgreSQL")

	if err := os.MkdirAll(name, 0755); err != nil {
		return "", fmt.Errorf("could not create directory %s: %w", name, err)
	}

	content := GenerateTemplate(name, platform, frontend, backend, database)
	outPath := filepath.Join(name, "app.human")
	if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("could not write %s: %w", outPath, err)
	}

	return outPath, nil
}

// Prompt asks the user to choose from options with a default value.
func Prompt(scanner *bufio.Scanner, out io.Writer, label string, options []string, defaultVal string) string {
	fmt.Fprintf(out, "%s (%s) [%s]: ", label, strings.Join(options, "/"), defaultVal)
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input != "" {
			for _, opt := range options {
				if strings.EqualFold(input, opt) {
					return opt
				}
			}
			return input
		}
	}
	return defaultVal
}

// GenerateTemplate produces a starter app.human file with the given configuration.
func GenerateTemplate(name, platform, frontend, backend, database string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "app %s is a %s application\n\n", name, platform)

	b.WriteString("# ── Data Models ──\n\n")
	b.WriteString("data User:\n")
	b.WriteString("  name is text, required\n")
	b.WriteString("  email is email, required, unique\n")
	b.WriteString("  password is text, required, encrypted\n\n")

	b.WriteString("# ── Pages ──\n\n")
	b.WriteString("page Home:\n")
	b.WriteString("  show heading \"Welcome to " + name + "\"\n\n")

	b.WriteString("# ── APIs ──\n\n")
	b.WriteString("api SignUp:\n")
	b.WriteString("  accepts name, email, password\n")
	b.WriteString("  create User with name, email, password\n")
	b.WriteString("  respond with success\n\n")

	b.WriteString("# ── Build ──\n\n")
	b.WriteString("build with:\n")
	fmt.Fprintf(&b, "  frontend using %s\n", frontend)
	fmt.Fprintf(&b, "  backend using %s\n", backend)
	fmt.Fprintf(&b, "  database using %s\n", database)
	b.WriteString("  deploy to Docker\n")

	return b.String()
}

// PromptForPorts interactively prompts the user to configure service ports.
// Returns a PortConfig with user-provided or default values.
// If running non-interactively (piped stdin), returns defaults without prompting.
func PromptForPorts(in io.Reader, out io.Writer) ir.PortConfig {
	// Check if stdin is a terminal (interactive)
	file, ok := in.(*os.File)
	if !ok || file.Fd() != 0 {
		// Not a terminal or not stdin, use defaults
		return ir.PortConfig{
			Frontend: 3000,
			Backend:  3001,
			Database: 5432,
		}
	}

	scanner := bufio.NewScanner(in)

	fmt.Fprintf(out, "\nConfigure service ports:\n")

	frontendPort := PromptForPort(scanner, out, "Frontend", 3000)
	backendPort := PromptForPort(scanner, out, "Backend", 3001)
	databasePort := PromptForPort(scanner, out, "Database", 5432)

	return ir.PortConfig{
		Frontend: frontendPort,
		Backend:  backendPort,
		Database: databasePort,
	}
}

// PromptForPort asks the user for a port number with a default value.
func PromptForPort(scanner *bufio.Scanner, out io.Writer, label string, defaultPort int) int {
	for {
		fmt.Fprintf(out, "  %s port [%d]: ", label, defaultPort)
		if scanner.Scan() {
			input := strings.TrimSpace(scanner.Text())
			if input == "" {
				return defaultPort
			}
			if port, err := strconv.Atoi(input); err == nil && port > 0 && port <= 65535 {
				return port
			}
			fmt.Fprintf(out, "    Invalid port. Please enter a number between 1 and 65535.\n")
			continue
		}
		return defaultPort
	}
}
