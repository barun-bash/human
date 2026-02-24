package cmdutil

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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
