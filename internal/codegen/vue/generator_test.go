package vue

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
		{"date", "string"},
		{"email", "string"},
		{"url", "string"},
		{"file", "string"},
		{"image", "string"},
		{"number", "number"},
		{"decimal", "number"},
		{"boolean", "boolean"},
		{"json", "Record<string, unknown>"},
		{"unknown_type", "string"},
	}

	for _, tt := range tests {
		got := tsType(tt.input)
		if got != tt.want {
			t.Errorf("tsType(%q): got %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestTsEnumType(t *testing.T) {
	got := tsEnumType([]string{"user", "admin"})
	want := `"user" | "admin"`
	if got != want {
		t.Errorf("tsEnumType: got %q, want %q", got, want)
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"GetTasks", "getTasks"},
		{"SignUp", "signUp"},
		{"Login", "login"},
		{"CreateTask", "createTask"},
		{"Sign Up", "signUp"},
		{"", ""},
	}

	for _, tt := range tests {
		got := toCamelCase(tt.input)
		if got != tt.want {
			t.Errorf("toCamelCase(%q): got %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestToKebabCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Dashboard", "dashboard"},
		{"GetTasks", "get-tasks"},
		{"TaskCard", "task-card"},
		{"SignUp", "sign-up"},
	}

	for _, tt := range tests {
		got := toKebabCase(tt.input)
		if got != tt.want {
			t.Errorf("toKebabCase(%q): got %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestHttpMethod(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"GetTasks", "GET"},
		{"GetProfile", "GET"},
		{"CreateTask", "POST"},
		{"SignUp", "POST"},
		{"Login", "POST"},
		{"UpdateTask", "PUT"},
		{"UpdateProfile", "PUT"},
		{"DeleteTask", "DELETE"},
	}

	for _, tt := range tests {
		got := httpMethod(tt.name)
		if got != tt.want {
			t.Errorf("httpMethod(%q): got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestApiPath(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"GetTasks", "/api/tasks"},
		{"CreateTask", "/api/task"},
		{"UpdateTask", "/api/task"},
		{"DeleteTask", "/api/task"},
		{"SignUp", "/api/sign-up"},
		{"Login", "/api/login"},
		{"GetProfile", "/api/profile"},
	}

	for _, tt := range tests {
		got := apiPath(tt.name)
		if got != tt.want {
			t.Errorf("apiPath(%q): got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestRoutePath(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"Home", "/"},
		{"Dashboard", "/dashboard"},
		{"Profile", "/profile"},
	}

	for _, tt := range tests {
		got := routePath(tt.name)
		if got != tt.want {
			t.Errorf("routePath(%q): got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"a hero section with the app name", "a-hero-section-with-the-app-name"},
		{"Hello World!", "hello-world"},
		{"foo  bar", "foo-bar"},
	}

	for _, tt := range tests {
		got := slugify(tt.input)
		if got != tt.want {
			t.Errorf("slugify(%q): got %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ── Types Generator ──

func TestGenerateTypes(t *testing.T) {
	app := &ir.Application{
		Data: []*ir.DataModel{
			{
				Name: "User",
				Fields: []*ir.DataField{
					{Name: "name", Type: "text", Required: true},
					{Name: "email", Type: "email", Required: true, Unique: true},
					{Name: "bio", Type: "text", Required: false},
					{Name: "role", Type: "enum", Required: true, EnumValues: []string{"user", "admin"}},
					{Name: "age", Type: "number", Required: true},
					{Name: "active", Type: "boolean", Required: true},
					{Name: "metadata", Type: "json", Required: false},
				},
				Relations: []*ir.Relation{
					{Kind: "has_many", Target: "Task"},
				},
			},
			{
				Name: "Task",
				Fields: []*ir.DataField{
					{Name: "title", Type: "text", Required: true},
				},
				Relations: []*ir.Relation{
					{Kind: "belongs_to", Target: "User"},
					{Kind: "has_many_through", Target: "Tag", Through: "TaskTag"},
				},
			},
		},
	}

	output := generateTypes(app)

	if !strings.Contains(output, "export interface User {") {
		t.Error("missing User interface")
	}
	if !strings.Contains(output, "export interface Task {") {
		t.Error("missing Task interface")
	}
	if !strings.Contains(output, "id: string;") {
		t.Error("missing id field")
	}
	if !strings.Contains(output, "name: string;") {
		t.Error("missing name field")
	}
	if !strings.Contains(output, "bio?: string;") {
		t.Error("missing optional bio field")
	}
	if !strings.Contains(output, `role: "user" | "admin";`) {
		t.Error("missing enum role field")
	}
	if !strings.Contains(output, "tasks?: Task[];") {
		t.Error("missing has_many tasks relation")
	}
}

// ── API Client Generator ──

func TestGenerateAPIClient(t *testing.T) {
	app := &ir.Application{
		APIs: []*ir.Endpoint{
			{Name: "SignUp", Params: []*ir.Param{{Name: "name"}, {Name: "email"}, {Name: "password"}}},
			{Name: "GetTasks", Auth: true},
		},
	}

	output := generateAPIClient(app)

	if !strings.Contains(output, "export async function signUp(") {
		t.Error("missing signUp function")
	}
	if !strings.Contains(output, "export async function getTasks(") {
		t.Error("missing getTasks function")
	}
	if !strings.Contains(output, "'POST', '/api/sign-up'") {
		t.Error("signUp should be POST")
	}
	if !strings.Contains(output, "'GET', '/api/tasks'") {
		t.Error("getTasks should be GET")
	}
	if !strings.Contains(output, "name: string; email: string; password: string") {
		t.Error("signUp should have name, email, password params")
	}
}

// ── Router Generator ──

func TestGenerateRouter(t *testing.T) {
	app := &ir.Application{
		Pages: []*ir.Page{
			{Name: "Home"},
			{Name: "Dashboard"},
			{Name: "Profile"},
		},
	}

	output := generateRouter(app)

	if !strings.Contains(output, "import { createRouter, createWebHistory } from 'vue-router'") {
		t.Error("missing vue-router import")
	}
	if !strings.Contains(output, "import HomePage from './pages/HomePage.vue'") {
		t.Error("missing HomePage import")
	}
	if !strings.Contains(output, `path: '/'`) {
		t.Error("missing Home route at /")
	}
	if !strings.Contains(output, `path: '/dashboard'`) {
		t.Error("missing Dashboard route")
	}
	if !strings.Contains(output, "export const router = createRouter") {
		t.Error("missing router export")
	}
}

// ── Page Generator ──

func TestGeneratePage(t *testing.T) {
	page := &ir.Page{
		Name: "Dashboard",
		Content: []*ir.Action{
			{Type: "display", Text: "a welcome message"},
			{Type: "query", Text: "fetch all tasks for the current user"},
			{Type: "loop", Text: "each task as a TaskCard"},
			{Type: "condition", Text: "while loading, show a spinner"},
			{Type: "condition", Text: "if no tasks exist, show an empty state"},
			{Type: "interact", Text: "clicking the create button navigates to CreateTask"},
		},
	}

	app := &ir.Application{}
	output := generatePage(page, app)

	if !strings.Contains(output, "<script setup lang=\"ts\">") {
		t.Error("missing setup script")
	}
	if !strings.Contains(output, "import { ref, onMounted } from 'vue'") {
		t.Error("missing vue imports")
	}
	if !strings.Contains(output, "import { useRouter } from 'vue-router'") {
		t.Error("missing router import")
	}
	if !strings.Contains(output, "const router = useRouter()") {
		t.Error("missing router ref")
	}
	if !strings.Contains(output, "v-if=\"loading\"") {
		t.Error("missing v-if loading directive")
	}
	if !strings.Contains(output, "v-for=\"(item, index) in data\"") {
		t.Error("missing v-for loop mapping")
	}
	if !strings.Contains(output, "@click=\"router.push('/create-task')\"") {
		t.Error("missing click handler")
	}
}

// ── Generate to Filesystem ──

func TestGenerateWritesFiles(t *testing.T) {
	app := &ir.Application{
		Name:     "TestApp",
		Platform: "web",
		Data: []*ir.DataModel{
			{Name: "User", Fields: []*ir.DataField{{Name: "email", Type: "email", Required: true}}},
		},
		Pages: []*ir.Page{
			{Name: "Home", Content: []*ir.Action{{Type: "display", Text: "welcome"}}},
		},
		Components: []*ir.Component{
			{Name: "Card", Content: []*ir.Action{{Type: "display", Text: "content"}}},
		},
		APIs: []*ir.Endpoint{
			{Name: "GetUsers"},
		},
	}

	dir := t.TempDir()
	g := Generator{}
	if err := g.Generate(app, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	expectedFiles := []string{
		"src/types/models.ts",
		"src/api/client.ts",
		"src/router.ts",
		"src/App.vue",
		"src/pages/HomePage.vue",
		"src/components/Card.vue",
	}

	for _, f := range expectedFiles {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}
}

// ── Full Integration Test ──

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
		"src/types/models.ts",
		"src/api/client.ts",
		"src/router.ts",
		"src/App.vue",
		"src/pages/HomePage.vue",
		"src/pages/DashboardPage.vue",
		"src/pages/ProfilePage.vue",
		"src/components/TaskCard.vue",
	}
	for _, f := range expectedFiles {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}

	modelsContent, err := os.ReadFile(filepath.Join(dir, "src", "types", "models.ts"))
	if err != nil {
		t.Fatalf("reading models.ts: %v", err)
	}
	models := string(modelsContent)
	interfaceCount := strings.Count(models, "export interface ")
	if interfaceCount != 4 {
		t.Errorf("models.ts: expected 4 interfaces, got %d", interfaceCount)
	}

	clientContent, err := os.ReadFile(filepath.Join(dir, "src", "api", "client.ts"))
	if err != nil {
		t.Fatalf("reading client.ts: %v", err)
	}
	client := string(clientContent)
	funcCount := strings.Count(client, "export async function ")
	if funcCount != 8 {
		t.Errorf("client.ts: expected 8 functions, got %d", funcCount)
	}

	routerContent, err := os.ReadFile(filepath.Join(dir, "src", "router.ts"))
	if err != nil {
		t.Fatalf("reading router.ts: %v", err)
	}
	routerTs := string(routerContent)
	routeCount := strings.Count(routerTs, "path: ")
	if routeCount != 3 {
		t.Errorf("router.ts: expected 3 routes, got %d", routeCount)
	}
	if !strings.Contains(routerTs, `path: '/'`) {
		t.Error("router.ts: Home should route to /")
	}

	t.Logf("Generated %d files to %s", len(expectedFiles), dir)
}
