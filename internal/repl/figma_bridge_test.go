package repl

import (
	"testing"
)

// ── parseFigmaURL tests ──

func TestParseFigmaURL_DesignURL(t *testing.T) {
	fileKey, nodeID, err := parseFigmaURL("https://www.figma.com/design/abc123/My-Design?node-id=1-42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fileKey != "abc123" {
		t.Errorf("fileKey = %q, want %q", fileKey, "abc123")
	}
	if nodeID != "1:42" {
		t.Errorf("nodeID = %q, want %q (dash should be converted to colon)", nodeID, "1:42")
	}
}

func TestParseFigmaURL_FileURL(t *testing.T) {
	fileKey, nodeID, err := parseFigmaURL("https://figma.com/file/xyz789/Some-File?node-id=10-200")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fileKey != "xyz789" {
		t.Errorf("fileKey = %q, want %q", fileKey, "xyz789")
	}
	if nodeID != "10:200" {
		t.Errorf("nodeID = %q, want %q", nodeID, "10:200")
	}
}

func TestParseFigmaURL_BoardURL(t *testing.T) {
	fileKey, _, err := parseFigmaURL("https://figma.com/board/boardKey123/My-Board")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fileKey != "boardKey123" {
		t.Errorf("fileKey = %q, want %q", fileKey, "boardKey123")
	}
}

func TestParseFigmaURL_BranchURL(t *testing.T) {
	fileKey, _, err := parseFigmaURL("https://figma.com/design/abc123/branch/branchKey456/My-Design")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Branch URLs use the branchKey as fileKey.
	if fileKey != "branchKey456" {
		t.Errorf("fileKey = %q, want %q (should use branchKey)", fileKey, "branchKey456")
	}
}

func TestParseFigmaURL_NoNodeID(t *testing.T) {
	fileKey, nodeID, err := parseFigmaURL("https://figma.com/design/abc123/My-Design")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fileKey != "abc123" {
		t.Errorf("fileKey = %q, want %q", fileKey, "abc123")
	}
	if nodeID != "" {
		t.Errorf("nodeID = %q, want empty string", nodeID)
	}
}

func TestParseFigmaURL_MultipleQueryParams(t *testing.T) {
	_, nodeID, err := parseFigmaURL("https://figma.com/design/abc123/My-Design?mode=dev&node-id=5-99&viewport=center")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nodeID != "5:99" {
		t.Errorf("nodeID = %q, want %q", nodeID, "5:99")
	}
}

func TestParseFigmaURL_NoProtocol(t *testing.T) {
	fileKey, _, err := parseFigmaURL("figma.com/design/abc123/My-Design")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fileKey != "abc123" {
		t.Errorf("fileKey = %q, want %q", fileKey, "abc123")
	}
}

func TestParseFigmaURL_HTTPProtocol(t *testing.T) {
	fileKey, _, err := parseFigmaURL("http://figma.com/design/abc123/My-Design")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fileKey != "abc123" {
		t.Errorf("fileKey = %q, want %q", fileKey, "abc123")
	}
}

func TestParseFigmaURL_InvalidTooShort(t *testing.T) {
	_, _, err := parseFigmaURL("https://figma.com/design")
	if err == nil {
		t.Error("expected error for URL with no fileKey, got nil")
	}
}

func TestParseFigmaURL_UnsupportedType(t *testing.T) {
	_, _, err := parseFigmaURL("https://figma.com/proto/abc123/My-Prototype")
	if err == nil {
		t.Error("expected error for unsupported URL type 'proto', got nil")
	}
}

func TestParseFigmaURL_CompletelyInvalid(t *testing.T) {
	_, _, err := parseFigmaURL("not-a-url")
	if err == nil {
		t.Error("expected error for invalid URL, got nil")
	}
}

// ── figmaResponseToFile tests ──

