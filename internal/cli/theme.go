package cli

import (
	"fmt"
	"strings"
)

// ColorRole identifies a semantic color in the theme.
type ColorRole int

const (
	RoleSuccess ColorRole = iota
	RoleError
	RoleWarn
	RoleInfo
	RoleAccent
	RoleHeading
	RoleMuted
	RolePrompt
)

// Theme maps color roles to ANSI escape sequences.
type Theme struct {
	Name   string
	Colors map[ColorRole]string
}

// Built-in themes using RGB true-color (24-bit) where possible.
// Brand colors from BRAND_GUIDELINES.md:
//
//	Accent:  #E85D3A (warm coral-red)
//	Success: #2D8C5A
//	Error:   #C43030
//	Warning: #D4940A
//	Muted:   #8C8C8C
var themes = map[string]*Theme{
	"default": {
		Name: "default",
		Colors: map[ColorRole]string{
			RoleSuccess: "\033[38;2;45;140;90m",  // #2D8C5A
			RoleError:   "\033[38;2;196;48;48m",  // #C43030
			RoleWarn:    "\033[38;2;212;148;10m",  // #D4940A
			RoleInfo:    "\033[36m",               // cyan (readable on both dark/light)
			RoleAccent:  "\033[38;2;232;93;58m",   // #E85D3A
			RoleHeading: "\033[1m",                // bold
			RoleMuted:   "\033[38;2;140;140;140m", // #8C8C8C
			RolePrompt:  "\033[38;2;232;93;58m",   // accent
		},
	},
	"dark": {
		Name: "dark",
		Colors: map[ColorRole]string{
			RoleSuccess: "\033[38;2;80;220;120m",
			RoleError:   "\033[38;2;255;80;80m",
			RoleWarn:    "\033[38;2;255;200;60m",
			RoleInfo:    "\033[38;2;100;180;220m",
			RoleAccent:  "\033[38;2;255;120;80m",
			RoleHeading: "\033[1;97m",
			RoleMuted:   "\033[38;2;120;120;120m",
			RolePrompt:  "\033[38;2;255;120;80m",
		},
	},
	"light": {
		Name: "light",
		Colors: map[ColorRole]string{
			RoleSuccess: "\033[38;2;30;100;60m",
			RoleError:   "\033[38;2;160;30;30m",
			RoleWarn:    "\033[38;2;160;110;0m",
			RoleInfo:    "\033[38;2;60;60;60m",
			RoleAccent:  "\033[38;2;200;70;40m",
			RoleHeading: "\033[1;30m",
			RoleMuted:   "\033[38;2;140;140;140m",
			RolePrompt:  "\033[38;2;200;70;40m",
		},
	},
	"minimal": {
		Name: "minimal",
		Colors: map[ColorRole]string{
			RoleSuccess: "",
			RoleError:   "",
			RoleWarn:    "",
			RoleInfo:    "",
			RoleAccent:  "",
			RoleHeading: "",
			RoleMuted:   "",
			RolePrompt:  "",
		},
	},
	"ocean": {
		Name: "ocean",
		Colors: map[ColorRole]string{
			RoleSuccess: "\033[38;2;80;200;180m",
			RoleError:   "\033[38;2;255;100;100m",
			RoleWarn:    "\033[38;2;255;200;100m",
			RoleInfo:    "\033[38;2;100;180;220m",
			RoleAccent:  "\033[38;2;60;150;255m",
			RoleHeading: "\033[1;38;2;60;150;255m",
			RoleMuted:   "\033[38;2;120;150;170m",
			RolePrompt:  "\033[38;2;60;150;255m",
		},
	},
	"forest": {
		Name: "forest",
		Colors: map[ColorRole]string{
			RoleSuccess: "\033[38;2;60;180;80m",
			RoleError:   "\033[38;2;220;60;60m",
			RoleWarn:    "\033[38;2;200;170;40m",
			RoleInfo:    "\033[38;2;120;160;100m",
			RoleAccent:  "\033[38;2;80;180;60m",
			RoleHeading: "\033[1;38;2;80;180;60m",
			RoleMuted:   "\033[38;2;120;140;110m",
			RolePrompt:  "\033[38;2;80;180;60m",
		},
	},
}

// currentTheme is the active theme.
var currentTheme = themes["default"]

// SetTheme changes the active theme. Returns an error if the name is unknown.
func SetTheme(name string) error {
	t, ok := themes[strings.ToLower(name)]
	if !ok {
		return fmt.Errorf("unknown theme %q — available: %s", name, strings.Join(ThemeNames(), ", "))
	}
	currentTheme = t
	return nil
}

// CurrentThemeName returns the name of the active theme.
func CurrentThemeName() string {
	return currentTheme.Name
}

// ThemeNames returns the list of available theme names in display order.
func ThemeNames() []string {
	return []string{"default", "dark", "light", "minimal", "ocean", "forest"}
}

// GetTheme returns the theme with the given name, or nil if not found.
func GetTheme(name string) *Theme {
	return themes[strings.ToLower(name)]
}

// Colorize wraps msg in the current theme's color for the given role.
func Colorize(role ColorRole, msg string) string {
	if !ColorEnabled {
		return msg
	}
	c := currentTheme.Colors[role]
	if c == "" {
		return msg
	}
	return c + msg + reset
}

// Accent formats text in the theme's accent color.
func Accent(msg string) string {
	return Colorize(RoleAccent, msg)
}

// Heading formats text in the theme's heading style.
func Heading(msg string) string {
	return Colorize(RoleHeading, msg)
}

// Muted formats text in the theme's muted color.
func Muted(msg string) string {
	return Colorize(RoleMuted, msg)
}

// ThemePreview returns a multi-line string showing color samples for a theme.
func ThemePreview(name string) string {
	t := themes[strings.ToLower(name)]
	if t == nil {
		return ""
	}

	if !ColorEnabled {
		return fmt.Sprintf("  %s (colors disabled)", name)
	}

	var b strings.Builder
	samples := []struct {
		role  ColorRole
		label string
	}{
		{RoleAccent, "accent"},
		{RoleSuccess, "success"},
		{RoleError, "error"},
		{RoleWarn, "warn"},
		{RoleInfo, "info"},
		{RoleMuted, "muted"},
	}

	for _, s := range samples {
		c := t.Colors[s.role]
		if c == "" {
			b.WriteString(fmt.Sprintf(" %s", s.label))
		} else {
			b.WriteString(fmt.Sprintf(" %s%s%s", c, s.label, reset))
		}
	}

	return b.String()
}
