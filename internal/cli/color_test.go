package cli

import (
	"strings"
	"testing"
)

func TestSuccessWithColor(t *testing.T) {
	ColorEnabled = true
	defer func() { ColorEnabled = false }()

	got := Success("all good")
	if !strings.Contains(got, "\033[32m") {
		t.Error("expected green ANSI code")
	}
	if !strings.Contains(got, "✓ all good") {
		t.Error("expected ✓ prefix and message")
	}
	if !strings.HasSuffix(got, "\033[0m") {
		t.Error("expected reset at end")
	}
}

func TestSuccessWithoutColor(t *testing.T) {
	ColorEnabled = false

	got := Success("all good")
	if got != "✓ all good" {
		t.Errorf("got %q, want %q", got, "✓ all good")
	}
}

func TestErrorWithColor(t *testing.T) {
	ColorEnabled = true
	defer func() { ColorEnabled = false }()

	got := Error("failed")
	if !strings.Contains(got, "\033[31m") {
		t.Error("expected red ANSI code")
	}
	if !strings.Contains(got, "✗ failed") {
		t.Error("expected ✗ prefix and message")
	}
}

func TestErrorWithoutColor(t *testing.T) {
	ColorEnabled = false

	got := Error("failed")
	if got != "✗ failed" {
		t.Errorf("got %q, want %q", got, "✗ failed")
	}
}

func TestWarnWithColor(t *testing.T) {
	ColorEnabled = true
	defer func() { ColorEnabled = false }()

	got := Warn("careful")
	if !strings.Contains(got, "\033[33m") {
		t.Error("expected yellow ANSI code")
	}
	if !strings.Contains(got, "⚠ careful") {
		t.Error("expected ⚠ prefix and message")
	}
}

func TestWarnWithoutColor(t *testing.T) {
	ColorEnabled = false

	got := Warn("careful")
	if got != "⚠ careful" {
		t.Errorf("got %q, want %q", got, "⚠ careful")
	}
}

func TestInfoWithColor(t *testing.T) {
	ColorEnabled = true
	defer func() { ColorEnabled = false }()

	got := Info("note")
	if !strings.Contains(got, "\033[36m") {
		t.Error("expected cyan ANSI code")
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
