package figma

import (
	"fmt"
	"math"
	"strings"
	"unicode"
)

// ToHex converts a Figma Color (0-1 float range) to a hex string like "#RRGGBB".
func (c Color) ToHex() string {
	r := int(math.Round(c.R * 255))
	g := int(math.Round(c.G * 255))
	b := int(math.Round(c.B * 255))
	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}

// toPascalCase converts a string like "my profile page" to "MyProfilePage".
// Non-alphanumeric characters are treated as word boundaries.
func toPascalCase(s string) string {
	var result strings.Builder
	upper := true
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			upper = true
			continue
		}
		if upper {
			result.WriteRune(unicode.ToUpper(r))
			upper = false
		} else {
			result.WriteRune(unicode.ToLower(r))
		}
	}
	return result.String()
}

// singularize converts a simple plural English word to singular.
// Handles common patterns: "ies" → "y", "ses/xes/zes" → drop "es", "s" → drop.
func singularize(s string) string {
	if len(s) < 3 {
		return s
	}
	lower := strings.ToLower(s)

	// Don't singularize words that aren't plural
	if !strings.HasSuffix(lower, "s") {
		return s
	}

	// "ies" → "y" (e.g., Categories → Category)
	if strings.HasSuffix(lower, "ies") {
		return s[:len(s)-3] + "y"
	}
	// "ses", "xes", "zes" → drop "es"
	if strings.HasSuffix(lower, "ses") || strings.HasSuffix(lower, "xes") || strings.HasSuffix(lower, "zes") {
		return s[:len(s)-2]
	}
	// "sses" would be caught above; "ches", "shes" → drop "es"
	if strings.HasSuffix(lower, "ches") || strings.HasSuffix(lower, "shes") {
		return s[:len(s)-2]
	}
	// General: drop trailing "s"
	if strings.HasSuffix(lower, "ss") {
		return s // "class" stays "class"
	}
	return s[:len(s)-1]
}

// indent returns a string of n*2 spaces for .human file indentation.
func indent(n int) string {
	return strings.Repeat("  ", n)
}

// extractTextContent recursively collects all text content from a node tree.
func extractTextContent(node *FigmaNode) string {
	if node == nil {
		return ""
	}
	if node.Type == "TEXT" && node.Characters != "" {
		return strings.TrimSpace(node.Characters)
	}
	var parts []string
	for _, child := range node.Children {
		if text := extractTextContent(child); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, " ")
}

// dominantColor finds the most common solid fill color across a slice of nodes.
// Returns the hex string of the most frequent color, or empty string if none found.
func dominantColor(nodes []*FigmaNode) string {
	counts := make(map[string]int)
	walkColors(nodes, counts)
	if len(counts) == 0 {
		return ""
	}
	var best string
	var bestCount int
	for hex, count := range counts {
		if count > bestCount {
			best = hex
			bestCount = count
		}
	}
	return best
}

func walkColors(nodes []*FigmaNode, counts map[string]int) {
	for _, node := range nodes {
		for _, fill := range node.Fills {
			if fill.Type == "SOLID" && fill.Visible {
				hex := fill.Color.ToHex()
				// Skip black, white, and near-white/near-black (likely background/text)
				if hex != "#000000" && hex != "#FFFFFF" && hex != "#FEFEFE" && hex != "#010101" {
					counts[hex]++
				}
			}
		}
		if len(node.Children) > 0 {
			walkColors(node.Children, counts)
		}
	}
}

// dominantFont finds the most common font family across a slice of nodes.
func dominantFont(nodes []*FigmaNode) string {
	counts := make(map[string]int)
	walkFonts(nodes, counts)
	if len(counts) == 0 {
		return ""
	}
	var best string
	var bestCount int
	for font, count := range counts {
		if count > bestCount {
			best = font
			bestCount = count
		}
	}
	return best
}

func walkFonts(nodes []*FigmaNode, counts map[string]int) {
	for _, node := range nodes {
		if node.Style != nil && node.Style.FontFamily != "" {
			counts[node.Style.FontFamily]++
		}
		if len(node.Children) > 0 {
			walkFonts(node.Children, counts)
		}
	}
}

// isDecorative determines whether a node is a decorative element that should
// be filtered from semantic classification (background rects, divider lines,
// decorative vectors).
func isDecorative(node *FigmaNode) bool {
	if node == nil {
		return true
	}
	name := strings.ToLower(node.Name)

	// Named decorative elements
	if strings.Contains(name, "background") || strings.Contains(name, "divider") ||
		strings.Contains(name, "separator") || strings.Contains(name, "decoration") ||
		strings.Contains(name, "overlay") {
		return true
	}

	// Small rectangles with no children are likely decorative
	if node.Type == "RECTANGLE" && len(node.Children) == 0 {
		if node.Width < 20 || node.Height < 20 {
			return true
		}
		// Full-width thin rectangles are dividers
		if node.Height < 5 && node.Width > 100 {
			return true
		}
	}

	// Tiny vectors are icons or decorations
	if node.Type == "VECTOR" && node.Width < 10 && node.Height < 10 {
		return true
	}

	// Lines (very thin rects or vectors)
	if node.Type == "LINE" {
		return true
	}

	return false
}

// hasSimilarChildren checks whether a node's children share a similar structure,
// indicating a list or repeating pattern. Returns true if 3+ children share
// the same type composition.
func hasSimilarChildren(node *FigmaNode) bool {
	if node == nil || len(node.Children) < 3 {
		return false
	}

	// Get the structural signature of each child
	signatures := make(map[string]int)
	for _, child := range node.Children {
		if isDecorative(child) {
			continue
		}
		sig := structuralSignature(child)
		signatures[sig]++
	}

	// If any signature appears in 3+ children, it's a repeating pattern
	for _, count := range signatures {
		if count >= 3 {
			return true
		}
	}
	return false
}

// structuralSignature creates a string describing a node's child type composition.
// Two nodes with the same signature have similar structure.
func structuralSignature(node *FigmaNode) string {
	if len(node.Children) == 0 {
		return node.Type
	}
	var types []string
	for _, child := range node.Children {
		if !isDecorative(child) {
			types = append(types, child.Type)
		}
	}
	return strings.Join(types, "+")
}
