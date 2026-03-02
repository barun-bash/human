package cmdutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/ir"
	"github.com/barun-bash/human/internal/parser"
)

func parseFile(t *testing.T, path string) *parser.Program {
	t.Helper()
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	prog, err := parser.Parse(string(source))
	if err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}
	return prog
}

// TestSplitProgram_FullApp splits the full taskflow example and verifies all 5 files.
func TestSplitProgram_FullApp(t *testing.T) {
	root := projectRoot()
	prog := parseFile(t, filepath.Join(root, "examples", "taskflow", "app.human"))

	files := SplitProgram(prog)

	// All 5 files should be present.
	expected := []string{"app.human", "frontend.human", "backend.human", "devops.human", "integrations.human"}
	for _, name := range expected {
		if _, ok := files[name]; !ok {
			t.Errorf("missing expected file: %s", name)
		}
	}

	// No extra files.
	if len(files) != len(expected) {
		t.Errorf("expected %d files, got %d", len(expected), len(files))
		for name := range files {
			t.Logf("  file: %s", name)
		}
	}

	// app.human should contain app declaration and build.
	if !strings.Contains(files["app.human"], "app TaskFlow") {
		t.Error("app.human missing app declaration")
	}
	if !strings.Contains(files["app.human"], "build with:") {
		t.Error("app.human missing build block")
	}

	// frontend.human should contain pages, components, theme.
	if !strings.Contains(files["frontend.human"], "page Home:") {
		t.Error("frontend.human missing Home page")
	}
	if !strings.Contains(files["frontend.human"], "page Dashboard:") {
		t.Error("frontend.human missing Dashboard page")
	}
	if !strings.Contains(files["frontend.human"], "component TaskCard:") {
		t.Error("frontend.human missing TaskCard component")
	}
	if !strings.Contains(files["frontend.human"], "theme:") {
		t.Error("frontend.human missing theme")
	}

	// backend.human should contain data models, APIs, auth, database, policies, error handlers.
	if !strings.Contains(files["backend.human"], "data User:") {
		t.Error("backend.human missing User model")
	}
	if !strings.Contains(files["backend.human"], "data Task:") {
		t.Error("backend.human missing Task model")
	}
	if !strings.Contains(files["backend.human"], "api SignUp:") {
		t.Error("backend.human missing SignUp API")
	}
	if !strings.Contains(files["backend.human"], "authentication:") {
		t.Error("backend.human missing authentication")
	}
	if !strings.Contains(files["backend.human"], "database:") {
		t.Error("backend.human missing database")
	}
	if !strings.Contains(files["backend.human"], "policy FreeUser:") {
		t.Error("backend.human missing FreeUser policy")
	}
	if !strings.Contains(files["backend.human"], "if database is unreachable:") {
		t.Error("backend.human missing error handler")
	}

	// devops.human should contain architecture, CI/CD workflows, environments, statements.
	if !strings.Contains(files["devops.human"], "architecture: monolith") {
		t.Error("devops.human missing architecture")
	}
	if !strings.Contains(files["devops.human"], "when code is pushed") {
		t.Error("devops.human missing CI/CD workflow")
	}
	if !strings.Contains(files["devops.human"], "environment staging:") {
		t.Error("devops.human missing staging environment")
	}

	// integrations.human should contain integrations and business workflows.
	if !strings.Contains(files["integrations.human"], "integrate with SendGrid:") {
		t.Error("integrations.human missing SendGrid integration")
	}
	if !strings.Contains(files["integrations.human"], "when a user signs up:") {
		t.Error("integrations.human missing business workflow")
	}
}

// TestSplitProgram_MinimalApp tests splitting an app with only app+build.
func TestSplitProgram_MinimalApp(t *testing.T) {
	source := `app MyApp is a web application

build with:
  frontend using React
  backend using Node
  database using PostgreSQL
`
	prog, err := parser.Parse(source)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	files := SplitProgram(prog)

	if len(files) != 1 {
		t.Errorf("expected 1 file (app.human), got %d", len(files))
	}
	if _, ok := files["app.human"]; !ok {
		t.Error("missing app.human")
	}
}

