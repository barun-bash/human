package angular

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

func TestToCamelCase(t *testing.T) {
	if got := toCamelCase("GetTasks"); got != "getTasks" {
		t.Errorf("toCamelCase failed: %q", got)
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

func TestGenerateApiService(t *testing.T) {
	app := &ir.Application{
		APIs: []*ir.Endpoint{
			{Name: "GetTasks"},
		},
	}
	out := generateApiService(app)
	if !strings.Contains(out, "@Injectable({ providedIn: 'root' })") {
		t.Error("missing Injectable")
	}
	if !strings.Contains(out, "private http = inject(HttpClient);") {
		t.Error("missing HttpClient injection")
	}
	if !strings.Contains(out, "getTasks(): Observable<ApiResponse<unknown>>") {
		t.Error("missing getTasks method")
	}
}

func TestGenerateAppConfig(t *testing.T) {
	out := generateAppConfig(&ir.Application{})
	if !strings.Contains(out, "provideRouter(routes)") {
		t.Error("missing provideRouter")
	}
	if !strings.Contains(out, "provideHttpClient()") {
		t.Error("missing provideHttpClient")
	}
}

func TestGeneratePage(t *testing.T) {
	app := &ir.Application{}
	page := &ir.Page{
		Name: "Dashboard",
		Content: []*ir.Action{
			{Type: "condition", Text: "while loading"},
			{Type: "loop", Text: "each task"},
		},
	}
	out := generatePage(page, app)
	if !strings.Contains(out, "standalone: true") {
		t.Error("missing standalone flag")
	}
	if !strings.Contains(out, "@if (loading())") {
		t.Error("missing modern @if control flow")
	}
	if !strings.Contains(out, "@for (item of data(); track item.id)") {
		t.Error("missing modern @for control flow")
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
	if !strings.Contains(out, "import type { Task } from '../../models/types';") {
		t.Error("missing Task import")
	}
	if !strings.Contains(out, "@Input() task!: Task;") {
		t.Error("missing typed @Input")
	}
	if !strings.Contains(out, "@Output() onClick = new EventEmitter<void>();") {
		t.Error("missing @Output event emitter")
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
		"angular.json",
		"tsconfig.json",
		"src/index.html",
		"src/main.ts",
		"src/app/app.config.ts",
		"src/app/app.routes.ts",
		"src/app/app.component.ts",
		"src/app/models/types.ts",
		"src/app/services/api.service.ts",
		"src/app/pages/dashboard/dashboard.component.ts",
		"src/app/components/task-card/task-card.component.ts",
		"src/app/pages/not-found/not-found.component.ts",
	}

	for _, f := range expectedFiles {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}

	routesContent, _ := os.ReadFile(filepath.Join(dir, "src/app/app.routes.ts"))
	if !strings.Contains(string(routesContent), "loadComponent") {
		t.Error("routes missing loadComponent")
	}

	apiContent, _ := os.ReadFile(filepath.Join(dir, "src/app/services/api.service.ts"))
	if !strings.Contains(string(apiContent), "createTask(params: { title: string; description: string; status: string; priority: string; dueDate: string }):") {
		t.Error("api client missing createTask param definitions")
	}

	// Verify Storybook deps and scripts in package.json
	pkgContent, _ := os.ReadFile(filepath.Join(dir, "package.json"))
	pkg := string(pkgContent)
	if !strings.Contains(pkg, "@storybook/angular") {
		t.Error("package.json missing @storybook/angular devDependency")
	}
	if !strings.Contains(pkg, `"storybook"`) {
		t.Error("package.json missing storybook script")
	}
	if !strings.Contains(pkg, `"build-storybook"`) {
		t.Error("package.json missing build-storybook script")
	}
}
