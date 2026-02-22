package figma

import (
	"strings"
)

// ClassifyNode determines what UI component a single Figma node represents.
// Heuristic priority: name-based > instance-based > structure-based > type-based > fallback.
func ClassifyNode(node *FigmaNode) ComponentType {
	if node == nil {
		return ComponentUnknown
	}

	// 1. Name-based classification (highest priority)
	if ct := classifyByName(node); ct != ComponentUnknown {
		return ct
	}

	// 2. Instance-based: INSTANCE nodes check component name
	if node.Type == "INSTANCE" && node.ComponentName != "" {
		if ct := classifyByName(&FigmaNode{Name: node.ComponentName}); ct != ComponentUnknown {
			return ct
		}
	}

	// 3. Structure-based classification
	if ct := classifyByStructure(node); ct != ComponentUnknown {
		return ct
	}

	// 4. Type-based classification
	if ct := classifyByType(node); ct != ComponentUnknown {
		return ct
	}

	// 5. Fallback
	if len(node.Children) > 0 {
		return ComponentSection
	}
	return ComponentUnknown
}

// classifyByName checks if the node name contains known component keywords.
func classifyByName(node *FigmaNode) ComponentType {
	name := strings.ToLower(node.Name)

	// Check each pattern group
	patterns := []struct {
		keywords []string
		result   ComponentType
	}{
		{[]string{"button", "btn", "cta"}, ComponentButton},
		{[]string{"card"}, ComponentCard},
		{[]string{"form"}, ComponentForm},
		{[]string{"input", "field", "text area", "textarea", "textfield"}, ComponentInput},
		{[]string{"sidebar", "side bar", "drawer", "side nav", "sidenav"}, ComponentSidebar},
		{[]string{"nav", "header", "topbar", "top bar", "menubar", "menu bar"}, ComponentNavbar},
		{[]string{"hero", "banner", "jumbotron"}, ComponentHero},
		{[]string{"footer"}, ComponentFooter},
		{[]string{"list"}, ComponentList},
		{[]string{"table", "datagrid", "data grid"}, ComponentTable},
		{[]string{"modal", "dialog", "popup", "pop up"}, ComponentModal},
		{[]string{"avatar", "profile pic", "profile image"}, ComponentAvatar},
		{[]string{"badge", "tag", "chip", "label"}, ComponentBadge},
		{[]string{"icon"}, ComponentIcon},
	}

	for _, p := range patterns {
		for _, keyword := range p.keywords {
			if strings.Contains(name, keyword) {
				return p.result
			}
		}
	}
	return ComponentUnknown
}

// classifyByStructure examines the node's properties and children to determine type.
func classifyByStructure(node *FigmaNode) ComponentType {
	// Button: small frame with centered text, fill color, and border radius
	if isButtonStructure(node) {
		return ComponentButton
	}

	// Card: frame with shadow and mixed content children
	if isCardStructure(node) {
		return ComponentCard
	}

	// Form: vertical auto-layout containing input-like children
	if isFormStructure(node) {
		return ComponentForm
	}

	// Navbar: horizontal layout at top with text/link children
	if isNavbarStructure(node) {
		return ComponentNavbar
	}

	// List: container with similar repeating children
	if hasSimilarChildren(node) {
		return ComponentList
	}

	return ComponentUnknown
}

// isButtonStructure checks for button-like structure:
// small frame, text content, fill color, border radius.
func isButtonStructure(node *FigmaNode) bool {
	if node.Type != "FRAME" && node.Type != "COMPONENT" && node.Type != "INSTANCE" {
		return false
	}
	// Must be small (button-sized)
	if node.Width > 400 || node.Height > 80 {
		return false
	}
	// Must have fill
	hasFill := false
	for _, f := range node.Fills {
		if f.Type == "SOLID" && f.Visible {
			hasFill = true
			break
		}
	}
	if !hasFill {
		return false
	}
	// Must have border radius
	if node.CornerRadius < 2 {
		return false
	}
	// Must contain text
	hasText := false
	for _, child := range node.Children {
		if child.Type == "TEXT" {
			hasText = true
			break
		}
	}
	return hasText
}

