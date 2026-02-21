package quality

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/ir"
)

func TestGenerateComponentTests(t *testing.T) {
	app := exampleApp(t)
	dir := t.TempDir()

	files, count, err := generateComponentTests(app, dir)
	if err != nil {
		t.Fatalf("generateComponentTests: %v", err)
	}

	if files == 0 {
		t.Fatal("expected component test files, got 0")
	}
	if count == 0 {
		t.Fatal("expected component test count > 0")
	}

	// Verify .test.tsx files were created
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading dir: %v", err)
	}

	tsxCount := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".test.tsx") {
			tsxCount++
		}
	}
	if tsxCount != files {
		t.Errorf("expected %d .test.tsx files, got %d", files, tsxCount)
	}
}

func TestGeneratePageTests_Display(t *testing.T) {
	page := &ir.Page{
		Name: "Home",
		Content: []*ir.Action{
			{Type: "display", Text: "show welcome message"},
		},
	}

	content, count := generatePageTests(page, &ir.Application{})
	if count < 2 {
		t.Errorf("expected at least 2 tests (render + display), got %d", count)
	}
	if !strings.Contains(content, "screen.getByText") {
		t.Error("missing screen.getByText assertion for display action")
	}
	if !strings.Contains(content, "welcome message") {
		t.Error("missing display text in test")
	}
}

func TestGeneratePageTests_Loading(t *testing.T) {
	page := &ir.Page{
		Name: "Dashboard",
		Content: []*ir.Action{
			{Type: "condition", Text: "while loading data"},
		},
	}

	content, count := generatePageTests(page, &ir.Application{})
	if count < 2 {
		t.Errorf("expected at least 2 tests (render + loading), got %d", count)
	}
	if !strings.Contains(content, "loading state") {
		t.Error("missing loading state test")
	}
	if !strings.Contains(content, "/loading/i") {
		t.Error("missing loading text assertion")
	}
}

func TestGeneratePageTests_EmptyState(t *testing.T) {
	page := &ir.Page{
		Name: "Tasks",
		Content: []*ir.Action{
			{Type: "condition", Text: "if no tasks match the filter"},
		},
	}

	content, count := generatePageTests(page, &ir.Application{})
	if count < 2 {
		t.Errorf("expected at least 2 tests (render + empty), got %d", count)
	}
	if !strings.Contains(content, "empty state") {
		t.Error("missing empty state test")
	}
}

func TestGeneratePageTests_Interaction(t *testing.T) {
	page := &ir.Page{
		Name: "Profile",
		Content: []*ir.Action{
			{Type: "interact", Text: "clicking the save button"},
		},
	}

	content, count := generatePageTests(page, &ir.Application{})
	if count < 2 {
		t.Errorf("expected at least 2 tests (render + click), got %d", count)
	}
	if !strings.Contains(content, "fireEvent.click") {
		t.Error("missing fireEvent.click in interaction test")
	}
}

func TestGeneratePageTests_Empty(t *testing.T) {
	page := &ir.Page{
		Name: "Blank",
	}

	_, count := generatePageTests(page, &ir.Application{})
	if count != 1 {
		t.Errorf("expected exactly 1 render test for empty page, got %d", count)
	}
}

func TestGeneratePageTests_MockedAPI(t *testing.T) {
	page := &ir.Page{
		Name: "Home",
	}

	content, _ := generatePageTests(page, &ir.Application{})
	if !strings.Contains(content, "jest.mock") {
		t.Error("missing jest.mock for API client")
	}
	if !strings.Contains(content, "../api/client") {
		t.Error("missing API client mock path")
	}
}

func TestGeneratePageTests_BrowserRouter(t *testing.T) {
	page := &ir.Page{
		Name: "Home",
	}

	content, _ := generatePageTests(page, &ir.Application{})
	if !strings.Contains(content, "BrowserRouter") {
		t.Error("missing BrowserRouter wrapper")
	}
}

func TestExtractDisplayText(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"show welcome message", "welcome message"},
		{"display user profile", "user profile"},
		{"render task list", "task list"},
		{"some other text", "some other text"},
	}
	for _, tt := range tests {
		got := extractDisplayText(tt.input)
		if got != tt.expect {
			t.Errorf("extractDisplayText(%q) = %q, want %q", tt.input, got, tt.expect)
		}
	}
}

func TestExtractClickTarget(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"clicking the save button", "save"},
		{"clicking delete", "delete"},
		{"click the submit button", "submit"},
		{"click on edit link", "edit"},
		{"no click here", ""},
	}
	for _, tt := range tests {
		got := extractClickTarget(tt.input)
		if got != tt.expect {
			t.Errorf("extractClickTarget(%q) = %q, want %q", tt.input, got, tt.expect)
		}
	}
}

func TestEscapeRegex(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"hello", "hello"},
		{"a.b", "a\\.b"},
		{"(a+b)", "\\(a\\+b\\)"},
		{"a\\b", "a\\\\b"},       // backslash first, then no double-escape
		{"$100", "\\$100"},
		{"a|b", "a\\|b"},
		{"[test]", "\\[test\\]"},
		{"", ""},
	}
	for _, tt := range tests {
		got := escapeRegex(tt.input)
		if got != tt.expect {
			t.Errorf("escapeRegex(%q) = %q, want %q", tt.input, got, tt.expect)
		}
	}
}

func TestEscapeJSString(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"hello", "hello"},
		{"it's", "it\\'s"},
		{`say "hi"`, `say \"hi\"`},
		{"", ""},
	}
	for _, tt := range tests {
		got := escapeJSString(tt.input)
		if got != tt.expect {
			t.Errorf("escapeJSString(%q) = %q, want %q", tt.input, got, tt.expect)
		}
	}
}

func TestGenerateComponentTests_OutputPath(t *testing.T) {
	app := &ir.Application{
		Pages: []*ir.Page{
			{Name: "Home"},
		},
	}
	dir := t.TempDir()

	_, _, err := generateComponentTests(app, dir)
	if err != nil {
		t.Fatalf("generateComponentTests: %v", err)
	}

	path := filepath.Join(dir, "home.test.tsx")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist", path)
	}
}
