package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"golang.org/x/term"
)

// SetupSignalHandler creates a context that cancels on first Ctrl+C.
// Second Ctrl+C calls os.Exit(1). Returns the cancellable context.
func SetupSignalHandler() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		cancel()
		// Second signal: hard exit.
		<-sigCh
		os.Exit(1)
	}()

	return ctx, cancel
}

// RunCancellable runs fn with context cancelled on ESC or first Ctrl+C.
// Second Ctrl+C exits the process. Returns nil if cancelled gracefully.
func RunCancellable(ctx context.Context, out io.Writer, fn func(ctx context.Context) error) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Set up Ctrl+C handler.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	defer signal.Stop(sigCh)

	var secondSigOnce sync.Once
	go func() {
		select {
		case <-sigCh:
			cancel()
			// Second signal: hard exit.
			secondSigOnce.Do(func() {
				go func() {
					<-sigCh
					os.Exit(1)
				}()
			})
		case <-ctx.Done():
		}
	}()

	// Try ESC detection on stdin if it's a terminal.
	if f, ok := out.(*os.File); ok && isTerminal(f) {
		startESCDetection(cancel)
	}

	err := fn(ctx)
	if ctx.Err() != nil {
		return nil // graceful cancellation
	}
	return err
}

// startESCDetection puts stdin in raw mode and watches for ESC (\x1b).
// Calls cancel() when ESC is detected. Restores terminal state on return.
func startESCDetection(cancel context.CancelFunc) {
	stdinFd := int(os.Stdin.Fd())
	if !term.IsTerminal(stdinFd) {
		return
	}

	oldState, err := term.MakeRaw(stdinFd)
	if err != nil {
		return
	}

	go func() {
		defer term.Restore(stdinFd, oldState)

		buf := make([]byte, 1)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil || n == 0 {
				return
			}
			if buf[0] == 0x1b { // ESC
				cancel()
				return
			}
			if buf[0] == 0x03 { // Ctrl+C in raw mode
				cancel()
				return
			}
		}
	}()
}

// Cancelled prints a cancellation message to the writer.
func Cancelled(out io.Writer) {
	if ColorEnabled {
		errColor := themeColor(RoleError, fallbackRed)
		fmt.Fprintf(out, "%s✗ Cancelled.%s\n", errColor, reset)
	} else {
		fmt.Fprintln(out, "✗ Cancelled.")
	}
}
