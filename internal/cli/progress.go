package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// ProgressBox displays a bordered progress box with a progress bar,
// percentage, and task list showing ✓ (done), spinner (in progress), ○ (pending).
type ProgressBox struct {
	out    io.Writer
	title  string
	stages []string
	done   []bool
	active int // index of currently running stage (-1 if none)
	failed int // index of failed stage (-1 if none)
	mu     sync.Mutex
	tty    bool
	lines  int // number of lines drawn (for cursor rewind)

	// Spinner animation
	stop     chan struct{}
	stopped  chan struct{}
	spinIdx  int
}

// NewProgressBox creates a progress display.
// stages is the list of stage names that will be reported.
func NewProgressBox(out io.Writer, title string, stages []string) *ProgressBox {
	tty := false
	if f, ok := out.(*os.File); ok {
		tty = isTerminal(f)
	}

	return &ProgressBox{
		out:     out,
		title:   title,
		stages:  stages,
		done:    make([]bool, len(stages)),
		active:  -1,
		failed:  -1,
		tty:     tty,
		stop:    make(chan struct{}),
		stopped: make(chan struct{}),
	}
}

// Start begins the progress display. On a TTY, starts the spinner animation.
func (p *ProgressBox) Start() {
	if p.tty {
		// Hide cursor.
		fmt.Fprint(p.out, "\033[?25l")
		p.draw()
		go p.animate()
	}
}

// Update marks the given stage as the currently active stage.
// All stages before it are marked as done.
func (p *ProgressBox) Update(stageName string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	idx := -1
	for i, s := range p.stages {
		if s == stageName {
			idx = i
			break
		}
	}

	if idx < 0 {
		// Unknown stage — append dynamically.
		p.stages = append(p.stages, stageName)
		p.done = append(p.done, false)
		idx = len(p.stages) - 1
	}

	// Mark all previous stages as done.
	for i := 0; i < idx; i++ {
		p.done[i] = true
	}
	p.active = idx

	if p.tty {
		p.draw()
	} else {
		// Non-TTY: print a simple line.
		doneCount := 0
		for _, d := range p.done {
			if d {
				doneCount++
			}
		}
		fmt.Fprintf(p.out, "  [%d/%d] %s\n", doneCount+1, len(p.stages), stageName)
	}
}

// FailStage marks the named stage as failed and stops the progress display.
func (p *ProgressBox) FailStage(name string) {
	if p.tty {
		close(p.stop)
		<-p.stopped
	}

	p.mu.Lock()
	for i, s := range p.stages {
		if s == name {
			p.failed = i
			break
		}
	}
	p.active = -1
	p.mu.Unlock()

	if p.tty {
		p.draw()
		fmt.Fprint(p.out, "\033[?25h")
		fmt.Fprintln(p.out)
	}
}

// Finish marks all stages as done and draws the final state.
func (p *ProgressBox) Finish() {
	if p.tty {
		close(p.stop)
		<-p.stopped
	}

	p.mu.Lock()
	for i := range p.done {
		p.done[i] = true
	}
	p.active = -1
	p.mu.Unlock()

	if p.tty {
		p.draw()
		// Show cursor.
		fmt.Fprint(p.out, "\033[?25h")
		fmt.Fprintln(p.out)
	}
}

func (p *ProgressBox) animate() {
	defer close(p.stopped)

	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-p.stop:
			return
		case <-ticker.C:
			p.mu.Lock()
			p.spinIdx++
			p.draw()
			p.mu.Unlock()
		}
	}
}

