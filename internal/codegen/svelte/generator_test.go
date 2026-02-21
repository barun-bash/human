package svelte

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/ir"
	"github.com/barun-bash/human/internal/parser"
)

func TestTsType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"text", "string"},
		{"number", "number"},
		{"boolean", "boolean"},
		{"json", "Record<string, unknown>"},
	}

	for _, tt := range tests {
		got := tsType(tt.input)
		if got != tt.want {
			t.Errorf("tsType(%q): got %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestToKebabCase(t *testing.T) {
	if got := toKebabCase("TaskCard"); got != "task-card" {
		t.Errorf("toKebabCase failed: %q", got)
	}
}

func TestGenerateTypes(t *testing.T) {
	app := &ir.Application{
		Data: []*ir.DataModel{
			{
				Name: "User",
				Fields: []*ir.DataField{
					{Name: "email", Type: "email", Required: true},
				},
			},
		},
	}
	out := generateTypes(app)
	if !strings.Contains(out, "export interface User {") {
		t.Error("missing User interface")
	}
	if !strings.Contains(out, "email: string;") {
		t.Error("missing email field")
	}
}

func TestGenerateApi(t *testing.T) {
	app := &ir.Application{
		APIs: []*ir.Endpoint{
			{Name: "GetTasks"},
		},
	}
	out := generateApi(app)
	if !strings.Contains(out, "export async function getTasks()") {
		t.Error("missing getTasks function")
	}
	if !strings.Contains(out, "fetch(`${API_BASE_URL}") {
		t.Error("missing fetch call")
	}
}

func TestGenerateLayout(t *testing.T) {
	app := &ir.Application{
		Pages: []*ir.Page{
			{Name: "Home"},
			{Name: "Dashboard"},
		},
	}
	out := generateLayout(app)
	if !strings.Contains(out, "let { children } = $props();") {
		t.Error("missing children $props")
	}
	if !strings.Contains(out, "<a href=\"/\">Home</a>") {
		t.Error("missing Home link")
	}
	if !strings.Contains(out, "<a href=\"/dashboard\">Dashboard</a>") {
		t.Error("missing Dashboard link")
	}
	if !strings.Contains(out, "{@render children()}") {
		t.Error("missing {@render children()}")
	}
}

func TestGeneratePage(t *testing.T) {
	app := &ir.Application{}
	page := &ir.Page{
		Name: "Dashboard",
		Content: []*ir.Action{
			{Type: "query", Text: "fetch tasks"},
			{Type: "condition", Text: "while loading"},
			{Type: "loop", Text: "each task as a TaskCard"},
		},
	}
	out := generatePage(page, app)
	if !strings.Contains(out, "let loading = $state(true);") {
		t.Error("missing $state rune")
	}
	if !strings.Contains(out, "$effect(() => {") {
		t.Error("missing $effect rune")
	}
	if !strings.Contains(out, "{#if loading}") {
		t.Error("missing {#if} block")
	}
	if !strings.Contains(out, "{#each data as item, index}") {
		t.Error("missing {#each} block")
	}
	if !strings.Contains(out, "<TaskCard task={item} />") {
		t.Error("missing nested component TaskCard")
	}
}

func TestGenerateComponent(t *testing.T) {
	app := &ir.Application{
		Data: []*ir.DataModel{{Name: "Task"}},
	}
	comp := &ir.Component{
		Name: "TaskCard",
		Props: []*ir.Prop{{Name: "task", Type: "Task"}},
		Content: []*ir.Action{{Type: "interact", Text: "click"}},
	}
	out := generateComponent(comp, app)
	if !strings.Contains(out, "import type { Task } from '$lib/types';") {
		t.Error("missing $lib/types import")
	}
	if !strings.Contains(out, "let { task, onclick }: { task: Task; onclick?: () => void; } = $props();") {
		t.Error("missing $props with typed onclick")
	}
	if !strings.Contains(out, "{onclick}") {
		t.Error("missing {onclick} attribute")
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
		"package.json",
		"svelte.config.js",
		"vite.config.ts",
		"tsconfig.json",
		"src/app.html",
		"src/app.d.ts",
		"src/lib/types.ts",
		"src/lib/api.ts",
		"src/routes/+layout.svelte",
		"src/routes/+page.svelte", // Home page
		"src/routes/dashboard/+page.svelte",
		"src/lib/components/TaskCard.svelte",
	}

	for _, f := range expectedFiles {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}

	pkgContent, _ := os.ReadFile(filepath.Join(dir, "package.json"))
	if !strings.Contains(string(pkgContent), "\"svelte\":") {
		t.Error("package.json missing svelte dependency")
	}
}
