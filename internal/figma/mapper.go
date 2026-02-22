package figma

import (
	"fmt"
	"strings"
)

// MapToHuman converts a classified page into Human language page block statements.
func MapToHuman(page *ClassifiedPage, appName string) string {
	if page == nil || len(page.Nodes) == 0 {
		return ""
	}

	pageName := toPascalCase(page.Name)
	var lines []string
	lines = append(lines, fmt.Sprintf("page %s:", pageName))

	for _, node := range page.Nodes {
		stmts := mapNode(node, 1)
		lines = append(lines, stmts...)
	}

	return strings.Join(lines, "\n")
}

// mapNode converts a single classified node into Human language statements.
func mapNode(node *ClassifiedNode, depth int) []string {
	if node == nil {
		return nil
	}

	prefix := indent(depth)
	var lines []string

	switch node.Type {
	case ComponentNavbar:
		lines = append(lines, mapNavbar(node, prefix)...)
	case ComponentHero:
		lines = append(lines, mapHero(node, prefix)...)
	case ComponentCard:
		lines = append(lines, mapCard(node, prefix, depth)...)
	case ComponentForm:
		lines = append(lines, mapForm(node, prefix)...)
	case ComponentButton:
		lines = append(lines, mapButton(node, prefix))
	case ComponentTable:
		lines = append(lines, mapTable(node, prefix)...)
	case ComponentList:
		lines = append(lines, mapList(node, prefix, depth)...)
	case ComponentHeading:
		lines = append(lines, mapHeading(node, prefix))
	case ComponentText:
		lines = append(lines, mapText(node, prefix))
	case ComponentFooter:
		lines = append(lines, mapFooter(node, prefix)...)
	case ComponentSidebar:
		lines = append(lines, mapSidebar(node, prefix)...)
	case ComponentModal:
		lines = append(lines, mapModal(node, prefix)...)
	case ComponentImage:
		lines = append(lines, fmt.Sprintf("%sshow an image", prefix))
	case ComponentInput:
		lines = append(lines, mapInput(node, prefix))
	case ComponentSection:
		// Recurse into section children
		for _, child := range node.Children {
			lines = append(lines, mapNode(child, depth)...)
		}
	default:
		// For unknown nodes with children, recurse
		for _, child := range node.Children {
			lines = append(lines, mapNode(child, depth)...)
		}
	}

	return lines
}

// mapNavbar extracts navigation links from a navbar component.
func mapNavbar(node *ClassifiedNode, prefix string) []string {
	var lines []string
	var linkTexts []string

	for _, child := range node.Children {
		if child.Text != "" && child.Type != ComponentImage && child.Type != ComponentIcon {
			linkTexts = append(linkTexts, child.Text)
		}
	}

	if len(linkTexts) == 0 {
		lines = append(lines, fmt.Sprintf("%sshow a navigation bar", prefix))
		return lines
	}

	lines = append(lines, fmt.Sprintf("%sshow a navigation bar", prefix))
	for _, text := range linkTexts {
		pageName := toPascalCase(text)
		lines = append(lines, fmt.Sprintf("%sclicking \"%s\" navigates to %s", prefix, text, pageName))
	}

	return lines
}

// mapHero extracts heading, subtext, and CTA from a hero section.
func mapHero(node *ClassifiedNode, prefix string) []string {
	var lines []string
	var heading, subtext, buttonText string

	for _, child := range node.Children {
		switch child.Type {
		case ComponentHeading:
			if heading == "" {
				heading = child.Text
			}
		case ComponentText:
			if subtext == "" {
				subtext = child.Text
			}
		case ComponentButton:
			if buttonText == "" {
				buttonText = child.Text
			}
		}
	}

	if heading != "" && subtext != "" {
		lines = append(lines, fmt.Sprintf("%sshow a hero section with \"%s\" and \"%s\"", prefix, heading, subtext))
	} else if heading != "" {
		lines = append(lines, fmt.Sprintf("%sshow a hero section with \"%s\"", prefix, heading))
	} else {
		lines = append(lines, fmt.Sprintf("%sshow a hero section", prefix))
	}

	if buttonText != "" {
		lines = append(lines, fmt.Sprintf("%sclicking \"%s\" navigates to SignUp", prefix, buttonText))
	}

	return lines
}

// mapCard generates statements for a card component.
func mapCard(node *ClassifiedNode, prefix string, depth int) []string {
	var lines []string

	// Check if this card is inside a list context (parent has similar children)
	// In that case, the list handler will wrap it
	var contentParts []string
	for _, child := range node.Children {
		if child.Text != "" {
			contentParts = append(contentParts, child.Text)
		}
	}

	if len(contentParts) > 0 {
		lines = append(lines, fmt.Sprintf("%sshow a card with %s", prefix, strings.Join(contentParts, ", ")))
	} else {
		lines = append(lines, fmt.Sprintf("%sshow a card", prefix))
		for _, child := range node.Children {
			lines = append(lines, mapNode(child, depth+1)...)
		}
	}

	return lines
}

// mapForm generates statements for a form with its inputs.
func mapForm(node *ClassifiedNode, prefix string) []string {
	var lines []string

	modelName := guessModelName(node)
	if modelName != "" {
		lines = append(lines, fmt.Sprintf("%sthere is a form to create %s", prefix, modelName))
	} else {
		lines = append(lines, fmt.Sprintf("%sthere is a form", prefix))
	}

	for _, child := range node.Children {
		if child.Type == ComponentInput || isInputLike(child) {
			lines = append(lines, mapInput(child, prefix))
		} else if child.Type == ComponentButton {
			lines = append(lines, mapButton(child, prefix))
		}
	}

	return lines
}

