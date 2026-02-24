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

// ProjectTemplate describes a project starter template.
type ProjectTemplate struct {
	Name        string // display name
	Key         string // selection key
	Description string // one-line description
	ExampleDir  string // if non-empty, copy from examples/<ExampleDir>/app.human
}

// AvailableTemplates returns the built-in project templates.
func AvailableTemplates() []ProjectTemplate {
	return []ProjectTemplate{
		{Name: "Blank", Key: "blank", Description: "Minimal starter with one page and one API"},
		{Name: "Blog", Key: "blog", Description: "Blog with posts, comments, and authentication", ExampleDir: "blog"},
		{Name: "E-Commerce", Key: "ecommerce", Description: "Online store with products, cart, and checkout", ExampleDir: "ecommerce"},
		{Name: "SaaS Dashboard", Key: "saas", Description: "SaaS app with subscription tiers and dashboard", ExampleDir: "saas"},
		{Name: "API Only", Key: "api", Description: "REST API backend with no frontend", ExampleDir: "api-only"},
		{Name: "Task Manager", Key: "taskflow", Description: "Full-featured task management app", ExampleDir: "taskflow"},
	}
}

// InitProject scaffolds a new Human project. It prompts for template, platform,
// frontend, backend, and database choices, creates the project directory, and
// writes a starter app.human file plus HUMAN.md and .gitignore.
// Returns the path to the generated .human file.
func InitProject(name string, in io.Reader, out io.Writer) (string, error) {
	if name == "" {
		dir, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("could not determine current directory")
		}
		name = filepath.Base(dir)
	}

	scanner := bufio.NewScanner(in)

	// Step 1: Template selection.
	templates := AvailableTemplates()
	fmt.Fprintln(out, "\nChoose a template:")
	for i, t := range templates {
		fmt.Fprintf(out, "  %d. %-16s %s\n", i+1, t.Name, t.Description)
	}
	fmt.Fprintln(out)

	templateIdx := 0 // default: blank
	fmt.Fprintf(out, "Template [1]: ")
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input != "" {
			if n, err := strconv.Atoi(input); err == nil && n >= 1 && n <= len(templates) {
				templateIdx = n - 1
			} else {
				// Try matching by key name.
				for i, t := range templates {
					if strings.EqualFold(input, t.Key) || strings.EqualFold(input, t.Name) {
						templateIdx = i
						break
					}
				}
			}
		}
	}
	chosen := templates[templateIdx]

	// Step 2: Stack selection (for blank template or override).
	var content string
	if chosen.ExampleDir != "" {
		// Try to load from examples directory.
		loaded, err := loadExampleTemplate(chosen.ExampleDir, name)
		if err == nil {
			content = loaded
		}
	}

	if content == "" {
		// Blank template or example not found — use wizard.
		platform := Prompt(scanner, out, "Platform", []string{"web", "mobile", "api"}, "web")
		frontend := Prompt(scanner, out, "Frontend", []string{"React", "Vue", "Angular", "Svelte", "None"}, "React")
		backend := Prompt(scanner, out, "Backend", []string{"Node", "Python", "Go"}, "Node")
		database := Prompt(scanner, out, "Database", []string{"PostgreSQL", "MySQL", "SQLite"}, "PostgreSQL")
		content = GenerateTemplate(name, platform, frontend, backend, database)
	}

	// Create project directory.
	if err := os.MkdirAll(name, 0755); err != nil {
		return "", fmt.Errorf("could not create directory %s: %w", name, err)
	}

	// Write app.human.
	outPath := filepath.Join(name, "app.human")
	if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("could not write %s: %w", outPath, err)
	}

	// Write HUMAN.md.
	humanMD := generateHumanMD(name, chosen)
	humanMDPath := filepath.Join(name, "HUMAN.md")
	os.WriteFile(humanMDPath, []byte(humanMD), 0644) // non-fatal

	// Write .gitignore.
	gitignore := generateGitignore()
	gitignorePath := filepath.Join(name, ".gitignore")
	os.WriteFile(gitignorePath, []byte(gitignore), 0644) // non-fatal

	return outPath, nil
}

// loadExampleTemplate reads an example template and replaces the app name.
func loadExampleTemplate(exampleDir, newName string) (string, error) {
	// Try relative to CWD (development).
	path := filepath.Join("examples", exampleDir, "app.human")
	data, err := os.ReadFile(path)
	if err != nil {
		// Try relative to executable.
		exe, _ := os.Executable()
		if exe != "" {
			path = filepath.Join(filepath.Dir(exe), "..", "examples", exampleDir, "app.human")
			data, err = os.ReadFile(path)
		}
		if err != nil {
			return "", err
		}
	}

	// Replace the original app name with the new project name.
	content := string(data)
	// The first line is typically: app <Name> is a ...
	lines := strings.SplitN(content, "\n", 2)
	if len(lines) >= 2 && strings.HasPrefix(strings.ToLower(lines[0]), "app ") {
		// Replace "app OldName" with "app NewName"
		parts := strings.SplitN(lines[0], " ", 3)
		if len(parts) >= 3 {
			content = fmt.Sprintf("app %s %s\n%s", newName, parts[2], lines[1])
		}
	}

	return content, nil
}

// generateHumanMD creates a starter HUMAN.md for the project.
func generateHumanMD(name string, template ProjectTemplate) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", name)
	fmt.Fprintf(&b, "Project created with Human compiler using the **%s** template.\n\n", template.Name)
	b.WriteString("## Instructions\n\n")
	b.WriteString("Add project-specific instructions here. These are passed to the LLM\n")
	b.WriteString("when using `/ask`, `/edit`, and `/suggest` commands.\n\n")
	b.WriteString("## Notes\n\n")
	b.WriteString("- Run `/build` to compile this project\n")
	b.WriteString("- Run `/check` to validate syntax\n")
	b.WriteString("- Run `/edit <instruction>` for AI-assisted editing\n")
	return b.String()
}

// generateGitignore creates a .gitignore for Human projects.
func generateGitignore() string {
	return `.human/output/
.human/intent/
.human/config.json
*.tmp
node_modules/
.env
`
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
