package figma

import (
	"fmt"
	"strings"
)

// GenerateFigmaPrompt creates an LLM prompt for a complete Figma file,
// describing detected components, inferred models, and Human syntax reference.
func GenerateFigmaPrompt(file *FigmaFile) string {
	if file == nil || len(file.Pages) == 0 {
		return ""
	}

	var sections []string

	sections = append(sections, fmt.Sprintf("# Design Analysis: %s\n", file.Name))

	// Classify and analyze
	var classifiedPages []*ClassifiedPage
	for _, page := range file.Pages {
		cp := ClassifyPage(page)
		if cp != nil {
			classifiedPages = append(classifiedPages, cp)
		}
	}

	// Components per page
	sections = append(sections, "## Components Detected\n")
	for _, cp := range classifiedPages {
		sections = append(sections, fmt.Sprintf("### Page: %s", cp.Name))
		componentSummary := summarizeComponents(cp.Nodes, 0)
		if componentSummary != "" {
			sections = append(sections, componentSummary)
		} else {
			sections = append(sections, "- No components detected")
		}
		sections = append(sections, "")
	}

	// Inferred models
	models := InferModels(classifiedPages)
	if len(models) > 0 {
		sections = append(sections, "## Inferred Data Models\n")
		for _, m := range models {
			sections = append(sections, fmt.Sprintf("### %s (from %s)", m.Name, m.Source))
			for _, f := range m.Fields {
				sections = append(sections, fmt.Sprintf("- %s: %s", f.Name, f.Type))
			}
			sections = append(sections, "")
		}
	}

	// Theme
	theme := extractTheme(file)
	if theme != nil {
		sections = append(sections, "## Theme\n")
		if theme.PrimaryColor != "" {
			sections = append(sections, fmt.Sprintf("- Primary color: %s", theme.PrimaryColor))
		}
		if theme.BodyFont != "" {
			sections = append(sections, fmt.Sprintf("- Font: %s", theme.BodyFont))
		}
		if theme.BorderRadius != "" {
			sections = append(sections, fmt.Sprintf("- Border radius: %s", theme.BorderRadius))
		}
		sections = append(sections, "")
	}

	// Syntax reference
	sections = append(sections, humanSyntaxReference())

	// Instructions
	sections = append(sections, generateInstructions(file, models))

	return strings.Join(sections, "\n")
}

// GeneratePagePrompt creates an LLM prompt for a single Figma page.
func GeneratePagePrompt(page *FigmaPage) string {
	if page == nil || len(page.Nodes) == 0 {
		return ""
	}

	var sections []string

	sections = append(sections, fmt.Sprintf("# Design Analysis: %s Page\n", page.Name))

	cp := ClassifyPage(page)
	if cp != nil {
		sections = append(sections, "## Components Detected\n")
		summary := summarizeComponents(cp.Nodes, 0)
		if summary != "" {
			sections = append(sections, summary)
		}

		models := InferModels([]*ClassifiedPage{cp})
		if len(models) > 0 {
			sections = append(sections, "\n## Inferred Data Models\n")
			for _, m := range models {
				sections = append(sections, fmt.Sprintf("### %s", m.Name))
				for _, f := range m.Fields {
					sections = append(sections, fmt.Sprintf("- %s: %s", f.Name, f.Type))
				}
			}
		}
	}

	sections = append(sections, "\n"+humanSyntaxReference())
	sections = append(sections, fmt.Sprintf(
		"\n## Instructions\n\nGenerate a Human language `page %s:` block that matches this design.\nUse the syntax patterns above. Include display statements, interactions, and conditionals.\n",
		toPascalCase(page.Name)))

	return strings.Join(sections, "\n")
}

// summarizeComponents creates a bullet-point summary of detected components.
func summarizeComponents(nodes []*ClassifiedNode, depth int) string {
	var lines []string
	prefix := strings.Repeat("  ", depth)

	for _, node := range nodes {
		if node.Type == ComponentUnknown {
			continue
		}
		desc := fmt.Sprintf("%s- %s", prefix, node.Type.String())
		if node.Text != "" {
			// Truncate long text
			text := node.Text
			if len(text) > 60 {
				text = text[:57] + "..."
			}
			desc += fmt.Sprintf(": \"%s\"", text)
		}
		lines = append(lines, desc)

		if len(node.Children) > 0 {
			childSummary := summarizeComponents(node.Children, depth+1)
			if childSummary != "" {
				lines = append(lines, childSummary)
			}
		}
	}

	return strings.Join(lines, "\n")
}

// humanSyntaxReference returns a condensed Human language syntax reference for LLM context.
func humanSyntaxReference() string {
	return `## Human Language Syntax Reference

### App Declaration
` + "```" + `
app MyApp is a web application
` + "```" + `

### Theme
` + "```" + `
theme:
  primary color is #6C5CE7
  font is Inter for body and Poppins for headings
  border radius is smooth
  spacing is comfortable
` + "```" + `

### Data Models
` + "```" + `
data User:
  has a name which is text
  has an email which is unique email
  has an optional bio which is text
  has a role which is either "user" or "admin"
  has a created datetime
` + "```" + `

### Pages
` + "```" + `
page Dashboard:
  show a heading "Welcome"
  show a list of recent tasks
  each task shows its title, status, and date
  there is a search bar that filters tasks by title
  clicking a task navigates to TaskDetail
  if no tasks match, show "No tasks found"
` + "```" + `

### Components
` + "```" + `
component TaskCard:
  accepts task as Task
  show the task title in bold
  show the status as a colored badge
` + "```" + `

### APIs
` + "```" + `
api CreateTask:
  requires authentication
  accepts title, description, and status
  check that title is not empty
  create a Task with the given fields
  respond with the created task
` + "```" + `

### Display Statements
- show a heading "text"
- show a list of Models
- each model shows its field1, field2
- show a card with content
- show a table of Models showing col1, col2
- show a hero section with "heading" and "subtext"
- show a navigation bar
- show a sidebar navigation with Link1, Link2

### Input Elements
- there is a text input for "label"
- there is a search bar that filters data by field
- there is a form to create Model

### Interactions
- clicking "Button" navigates to Page
- clicking "Button" does action
`
}

// generateInstructions creates the final instruction block for the LLM.
func generateInstructions(file *FigmaFile, models []*InferredModel) string {
	var lines []string
	lines = append(lines, "## Instructions\n")
	lines = append(lines, "Generate a complete .human file that matches the detected design. Include:\n")
	lines = append(lines, "1. App declaration with appropriate name and platform")
	lines = append(lines, "2. Theme block matching the detected colors, fonts, and spacing")

	if len(models) > 0 {
		var names []string
		for _, m := range models {
			names = append(names, m.Name)
		}
		lines = append(lines, fmt.Sprintf("3. Data models: %s", strings.Join(names, ", ")))
	} else {
		lines = append(lines, "3. Data models inferred from forms, cards, and tables")
	}

	lines = append(lines, fmt.Sprintf("4. Page blocks for each of the %d pages", len(file.Pages)))
	lines = append(lines, "5. CRUD API stubs for each data model")
	lines = append(lines, "6. Build target specification")
	lines = append(lines, "\nUse the Human Language Syntax Reference above for correct syntax.")
	lines = append(lines, "Ensure all page content matches the component structure detected in the design.")

	return strings.Join(lines, "\n")
}
