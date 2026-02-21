package storybook

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/ir"
	"github.com/barun-bash/human/internal/parser"
)

func TestInventory(t *testing.T) {
	app := &ir.Application{
		Components: []*ir.Component{
			{
				Name: "TaskCard",
				Props: []*ir.Prop{{Name: "task", Type: "Task"}},
				Content: []*ir.Action{{Type: "interact", Text: "click"}},
			},
		},
		Pages: []*ir.Page{
			{
				Name: "Dashboard",
				Content: []*ir.Action{
					{Type: "condition", Text: "while loading"},
					{Type: "condition", Text: "if no tasks"},
				},
			},
		},
	}

	inv := BuildInventory(app)

	if len(inv.Components) != 1 || inv.Components[0].Name != "TaskCard" {
		t.Errorf("inventory failed to capture component")
	}
	if !inv.Components[0].HasClick {
		t.Errorf("inventory failed to detect click handler")
	}

	if len(inv.Pages) != 1 || inv.Pages[0].Name != "Dashboard" {
		t.Errorf("inventory failed to capture page")
	}
	if !inv.Pages[0].HasLoading {
		t.Errorf("inventory failed to detect loading state")
	}
	if !inv.Pages[0].HasEmpty {
		t.Errorf("inventory failed to detect empty state")
	}
}

func TestGenerateMockData(t *testing.T) {
	app := &ir.Application{
		Data: []*ir.DataModel{
			{
				Name: "User",
				Fields: []*ir.DataField{
					{Name: "email", Type: "email"},
					{Name: "age", Type: "number"},
					{Name: "role", Type: "enum", EnumValues: []string{"admin", "user"}},
				},
			},
		},
	}

	out := generateMockData(app)
	if !strings.Contains(out, "export const mockUser") {
		t.Error("missing mock factory")
	}
	if !strings.Contains(out, "'jane.doe@example.com'") {
		t.Error("missing email dummy data")
	}
	if !strings.Contains(out, "28") {
		t.Error("missing age dummy data")
	}
	if !strings.Contains(out, "'admin'") {
		t.Error("missing enum dummy data")
	}
}

func TestFullIntegration(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	humanFile := filepath.Join(root, "examples", "taskflow", "app.human")

	source, err := os.ReadFile(humanFile)
	if err != nil {
		t.Fatalf("failed to read app.human: %v", err)
	}

	prog, err := parser.Parse(string(source))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	app, err := ir.Build(prog)
	if err != nil {
		t.Fatalf("IR build error: %v", err)
	}

	dir := t.TempDir()
	g := Generator{}
	if err := g.Generate(app, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	expectedFiles := []string{
		".storybook/main.ts",
		".storybook/preview.ts",
		"storybook-dependencies.json",
		"src/stories/Introduction.mdx",
		"src/mocks/data.ts",
		"src/stories/components/TaskCard.stories.tsx",
		"src/stories/pages/Dashboard.stories.tsx",
	}

	for _, f := range expectedFiles {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}

	compContent, _ := os.ReadFile(filepath.Join(dir, "src/stories/components/TaskCard.stories.tsx"))
	if !strings.Contains(string(compContent), "task: mocks.mockTask()") {
		t.Error("component story missing mock data usage")
	}
}