func TestFigmaResponseToFile_DocumentFormat(t *testing.T) {
	// Simulates get_metadata response with document.children pages.
	json := `{
		"name": "My App",
		"document": {
			"children": [
				{
					"name": "Login Page",
					"type": "CANVAS",
					"children": [
						{
							"id": "1:1",
							"name": "Header",
							"type": "FRAME",
							"absoluteBoundingBox": {"width": 1440, "height": 80}
						},
						{
							"id": "1:2",
							"name": "Login Form",
							"type": "FRAME",
							"absoluteBoundingBox": {"width": 400, "height": 300}
						}
					]
				},
				{
					"name": "Dashboard",
					"type": "CANVAS",
					"children": [
						{
							"id": "2:1",
							"name": "Sidebar",
							"type": "FRAME",
							"layoutMode": "VERTICAL",
							"absoluteBoundingBox": {"width": 240, "height": 900}
						}
					]
				}
			]
		}
	}`

	file, err := figmaResponseToFile("Fallback", json)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if file.Name != "My App" {
		t.Errorf("file.Name = %q, want %q", file.Name, "My App")
	}
	if len(file.Pages) != 2 {
		t.Fatalf("len(Pages) = %d, want 2", len(file.Pages))
	}
	if file.Pages[0].Name != "Login Page" {
		t.Errorf("Pages[0].Name = %q, want %q", file.Pages[0].Name, "Login Page")
	}
	if len(file.Pages[0].Nodes) != 2 {
		t.Errorf("len(Pages[0].Nodes) = %d, want 2", len(file.Pages[0].Nodes))
	}
	if file.Pages[1].Name != "Dashboard" {
		t.Errorf("Pages[1].Name = %q, want %q", file.Pages[1].Name, "Dashboard")
	}
	if len(file.Pages[1].Nodes) != 1 {
		t.Errorf("len(Pages[1].Nodes) = %d, want 1", len(file.Pages[1].Nodes))
	}

	// Verify node properties.
	header := file.Pages[0].Nodes[0]
	if header.ID != "1:1" || header.Name != "Header" || header.Type != "FRAME" {
		t.Errorf("header node = {%q, %q, %q}, want {1:1, Header, FRAME}", header.ID, header.Name, header.Type)
	}
	if header.Width != 1440 || header.Height != 80 {
		t.Errorf("header size = %gx%g, want 1440x80", header.Width, header.Height)
	}

	sidebar := file.Pages[1].Nodes[0]
	if sidebar.LayoutMode != "VERTICAL" {
		t.Errorf("sidebar.LayoutMode = %q, want %q", sidebar.LayoutMode, "VERTICAL")
	}
}

func TestFigmaResponseToFile_DocumentFallbackName(t *testing.T) {
	// When the JSON doesn't include a name, the fallback name should be used.
	json := `{
		"document": {
			"children": [
				{"name": "Page 1", "type": "CANVAS", "children": [
					{"id": "1:1", "name": "Frame", "type": "FRAME"}
				]}
			]
		}
	}`

	file, err := figmaResponseToFile("MyFallback", json)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if file.Name != "MyFallback" {
		t.Errorf("file.Name = %q, want %q (fallback)", file.Name, "MyFallback")
	}
}

func TestFigmaResponseToFile_FlatNodeList(t *testing.T) {
	// Simulates get_design_context response as an array of nodes.
	json := `[
		{
			"id": "10:1",
			"name": "Button",
			"type": "COMPONENT",
			"cornerRadius": 8,
			"absoluteBoundingBox": {"width": 120, "height": 40}
		},
		{
			"id": "10:2",
			"name": "Card",
			"type": "FRAME",
			"layoutMode": "VERTICAL",
			"itemSpacing": 16,
			"paddingTop": 24,
			"paddingBottom": 24,
			"paddingLeft": 16,
			"paddingRight": 16,
			"absoluteBoundingBox": {"width": 320, "height": 200}
		}
	]`

	file, err := figmaResponseToFile("DesignContext", json)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if file.Name != "DesignContext" {
		t.Errorf("file.Name = %q, want %q", file.Name, "DesignContext")
	}
	if len(file.Pages) != 1 {
		t.Fatalf("len(Pages) = %d, want 1", len(file.Pages))
	}
	if file.Pages[0].Name != "Design" {
		t.Errorf("Pages[0].Name = %q, want %q", file.Pages[0].Name, "Design")
	}
	if len(file.Pages[0].Nodes) != 2 {
		t.Fatalf("len(Nodes) = %d, want 2", len(file.Pages[0].Nodes))
	}

	btn := file.Pages[0].Nodes[0]
	if btn.CornerRadius != 8 {
		t.Errorf("Button.CornerRadius = %g, want 8", btn.CornerRadius)
	}

	card := file.Pages[0].Nodes[1]
	if card.ItemSpacing != 16 {
		t.Errorf("Card.ItemSpacing = %g, want 16", card.ItemSpacing)
	}
	if card.PaddingTop != 24 || card.PaddingLeft != 16 {
		t.Errorf("Card padding = {top:%g, left:%g}, want {24, 16}", card.PaddingTop, card.PaddingLeft)
	}
}

