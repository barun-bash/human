package cli

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Spinner displays an animated waiting indicator on a TTY.
// It cycles through frames and erases itself when stopped.
type Spinner struct {
	out     io.Writer
	message string
	frames  []string
	mu      sync.Mutex
	stop    chan struct{}
	done    chan struct{}
	running bool
}

var defaultFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// NewSpinner creates a spinner that writes to the given writer.
// The message is displayed after the spinner character.
func NewSpinner(out io.Writer, message string) *Spinner {
	return &Spinner{
		out:     out,
		message: message,
		frames:  defaultFrames,
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
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
	color := themeColor(RoleInfo, fallbackCyan)

	for {
		select {
		case <-s.stop:
			// Clear the spinner line.
			fmt.Fprintf(s.out, "\r\033[K")
			return
		case <-ticker.C:
			frame := s.frames[idx%len(s.frames)]
			if ColorEnabled {
				fmt.Fprintf(s.out, "\r  %s%s%s %s", color, frame, reset, s.message)
			} else {
				fmt.Fprintf(s.out, "\r  %s %s", frame, s.message)
			}
			idx++
		}
	}
}

// isTerminal checks if a file descriptor is a terminal.
func isTerminal(f *os.File) bool {
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}
