package cli

import (
	"strings"
	"testing"
)

func TestSuccessWithColor(t *testing.T) {
	ColorEnabled = true
	defer func() { ColorEnabled = false }()
	_ = SetTheme("default")

	got := Success("all good")
	if !strings.Contains(got, "\033[") {
		t.Error("expected ANSI escape code")
	}
	if !strings.Contains(got, "\u2713 all good") {
		t.Error("expected check prefix and message")
	}
	if !strings.HasSuffix(got, reset) {
		t.Error("expected reset at end")
	}
}

func TestSuccessWithoutColor(t *testing.T) {
	ColorEnabled = false

	got := Success("all good")
	if got != "\u2713 all good" {
		t.Errorf("got %q, want %q", got, "\u2713 all good")
	}
}

func TestErrorWithColor(t *testing.T) {
	ColorEnabled = true
	defer func() { ColorEnabled = false }()
	_ = SetTheme("default")

	got := Error("failed")
	if !strings.Contains(got, "\033[") {
		t.Error("expected ANSI escape code")
	}
	if !strings.Contains(got, "\u2717 failed") {
		t.Error("expected cross prefix and message")
	}
}

func TestErrorWithoutColor(t *testing.T) {
	ColorEnabled = false

	got := Error("failed")
	if got != "\u2717 failed" {
		t.Errorf("got %q, want %q", got, "\u2717 failed")
	}
}

func TestWarnWithColor(t *testing.T) {
	ColorEnabled = true
	defer func() { ColorEnabled = false }()
	_ = SetTheme("default")

	got := Warn("careful")
	if !strings.Contains(got, "\033[") {
		t.Error("expected ANSI escape code")
	}
	if !strings.Contains(got, "\u26a0 careful") {
		t.Error("expected warning prefix and message")
	}
}

func TestWarnWithoutColor(t *testing.T) {
	ColorEnabled = false

	got := Warn("careful")
	if got != "\u26a0 careful" {
		t.Errorf("got %q, want %q", got, "\u26a0 careful")
	}
}

func TestInfoWithColor(t *testing.T) {
	ColorEnabled = true
	defer func() { ColorEnabled = false }()
	_ = SetTheme("default")

	got := Info("note")
	if !strings.Contains(got, "\033[") {
		t.Error("expected ANSI escape code")
	}
	if !strings.Contains(got, "note") {
		t.Error("expected message")
	}
}

func TestInfoWithoutColor(t *testing.T) {
	ColorEnabled = false

	got := Info("note")
	if got != "note" {
		t.Errorf("got %q, want %q", got, "note")
	}
}

func TestInitColorEnabledRespectsNO_COLOR(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	got := initColorEnabled()
	if got {
		t.Error("expected colors disabled when NO_COLOR is set")
	}
}

func TestInitColorEnabledNonTTY(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	// In test environments stdout is not a TTY, so colors should be off
	got := initColorEnabled()
	if got {
		t.Error("expected colors disabled when stdout is not a TTY")
	}
}

func TestSuccessWithMinimalTheme(t *testing.T) {
	ColorEnabled = true
	defer func() { ColorEnabled = false }()
	_ = SetTheme("minimal")
	defer func() { _ = SetTheme("default") }()

	got := Success("ok")
	// Minimal theme has empty colors, so fallback green is used.
	if !strings.Contains(got, "\u2713 ok") {
		t.Errorf("expected check prefix, got %q", got)
	}
}
