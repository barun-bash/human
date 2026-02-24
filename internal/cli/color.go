package cli

import (
	"fmt"
	"os"
)

// ANSI reset code, shared across the package.
const reset = "\033[0m"

// Fallback ANSI color codes used when a theme color is empty.
const (
	fallbackRed    = "\033[31m"
	fallbackGreen  = "\033[32m"
	fallbackYellow = "\033[33m"
	fallbackCyan   = "\033[36m"
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

// themeColor returns the current theme's color for the role, falling back
// to the provided default if the theme has no color set.
func themeColor(role ColorRole, fallback string) string {
	c := currentTheme.Colors[role]
	if c != "" {
		return c
	}
	return fallback
}

// Success formats a message with a green check prefix.
func Success(msg string) string {
	if ColorEnabled {
		return fmt.Sprintf("%s\u2713 %s%s", themeColor(RoleSuccess, fallbackGreen), msg, reset)
	}
	return "\u2713 " + msg
}

// Error formats a message with a red cross prefix.
func Error(msg string) string {
	if ColorEnabled {
		return fmt.Sprintf("%s\u2717 %s%s", themeColor(RoleError, fallbackRed), msg, reset)
	}
	return "\u2717 " + msg
}

// Warn formats a message with a yellow warning prefix.
func Warn(msg string) string {
	if ColorEnabled {
		return fmt.Sprintf("%s\u26a0 %s%s", themeColor(RoleWarn, fallbackYellow), msg, reset)
	}
	return "\u26a0 " + msg
}

// Info formats a message with the theme's info color (no prefix).
func Info(msg string) string {
	if ColorEnabled {
		return fmt.Sprintf("%s%s%s", themeColor(RoleInfo, fallbackCyan), msg, reset)
	}
	return msg
}
