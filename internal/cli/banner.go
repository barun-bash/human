package cli

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"
	"time"
)

// logoLetters stores each letter of HUMAN_ as 6-row ASCII block art.
// The underscore is a full-width block character — part of the brand identity.
var logoLetters = [6][6]string{
	// H
	{
		"██╗  ██╗",
		"██║  ██║",
		"███████║",
		"██╔══██║",
		"██║  ██║",
		"╚═╝  ╚═╝",
	},
	// U
	{
		"██╗   ██╗",
		"██║   ██║",
		"██║   ██║",
		"██║   ██║",
		"╚██████╔╝",
		" ╚═════╝ ",
	},
	// M
	{
		"███╗   ███╗",
		"████╗ ████║",
		"██╔████╔██║",
		"██║╚██╔╝██║",
		"██║ ╚═╝ ██║",
		"╚═╝     ╚═╝",
	},
	// A
	{
		" █████╗ ",
		"██╔══██╗",
		"███████║",
		"██╔══██║",
		"██║  ██║",
		"╚═╝  ╚═╝",
	},
	// N
	{
		"███╗   ██╗",
		"████╗  ██║",
		"██╔██╗ ██║",
		"██║╚██╗██║",
		"██║ ╚████║",
		"╚═╝  ╚═══╝",
	},
	// _ (full-width block underscore — the brand cursor)
	{
		"         ",
		"         ",
		"         ",
		"         ",
		"████████╗",
		"╚═══════╝",
	},
}

const logoRows = 6

// tips is the pool of startup tips shown randomly on launch.
var tips = []string{
	"Use /ask to generate a .human file from a description",
	"Run /suggest to get AI improvement ideas for your project",
	"Use /build --watch to auto-rebuild on file changes",
	"Run /check to validate your .human file without building",
	"Use /deploy --dry-run to preview deployment without executing",
	"Run /eject to export as standalone code with no Human dependency",
	"Use /review to open your .human file in your editor",
	"Run /examples to browse example projects",
	"Use /edit for interactive AI-assisted editing",
	"Run /audit to view the security report for your project",
	"Use /status to check project and Docker container status",
	"Try /theme list to see available color themes",
	"Run /config set plan_mode off to skip plan confirmation",
	"Use /new to scaffold a new Human project interactively",
}

// BannerInfo holds the data for the startup info block.
type BannerInfo struct {
	LLMStatus   string // e.g. "Connected to Anthropic (claude-sonnet-4.5)" or ""
	MCPStatus   string // e.g. "Figma, GitHub" or ""
	ProjectFile string // e.g. "examples/timekeeper/app.human" or ""
	ProjectName string // e.g. "timekeeper" or ""
	FirstRun    bool   // true on first launch
}

// PrintBanner renders the HUMAN_ logo and info block.
// When animate is true and the writer is a TTY, the logo types in letter by
// letter with a blinking underscore. Otherwise, the static logo is printed.
func PrintBanner(w io.Writer, version string, animate bool, info *BannerInfo) {
	if animate && isTTY(w) {
		printAnimatedLogo(w)
	} else {
		printStaticLogo(w)
	}
	fmt.Fprintln(w)
	printInfoBlock(w, version, info)
}

// buildLogoLines composes full logo lines showing the first numLetters letters.
func buildLogoLines(numLetters int) [logoRows]string {
	var lines [logoRows]string
	for row := 0; row < logoRows; row++ {
		parts := make([]string, numLetters)
		for i := 0; i < numLetters; i++ {
			parts[i] = logoLetters[i][row]
		}
		lines[row] = strings.Join(parts, " ")
	}
	return lines
}

