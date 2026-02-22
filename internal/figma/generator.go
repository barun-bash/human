package figma

import (
	"fmt"
	"strings"

	"github.com/barun-bash/human/internal/parser"
)

// GenerateHumanFile converts a Figma file into a complete .human source file.
// Pipeline: classify → infer models → extract theme → detect components →
// map pages → generate APIs → assemble → validate.
func GenerateHumanFile(file *FigmaFile, config *GenerateConfig) (string, error) {
	if file == nil || len(file.Pages) == 0 {
		return "", fmt.Errorf("figma file is empty or has no pages")
	}

	if config == nil {
		config = defaultConfig(file.Name)
	}

	// Step 1: Classify all pages
	var classifiedPages []*ClassifiedPage
	for _, page := range file.Pages {
		cp := ClassifyPage(page)
		if cp != nil {
			classifiedPages = append(classifiedPages, cp)
		}
	}

	// Step 2: Infer data models
	models := InferModels(classifiedPages)

	// Step 3: Extract theme from design
	theme := extractTheme(file)

	// Step 4: Detect reusable components
	components := detectReusableComponents(classifiedPages)

	// Step 5: Map pages to Human syntax
	var pageBlocks []string
	for _, cp := range classifiedPages {
		block := MapToHuman(cp, config.AppName)
		if block != "" {
			pageBlocks = append(pageBlocks, block)
		}
	}

	// Step 6: Generate CRUD API stubs per model
	var apiBlocks []string
	for _, model := range models {
		apiBlocks = append(apiBlocks, generateCRUDAPIs(model)...)
	}

	// Step 7: Assemble complete .human file
	output := assembleHumanFile(config, theme, pageBlocks, components, models, apiBlocks)

	// Step 8: Validate via parser (warn but still return)
	if _, err := parser.Parse(output); err != nil {
		return output, fmt.Errorf("generated .human file has syntax issues (output still usable): %w", err)
	}

	return output, nil
}

// defaultConfig returns a default configuration when none is provided.
func defaultConfig(fileName string) *GenerateConfig {
	name := toPascalCase(fileName)
	if name == "" {
		name = "MyApp"
	}
	return &GenerateConfig{
		AppName:  name,
		Platform: "web",
		Frontend: "React",
		Backend:  "Node",
		Database: "PostgreSQL",
	}
}

// extractTheme walks the Figma file to extract dominant visual properties.
func extractTheme(file *FigmaFile) *extractedTheme {
	var allNodes []*FigmaNode
	for _, page := range file.Pages {
		allNodes = append(allNodes, page.Nodes...)
	}

	theme := &extractedTheme{}

	// Extract dominant color as primary
	primary := dominantColor(allNodes)
	if primary != "" {
		theme.PrimaryColor = primary
	}

	// Extract dominant font
	font := dominantFont(allNodes)
	if font != "" {
		theme.BodyFont = font
		theme.HeadingFont = font
	}

	// Determine border radius style from most common corner radius
	theme.BorderRadius = inferBorderRadiusStyle(allNodes)
	theme.Spacing = "comfortable"

	return theme
}

// inferBorderRadiusStyle determines the border radius style from node corner radii.
func inferBorderRadiusStyle(nodes []*FigmaNode) string {
	var total, count float64
	walkRadius(nodes, &total, &count)
	if count == 0 {
		return "smooth"
	}
	avg := total / count
	switch {
	case avg < 2:
		return "sharp"
	case avg > 12:
		return "rounded"
	default:
		return "smooth"
	}
}

func walkRadius(nodes []*FigmaNode, total, count *float64) {
	for _, node := range nodes {
		if node.CornerRadius > 0 {
			*total += node.CornerRadius
			*count++
		}
		if len(node.Children) > 0 {
			walkRadius(node.Children, total, count)
		}
	}
}

// detectReusableComponents finds classified nodes that appear on multiple pages
// with the same structure, suggesting they should be extracted as components.
func detectReusableComponents(pages []*ClassifiedPage) []string {
	if len(pages) < 2 {
		return nil
	}

	// Track component signatures per page
	type occurrence struct {
		signature string
		text      string
		pages     map[string]bool
	}
	seen := make(map[string]*occurrence)

	for _, page := range pages {
		for _, node := range page.Nodes {
			sig := componentSignature(node)
			if sig == "" {
				continue
			}
			if occ, ok := seen[sig]; ok {
				occ.pages[page.Name] = true
			} else {
				seen[sig] = &occurrence{
					signature: sig,
					text:      node.Text,
					pages:     map[string]bool{page.Name: true},
				}
			}
		}
	}

	// Components appearing on 2+ pages are reusable
	var components []string
	for _, occ := range seen {
		if len(occ.pages) >= 2 {
			compName := toPascalCase(occ.text)
			if compName == "" {
				continue
			}
			comp := fmt.Sprintf("component %s:\n  show \"%s\"", compName, occ.text)
			components = append(components, comp)
		}
	}
	return components
}