func (p *ProgressBox) draw() {
	// Calculate progress.
	total := len(p.stages)
	doneCount := 0
	for _, d := range p.done {
		if d {
			doneCount++
		}
	}
	if p.active >= 0 {
		// Count active as half done for percentage.
		doneCount = p.active
	}

	pct := 0
	if total > 0 {
		pct = (doneCount * 100) / total
	}

	// Erase previous draw.
	if p.lines > 0 {
		fmt.Fprintf(p.out, "\033[%dA", p.lines)
		for i := 0; i < p.lines; i++ {
			fmt.Fprint(p.out, "\033[K\n")
		}
		fmt.Fprintf(p.out, "\033[%dA", p.lines)
	}

	width := 52
	infoColor := themeColor(RoleInfo, fallbackCyan)
	successColor := themeColor(RoleSuccess, fallbackGreen)
	mutedColor := themeColor(RoleMuted, "\033[90m")
	spinFrames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

	var lines int

	// Top border.
	titleLine := fmt.Sprintf("─ %s ", p.title)
	pad := width - len([]rune(titleLine)) - 2
	if pad < 0 {
		pad = 0
	}
	if ColorEnabled {
		fmt.Fprintf(p.out, "%s┌%s%s┐%s\n", mutedColor, titleLine, strings.Repeat("─", pad), reset)
	} else {
		fmt.Fprintf(p.out, "┌%s%s┐\n", titleLine, strings.Repeat("─", pad))
	}
	lines++

	// Progress bar line.
	barWidth := 20
	filled := (pct * barWidth) / 100
	empty := barWidth - filled
	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	barLine := fmt.Sprintf(" [%s] %d%%", bar, pct)
	barPad := width - len([]rune(barLine)) - 2
	if barPad < 0 {
		barPad = 0
	}
	if ColorEnabled {
		fmt.Fprintf(p.out, "%s│%s %s[%s%s%s%s%s] %d%%%s%s│%s\n",
			mutedColor, reset,
			"", infoColor, strings.Repeat("█", filled), mutedColor, strings.Repeat("░", empty), reset,
			pct, strings.Repeat(" ", barPad-1), mutedColor, reset)
	} else {
		fmt.Fprintf(p.out, "│%s%s│\n", barLine, strings.Repeat(" ", barPad))
	}
	lines++

	// Empty line.
	if ColorEnabled {
		fmt.Fprintf(p.out, "%s│%s%s%s│%s\n", mutedColor, reset, strings.Repeat(" ", width-2), mutedColor, reset)
	} else {
		fmt.Fprintf(p.out, "│%s│\n", strings.Repeat(" ", width-2))
	}
	lines++

	errorColor := themeColor(RoleError, fallbackRed)

	// Stage list.
	for i, stage := range p.stages {
		var marker string
		if i == p.failed {
			if ColorEnabled {
				marker = errorColor + "✗" + reset
			} else {
				marker = "✗"
			}
		} else if p.done[i] {
			if ColorEnabled {
				marker = successColor + "✓" + reset
			} else {
				marker = "✓"
			}
		} else if i == p.active {
			frame := spinFrames[p.spinIdx%len(spinFrames)]
			if ColorEnabled {
				marker = infoColor + frame + reset
			} else {
				marker = frame
			}
		} else {
			if ColorEnabled {
				marker = mutedColor + "○" + reset
			} else {
				marker = "○"
			}
		}

		// Truncate stage name if too long.
		maxStage := width - 7
		displayStage := stage
		if len([]rune(displayStage)) > maxStage {
			displayStage = string([]rune(displayStage)[:maxStage-3]) + "..."
		}

		// Calculate padding (using visible character widths, not ANSI).
		visibleLen := 3 + len([]rune(displayStage)) // " X stagename"
		padding := width - 2 - visibleLen
		if padding < 0 {
			padding = 0
		}

		if ColorEnabled {
			fmt.Fprintf(p.out, "%s│%s %s %s%s%s│%s\n", mutedColor, reset, marker, displayStage, strings.Repeat(" ", padding), mutedColor, reset)
		} else {
			fmt.Fprintf(p.out, "│ %s %s%s│\n", marker, displayStage, strings.Repeat(" ", padding))
		}
		lines++
	}

	// Bottom border.
	if ColorEnabled {
		fmt.Fprintf(p.out, "%s└%s┘%s\n", mutedColor, strings.Repeat("─", width-2), reset)
	} else {
		fmt.Fprintf(p.out, "└%s┘\n", strings.Repeat("─", width-2))
	}
	lines++

	p.lines = lines
}

// Step represents a named step to execute with a progress display.
type Step struct {
	Name string
	Fn   func() error
}

// WithSteps runs a sequence of steps with a progress box.
func WithSteps(out io.Writer, title string, steps []Step) error {
	names := make([]string, len(steps))
	for i, s := range steps {
		names[i] = s.Name
	}

	box := NewProgressBox(out, title, names)
	box.Start()

	for _, step := range steps {
		box.Update(step.Name)
		if err := step.Fn(); err != nil {
			box.FailStage(step.Name)
			return err
		}
	}

	box.Finish()
	return nil
}
