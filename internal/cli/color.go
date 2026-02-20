package cli

import (
	"fmt"
	"os"
)

// ANSI color codes
const (
	reset  = "\033[0m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	cyan   = "\033[36m"
)

// ColorEnabled controls whether ANSI color codes are emitted.
// It defaults to true if stdout is a terminal and NO_COLOR is not set.
var ColorEnabled = initColorEnabled()

func initColorEnabled() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	stat, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// Success formats a message with a green ✓ prefix.
func Success(msg string) string {
	if ColorEnabled {
		return fmt.Sprintf("%s✓ %s%s", green, msg, reset)
	}
	return "✓ " + msg
}

// Error formats a message with a red ✗ prefix.
func Error(msg string) string {
	if ColorEnabled {
		return fmt.Sprintf("%s✗ %s%s", red, msg, reset)
	}
	return "✗ " + msg
}

// Warn formats a message with a yellow ⚠ prefix.
func Warn(msg string) string {
	if ColorEnabled {
		return fmt.Sprintf("%s⚠ %s%s", yellow, msg, reset)
	}
	return "⚠ " + msg
}

// Info formats a message with cyan color (no prefix).
func Info(msg string) string {
	if ColorEnabled {
		return fmt.Sprintf("%s%s%s", cyan, msg, reset)
	}
	return msg
}
