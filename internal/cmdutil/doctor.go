package cmdutil

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/barun-bash/human/internal/cli"
	"github.com/barun-bash/human/internal/config"
	"github.com/barun-bash/human/internal/version"
)

// DoctorCheck is a single health check with its result.
type DoctorCheck struct {
	Name   string
	Status string // "ok", "warn", "fail"
	Detail string
	Fix    string // suggested fix (empty if ok)
}

// RunDoctor performs environment, configuration, and project health checks.
func RunDoctor(out io.Writer) {
	fmt.Fprintln(out)

	// Environment checks.
	envChecks := checkEnvironment()
	printSection(out, "Environment", envChecks)

	// Configuration checks.
	cfgChecks := checkConfiguration()
	printSection(out, "Configuration", cfgChecks)

	// Project checks.
	projChecks := checkProject()
	printSection(out, "Project", projChecks)

	// Summary.
	allChecks := append(envChecks, cfgChecks...)
	allChecks = append(allChecks, projChecks...)

	fails := 0
	warns := 0
	for _, c := range allChecks {
		switch c.Status {
		case "fail":
			fails++
		case "warn":
			warns++
		}
	}

	fmt.Fprintln(out)
	if fails > 0 {
		fmt.Fprintln(out, cli.Error(fmt.Sprintf("Found %d issue(s) that need fixing.", fails)))
	} else if warns > 0 {
		fmt.Fprintln(out, cli.Warn(fmt.Sprintf("Ready with %d warning(s).", warns)))
	} else {
		fmt.Fprintln(out, cli.Success("All checks passed. Ready to build."))
	}
	fmt.Fprintln(out)
}

func checkEnvironment() []DoctorCheck {
	var checks []DoctorCheck

	// Human compiler version.
	checks = append(checks, DoctorCheck{
		Name:   "Human compiler",
		Status: "ok",
		Detail: fmt.Sprintf("v%s", version.Info()),
	})

	// Go runtime.
	checks = append(checks, DoctorCheck{
		Name:   "Go runtime",
		Status: "ok",
		Detail: runtime.Version(),
	})

	// Docker.
	if path, err := exec.LookPath("docker"); err == nil {
		ver := getCommandVersion("docker", "--version")
		checks = append(checks, DoctorCheck{
			Name:   "Docker",
			Status: "ok",
			Detail: ver,
		})
		_ = path
	} else {
		checks = append(checks, DoctorCheck{
			Name:   "Docker",
			Status: "warn",
			Detail: "not found",
			Fix:    "Install Docker from https://docker.com (needed for deploy)",
		})
	}

	// Node.js.
	if _, err := exec.LookPath("node"); err == nil {
		ver := getCommandVersion("node", "--version")
		checks = append(checks, DoctorCheck{
			Name:   "Node.js",
			Status: "ok",
			Detail: ver,
		})
	} else {
		checks = append(checks, DoctorCheck{
			Name:   "Node.js",
			Status: "warn",
			Detail: "not found",
			Fix:    "Install Node.js from https://nodejs.org (needed for frontend)",
		})
	}

	// Python.
	if _, err := exec.LookPath("python3"); err == nil {
		ver := getCommandVersion("python3", "--version")
		checks = append(checks, DoctorCheck{
			Name:   "Python",
			Status: "ok",
			Detail: ver,
		})
	} else {
		checks = append(checks, DoctorCheck{
			Name:   "Python",
			Status: "warn",
			Detail: "not found",
			Fix:    "Install Python from https://python.org (needed for Python backend)",
		})
	}

	// Terraform.
	if _, err := exec.LookPath("terraform"); err == nil {
		ver := getCommandVersion("terraform", "--version")
		// Terraform --version may output multiple lines; take the first.
		if idx := strings.Index(ver, "\n"); idx > 0 {
			ver = ver[:idx]
		}
		checks = append(checks, DoctorCheck{
			Name:   "Terraform",
			Status: "ok",
			Detail: ver,
		})
	} else {
		checks = append(checks, DoctorCheck{
			Name:   "Terraform",
			Status: "warn",
			Detail: "not found",
			Fix:    "Install Terraform from https://terraform.io (needed for AWS/GCP deploy)",
		})
	}

	return checks
}