func printAnimatedLogo(w io.Writer) {
	accent, rst := accentCodes()

	// Clear screen, cursor to top-left.
	fmt.Fprint(w, "\033[2J\033[H")

	// Reveal letters H-U-M-A-N one at a time (5 stages x 80ms = 400ms).
	for stage := 1; stage <= 5; stage++ {
		fmt.Fprint(w, "\033[H") // cursor home — overwrite in place
		lines := buildLogoLines(stage)
		for _, line := range lines {
			fmt.Fprintf(w, "  %s%s%s\033[K\n", accent, line, rst)
		}
		time.Sleep(80 * time.Millisecond)
	}

	// Blink the full-block underscore 2 times (2 x 500ms = 1000ms). Total ~1.4s.
	for i := 0; i < 2; i++ {
		// Show underscore (all 6 letters)
		printLogoFrame(w, 6, accent, rst)
		time.Sleep(250 * time.Millisecond)
		// Hide underscore (just 5 letters)
		printLogoFrame(w, 5, accent, rst)
		time.Sleep(250 * time.Millisecond)
	}

	// Final: underscore stays solid.
	printLogoFrame(w, 6, accent, rst)
}

// printLogoFrame reprints all logo rows from cursor home for n letters.
func printLogoFrame(w io.Writer, n int, accent, rst string) {
	fmt.Fprint(w, "\033[H")
	lines := buildLogoLines(n)
	for _, line := range lines {
		fmt.Fprintf(w, "  %s%s%s\033[K\n", accent, line, rst)
	}
}

func printStaticLogo(w io.Writer) {
	accent, rst := accentCodes()
	lines := buildLogoLines(6) // all 6 letters including block underscore
	for _, line := range lines {
		fmt.Fprintf(w, "  %s%s%s\n", accent, line, rst)
	}
}

func printInfoBlock(w io.Writer, version string, info *BannerInfo) {
	if info == nil {
		info = &BannerInfo{}
	}

	fmt.Fprintf(w, "  %s  v%s\n", Muted("Version:"), version)

	if info.LLMStatus != "" {
		fmt.Fprintf(w, "  %s      %s\n", Muted("LLM:"), info.LLMStatus)
	} else {
		fmt.Fprintf(w, "  %s      %s\n", Muted("LLM:"), Muted("Not configured. Run /connect"))
	}

	if info.MCPStatus != "" {
		fmt.Fprintf(w, "  %s      %s\n", Muted("MCP:"), info.MCPStatus)
	} else {
		fmt.Fprintf(w, "  %s      %s\n", Muted("MCP:"), Muted("0 servers connected"))
	}

	if info.ProjectFile != "" {
		fmt.Fprintf(w, "  %s  %s %s\n", Muted("Project:"), info.ProjectName, Muted("("+info.ProjectFile+")"))
	} else {
		fmt.Fprintf(w, "  %s  %s\n", Muted("Project:"), Muted("No project. Run /open or /new"))
	}

	fmt.Fprintf(w, "  %s      %s\n", Muted("Tip:"), tips[rand.Intn(len(tips))])
	fmt.Fprintln(w)

	if info.FirstRun {
		printFirstRunWelcome(w)
	}
}

func printFirstRunWelcome(w io.Writer) {
	fmt.Fprintln(w, Accent("  Welcome to Human! Let's get you started."))
	fmt.Fprintf(w, "  %s Run /connect to set up your AI provider\n", Accent("\u2192"))
	fmt.Fprintf(w, "  %s Run /new to create your first app\n", Accent("\u2192"))
	fmt.Fprintf(w, "  %s Run /examples to see what Human can build\n", Accent("\u2192"))
	fmt.Fprintln(w)
}

// accentCodes returns the accent color escape and reset, respecting ColorEnabled.
func accentCodes() (string, string) {
	if !ColorEnabled {
		return "", ""
	}
	c := currentTheme.Colors[RoleAccent]
	if c == "" {
		return "", ""
	}
	return c, reset
}

// isTTY returns true if w is a terminal.
func isTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// RandomTip returns a random startup tip.
func RandomTip() string {
	return tips[rand.Intn(len(tips))]
}