// componentSignature creates a signature for detecting duplicate components.
func componentSignature(node *ClassifiedNode) string {
	if node == nil {
		return ""
	}
	var parts []string
	parts = append(parts, node.Type.String())
	for _, child := range node.Children {
		parts = append(parts, child.Type.String())
	}
	return strings.Join(parts, ":")
}

// generateCRUDAPIs produces Create, GetAll, GetByID, Update, Delete API blocks.
func generateCRUDAPIs(model *InferredModel) []string {
	name := model.Name

	var fieldNames []string
	for _, f := range model.Fields {
		fieldNames = append(fieldNames, f.Name)
	}
	fields := strings.Join(fieldNames, ", ")

	var apis []string

	// Create
	apis = append(apis, fmt.Sprintf(
		"api Create%s:\n  requires authentication\n  accepts %s\n  create a %s with the given fields\n  respond with the created %s",
		name, fields, name, strings.ToLower(name)))

	// GetAll
	apis = append(apis, fmt.Sprintf(
		"api GetAll%ss:\n  fetch all %ss\n  respond with the %ss",
		name, name, strings.ToLower(name)))

	// GetByID
	apis = append(apis, fmt.Sprintf(
		"api Get%s:\n  accepts id\n  fetch the %s with the given id\n  respond with the %s",
		name, name, strings.ToLower(name)))

	// Update
	apis = append(apis, fmt.Sprintf(
		"api Update%s:\n  requires authentication\n  accepts id, %s\n  update the %s with the given fields\n  respond with the updated %s",
		name, fields, name, strings.ToLower(name)))

	// Delete
	apis = append(apis, fmt.Sprintf(
		"api Delete%s:\n  requires authentication\n  accepts id\n  delete the %s with the given id\n  respond with success",
		name, name))

	return apis
}

// assembleHumanFile puts together all parts into a complete .human file.
func assembleHumanFile(
	config *GenerateConfig,
	theme *extractedTheme,
	pageBlocks []string,
	components []string,
	models []*InferredModel,
	apiBlocks []string,
) string {
	var sections []string

	// App declaration
	platform := config.Platform
	if platform == "" {
		platform = "web"
	}
	sections = append(sections, fmt.Sprintf("app %s is a %s application", config.AppName, platform))

	// Theme
	if theme != nil {
		themeBlock := generateThemeBlock(theme)
		if themeBlock != "" {
			sections = append(sections, themeBlock)
		}
	}

	// Design reference
	if config.DesignFile != "" {
		sections = append(sections, fmt.Sprintf("design dashboard from \"%s\"", config.DesignFile))
	}

	// Data models
	for _, model := range models {
		sections = append(sections, generateDataBlock(model))
	}

	// Pages
	sections = append(sections, pageBlocks...)

	// Components
	sections = append(sections, components...)

	// APIs
	sections = append(sections, apiBlocks...)

	// Build target
	sections = append(sections, generateBuildBlock(config))

	return strings.Join(sections, "\n\n") + "\n"
}

// generateThemeBlock creates a theme block from extracted theme properties.
func generateThemeBlock(theme *extractedTheme) string {
	var lines []string
	lines = append(lines, "theme:")

	if theme.PrimaryColor != "" {
		lines = append(lines, fmt.Sprintf("  primary color is %s", theme.PrimaryColor))
	}
	if theme.SecondaryColor != "" {
		lines = append(lines, fmt.Sprintf("  secondary color is %s", theme.SecondaryColor))
	}
	if theme.BodyFont != "" && theme.HeadingFont != "" && theme.BodyFont != theme.HeadingFont {
		lines = append(lines, fmt.Sprintf("  font is %s for body and %s for headings", theme.BodyFont, theme.HeadingFont))
	} else if theme.BodyFont != "" {
		lines = append(lines, fmt.Sprintf("  font is %s", theme.BodyFont))
	}
	if theme.BorderRadius != "" {
		lines = append(lines, fmt.Sprintf("  border radius is %s", theme.BorderRadius))
	}
	if theme.Spacing != "" {
		lines = append(lines, fmt.Sprintf("  spacing is %s", theme.Spacing))
	}

	if len(lines) == 1 {
		return ""
	}
	return strings.Join(lines, "\n")
}

// generateDataBlock creates a data model block.
func generateDataBlock(model *InferredModel) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("data %s:", model.Name))

	for _, field := range model.Fields {
		lines = append(lines, fmt.Sprintf("  has a %s which is %s", field.Name, field.Type))
	}

	return strings.Join(lines, "\n")
}

// generateBuildBlock creates a build target block.
func generateBuildBlock(config *GenerateConfig) string {
	var lines []string
	lines = append(lines, "build with:")

	if config.Frontend != "" {
		lines = append(lines, fmt.Sprintf("  frontend using %s", config.Frontend))
	}
	if config.Backend != "" {
		lines = append(lines, fmt.Sprintf("  backend using %s", config.Backend))
	}
	if config.Database != "" {
		lines = append(lines, fmt.Sprintf("  database using %s", config.Database))
	}

	return strings.Join(lines, "\n")
}
