// Package figma provides intelligence for mapping Figma design elements
// to Human language concepts. It classifies visual components, infers
// data models, and generates .human source files from Figma JSON.
package figma

// FigmaFile represents a complete Figma file with multiple pages.
type FigmaFile struct {
	Name  string
	Pages []*FigmaPage
}

// FigmaPage represents a single page (canvas) in a Figma file.
type FigmaPage struct {
	Name  string
	Nodes []*FigmaNode
}

// FigmaNode represents a node in the Figma document tree.
// This mirrors the Figma REST API response structure.
type FigmaNode struct {
	ID         string
	Name       string
	Type       string // FRAME, TEXT, RECTANGLE, VECTOR, GROUP, INSTANCE, COMPONENT, etc.
	Children   []*FigmaNode
	Characters string // text content for TEXT nodes

	// Layout properties
	LayoutMode    string  // HORIZONTAL, VERTICAL, NONE
	PrimaryAxis   string  // MIN, CENTER, MAX, SPACE_BETWEEN
	CounterAxis   string  // MIN, CENTER, MAX
	ItemSpacing   float64 // spacing between children
	PaddingLeft   float64
	PaddingRight  float64
	PaddingTop    float64
	PaddingBottom float64

	// Visual properties
	Fills        []Paint
	Strokes      []Paint
	Effects      []Effect
	CornerRadius float64
	Opacity      float64

	// Text properties
	Style *TextStyle

	// Size
	Width  float64
	Height float64

	// Component metadata
	ComponentID   string // for INSTANCE nodes, the component it references
	ComponentName string // resolved component name
}

// Paint represents a fill or stroke on a Figma node.
type Paint struct {
	Type    string // SOLID, GRADIENT_LINEAR, IMAGE, etc.
	Color   Color
	Visible bool
	Opacity float64
}

// Color represents an RGBA color with Figma's 0-1 float range.
type Color struct {
	R float64 // 0.0 to 1.0
	G float64
	B float64
	A float64
}

// Effect represents a visual effect (shadow, blur) on a node.
type Effect struct {
	Type    string  // DROP_SHADOW, INNER_SHADOW, LAYER_BLUR, BACKGROUND_BLUR
	Visible bool
	Radius  float64
	Color   Color
	OffsetX float64
	OffsetY float64
}

// TextStyle holds typography properties for TEXT nodes.
type TextStyle struct {
	FontFamily    string
	FontSize      float64
	FontWeight    float64
	LineHeight    float64
	LetterSpacing float64
	TextAlign     string // LEFT, CENTER, RIGHT, JUSTIFIED
}

// ComponentType identifies what kind of UI element a Figma node represents.
type ComponentType int

const (
	ComponentUnknown ComponentType = iota
	ComponentButton
	ComponentCard
	ComponentForm
	ComponentInput
	ComponentNavbar
	ComponentSidebar
	ComponentHero
	ComponentFooter
	ComponentList
	ComponentTable
	ComponentModal
	ComponentImage
	ComponentHeading
	ComponentText
	ComponentIcon
	ComponentBadge
	ComponentAvatar
	ComponentSection
)

// String returns the human-readable name for a ComponentType.
func (ct ComponentType) String() string {
	names := [...]string{
		"unknown", "button", "card", "form", "input", "navbar",
		"sidebar", "hero", "footer", "list", "table", "modal",
		"image", "heading", "text", "icon", "badge", "avatar", "section",
	}
	if int(ct) < len(names) {
		return names[ct]
	}
	return "unknown"
}

// ClassifiedNode is a FigmaNode annotated with its detected component type
// and recursively classified children.
type ClassifiedNode struct {
	Node     *FigmaNode
	Type     ComponentType
	Children []*ClassifiedNode
	Text     string // extracted text content from the node tree
}

// ClassifiedPage holds the classification results for an entire page.
type ClassifiedPage struct {
	Name  string
	Nodes []*ClassifiedNode
}

// InferredModel represents a data model extracted from design patterns.
type InferredModel struct {
	Name   string
	Fields []*InferredField
	Source string // where it was inferred from: "form", "card", "table"
}

// InferredField represents a single field within an inferred model.
type InferredField struct {
	Name string
	Type string // Human type: text, number, email, date, etc.
}

// GenerateConfig holds configuration for .human file generation.
type GenerateConfig struct {
	AppName    string // application name
	Platform   string // web, mobile, desktop, api
	Frontend   string // React, Vue, Angular, Svelte
	Backend    string // Node, Python, Go
	Database   string // PostgreSQL, MySQL, MongoDB
	DesignFile string // path to the design file
}

// extractedTheme holds visual theme properties extracted from Figma nodes.
type extractedTheme struct {
	PrimaryColor   string // hex color
	SecondaryColor string
	BodyFont       string
	HeadingFont    string
	BorderRadius   string // smooth, sharp, rounded
	Spacing        string // comfortable, compact, spacious
}