// isCardStructure checks for card-like structure:
// frame with shadow effect and multiple different child types.
func isCardStructure(node *FigmaNode) bool {
	if node.Type != "FRAME" && node.Type != "COMPONENT" && node.Type != "INSTANCE" {
		return false
	}
	// Must have shadow
	hasShadow := false
	for _, e := range node.Effects {
		if e.Type == "DROP_SHADOW" && e.Visible {
			hasShadow = true
			break
		}
	}
	if !hasShadow {
		return false
	}
	// Must have mixed content (at least 2 non-decorative children)
	contentCount := 0
	for _, child := range node.Children {
		if !isDecorative(child) {
			contentCount++
		}
	}
	return contentCount >= 2
}

// isFormStructure checks for form-like structure:
// vertical auto-layout containing input elements.
func isFormStructure(node *FigmaNode) bool {
	if node.LayoutMode != "VERTICAL" {
		return false
	}
	if len(node.Children) < 2 {
		return false
	}
	// Count children that look like inputs
	inputCount := 0
	for _, child := range node.Children {
		name := strings.ToLower(child.Name)
		if strings.Contains(name, "input") || strings.Contains(name, "field") {
			inputCount++
			continue
		}
		// Frame children with border/stroke that look like input boxes
		if (child.Type == "FRAME" || child.Type == "INSTANCE") && len(child.Strokes) > 0 {
			inputCount++
		}
	}
	return inputCount >= 2
}

// isNavbarStructure checks for navbar-like structure:
// horizontal layout with text or link children, positioned at top area.
func isNavbarStructure(node *FigmaNode) bool {
	if node.LayoutMode != "HORIZONTAL" {
		return false
	}
	if len(node.Children) < 2 {
		return false
	}
	// Should be wide relative to height (navbar shape)
	if node.Width < 200 || node.Height > 120 {
		return false
	}
	// Should contain text children (links)
	textCount := 0
	for _, child := range node.Children {
		if child.Type == "TEXT" {
			textCount++
		}
	}
	return textCount >= 2
}

// classifyByType uses the Figma node type and properties for classification.
func classifyByType(node *FigmaNode) ComponentType {
	switch node.Type {
	case "TEXT":
		if node.Style != nil {
			if node.Style.FontSize >= 24 {
				return ComponentHeading
			}
		}
		return ComponentText

	case "VECTOR":
		if node.Width < 48 && node.Height < 48 {
			return ComponentIcon
		}

	case "FRAME", "COMPONENT":
		// Frame with image fill
		for _, fill := range node.Fills {
			if fill.Type == "IMAGE" {
				return ComponentImage
			}
		}
	}

	return ComponentUnknown
}

// ClassifyTree recursively classifies a node and all its children,
// filtering out decorative elements.
func ClassifyTree(node *FigmaNode) *ClassifiedNode {
	if node == nil || isDecorative(node) {
		return nil
	}

	classified := &ClassifiedNode{
		Node: node,
		Type: ClassifyNode(node),
		Text: extractTextContent(node),
	}

	for _, child := range node.Children {
		if childClassified := ClassifyTree(child); childClassified != nil {
			classified.Children = append(classified.Children, childClassified)
		}
	}

	return classified
}

// ClassifyPage classifies all top-level nodes on a Figma page.
func ClassifyPage(page *FigmaPage) *ClassifiedPage {
	if page == nil {
		return nil
	}

	result := &ClassifiedPage{Name: page.Name}
	for _, node := range page.Nodes {
		if classified := ClassifyTree(node); classified != nil {
			result.Nodes = append(result.Nodes, classified)
		}
	}
	return result
}