func TestFigmaResponseToFile_SingleNode(t *testing.T) {
	// Simulates a response with a single root node tree.
	json := `{
		"id": "0:1",
		"name": "App Frame",
		"type": "FRAME",
		"layoutMode": "HORIZONTAL",
		"absoluteBoundingBox": {"width": 1440, "height": 900},
		"children": [
			{
				"id": "0:2",
				"name": "Title",
				"type": "TEXT",
				"characters": "Hello World",
				"absoluteBoundingBox": {"width": 200, "height": 32},
				"style": {
					"fontFamily": "Inter",
					"fontSize": 24,
					"fontWeight": 700,
					"lineHeightPx": 32,
					"letterSpacing": -0.5,
					"textAlignHorizontal": "LEFT"
				}
			},
			{
				"id": "0:3",
				"name": "Button",
				"type": "RECTANGLE",
				"cornerRadius": 12,
				"opacity": 0.9,
				"componentId": "comp:btn-primary",
				"absoluteBoundingBox": {"width": 160, "height": 48}
			}
		]
	}`

	file, err := figmaResponseToFile("SingleNode", json)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(file.Pages) != 1 || len(file.Pages[0].Nodes) != 1 {
		t.Fatalf("expected 1 page with 1 root node, got %d pages", len(file.Pages))
	}

	root := file.Pages[0].Nodes[0]
	if root.Name != "App Frame" || root.LayoutMode != "HORIZONTAL" {
		t.Errorf("root = {%q, %q}, want {App Frame, HORIZONTAL}", root.Name, root.LayoutMode)
	}
	if len(root.Children) != 2 {
		t.Fatalf("root.Children = %d, want 2", len(root.Children))
	}

	// Verify text node with style.
	title := root.Children[0]
	if title.Characters != "Hello World" {
		t.Errorf("title.Characters = %q, want %q", title.Characters, "Hello World")
	}
	if title.Style == nil {
		t.Fatal("title.Style is nil, expected TextStyle")
	}
	if title.Style.FontFamily != "Inter" {
		t.Errorf("FontFamily = %q, want %q", title.Style.FontFamily, "Inter")
	}
	if title.Style.FontSize != 24 {
		t.Errorf("FontSize = %g, want 24", title.Style.FontSize)
	}
	if title.Style.FontWeight != 700 {
		t.Errorf("FontWeight = %g, want 700", title.Style.FontWeight)
	}
	if title.Style.LineHeight != 32 {
		t.Errorf("LineHeight = %g, want 32", title.Style.LineHeight)
	}
	if title.Style.LetterSpacing != -0.5 {
		t.Errorf("LetterSpacing = %g, want -0.5", title.Style.LetterSpacing)
	}
	if title.Style.TextAlign != "LEFT" {
		t.Errorf("TextAlign = %q, want %q", title.Style.TextAlign, "LEFT")
	}

	// Verify non-text node (no style).
	btn := root.Children[1]
	if btn.CornerRadius != 12 {
		t.Errorf("btn.CornerRadius = %g, want 12", btn.CornerRadius)
	}
	if btn.Opacity != 0.9 {
		t.Errorf("btn.Opacity = %g, want 0.9", btn.Opacity)
	}
	if btn.ComponentID != "comp:btn-primary" {
		t.Errorf("btn.ComponentID = %q, want %q", btn.ComponentID, "comp:btn-primary")
	}
	if btn.Style != nil {
		t.Errorf("btn.Style should be nil for non-text node, got %+v", btn.Style)
	}
}

func TestFigmaResponseToFile_InvalidJSON(t *testing.T) {
	_, err := figmaResponseToFile("Bad", "not json at all")
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestFigmaResponseToFile_EmptyObject(t *testing.T) {
	// An empty JSON object {} parses as a single node with all zero values.
	// This is expected — the parser doesn't validate node content.
	file, err := figmaResponseToFile("Empty", "{}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(file.Pages) != 1 || len(file.Pages[0].Nodes) != 1 {
		t.Errorf("expected 1 page with 1 empty node, got %d pages", len(file.Pages))
	}
	node := file.Pages[0].Nodes[0]
	if node.Name != "" || node.Type != "" {
		t.Errorf("expected empty node fields, got Name=%q Type=%q", node.Name, node.Type)
	}
}

func TestFigmaResponseToFile_EmptyArray(t *testing.T) {
	// Empty array should succeed but with a single page containing no nodes.
	// Actually, the code checks len(nodes) > 0, so empty array falls through.
	_, err := figmaResponseToFile("Empty", "[]")
	if err == nil {
		t.Error("expected error for empty array, got nil")
	}
}

func TestFigmaResponseToFile_NestedChildren(t *testing.T) {
	// Verify deep nesting works (3 levels).
	json := `{
		"id": "root",
		"name": "Root",
		"type": "FRAME",
		"children": [
			{
				"id": "child-1",
				"name": "Container",
				"type": "FRAME",
				"children": [
					{
						"id": "grandchild-1",
						"name": "Label",
						"type": "TEXT",
						"characters": "Nested"
					}
				]
			}
		]
	}`

	file, err := figmaResponseToFile("Nested", json)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	root := file.Pages[0].Nodes[0]
	if len(root.Children) != 1 {
		t.Fatalf("root.Children = %d, want 1", len(root.Children))
	}
	container := root.Children[0]
	if container.Name != "Container" {
		t.Errorf("container.Name = %q, want %q", container.Name, "Container")
	}
	if len(container.Children) != 1 {
		t.Fatalf("container.Children = %d, want 1", len(container.Children))
	}
	label := container.Children[0]
	if label.Name != "Label" || label.Characters != "Nested" {
		t.Errorf("label = {%q, %q}, want {Label, Nested}", label.Name, label.Characters)
	}
}