// mapButton generates a statement for a button.
func mapButton(node *ClassifiedNode, prefix string) string {
	text := node.Text
	if text == "" {
		text = "Submit"
	}

	// Infer action from button text
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "sign up") || strings.Contains(lower, "register"):
		return fmt.Sprintf("%sclicking \"%s\" navigates to SignUp", prefix, text)
	case strings.Contains(lower, "log in") || strings.Contains(lower, "sign in"):
		return fmt.Sprintf("%sclicking \"%s\" navigates to Login", prefix, text)
	case strings.Contains(lower, "submit") || strings.Contains(lower, "save") || strings.Contains(lower, "create"):
		return fmt.Sprintf("%sclicking \"%s\" does submit the form", prefix, text)
	case strings.Contains(lower, "delete") || strings.Contains(lower, "remove"):
		return fmt.Sprintf("%sclicking \"%s\" does delete the item", prefix, text)
	default:
		pageName := toPascalCase(text)
		return fmt.Sprintf("%sclicking \"%s\" navigates to %s", prefix, text, pageName)
	}
}

// mapTable generates statements for a table component.
func mapTable(node *ClassifiedNode, prefix string) []string {
	var lines []string

	modelName := guessModelName(node)
	var columns []string

	// Extract column names from first row
	if len(node.Children) > 0 {
		for _, cell := range node.Children[0].Children {
			if cell.Text != "" {
				columns = append(columns, strings.ToLower(cell.Text))
			}
		}
	}

	if modelName != "" && len(columns) > 0 {
		lines = append(lines, fmt.Sprintf("%sshow a table of %ss showing %s",
			prefix, modelName, strings.Join(columns, ", ")))
	} else if modelName != "" {
		lines = append(lines, fmt.Sprintf("%sshow a table of %ss", prefix, modelName))
	} else {
		lines = append(lines, fmt.Sprintf("%sshow a table", prefix))
	}

	return lines
}

// mapList generates statements for a list component.
func mapList(node *ClassifiedNode, prefix string, depth int) []string {
	var lines []string

	// Try to identify what's being listed
	modelName := guessModelName(node)
	if modelName != "" {
		lines = append(lines, fmt.Sprintf("%sshow a list of %ss", prefix, modelName))

		// If children are cards, describe what each item shows
		if len(node.Children) > 0 {
			first := node.Children[0]
			var fieldNames []string
			for _, child := range first.Children {
				if child.Text != "" {
					fieldNames = append(fieldNames, strings.ToLower(child.Text))
				}
			}
			if len(fieldNames) > 0 {
				lowerModel := strings.ToLower(modelName)
				lines = append(lines, fmt.Sprintf("%seach %s shows its %s",
					prefix, lowerModel, strings.Join(fieldNames, ", ")))
			}
		}
	} else {
		lines = append(lines, fmt.Sprintf("%sshow a list of items", prefix))
	}

	return lines
}

// mapHeading generates a statement for a heading.
func mapHeading(node *ClassifiedNode, prefix string) string {
	if node.Text != "" {
		return fmt.Sprintf("%sshow a heading \"%s\"", prefix, node.Text)
	}
	return fmt.Sprintf("%sshow a heading", prefix)
}

// mapText generates a statement for text content.
func mapText(node *ClassifiedNode, prefix string) string {
	if node.Text != "" {
		return fmt.Sprintf("%sshow \"%s\"", prefix, node.Text)
	}
	return ""
}

// mapFooter generates statements for a footer component.
func mapFooter(node *ClassifiedNode, prefix string) []string {
	var lines []string
	var parts []string

	for _, child := range node.Children {
		if child.Text != "" {
			parts = append(parts, fmt.Sprintf("\"%s\"", child.Text))
		}
	}

	if len(parts) > 0 {
		lines = append(lines, fmt.Sprintf("%sshow a footer with %s", prefix, strings.Join(parts, " and ")))
	} else {
		lines = append(lines, fmt.Sprintf("%sshow a footer", prefix))
	}

	return lines
}

// mapSidebar generates statements for a sidebar navigation.
func mapSidebar(node *ClassifiedNode, prefix string) []string {
	var lines []string
	var linkTexts []string

	for _, child := range node.Children {
		if child.Text != "" {
			linkTexts = append(linkTexts, child.Text)
		}
	}

	if len(linkTexts) > 0 {
		lines = append(lines, fmt.Sprintf("%sshow a sidebar navigation with %s",
			prefix, strings.Join(linkTexts, ", ")))
	} else {
		lines = append(lines, fmt.Sprintf("%sshow a sidebar navigation", prefix))
	}

	return lines
}

// mapModal generates statements for a modal dialog.
func mapModal(node *ClassifiedNode, prefix string) []string {
	var lines []string
	modalName := node.Text
	if modalName == "" {
		modalName = node.Node.Name
	}
	lines = append(lines, fmt.Sprintf("%sclicking trigger opens %s", prefix, modalName))
	return lines
}

// mapInput generates a statement for a form input.
func mapInput(node *ClassifiedNode, prefix string) string {
	fieldName := extractFieldName(node)
	if fieldName != "" {
		return fmt.Sprintf("%sthere is a text input for \"%s\"", prefix, strings.ReplaceAll(fieldName, "_", " "))
	}
	return fmt.Sprintf("%sthere is a text input", prefix)
}
