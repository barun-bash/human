package repl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/barun-bash/human/internal/cli"
	"github.com/barun-bash/human/internal/config"
	"github.com/barun-bash/human/internal/llm/prompts"
	"github.com/barun-bash/human/internal/mcp"
	"github.com/barun-bash/human/internal/readline"
)

// REPL is the interactive Human compiler shell.
type REPL struct {
	projectFile string
	projectName string
	version     string
	in          io.Reader
	out         io.Writer
	errOut      io.Writer
	scanner     *bufio.Scanner          // used for scanLine() sub-prompts
	rl          *readline.Instance      // nil when stdin is not a terminal
	history     *History
	commands    map[string]*Command
	aliases     map[string]string
	running         bool
	settings        *config.GlobalSettings
	lastSuggestions []prompts.Suggestion   // cached from last /suggest, cleared on source change
	mcpClients      map[string]*mcp.Client // live MCP server connections
	instructions    string                 // project instructions from HUMAN.md
	lastDir         string                 // previous working directory for /cd -

	// Update check state (populated by checkUpdateBackground).
	updateInfo *UpdateInfo
	updateMu   sync.Mutex
	updateDone chan struct{}
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
		version:    version,
		in:         os.Stdin,
		out:        os.Stdout,
		errOut:     os.Stderr,
		commands:   make(map[string]*Command),
		aliases:    make(map[string]string),
		mcpClients: make(map[string]*mcp.Client),
	}
	for _, opt := range opts {
		opt(r)
	}

	// Try to create a readline instance if stdin is a real file (terminal).
	if f, ok := r.in.(*os.File); ok {
		r.rl = readline.New(f, r.out)
	}

	r.scanner = bufio.NewScanner(r.in)
	r.history = NewHistory()
	r.registerCommands()
	r.loadSettings()
	return r
}

// Run starts the REPL loop: banner, prompt, read, dispatch, repeat.
func (r *REPL) Run() {
	r.autoDetectProject()
	r.autoConnectMCP()
	r.checkUpdateBackground()
	r.printBanner()
	r.showUpdateNotification()
	r.running = true

	if r.rl != nil && r.rl.IsTTY() {
		r.runReadline()
	} else {
		r.runScanner()
	}

	r.closeMCPClients()
	r.history.Save()
}

// runReadline runs the main loop using the readline instance (interactive terminal).
func (r *REPL) runReadline() {
	r.rl.SetCompleter(r.buildCompleter())

	for r.running {
		// Update readline state each iteration (prompt may change after /open).
		r.rl.SetPrompt(r.promptString())
		r.rl.SetHistory(r.history.Entries())

		line, err := r.rl.ReadLine()
		if err != nil {
			// EOF (Ctrl+D) or read error.
			fmt.Fprintln(r.out, "Goodbye.")
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		r.history.Add(line)
		r.execute(line)
	}
}

// runScanner runs the main loop using bufio.Scanner (non-TTY / test mode).
func (r *REPL) runScanner() {
	for r.running {
		r.printPrompt()
		if !r.scanner.Scan() {
			// EOF (Ctrl+D)
			fmt.Fprintln(r.out)
			fmt.Fprintln(r.out, "Goodbye.")
			break
		}

		line := strings.TrimSpace(r.scanner.Text())
		if line == "" {
			continue
		}

		r.history.Add(line)
		r.execute(line)
	}
}

// scanLine reads one line from the shared scanner. Returns the trimmed line
// and false if EOF was reached. Command handlers should use this instead of
// creating their own scanners on r.in.
func (r *REPL) scanLine() (string, bool) {
	if !r.scanner.Scan() {
		return "", false
	}
	return strings.TrimSpace(r.scanner.Text()), true
}

// promptString returns the prompt string (with or without ANSI colors).
func (r *REPL) promptString() string {
	name := "human"
	if r.projectName != "" {
		name = r.projectName
	}

	if cli.ColorEnabled {
		return cli.Accent(name+"_>") + " "
	}
	return name + "_> "
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
// Also loads HUMAN.md from the project directory if it exists.
func (r *REPL) setProject(file string) {
	r.projectFile = file
	base := filepath.Base(file)
	r.projectName = strings.TrimSuffix(base, filepath.Ext(base))
	r.loadInstructions()
}

// loadInstructions reads HUMAN.md from the project directory (same directory
// as the .human file). If found, the content is cached in r.instructions and
// passed as context to all LLM operations.
func (r *REPL) loadInstructions() {
	r.instructions = ""
	if r.projectFile == "" {
		return
	}

	dir := filepath.Dir(r.projectFile)
	path := filepath.Join(dir, "HUMAN.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return // file doesn't exist or unreadable — not an error
	}

	content := strings.TrimSpace(string(data))
	if content == "" {
		return
	}

	r.instructions = content
	fmt.Fprintln(r.out, cli.Muted("  Loaded project instructions from HUMAN.md"))
}

// instructionsPath returns the path to HUMAN.md for the current project.
// Returns "" if no project is loaded.
func (r *REPL) instructionsPath() string {
	if r.projectFile == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(r.projectFile), "HUMAN.md")
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

	// Try to determine LLM status from project config, then global config.
	cwd, err := os.Getwd()
	if err == nil {
		if cfg, err := config.Load(cwd); err == nil && cfg.LLM != nil {
			info.LLMStatus = fmt.Sprintf("%s (%s)", cfg.LLM.Provider, cfg.LLM.Model)
		}
	}
	if info.LLMStatus == "" {
		if gc, err := config.LoadGlobalConfig(); err == nil && gc.LLM != nil {
			info.LLMStatus = fmt.Sprintf("%s (%s)", gc.LLM.Provider, gc.LLM.Model)
		}
	}

	info.MCPStatus = r.mcpBannerStatus()

	cli.PrintBanner(r.out, r.version, animate, info)

	// Mark first run as done.
	if firstRun {
		r.settings.FirstRunDone = true
		_ = config.SaveGlobal(r.settings)
	}
}

// printPrompt displays the branded prompt (used in non-TTY/scanner mode).
func (r *REPL) printPrompt() {
	fmt.Fprint(r.out, r.promptString())
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

// clearSuggestions invalidates cached /suggest results. Called when the
// source file changes (via /edit, /undo, /open, /ask).
func (r *REPL) clearSuggestions() {
	r.lastSuggestions = nil
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
