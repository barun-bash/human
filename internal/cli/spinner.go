package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Spinner displays an animated waiting indicator on a TTY.
// It cycles through frames and erases itself when stopped.
type Spinner struct {
	out       io.Writer
	message   string
	frames    []string
	mu        sync.Mutex
	stop      chan struct{}
	done      chan struct{}
	running   bool
	startTime time.Time
	showDelay time.Duration // delay before showing spinner (default 200ms)
	failed    bool         // true if Fail() was called
}

var defaultFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// NewSpinner creates a spinner that writes to the given writer.
// The message is displayed after the spinner character.
func NewSpinner(out io.Writer, message string) *Spinner {
	return &Spinner{
		out:       out,
		message:   message,
		frames:    defaultFrames,
		stop:      make(chan struct{}),
		done:      make(chan struct{}),
		showDelay: 200 * time.Millisecond,
	}
}

// SetMessage updates the spinner message while it's running.
func (s *Spinner) SetMessage(msg string) {
	s.mu.Lock()
	s.message = msg
	s.mu.Unlock()
}

// Fail stops the spinner and shows a red ✗ with the given message.
func (s *Spinner) Fail(msg string) {
	s.mu.Lock()
	s.failed = true
	s.mu.Unlock()

	s.Stop()

	if ColorEnabled {
		errColor := themeColor(RoleError, fallbackRed)
		fmt.Fprintf(s.out, "\r\033[K  %s✗ %s%s\n", errColor, msg, reset)
	} else {
		fmt.Fprintf(s.out, "\r\033[K  ✗ %s\n", msg)
	}
}

// Start begins the spinner animation in a goroutine.
// No-op if the writer is not a TTY or if already running.
func (s *Spinner) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return
	}

	s.startTime = time.Now()

	// Only animate on a real terminal.
	if f, ok := s.out.(*os.File); !ok || !isTerminal(f) {
		// Non-TTY: print a static message instead.
		fmt.Fprintf(s.out, "  %s\n", s.message)
		close(s.done)
		return
	}

	s.running = true
	go s.animate()
}

// Stop halts the spinner and clears the line.
func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		<-s.done // wait for non-TTY case
		return
	}
	s.running = false
	s.mu.Unlock()

	close(s.stop)
	<-s.done
}

func (s *Spinner) animate() {
	defer close(s.done)

	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	idx := 0
	shown := false

	for {
		select {
		case <-s.stop:
			if shown {
				// Clear the spinner line.
				fmt.Fprintf(s.out, "\r\033[K")
			}
			return
		case <-ticker.C:
			s.mu.Lock()
			elapsed := time.Since(s.startTime)
			msg := s.message
			s.mu.Unlock()

			// Delay: don't show spinner until showDelay has passed.
			if !shown && elapsed < s.showDelay {
				idx++
				continue
			}
			shown = true

			frame := s.frames[idx%len(s.frames)]
			suffix := ""

			// After 3s: show elapsed time in dim gray.
			if elapsed >= 3*time.Second {
				mutedColor := themeColor(RoleMuted, "\033[90m")
				if ColorEnabled {
					suffix = fmt.Sprintf(" %s(%.1fs)%s", mutedColor, elapsed.Seconds(), reset)
				} else {
					suffix = fmt.Sprintf(" (%.1fs)", elapsed.Seconds())
				}
			}

			// After 30s: append cancel hint.
			if elapsed >= 30*time.Second {
				mutedColor := themeColor(RoleMuted, "\033[90m")
				if ColorEnabled {
					suffix += fmt.Sprintf(" %sPress ESC to cancel%s", mutedColor, reset)
				} else {
					suffix += " Press ESC to cancel"
				}
			}

			color := themeColor(RoleInfo, fallbackCyan)
			if ColorEnabled {
				fmt.Fprintf(s.out, "\r\033[K  %s%s%s %s%s", color, frame, reset, msg, suffix)
			} else {
				fmt.Fprintf(s.out, "\r\033[K  %s %s%s", frame, msg, suffix)
			}
			idx++
		}
	}
}

// WithSpinner runs fn with an animated spinner. Shows nothing if fn completes in <200ms.
func WithSpinner(out io.Writer, message string, fn func() error) error {
	s := NewSpinner(out, message)
	s.Start()
	err := fn()
	if err != nil {
		s.Fail(err.Error())
		return err
	}
	s.Stop()
	return nil
}

// WithSpinnerCtx is like WithSpinner but cancellable via context.
func WithSpinnerCtx(ctx context.Context, out io.Writer, message string, fn func(ctx context.Context) error) error {
	s := NewSpinner(out, message)
	s.Start()
	err := fn(ctx)
	if ctx.Err() != nil {
		s.Fail("Cancelled.")
		return nil
	}
	if err != nil {
		s.Fail(err.Error())
		return err
	}
	s.Stop()
	return nil
}

// isTerminal checks if a file descriptor is a terminal.
func isTerminal(f *os.File) bool {
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}
