package repl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/barun-bash/human/internal/cli"
	"github.com/barun-bash/human/internal/config"
)

// REPL is the interactive Human compiler shell.
type REPL struct {
	projectFile string
	projectName string
	version     string
	in          io.Reader
	out         io.Writer
	errOut      io.Writer
	history     *History
	commands    map[string]*Command
	aliases     map[string]string
	running     bool
	settings    *config.GlobalSettings
}

// Option configures the REPL.
type Option func(*REPL)

// WithInput sets the input reader (default: os.Stdin).
func WithInput(r io.Reader) Option {
	return func(repl *REPL) { repl.in = r }
}

// WithOutput sets the output writer (default: os.Stdout).
func WithOutput(w io.Writer) Option {
	return func(repl *REPL) { repl.out = w }
}

// WithErrOutput sets the error output writer (default: os.Stderr).
func WithErrOutput(w io.Writer) Option {
	return func(repl *REPL) { repl.errOut = w }
}

// New creates a REPL with the given version and options.
func New(version string, opts ...Option) *REPL {
	r := &REPL{
		version:  version,
		in:       os.Stdin,
		out:      os.Stdout,
		errOut:   os.Stderr,
		commands: make(map[string]*Command),
		aliases:  make(map[string]string),
	}
	for _, opt := range opts {
		opt(r)
	}
	r.history = NewHistory()
	r.registerCommands()
	r.loadSettings()
	return r
}

// Run starts the REPL loop: banner, prompt, read, dispatch, repeat.
func (r *REPL) Run() {
	r.autoDetectProject()
	r.printBanner()
	r.running = true

	scanner := bufio.NewScanner(r.in)
	for r.running {
		r.printPrompt()
		if !scanner.Scan() {
			// EOF (Ctrl+D)
			fmt.Fprintln(r.out)
			fmt.Fprintln(r.out, "Goodbye.")
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		r.history.Add(line)
		r.execute(line)
	}

	r.history.Save()
}

// loadSettings loads global settings and applies them (theme, etc.).
func (r *REPL) loadSettings() {
	s, err := config.LoadGlobal()
	if err != nil {
		// Non-fatal: use defaults.
		s = &config.GlobalSettings{}
	}
	r.settings = s

	// Apply theme from settings.
	if s.Theme != "" {
		_ = cli.SetTheme(s.Theme) // ignore error, keep default
	}
}

// autoDetectProject checks if there's a single .human file in the current
// directory and loads it automatically.
func (r *REPL) autoDetectProject() {
	matches, _ := filepath.Glob("*.human")
	// Filter out directories (e.g. .human/)
	var files []string
	for _, m := range matches {
		info, err := os.Stat(m)
		if err == nil && !info.IsDir() {
			files = append(files, m)
		}
	}
	if len(files) == 1 {
		r.setProject(files[0])
	}
}

// setProject sets the loaded project file and derives the project name.
func (r *REPL) setProject(file string) {
	r.projectFile = file
	base := filepath.Base(file)
	r.projectName = strings.TrimSuffix(base, filepath.Ext(base))
}

// printBanner displays the branded HUMAN_ startup banner.
func (r *REPL) printBanner() {
	animate := r.settings.AnimateEnabled()
	firstRun := !r.settings.FirstRunDone

	info := &cli.BannerInfo{
		ProjectFile: r.projectFile,
		ProjectName: r.projectName,
		FirstRun:    firstRun,
	}

	// Try to determine LLM status from project config.
	cwd, err := os.Getwd()
	if err == nil {
		if cfg, err := config.Load(cwd); err == nil && cfg.LLM != nil {
			info.LLMStatus = fmt.Sprintf("%s (%s)", cfg.LLM.Provider, cfg.LLM.Model)
		}
	}

	cli.PrintBanner(r.out, r.version, animate, info)

	// Mark first run as done.
	if firstRun {
		r.settings.FirstRunDone = true
		_ = config.SaveGlobal(r.settings)
	}
}

// printPrompt displays the branded prompt with underscore: human_> or project_>
func (r *REPL) printPrompt() {
	name := "human"
	if r.projectName != "" {
		name = r.projectName
	}

	if cli.ColorEnabled {
		accent := cli.Accent(name + "_>")
		fmt.Fprintf(r.out, "%s ", accent)
	} else {
		fmt.Fprintf(r.out, "%s_> ", name)
	}
}

// execute dispatches a line of input to the appropriate command handler.
func (r *REPL) execute(line string) {
	if !strings.HasPrefix(line, "/") {
		fmt.Fprintln(r.out, "Commands start with /. Type /help for a list.")
		return
	}

	parts := strings.Fields(line)
	name := strings.ToLower(parts[0])
	args := parts[1:]

	// Resolve aliases
	if target, ok := r.aliases[name]; ok {
		name = target
	}

	cmd, ok := r.commands[name]
	if !ok {
		r.suggestCommand(name)
		return
	}

	cmd.Handler(r, args)
}

// suggestCommand shows an error and suggests the closest known command.
func (r *REPL) suggestCommand(name string) {
	fmt.Fprintf(r.errOut, "Unknown command: %s\n", name)

	candidates := make([]string, 0, len(r.commands))
	for k := range r.commands {
		candidates = append(candidates, k)
	}

	if closest := findClosest(name, candidates, 0.5); closest != "" {
		fmt.Fprintf(r.errOut, "Did you mean %s?\n", closest)
	} else {
		fmt.Fprintln(r.errOut, "Type /help for a list of commands.")
	}
}

// requireProject checks that a project file is loaded and prints an error if not.
// Returns true if a project is loaded.
func (r *REPL) requireProject() bool {
	if r.projectFile == "" {
		fmt.Fprintln(r.errOut, "No project loaded. Use /open <file.human> to load one.")
		return false
	}
	return true
}
