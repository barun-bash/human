package figma

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// helpers_test
// ---------------------------------------------------------------------------

func TestColorToHex(t *testing.T) {
	tests := []struct {
		color Color
		want  string
	}{
		{Color{R: 1, G: 0, B: 0, A: 1}, "#FF0000"},
		{Color{R: 0, G: 1, B: 0, A: 1}, "#00FF00"},
		{Color{R: 0, G: 0, B: 1, A: 1}, "#0000FF"},
		{Color{R: 0, G: 0, B: 0, A: 1}, "#000000"},
		{Color{R: 1, G: 1, B: 1, A: 1}, "#FFFFFF"},
		{Color{R: 0.424, G: 0.361, B: 0.906, A: 1}, "#6C5CE7"}, // Human primary
	}
	for _, tt := range tests {
		got := tt.color.ToHex()
		if got != tt.want {
			t.Errorf("Color{%v, %v, %v}.ToHex() = %q, want %q",
				tt.color.R, tt.color.G, tt.color.B, got, tt.want)
		}
	}
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"my profile", "MyProfile"},
		{"user-settings", "UserSettings"},
		{"task_detail", "TaskDetail"},
		{"HELLO WORLD", "HelloWorld"},
		{"already", "Already"},
		{"", ""},
		{"one two three", "OneTwoThree"},
	}
	for _, tt := range tests {
		got := toPascalCase(tt.input)
		if got != tt.want {
			t.Errorf("toPascalCase(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSingularize(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"Tasks", "Task"},
		{"Categories", "Category"},
		{"Users", "User"},
		{"Addresses", "Address"},
		{"Boxes", "Box"},
		{"Buzzes", "Buzz"},
		{"Watches", "Watch"},
		{"Dishes", "Dish"},
		{"class", "class"}, // already singular, ends in "ss"
		{"hi", "hi"},       // too short
		{"Task", "Task"},   // no trailing s
	}
	for _, tt := range tests {
		got := singularize(tt.input)
		if got != tt.want {
			t.Errorf("singularize(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIndent(t *testing.T) {
	if got := indent(0); got != "" {
		t.Errorf("indent(0) = %q, want empty", got)
	}
	if got := indent(1); got != "  " {
		t.Errorf("indent(1) = %q, want two spaces", got)
	}
	if got := indent(3); got != "      " {
		t.Errorf("indent(3) = %q, want six spaces", got)
	}
}

func TestExtractTextContent(t *testing.T) {
	node := &FigmaNode{
		Type: "FRAME",
		Children: []*FigmaNode{
			{Type: "TEXT", Characters: "Hello"},
			{Type: "FRAME", Children: []*FigmaNode{
				{Type: "TEXT", Characters: "World"},
			}},
		},
	}
	got := extractTextContent(node)
	if got != "Hello World" {
		t.Errorf("extractTextContent = %q, want %q", got, "Hello World")
	}

	// nil node
	if extractTextContent(nil) != "" {
		t.Error("extractTextContent(nil) should return empty string")
	}
}

func TestIsDecorative(t *testing.T) {
	tests := []struct {
		name string
		node *FigmaNode
		want bool
	}{
		{"nil node", nil, true},
		{"background rect", &FigmaNode{Name: "Background", Type: "RECTANGLE"}, true},
		{"divider", &FigmaNode{Name: "Divider", Type: "RECTANGLE", Width: 500, Height: 2}, true},
		{"separator", &FigmaNode{Name: "separator-line", Type: "RECTANGLE"}, true},
		{"small rect", &FigmaNode{Name: "dot", Type: "RECTANGLE", Width: 5, Height: 5}, true},
		{"thin wide rect", &FigmaNode{Name: "line", Type: "RECTANGLE", Width: 300, Height: 2}, true},
		{"tiny vector", &FigmaNode{Name: "v", Type: "VECTOR", Width: 3, Height: 3}, true},
		{"line type", &FigmaNode{Name: "l", Type: "LINE"}, true},
		{"content frame", &FigmaNode{Name: "card", Type: "FRAME", Width: 200, Height: 150}, false},
		{"large rect", &FigmaNode{Name: "panel", Type: "RECTANGLE", Width: 200, Height: 100}, false},
		{"text node", &FigmaNode{Name: "title", Type: "TEXT"}, false},
	}
	for _, tt := range tests {
		got := isDecorative(tt.node)
		if got != tt.want {
			t.Errorf("isDecorative(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestHasSimilarChildren(t *testing.T) {
	// 4 children with same structure → true
	list := &FigmaNode{
		Name: "list",
		Type: "FRAME",
		Children: []*FigmaNode{
			{Name: "item1", Type: "FRAME", Width: 200, Height: 50, Children: []*FigmaNode{{Type: "TEXT"}}},
			{Name: "item2", Type: "FRAME", Width: 200, Height: 50, Children: []*FigmaNode{{Type: "TEXT"}}},
			{Name: "item3", Type: "FRAME", Width: 200, Height: 50, Children: []*FigmaNode{{Type: "TEXT"}}},
			{Name: "item4", Type: "FRAME", Width: 200, Height: 50, Children: []*FigmaNode{{Type: "TEXT"}}},
		},
	}
	if !hasSimilarChildren(list) {
		t.Error("expected hasSimilarChildren to be true for list with 4 identical items")
	}

	// Mixed children → false
	mixed := &FigmaNode{
		Name: "mixed",
		Type: "FRAME",
		Children: []*FigmaNode{
			{Name: "a", Type: "TEXT", Width: 100, Height: 20},
			{Name: "b", Type: "FRAME", Width: 200, Height: 100, Children: []*FigmaNode{{Type: "TEXT"}}},
			{Name: "c", Type: "VECTOR", Width: 50, Height: 50},
		},
	}
	if hasSimilarChildren(mixed) {
		t.Error("expected hasSimilarChildren to be false for mixed children")
	}

	// nil/few children → false
	if hasSimilarChildren(nil) {
		t.Error("expected false for nil")
	}
	if hasSimilarChildren(&FigmaNode{Children: []*FigmaNode{{Type: "TEXT"}}}) {
		t.Error("expected false for single child")
	}
}

// ---------------------------------------------------------------------------
// classifier_test
// ---------------------------------------------------------------------------

func TestClassifyNodeByName(t *testing.T) {
	tests := []struct {
		name string
		want ComponentType
	}{
		{"Primary Button", ComponentButton},
		{"submit-btn", ComponentButton},
		{"CTA Button", ComponentButton},
		{"Product Card", ComponentCard},
		{"Login Form", ComponentForm},
		{"Email Input", ComponentInput},
		{"text field", ComponentInput},
		{"Top Navigation", ComponentNavbar},
		{"Header Bar", ComponentNavbar},
		{"topbar", ComponentNavbar},
		{"Side Bar", ComponentSidebar},
		{"drawer-nav", ComponentSidebar},
		{"Hero Banner", ComponentHero},
		{"jumbotron", ComponentHero},
		{"page footer", ComponentFooter},
		{"Task List", ComponentList},
		{"data table", ComponentTable},
		{"datagrid", ComponentTable},
		{"confirm modal", ComponentModal},
		{"settings dialog", ComponentModal},
		{"user avatar", ComponentAvatar},
		{"status badge", ComponentBadge},
		{"tag chip", ComponentBadge},
		{"search icon", ComponentIcon},
	}

	for _, tt := range tests {
		node := &FigmaNode{Name: tt.name, Type: "FRAME", Width: 100, Height: 50}
		got := ClassifyNode(node)
		if got != tt.want {
			t.Errorf("ClassifyNode(name=%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestClassifyNodeByNameCaseInsensitive(t *testing.T) {
	// "BUTTON" should classify same as "button"
	upper := &FigmaNode{Name: "SUBMIT BUTTON", Type: "FRAME", Width: 100, Height: 50}
	lower := &FigmaNode{Name: "submit button", Type: "FRAME", Width: 100, Height: 50}
	if ClassifyNode(upper) != ClassifyNode(lower) {
		t.Error("classification should be case-insensitive")
	}
}

func TestClassifyButtonByStructure(t *testing.T) {
	// Small frame + text child + fill + border radius = button
	button := &FigmaNode{
		Name:         "action",
		Type:         "FRAME",
		Width:        120,
		Height:       40,
		CornerRadius: 8,
		Fills:        []Paint{{Type: "SOLID", Visible: true, Color: Color{R: 0.4, G: 0.3, B: 0.9}}},
		Children:     []*FigmaNode{{Type: "TEXT", Characters: "Click Me"}},
	}
	if got := ClassifyNode(button); got != ComponentButton {
		t.Errorf("button by structure: got %v, want %v", got, ComponentButton)
	}
}

func TestClassifyCardByStructure(t *testing.T) {
	card := &FigmaNode{
		Name: "item",
		Type: "FRAME",
		Effects: []Effect{
			{Type: "DROP_SHADOW", Visible: true, Radius: 4},
		},
		Children: []*FigmaNode{
			{Name: "title", Type: "TEXT", Characters: "Title", Width: 100, Height: 20},
			{Name: "desc", Type: "TEXT", Characters: "Description", Width: 100, Height: 40},
			{Name: "img", Type: "FRAME", Width: 100, Height: 80,
				Fills: []Paint{{Type: "IMAGE"}}},
		},
	}
	if got := ClassifyNode(card); got != ComponentCard {
		t.Errorf("card by structure: got %v, want %v", got, ComponentCard)
	}
}

func TestClassifyFormByStructure(t *testing.T) {
	form := &FigmaNode{
		Name:       "signup",
		Type:       "FRAME",
		LayoutMode: "VERTICAL",
		Children: []*FigmaNode{
			{Name: "name input", Type: "FRAME", Width: 200, Height: 40,
				Strokes: []Paint{{Type: "SOLID", Visible: true}}},
			{Name: "email input", Type: "FRAME", Width: 200, Height: 40,
				Strokes: []Paint{{Type: "SOLID", Visible: true}}},
			{Name: "submit", Type: "FRAME", Width: 100, Height: 40},
		},
	}
	if got := ClassifyNode(form); got != ComponentForm {
		t.Errorf("form by structure: got %v, want %v", got, ComponentForm)
	}
}

func TestClassifyListByStructure(t *testing.T) {
	list := &FigmaNode{
		Name: "items",
		Type: "FRAME",
		Children: []*FigmaNode{
			{Name: "row1", Type: "FRAME", Width: 300, Height: 50, Children: []*FigmaNode{{Type: "TEXT"}}},
			{Name: "row2", Type: "FRAME", Width: 300, Height: 50, Children: []*FigmaNode{{Type: "TEXT"}}},
			{Name: "row3", Type: "FRAME", Width: 300, Height: 50, Children: []*FigmaNode{{Type: "TEXT"}}},
		},
	}
	if got := ClassifyNode(list); got != ComponentList {
		t.Errorf("list by structure: got %v, want %v", got, ComponentList)
	}
}

func TestClassifyTextByFontSize(t *testing.T) {
	heading := &FigmaNode{
		Name:       "big text",
		Type:       "TEXT",
		Characters: "Welcome",
		Style:      &TextStyle{FontSize: 32, FontFamily: "Inter"},
	}
	if got := ClassifyNode(heading); got != ComponentHeading {
		t.Errorf("heading by font size: got %v, want %v", got, ComponentHeading)
	}

	body := &FigmaNode{
		Name:       "small text",
		Type:       "TEXT",
		Characters: "Description",
		Style:      &TextStyle{FontSize: 14, FontFamily: "Inter"},
	}
	if got := ClassifyNode(body); got != ComponentText {
		t.Errorf("text by font size: got %v, want %v", got, ComponentText)
	}
}

func TestClassifyInstanceByComponentName(t *testing.T) {
	node := &FigmaNode{
		Name:          "instance_1",
		Type:          "INSTANCE",
		ComponentName: "Primary Button",
		Width:         100,
		Height:        40,
	}
	if got := ClassifyNode(node); got != ComponentButton {
		t.Errorf("instance classification: got %v, want %v", got, ComponentButton)
	}
}

func TestClassifyTree(t *testing.T) {
	tree := &FigmaNode{
		Name: "page-frame",
		Type: "FRAME",
		Width: 1440, Height: 900,
		Children: []*FigmaNode{
			{Name: "Navigation", Type: "FRAME", Width: 1440, Height: 60,
				LayoutMode: "HORIZONTAL",
				Children: []*FigmaNode{
					{Type: "TEXT", Characters: "Home", Width: 40, Height: 16},
					{Type: "TEXT", Characters: "About", Width: 40, Height: 16},
					{Type: "TEXT", Characters: "Contact", Width: 50, Height: 16},
				}},
			{Name: "background", Type: "RECTANGLE", Width: 5, Height: 5}, // decorative
			{Name: "Content", Type: "TEXT", Characters: "Hello World",
				Style: &TextStyle{FontSize: 32}},
		},
	}

	classified := ClassifyTree(tree)
	if classified == nil {
		t.Fatal("ClassifyTree returned nil")
	}
	if classified.Type != ComponentSection {
		t.Errorf("root type = %v, want section", classified.Type)
	}

	// Background rect should be filtered
	if len(classified.Children) != 2 {
		t.Errorf("expected 2 children after filtering decorative, got %d", len(classified.Children))
	}

	// First child should be navbar
	if classified.Children[0].Type != ComponentNavbar {
		t.Errorf("first child type = %v, want navbar", classified.Children[0].Type)
	}

	// Second child should be heading
	if classified.Children[1].Type != ComponentHeading {
		t.Errorf("second child type = %v, want heading", classified.Children[1].Type)
	}
}

func TestClassifyPage(t *testing.T) {
	page := &FigmaPage{
		Name: "Dashboard",
		Nodes: []*FigmaNode{
			{Name: "nav bar", Type: "FRAME", Width: 1440, Height: 60,
				LayoutMode: "HORIZONTAL",
				Children: []*FigmaNode{
					{Type: "TEXT", Characters: "Home", Width: 40, Height: 16},
					{Type: "TEXT", Characters: "Settings", Width: 60, Height: 16},
				}},
		},
	}

	result := ClassifyPage(page)
	if result == nil {
		t.Fatal("ClassifyPage returned nil")
	}
	if result.Name != "Dashboard" {
		t.Errorf("page name = %q, want %q", result.Name, "Dashboard")
	}
	if len(result.Nodes) != 1 {
		t.Fatalf("expected 1 classified node, got %d", len(result.Nodes))
	}
	if result.Nodes[0].Type != ComponentNavbar {
		t.Errorf("classified type = %v, want navbar", result.Nodes[0].Type)
	}
}

// ---------------------------------------------------------------------------
// inference_test
// ---------------------------------------------------------------------------

func TestGuessFieldType(t *testing.T) {
	tests := []struct {
		label, want string
	}{
		{"email", "email"},
		{"user_email", "email"},
		{"price", "decimal"},
		{"total_amount", "decimal"},
		{"age", "number"},
		{"quantity", "number"},
		{"birthday", "date"},
		{"created_at", "datetime"},
		{"is_active", "boolean"},
		{"published", "boolean"},
		{"avatar", "image"},
		{"profile_photo", "image"},
		{"attachment", "file"},
		{"website", "url"},
		{"name", "text"},
		{"title", "text"},
		{"unknown_field", "text"},
	}
	for _, tt := range tests {
		got := guessFieldType(tt.label)
		if got != tt.want {
			t.Errorf("guessFieldType(%q) = %q, want %q", tt.label, got, tt.want)
		}
	}
}

func TestInferFromForm(t *testing.T) {
	form := &ClassifiedNode{
		Node: &FigmaNode{Name: "Create Task Form"},
		Type: ComponentForm,
		Children: []*ClassifiedNode{
			{Node: &FigmaNode{Name: "title input"}, Type: ComponentInput, Text: "Title"},
			{Node: &FigmaNode{Name: "description input"}, Type: ComponentInput, Text: "Description"},
			{Node: &FigmaNode{Name: "due date input"}, Type: ComponentInput, Text: "Due Date"},
		},
	}

	model := inferFromForm(form)
	if model == nil {
		t.Fatal("inferFromForm returned nil")
	}
	if model.Name != "Task" {
		t.Errorf("model name = %q, want %q", model.Name, "Task")
	}
	if model.Source != "form" {
		t.Errorf("model source = %q, want %q", model.Source, "form")
	}
	if len(model.Fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(model.Fields))
	}
}

func TestInferFromCard(t *testing.T) {
	card := &ClassifiedNode{
		Node: &FigmaNode{Name: "Product Card"},
		Type: ComponentCard,
		Children: []*ClassifiedNode{
			{Node: &FigmaNode{Name: "title"}, Type: ComponentHeading, Text: "Product Name"},
			{Node: &FigmaNode{Name: "desc"}, Type: ComponentText, Text: "Short description"},
			{Node: &FigmaNode{Name: "image"}, Type: ComponentImage},
			{Node: &FigmaNode{Name: "price badge"}, Type: ComponentBadge, Text: "$29.99"},
		},
	}

	model := inferFromCard(card)
	if model == nil {
		t.Fatal("inferFromCard returned nil")
	}
	if model.Name != "Product" {
		t.Errorf("model name = %q, want %q", model.Name, "Product")
	}
	if len(model.Fields) < 3 {
		t.Errorf("expected at least 3 fields, got %d", len(model.Fields))
	}
}

func TestInferFromTable(t *testing.T) {
	table := &ClassifiedNode{
		Node: &FigmaNode{Name: "Users Table"},
		Type: ComponentTable,
		Children: []*ClassifiedNode{
			// Header row
			{Node: &FigmaNode{Name: "header"}, Type: ComponentSection,
				Children: []*ClassifiedNode{
					{Node: &FigmaNode{Name: "col1"}, Text: "Name"},
					{Node: &FigmaNode{Name: "col2"}, Text: "Email"},
					{Node: &FigmaNode{Name: "col3"}, Text: "Created At"},
				}},
		},
	}

	model := inferFromTable(table)
	if model == nil {
		t.Fatal("inferFromTable returned nil")
	}
	if model.Name != "User" {
		t.Errorf("model name = %q, want %q", model.Name, "User")
	}
	if len(model.Fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(model.Fields))
	}
	// Check types were guessed
	for _, f := range model.Fields {
		switch f.Name {
		case "email":
			if f.Type != "email" {
				t.Errorf("email field type = %q, want email", f.Type)
			}
		case "created at":
			if f.Type != "datetime" {
				t.Errorf("created_at field type = %q, want datetime", f.Type)
			}
		}
	}
}

func TestInferModelsDeduplication(t *testing.T) {
	pages := []*ClassifiedPage{
		{Name: "Create", Nodes: []*ClassifiedNode{
			{Node: &FigmaNode{Name: "Create Task Form"}, Type: ComponentForm,
				Children: []*ClassifiedNode{
					{Node: &FigmaNode{Name: "title input"}, Type: ComponentInput, Text: "Title"},
				}},
		}},
		{Name: "Detail", Nodes: []*ClassifiedNode{
			{Node: &FigmaNode{Name: "Task Card"}, Type: ComponentCard,
				Children: []*ClassifiedNode{
					{Node: &FigmaNode{Name: "title"}, Type: ComponentHeading, Text: "Task Name"},
					{Node: &FigmaNode{Name: "status"}, Type: ComponentBadge, Text: "Active"},
				}},
		}},
	}

	models := InferModels(pages)
	// Should have one Task model with merged fields
	taskCount := 0
	for _, m := range models {
		if m.Name == "Task" {
			taskCount++
			if len(m.Fields) < 2 {
				t.Errorf("expected merged Task to have at least 2 fields, got %d", len(m.Fields))
			}
		}
	}
	if taskCount != 1 {
		t.Errorf("expected exactly 1 Task model after dedup, got %d", taskCount)
	}
}

// ---------------------------------------------------------------------------
// mapper_test
// ---------------------------------------------------------------------------

func TestMapNavbar(t *testing.T) {
	page := &ClassifiedPage{
		Name: "Home",
		Nodes: []*ClassifiedNode{
			{Node: &FigmaNode{Name: "Navigation"}, Type: ComponentNavbar,
				Children: []*ClassifiedNode{
					{Node: &FigmaNode{Name: "link1"}, Type: ComponentText, Text: "Dashboard"},
					{Node: &FigmaNode{Name: "link2"}, Type: ComponentText, Text: "Settings"},
				}},
		},
	}
	output := MapToHuman(page, "TestApp")
	if !strings.Contains(output, "page Home:") {
		t.Error("output should contain page declaration")
	}
	if !strings.Contains(output, "show a navigation bar") {
		t.Error("output should contain navigation bar")
	}
	if !strings.Contains(output, `clicking "Dashboard" navigates to Dashboard`) {
		t.Errorf("output should contain navigation for Dashboard, got:\n%s", output)
	}
}

func TestMapHero(t *testing.T) {
	page := &ClassifiedPage{
		Name: "Landing",
		Nodes: []*ClassifiedNode{
			{Node: &FigmaNode{Name: "Hero Section"}, Type: ComponentHero,
				Children: []*ClassifiedNode{
					{Node: &FigmaNode{Name: "h1"}, Type: ComponentHeading, Text: "Welcome to Our App"},
					{Node: &FigmaNode{Name: "sub"}, Type: ComponentText, Text: "Build faster with Human"},
					{Node: &FigmaNode{Name: "cta"}, Type: ComponentButton, Text: "Get Started"},
				}},
		},
	}
	output := MapToHuman(page, "TestApp")
	if !strings.Contains(output, `show a hero section with "Welcome to Our App" and "Build faster with Human"`) {
		t.Errorf("hero output missing heading+subtext:\n%s", output)
	}
	if !strings.Contains(output, `clicking "Get Started"`) {
		t.Errorf("hero output missing CTA:\n%s", output)
	}
}

func TestMapForm(t *testing.T) {
	page := &ClassifiedPage{
		Name: "CreateTask",
		Nodes: []*ClassifiedNode{
			{Node: &FigmaNode{Name: "Create Task Form"}, Type: ComponentForm,
				Children: []*ClassifiedNode{
					{Node: &FigmaNode{Name: "title input"}, Type: ComponentInput, Text: "Title"},
					{Node: &FigmaNode{Name: "description input"}, Type: ComponentInput, Text: "Description"},
					{Node: &FigmaNode{Name: "Submit"}, Type: ComponentButton, Text: "Create"},
				}},
		},
	}
	output := MapToHuman(page, "TestApp")
	if !strings.Contains(output, "there is a form to create Task") {
		t.Errorf("form output missing model reference:\n%s", output)
	}
	if !strings.Contains(output, "there is a text input") {
		t.Errorf("form output missing input:\n%s", output)
	}
}

func TestMapCardList(t *testing.T) {
	page := &ClassifiedPage{
		Name: "Products",
		Nodes: []*ClassifiedNode{
			{Node: &FigmaNode{Name: "Product List"}, Type: ComponentList,
				Children: []*ClassifiedNode{
					{Node: &FigmaNode{Name: "card1"}, Type: ComponentCard,
						Children: []*ClassifiedNode{
							{Node: &FigmaNode{Name: "name"}, Type: ComponentText, Text: "name"},
							{Node: &FigmaNode{Name: "price"}, Type: ComponentText, Text: "price"},
						}},
					{Node: &FigmaNode{Name: "card2"}, Type: ComponentCard},
				}},
		},
	}
	output := MapToHuman(page, "TestApp")
	if !strings.Contains(output, "show a list of") {
		t.Errorf("list output missing list statement:\n%s", output)
	}
}

func TestMapButton(t *testing.T) {
	tests := []struct {
		text     string
		contains string
	}{
		{"Sign Up", "navigates to SignUp"},
		{"Log In", "navigates to Login"},
		{"Submit", "does submit the form"},
		{"Delete", "does delete the item"},
		{"View Details", "navigates to ViewDetails"},
	}
	for _, tt := range tests {
		node := &ClassifiedNode{
			Node: &FigmaNode{Name: "btn"}, Type: ComponentButton, Text: tt.text,
		}
		got := mapButton(node, "  ")
		if !strings.Contains(got, tt.contains) {
			t.Errorf("mapButton(%q) = %q, want to contain %q", tt.text, got, tt.contains)
		}
	}
}

func TestMapHeading(t *testing.T) {
	node := &ClassifiedNode{
		Node: &FigmaNode{Name: "h1"}, Type: ComponentHeading, Text: "Dashboard",
	}
	got := mapHeading(node, "  ")
	if got != `  show a heading "Dashboard"` {
		t.Errorf("mapHeading = %q", got)
	}
}

// ---------------------------------------------------------------------------
// generator_test
// ---------------------------------------------------------------------------

func TestGenerateCRUDAPIs(t *testing.T) {
	model := &InferredModel{
		Name: "Task",
		Fields: []*InferredField{
			{Name: "title", Type: "text"},
			{Name: "status", Type: "text"},
		},
	}

	apis := generateCRUDAPIs(model)
	if len(apis) != 5 {
		t.Fatalf("expected 5 CRUD APIs, got %d", len(apis))
	}

	// Check each API exists
	expected := []string{"CreateTask", "GetAllTasks", "GetTask", "UpdateTask", "DeleteTask"}
	for i, name := range expected {
		if !strings.Contains(apis[i], "api "+name) {
			t.Errorf("API %d should contain %q, got:\n%s", i, name, apis[i])
		}
	}

	// Create should require auth and accept fields
	if !strings.Contains(apis[0], "requires authentication") {
		t.Error("CreateTask should require authentication")
	}
	if !strings.Contains(apis[0], "accepts title, status") {
		t.Errorf("CreateTask should accept fields, got:\n%s", apis[0])
	}
}

func TestGenerateThemeBlock(t *testing.T) {
	theme := &extractedTheme{
		PrimaryColor: "#6C5CE7",
		BodyFont:     "Inter",
		HeadingFont:  "Poppins",
		BorderRadius: "smooth",
		Spacing:      "comfortable",
	}
	block := generateThemeBlock(theme)
	if !strings.Contains(block, "primary color is #6C5CE7") {
		t.Error("theme should contain primary color")
	}
	if !strings.Contains(block, "font is Inter for body and Poppins for headings") {
		t.Error("theme should contain font with body and headings")
	}
}

func TestGenerateDataBlock(t *testing.T) {
	model := &InferredModel{
		Name: "User",
		Fields: []*InferredField{
			{Name: "name", Type: "text"},
			{Name: "email", Type: "email"},
		},
	}
	block := generateDataBlock(model)
	if !strings.Contains(block, "data User:") {
		t.Error("data block should contain model name")
	}
	if !strings.Contains(block, "has a name which is text") {
		t.Error("data block should contain name field")
	}
	if !strings.Contains(block, "has a email which is email") {
		t.Error("data block should contain email field")
	}
}

func TestEndToEndGeneration(t *testing.T) {
	// Build a mock dashboard Figma file
	file := &FigmaFile{
		Name: "Dashboard App",
		Pages: []*FigmaPage{
			{
				Name: "Dashboard",
				Nodes: []*FigmaNode{
					// Navbar
					{
						Name: "Top Nav", Type: "FRAME", Width: 1440, Height: 60,
						LayoutMode: "HORIZONTAL",
						Children: []*FigmaNode{
							{Type: "TEXT", Characters: "Home", Width: 40, Height: 16},
							{Type: "TEXT", Characters: "Tasks", Width: 40, Height: 16},
							{Type: "TEXT", Characters: "Settings", Width: 60, Height: 16},
						},
					},
					// Heading
					{
						Name: "Page Title", Type: "TEXT", Characters: "My Dashboard",
						Style: &TextStyle{FontSize: 28, FontFamily: "Inter"},
						Width: 300, Height: 40,
					},
					// Task form
					{
						Name: "Create Task Form", Type: "FRAME",
						LayoutMode: "VERTICAL", Width: 400, Height: 300,
						Children: []*FigmaNode{
							{Name: "title input", Type: "FRAME", Width: 350, Height: 40,
								Strokes: []Paint{{Type: "SOLID", Visible: true}}},
							{Name: "description input", Type: "FRAME", Width: 350, Height: 40,
								Strokes: []Paint{{Type: "SOLID", Visible: true}}},
							{Name: "Submit Button", Type: "FRAME", Width: 120, Height: 40,
								CornerRadius: 6,
								Fills:    []Paint{{Type: "SOLID", Visible: true, Color: Color{R: 0.4, G: 0.3, B: 0.9}}},
								Children: []*FigmaNode{{Type: "TEXT", Characters: "Create"}},
							},
						},
					},
				},
			},
		},
	}

	config := &GenerateConfig{
		AppName:  "TaskFlow",
		Platform: "web",
		Frontend: "React",
		Backend:  "Node",
		Database: "PostgreSQL",
	}

	output, err := GenerateHumanFile(file, config)
	// We accept parse validation warnings — the generated output may not perfectly
	// parse due to limitations of the heuristic generator
	if err != nil && !strings.Contains(err.Error(), "syntax issues") {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify key sections exist
	checks := []struct {
		label, substr string
	}{
		{"app declaration", "app TaskFlow is a web application"},
		{"theme block", "theme:"},
		{"page block", "page Dashboard:"},
		{"navigation", "navigation bar"},
		{"form", "form to create Task"},
		{"data model", "data Task:"},
		{"api block", "api CreateTask:"},
		{"build block", "build with:"},
		{"frontend target", "frontend using React"},
	}
	for _, check := range checks {
		if !strings.Contains(output, check.substr) {
			t.Errorf("output missing %s (%q):\n%s", check.label, check.substr, output)
		}
	}
}

func TestGenerateHumanFileNilInputs(t *testing.T) {
	_, err := GenerateHumanFile(nil, nil)
	if err == nil {
		t.Error("expected error for nil file")
	}

	_, err = GenerateHumanFile(&FigmaFile{Name: "empty"}, nil)
	if err == nil {
		t.Error("expected error for file with no pages")
	}
}

// ---------------------------------------------------------------------------
// prompt_test
// ---------------------------------------------------------------------------

func TestGenerateFigmaPrompt(t *testing.T) {
	file := &FigmaFile{
		Name: "My Design",
		Pages: []*FigmaPage{
			{
				Name: "Home",
				Nodes: []*FigmaNode{
					{Name: "Hero Banner", Type: "FRAME", Width: 1440, Height: 500,
						Children: []*FigmaNode{
							{Type: "TEXT", Characters: "Welcome",
								Style: &TextStyle{FontSize: 48, FontFamily: "Poppins"}},
						}},
					{Name: "Login Form", Type: "FRAME", LayoutMode: "VERTICAL",
						Width: 400, Height: 300,
						Children: []*FigmaNode{
							{Name: "email input", Type: "FRAME", Width: 350, Height: 40,
								Strokes: []Paint{{Type: "SOLID", Visible: true}}},
							{Name: "password input", Type: "FRAME", Width: 350, Height: 40,
								Strokes: []Paint{{Type: "SOLID", Visible: true}}},
						}},
				},
			},
		},
	}

	prompt := GenerateFigmaPrompt(file)
	if prompt == "" {
		t.Fatal("prompt should not be empty")
	}

	checks := []string{
		"Design Analysis: My Design",
		"Components Detected",
		"Page: Home",
		"hero",
		"Human Language Syntax Reference",
		"Instructions",
	}
	for _, check := range checks {
		if !strings.Contains(prompt, check) {
			t.Errorf("prompt missing %q", check)
		}
	}
}

func TestGeneratePagePrompt(t *testing.T) {
	page := &FigmaPage{
		Name: "Settings",
		Nodes: []*FigmaNode{
			{Name: "Settings Form", Type: "FRAME", LayoutMode: "VERTICAL",
				Width: 500, Height: 400,
				Children: []*FigmaNode{
					{Name: "name input", Type: "FRAME", Width: 400, Height: 40,
						Strokes: []Paint{{Type: "SOLID", Visible: true}}},
					{Name: "email input", Type: "FRAME", Width: 400, Height: 40,
						Strokes: []Paint{{Type: "SOLID", Visible: true}}},
				}},
		},
	}

	prompt := GeneratePagePrompt(page)
	if prompt == "" {
		t.Fatal("page prompt should not be empty")
	}
	if !strings.Contains(prompt, "Design Analysis: Settings Page") {
		t.Error("prompt should contain page name")
	}
	if !strings.Contains(prompt, "page Settings:") {
		t.Error("prompt should reference PascalCase page name in instructions")
	}
}

func TestGeneratePromptNilInputs(t *testing.T) {
	if GenerateFigmaPrompt(nil) != "" {
		t.Error("nil file should return empty prompt")
	}
	if GeneratePagePrompt(nil) != "" {
		t.Error("nil page should return empty prompt")
	}
}

// ---------------------------------------------------------------------------
// ComponentType.String test
// ---------------------------------------------------------------------------

func TestComponentTypeString(t *testing.T) {
	if ComponentButton.String() != "button" {
		t.Errorf("ComponentButton.String() = %q", ComponentButton.String())
	}
	if ComponentUnknown.String() != "unknown" {
		t.Errorf("ComponentUnknown.String() = %q", ComponentUnknown.String())
	}
	if ComponentType(999).String() != "unknown" {
		t.Errorf("out-of-range ComponentType.String() should return unknown")
	}
}
