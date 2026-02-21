package themes

import (
	"github.com/barun-bash/human/internal/ir"
)

// spacingValues maps spacing names to px values (small, medium, large).
var spacingValues = map[string][3]string{
	"compact":     {"4px", "8px", "12px"},
	"comfortable": {"8px", "16px", "24px"},
	"spacious":    {"12px", "24px", "36px"},
}

// borderRadiusValues maps border radius names to px values.
var borderRadiusValues = map[string]string{
	"sharp":   "0px",
	"smooth":  "6px",
	"rounded": "12px",
	"pill":    "9999px",
}

// systemDefaults holds default color tokens per design system.
var systemDefaults = map[string]map[string]string{
	"material": {
		"primary":    "#1976d2",
		"secondary":  "#9c27b0",
		"background": "#ffffff",
		"surface":    "#f5f5f5",
		"text":       "#212121",
		"error":      "#d32f2f",
	},
	"shadcn": {
		"primary":    "#0f172a",
		"secondary":  "#64748b",
		"background": "#ffffff",
		"surface":    "#f8fafc",
		"text":       "#0f172a",
		"error":      "#ef4444",
	},
	"ant": {
		"primary":    "#1677ff",
		"secondary":  "#722ed1",
		"background": "#ffffff",
		"surface":    "#f5f5f5",
		"text":       "#000000e0",
		"error":      "#ff4d4f",
	},
	"chakra": {
		"primary":    "#3182ce",
		"secondary":  "#805ad5",
		"background": "#ffffff",
		"surface":    "#f7fafc",
		"text":       "#1a202c",
		"error":      "#e53e3e",
	},
	"bootstrap": {
		"primary":    "#0d6efd",
		"secondary":  "#6c757d",
		"background": "#ffffff",
		"surface":    "#f8f9fa",
		"text":       "#212529",
		"error":      "#dc3545",
	},
	"tailwind": {
		"primary":    "#3b82f6",
		"secondary":  "#8b5cf6",
		"background": "#ffffff",
		"surface":    "#f9fafb",
		"text":       "#111827",
		"error":      "#ef4444",
	},
	"untitled": {
		"primary":    "#7f56d9",
		"secondary":  "#6941c6",
		"background": "#ffffff",
		"surface":    "#f9fafb",
		"text":       "#101828",
		"error":      "#f04438",
	},
}

// defaultNeutral is the fallback when no design system is specified.
var defaultNeutral = map[string]string{
	"primary":    "#3b82f6",
	"secondary":  "#8b5cf6",
	"background": "#ffffff",
	"surface":    "#f9fafb",
	"text":       "#111827",
	"error":      "#ef4444",
}

// DefaultTokens returns CSS custom property defaults for a design system.
func DefaultTokens(systemID string) map[string]string {
	if tokens, ok := systemDefaults[systemID]; ok {
		return tokens
	}
	return defaultNeutral
}

// MergeTokens combines user-provided colors/fonts with system defaults.
// User values take precedence over defaults.
func MergeTokens(systemID string, theme *ir.Theme) map[string]string {
	tokens := make(map[string]string)

	// Start with defaults
	defaults := DefaultTokens(systemID)
	for k, v := range defaults {
		tokens["--color-"+k] = v
	}

	// Override with user-provided values
	if theme != nil {
		for k, v := range theme.Colors {
			tokens["--color-"+k] = v
		}
		for k, v := range theme.Fonts {
			tokens["--font-"+k] = v
		}

		// Spacing tokens from user preference
		if theme.Spacing != "" {
			if vals, ok := spacingValues[theme.Spacing]; ok {
				tokens["--spacing-sm"] = vals[0]
				tokens["--spacing-md"] = vals[1]
				tokens["--spacing-lg"] = vals[2]
			}
		}

		// Border radius from user preference
		if theme.BorderRadius != "" {
			if val, ok := borderRadiusValues[theme.BorderRadius]; ok {
				tokens["--radius"] = val
			}
		}
	}

	// Always provide spacing and radius defaults if not already set
	if _, ok := tokens["--spacing-sm"]; !ok {
		tokens["--spacing-sm"] = "8px"
		tokens["--spacing-md"] = "16px"
		tokens["--spacing-lg"] = "24px"
	}
	if _, ok := tokens["--radius"]; !ok {
		tokens["--radius"] = "6px" // default smooth
	}

	return tokens
}