// TestSplitProgram_FrontendOnly tests splitting an app with frontend content.
func TestSplitProgram_FrontendOnly(t *testing.T) {
	source := `app MyApp is a web application

page Home:
  show heading "Hello"

build with:
  frontend using React
  backend using Node
  database using PostgreSQL
`
	prog, err := parser.Parse(source)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	files := SplitProgram(prog)

	if _, ok := files["app.human"]; !ok {
		t.Error("missing app.human")
	}
	if _, ok := files["frontend.human"]; !ok {
		t.Error("missing frontend.human")
	}
	if _, ok := files["backend.human"]; ok {
		t.Error("unexpected backend.human for frontend-only app")
	}
}

// TestSplitProgram_BackendOnly tests splitting an app with only backend content.
func TestSplitProgram_BackendOnly(t *testing.T) {
	source := `app MyApp is a web application

data User:
  has a name which is text

api GetUsers:
  fetch all User
  respond with items

build with:
  frontend using React
  backend using Node
  database using PostgreSQL
`
	prog, err := parser.Parse(source)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	files := SplitProgram(prog)

	if _, ok := files["app.human"]; !ok {
		t.Error("missing app.human")
	}
	if _, ok := files["backend.human"]; !ok {
		t.Error("missing backend.human")
	}
	if _, ok := files["frontend.human"]; ok {
		t.Error("unexpected frontend.human for backend-only app")
	}
}

// TestIsDevOpsWorkflow tests CI/CD vs business workflow classification.
func TestIsDevOpsWorkflow(t *testing.T) {
	tests := []struct {
		event  string
		devops bool
	}{
		{"code is pushed to a feature branch", true},
		{"code is merged to main", true},
		{"code is merged to staging", true},
		{"a pull request is opened", true},
		{"branch is created", true},
		{"a user signs up", false},
		{"a task becomes overdue", false},
		{"a task is marked done", false},
		{"an order is placed", false},
		{"payment fails", false},
	}

	for _, tt := range tests {
		t.Run(tt.event, func(t *testing.T) {
			got := isDevOpsWorkflow(tt.event)
			if got != tt.devops {
				t.Errorf("isDevOpsWorkflow(%q) = %v, want %v", tt.event, got, tt.devops)
			}
		})
	}
}

// TestSplitRoundTrip verifies that split → merge → IR matches the original.
func TestSplitRoundTrip(t *testing.T) {
	root := projectRoot()
	singleFile := filepath.Join(root, "examples", "taskflow", "app.human")

	if _, err := os.Stat(singleFile); err != nil {
		t.Skipf("single-file example not found: %v", err)
	}

	// Parse original.
	prog := parseFile(t, singleFile)

	// Build original IR.
	origApp, err := ir.Build(prog)
	if err != nil {
		t.Fatalf("building original IR: %v", err)
	}
	origYAML, err := ir.ToYAML(origApp)
	if err != nil {
		t.Fatalf("serializing original IR: %v", err)
	}

	// Split to temp dir.
	tmpDir := t.TempDir()
	created, err := SplitToDir(prog, tmpDir)
	if err != nil {
		t.Fatalf("splitting: %v", err)
	}
	if len(created) == 0 {
		t.Fatal("no files created")
	}

	// Discover, parse, and merge the split files.
	files, err := parser.DiscoverFiles(tmpDir)
	if err != nil {
		t.Fatalf("discovering split files: %v", err)
	}

	programs, err := parser.ParseFiles(files)
	if err != nil {
		t.Fatalf("parsing split files: %v", err)
	}

	merged, err := parser.MergePrograms(programs)
	if err != nil {
		t.Fatalf("merging split files: %v", err)
	}

	// Build merged IR.
	mergedApp, err := ir.Build(merged)
	if err != nil {
		t.Fatalf("building merged IR: %v", err)
	}
	mergedYAML, err := ir.ToYAML(mergedApp)
	if err != nil {
		t.Fatalf("serializing merged IR: %v", err)
	}

	// Compare.
	if origYAML != mergedYAML {
		origLines := strings.Split(origYAML, "\n")
		mergedLines := strings.Split(mergedYAML, "\n")
		for i := 0; i < len(origLines) && i < len(mergedLines); i++ {
			if origLines[i] != mergedLines[i] {
				t.Fatalf("IR mismatch at line %d:\n  original: %s\n  merged:   %s", i+1, origLines[i], mergedLines[i])
			}
		}
		if len(origLines) != len(mergedLines) {
			t.Fatalf("IR line count mismatch: original=%d, merged=%d", len(origLines), len(mergedLines))
		}
		t.Fatal("IR YAML content differs (unknown line)")
	}
}
