package cli

import (
	"strings"
	"testing"
)

func TestSetThemeValid(t *testing.T) {
	for _, name := range ThemeNames() {
		if err := SetTheme(name); err != nil {
			t.Errorf("SetTheme(%q) returned error: %v", name, err)
		}
		if CurrentThemeName() != name {
			t.Errorf("after SetTheme(%q), CurrentThemeName() = %q", name, CurrentThemeName())
		}
	}
	// Reset to default.
	_ = SetTheme("default")
}

func TestSetThemeCaseInsensitive(t *testing.T) {
	if err := SetTheme("OCEAN"); err != nil {
		t.Errorf("SetTheme(OCEAN) failed: %v", err)
	}
	if CurrentThemeName() != "ocean" {
		t.Errorf("expected ocean, got %s", CurrentThemeName())
	}
	_ = SetTheme("default")
}

func TestSetThemeUnknown(t *testing.T) {
	err := SetTheme("neon")
	if err == nil {
		t.Fatal("expected error for unknown theme")
	}
	if !strings.Contains(err.Error(), "neon") {
		t.Errorf("error should mention theme name, got: %v", err)
	}
}

func TestThemeNamesNotEmpty(t *testing.T) {
	names := ThemeNames()
	if len(names) < 6 {
		t.Errorf("expected at least 6 themes, got %d", len(names))
	}
	if names[0] != "default" {
		t.Errorf("first theme should be default, got %q", names[0])
	}
}

func TestColorizeWithColor(t *testing.T) {
	ColorEnabled = true
	defer func() { ColorEnabled = false }()
	_ = SetTheme("default")

	got := Colorize(RoleAccent, "hello")
	if !strings.Contains(got, "hello") {
		t.Error("expected message in output")
	}
	if !strings.HasSuffix(got, reset) {
		t.Error("expected reset at end")
	}
	if !strings.HasPrefix(got, "\033[") {
		t.Error("expected ANSI escape at start")
	}
}

func TestColorizeWithoutColor(t *testing.T) {
	ColorEnabled = false

	got := Colorize(RoleAccent, "hello")
	if got != "hello" {
		t.Errorf("expected plain text, got %q", got)
	}
}

func TestColorizeMinimalTheme(t *testing.T) {
	ColorEnabled = true
	defer func() { ColorEnabled = false }()
	_ = SetTheme("minimal")
	defer func() { _ = SetTheme("default") }()

	got := Colorize(RoleAccent, "hello")
	if got != "hello" {
		t.Errorf("minimal theme should produce plain text, got %q", got)
	}
}

func TestAccentMutedHeading(t *testing.T) {
	ColorEnabled = true
	defer func() { ColorEnabled = false }()
	_ = SetTheme("default")

	if !strings.Contains(Accent("x"), "x") {
		t.Error("Accent should contain message")
	}
	if !strings.Contains(Muted("x"), "x") {
		t.Error("Muted should contain message")
	}
	if !strings.Contains(Heading("x"), "x") {
		t.Error("Heading should contain message")
	}
}

func TestThemePreviewDefault(t *testing.T) {
	ColorEnabled = true
	defer func() { ColorEnabled = false }()

	p := ThemePreview("default")
	if !strings.Contains(p, "accent") {
		t.Error("preview should contain 'accent'")
	}
	if !strings.Contains(p, "success") {
		t.Error("preview should contain 'success'")
	}
}

func TestThemePreviewUnknown(t *testing.T) {
	p := ThemePreview("nonexistent")
	if p != "" {
		t.Errorf("expected empty preview for unknown theme, got %q", p)
	}
}

func TestGetTheme(t *testing.T) {
	if GetTheme("default") == nil {
		t.Error("GetTheme(default) should not be nil")
	}
	if GetTheme("nonexistent") != nil {
		t.Error("GetTheme(nonexistent) should be nil")
	}
}

func TestAllThemesHaveAllRoles(t *testing.T) {
	roles := []ColorRole{RoleSuccess, RoleError, RoleWarn, RoleInfo, RoleAccent, RoleHeading, RoleMuted, RolePrompt}
	for _, name := range ThemeNames() {
		theme := GetTheme(name)
		if theme == nil {
			t.Errorf("theme %q not found", name)
			continue
		}
		for _, role := range roles {
			if _, ok := theme.Colors[role]; !ok {
				t.Errorf("theme %q missing role %d", name, role)
			}
		}
	}
}