func checkConfiguration() []DoctorCheck {
	var checks []DoctorCheck

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	// Project config.
	cfg, err := config.Load(cwd)
	if err != nil {
		checks = append(checks, DoctorCheck{
			Name:   "Project config",
			Status: "fail",
			Detail: err.Error(),
			Fix:    "Run 'human init' to create a project",
		})
	} else {
		configPath := filepath.Join(cwd, ".human", "config.json")
		if _, err := os.Stat(configPath); err == nil {
			checks = append(checks, DoctorCheck{
				Name:   "Project config",
				Status: "ok",
				Detail: ".human/config.json",
			})
		} else {
			checks = append(checks, DoctorCheck{
				Name:   "Project config",
				Status: "warn",
				Detail: "no config file (using defaults)",
				Fix:    "Run 'human init' or create .human/config.json",
			})
		}

		// LLM configured.
		if cfg.LLM != nil {
			checks = append(checks, DoctorCheck{
				Name:   "LLM provider",
				Status: "ok",
				Detail: fmt.Sprintf("%s (%s)", cfg.LLM.Provider, cfg.LLM.Model),
			})
		} else if os.Getenv("ANTHROPIC_API_KEY") != "" || os.Getenv("OPENAI_API_KEY") != "" {
			checks = append(checks, DoctorCheck{
				Name:   "LLM provider",
				Status: "ok",
				Detail: "detected from environment",
			})
		} else {
			checks = append(checks, DoctorCheck{
				Name:   "LLM provider",
				Status: "warn",
				Detail: "not configured (optional)",
				Fix:    "Run 'human connect' or set ANTHROPIC_API_KEY",
			})
		}
	}

	return checks
}

func checkProject() []DoctorCheck {
	var checks []DoctorCheck

	// Find .human files in current directory.
	matches, _ := filepath.Glob("*.human")
	var files []string
	for _, m := range matches {
		info, err := os.Stat(m)
		if err == nil && !info.IsDir() {
			files = append(files, m)
		}
	}

	if len(files) == 0 {
		checks = append(checks, DoctorCheck{
			Name:   ".human files",
			Status: "warn",
			Detail: "no .human files found",
			Fix:    "Create a .human file or cd to your project directory",
		})
		return checks
	}

	for _, file := range files {
		result, err := ParseAndAnalyze(file)
		if err != nil {
			checks = append(checks, DoctorCheck{
				Name:   file,
				Status: "fail",
				Detail: err.Error(),
			})
			continue
		}

		errCount := len(result.Errs.Errors())
		warnCount := len(result.Errs.Warnings())

		if errCount > 0 {
			checks = append(checks, DoctorCheck{
				Name:   file,
				Status: "fail",
				Detail: fmt.Sprintf("%d error(s), %d warning(s)", errCount, warnCount),
				Fix:    fmt.Sprintf("Run 'human check %s' for details", file),
			})
		} else if warnCount > 0 {
			checks = append(checks, DoctorCheck{
				Name:   file,
				Status: "warn",
				Detail: fmt.Sprintf("valid (%d warning(s))", warnCount),
			})
		} else {
			checks = append(checks, DoctorCheck{
				Name:   file,
				Status: "ok",
				Detail: "valid",
			})
		}
	}

	return checks
}

func printSection(out io.Writer, title string, checks []DoctorCheck) {
	header := fmt.Sprintf("── %s ", title)
	pad := 50 - len([]rune(header))
	if pad < 0 {
		pad = 0
	}
	fmt.Fprintf(out, "%s%s\n", cli.Heading(header), strings.Repeat("─", pad))

	for _, c := range checks {
		var marker string
		switch c.Status {
		case "ok":
			marker = cli.Success("")
			// Strip the trailing space from ✓ prefix.
			marker = strings.TrimSpace(marker)
		case "warn":
			marker = cli.Warn("")
			marker = strings.TrimSpace(marker)
		case "fail":
			marker = cli.Error("")
			marker = strings.TrimSpace(marker)
		}

		fmt.Fprintf(out, "%s %s %s\n", marker, c.Name, cli.Muted(c.Detail))
		if c.Fix != "" {
			fmt.Fprintf(out, "  %s\n", cli.Muted("Fix: "+c.Fix))
		}
	}
	fmt.Fprintln(out)
}

func getCommandVersion(name string, args ...string) string {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return "installed"
	}
	return strings.TrimSpace(string(out))
}
