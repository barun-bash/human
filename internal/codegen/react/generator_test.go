package react

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/ir"
	"github.com/barun-bash/human/internal/parser"
)

// ── Helper Utilities ──

func TestTsType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"text", "string"},
		{"date", "string"},
		{"datetime", "string"},
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

	// Check interfaces exist
	if !strings.Contains(output, "export interface User {") {
		t.Error("missing User interface")
	}
	if !strings.Contains(output, "export interface Task {") {
		t.Error("missing Task interface")
	}

	// Check id field
	if !strings.Contains(output, "id: string;") {
		t.Error("missing id field")
	}

	// Check field types
	if !strings.Contains(output, "name: string;") {
		t.Error("missing name field")
	}
	if !strings.Contains(output, "email: string;") {
		t.Error("missing email field")
	}
	if !strings.Contains(output, "age: number;") {
		t.Error("missing age field")
	}
	if !strings.Contains(output, "active: boolean;") {
		t.Error("missing active field")
	}

	// Check optional field
	if !strings.Contains(output, "bio?: string;") {
		t.Error("missing optional bio field")
	}
	if !strings.Contains(output, "metadata?: Record<string, unknown>;") {
		t.Error("missing optional metadata field")
	}

	// Check enum type
	if !strings.Contains(output, `role: "user" | "admin";`) {
		t.Error("missing enum role field")
	}

	// Check has_many relation
	if !strings.Contains(output, "tasks?: Task[];") {
		t.Error("missing has_many tasks relation")
	}

	// Check belongs_to relation
	if !strings.Contains(output, "userId: string;") {
		t.Error("missing belongs_to userId")
	}
	if !strings.Contains(output, "user?: User;") {
		t.Error("missing belongs_to user reference")
	}

	// Check has_many_through relation
	if !strings.Contains(output, "tags?: Tag[];") {
		t.Error("missing has_many_through tags relation")
	}
}

// ── API Client Generator ──

func TestGenerateAPIClient(t *testing.T) {
	app := &ir.Application{
		APIs: []*ir.Endpoint{
			{Name: "SignUp", Params: []*ir.Param{{Name: "name"}, {Name: "email"}, {Name: "password"}}},
			{Name: "Login", Auth: false, Params: []*ir.Param{{Name: "email"}, {Name: "password"}}},
			{Name: "GetTasks", Auth: true},
			{Name: "CreateTask", Auth: true, Params: []*ir.Param{{Name: "title"}}},
			{Name: "UpdateTask", Auth: true, Params: []*ir.Param{{Name: "task_id"}}},
			{Name: "DeleteTask", Auth: true, Params: []*ir.Param{{Name: "task_id"}}},
		},
	}

	output := generateAPIClient(app)

	// Check shared infrastructure
	if !strings.Contains(output, "API_BASE_URL") {
		t.Error("missing API_BASE_URL")
	}
	if !strings.Contains(output, "ApiResponse<T>") {
		t.Error("missing ApiResponse type")
	}
	if !strings.Contains(output, "async function request<T>") {
		t.Error("missing request helper")
	}
	if !strings.Contains(output, "localStorage.getItem('token')") {
		t.Error("missing auth token handling")
	}

	// Check function names (camelCase)
	if !strings.Contains(output, "export async function signUp(") {
		t.Error("missing signUp function")
	}
	if !strings.Contains(output, "export async function login(") {
		t.Error("missing login function")
	}
	if !strings.Contains(output, "export async function getTasks(") {
		t.Error("missing getTasks function")
	}

	// Check HTTP methods
	if !strings.Contains(output, "'POST', '/api/sign-up'") {
		t.Error("signUp should be POST")
	}
	if !strings.Contains(output, "'GET', '/api/tasks'") {
		t.Error("getTasks should be GET")
	}
	if !strings.Contains(output, "'PUT', '/api/task'") {
		t.Error("updateTask should be PUT")
	}
	if !strings.Contains(output, "'DELETE', '/api/task'") {
		t.Error("deleteTask should be DELETE")
	}

	// Check params
	if !strings.Contains(output, "name: string; email: string; password: string") {
		t.Error("signUp should have name, email, password params")
	}
}

// ── App (Router) Generator ──

func TestGenerateApp(t *testing.T) {
	app := &ir.Application{
		Pages: []*ir.Page{
			{Name: "Home"},
			{Name: "Dashboard"},
			{Name: "Profile"},
		},
	}

	output := generateApp(app)

	// Check imports
	if !strings.Contains(output, "import { BrowserRouter, Routes, Route } from 'react-router-dom'") {
		t.Error("missing react-router-dom import")
	}
	if !strings.Contains(output, "import HomePage from './pages/HomePage'") {
		t.Error("missing HomePage import")
	}
	if !strings.Contains(output, "import DashboardPage from './pages/DashboardPage'") {
		t.Error("missing DashboardPage import")
	}
	if !strings.Contains(output, "import ProfilePage from './pages/ProfilePage'") {
		t.Error("missing ProfilePage import")
	}

	// Check routes
	if !strings.Contains(output, `path="/"`) {
		t.Error("missing Home route at /")
	}
	if !strings.Contains(output, `path="/dashboard"`) {
		t.Error("missing Dashboard route")
	}
	if !strings.Contains(output, `path="/profile"`) {
		t.Error("missing Profile route")
	}
	if !strings.Contains(output, "element={<HomePage />}") {
		t.Error("missing HomePage element")
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

	// Check function name
	if !strings.Contains(output, "export default function DashboardPage()") {
		t.Error("missing DashboardPage function")
	}

	// Check hooks
	if !strings.Contains(output, "useState") {
		t.Error("missing useState import")
	}
	if !strings.Contains(output, "useEffect") {
		t.Error("missing useEffect import")
	}
	if !strings.Contains(output, "useNavigate") {
		t.Error("missing useNavigate for interact+navigate action")
	}

	// Check JSX
	if !strings.Contains(output, `className="dashboard-page"`) {
		t.Error("missing page wrapper className")
	}
	if !strings.Contains(output, "loading &&") {
		t.Error("missing loading state guard")
	}
	if !strings.Contains(output, "data.length === 0") {
		t.Error("missing empty state check")
	}
	if !strings.Contains(output, "data.map(") {
		t.Error("missing loop mapping")
	}
}

// ── Page Generator with Data Model ──

func TestGeneratePageWithModel(t *testing.T) {
	page := &ir.Page{
		Name: "Dashboard",
		Content: []*ir.Action{
			{Type: "display", Text: "show a hero section with the app name"},
			{Type: "display", Text: "show a greeting with the user's name"},
			{Type: "display", Text: "show a summary card with total tasks, completed tasks, and overdue tasks"},
			{Type: "display", Text: "show a Get Started button"},
			{Type: "loop", Text: "each task shows its title, status, priority, and due date"},
			{Type: "condition", Text: "while loading, show a spinner"},
			{Type: "condition", Text: "if no tasks match, show No tasks found. Create your first task!"},
			{Type: "input", Text: "there is a search bar that filters tasks by title"},
			{Type: "input", Text: "there is a floating button to add a new task"},
		},
	}

	app := &ir.Application{
		Name: "TaskFlow",
		Data: []*ir.DataModel{
			{
				Name: "Task",
				Fields: []*ir.DataField{
					{Name: "title", Type: "text"},
					{Name: "status", Type: "enum"},
					{Name: "priority", Type: "enum"},
					{Name: "due", Type: "date"},
				},
			},
		},
	}

	output := generatePage(page, app)

	// Model-aware state
	if !strings.Contains(output, "useState<Task[]>") {
		t.Error("missing typed Task[] state")
	}
	if !strings.Contains(output, "import { Task }") {
		t.Error("missing Task type import")
	}
	if !strings.Contains(output, "useEffect") {
		t.Error("missing useEffect for data fetching")
	}

	// Hero section renders as <section>
	if !strings.Contains(output, `<section className="hero">`) {
		t.Error("hero section should render as <section>")
	}
	if !strings.Contains(output, "<h1>TaskFlow</h1>") {
		t.Error("hero section should include app name")
	}

	// Greeting renders as <h2>
	if !strings.Contains(output, `className="greeting"`) {
		t.Error("greeting should render with greeting class")
	}

	// Summary cards render as stat cards
	if !strings.Contains(output, `className="summary-cards"`) {
		t.Error("summary should render as summary-cards")
	}
	if !strings.Contains(output, `className="stat-card"`) {
		t.Error("should have stat-card elements")
	}

	// Button display
	if !strings.Contains(output, `<button className="btn">Get Started</button>`) {
		t.Error("button display should render as <button>")
	}

	// Loop with typed fields
	if !strings.Contains(output, "tasks.map((task)") {
		t.Error("loop should use model-aware variable names")
	}
	if !strings.Contains(output, "task.title") {
		t.Error("loop should reference task.title")
	}
	if !strings.Contains(output, "task.status") {
		t.Error("loop should reference task.status")
	}

	// Loading spinner
	if !strings.Contains(output, "loading &&") {
		t.Error("loading state should render spinner")
	}

	// Empty state with custom message
	if !strings.Contains(output, "tasks.length === 0") {
		t.Error("empty state should check tasks.length")
	}
	if !strings.Contains(output, "No tasks found. Create your first task!") {
		t.Error("empty state should use custom message")
	}

	// Search input
	if !strings.Contains(output, `type="search"`) {
		t.Error("search input should render as search type")
	}

	// FAB
	if !strings.Contains(output, `className="fab"`) {
		t.Error("floating button should render as FAB")
	}
}

func TestGeneratePageProfile(t *testing.T) {
	page := &ir.Page{
		Name: "Profile",
		Content: []*ir.Action{
			{Type: "display", Text: "show the user's name, email, and avatar"},
			{Type: "input", Text: "there is a form to update name and bio"},
			{Type: "input", Text: "there is a file upload for avatar"},
			{Type: "interact", Text: "clicking Save updates the user profile"},
			{Type: "interact", Text: "clicking Change Password opens a password change form"},
			{Type: "condition", Text: "if the update succeeds, show Profile updated successfully"},
			{Type: "condition", Text: "if there is an error, show the error message"},
		},
	}

	app := &ir.Application{}
	output := generatePage(page, app)

	// Field group for user's data
	if !strings.Contains(output, `className="field-group"`) {
		t.Error("user fields should render as field-group")
	}
	if !strings.Contains(output, "Name") && !strings.Contains(output, "name") {
		t.Error("should display name field")
	}

	// Form
	if !strings.Contains(output, "<form") {
		t.Error("should render a form")
	}

	// File upload
	if !strings.Contains(output, `type="file"`) {
		t.Error("should render file upload input")
	}

	// Save button
	if !strings.Contains(output, ">Save</button>") {
		t.Error("should render Save button")
	}

	// Change Password button
	if !strings.Contains(output, ">Change Password</button>") {
		t.Error("should render Change Password button")
	}

	// Success message
	if !strings.Contains(output, "Profile updated successfully") {
		t.Error("should include custom success message")
	}

	// Error display
	if !strings.Contains(output, "alert-error") {
		t.Error("should render error alert")
	}
}

// ── Component Generator ──

func TestGenerateComponent(t *testing.T) {
	comp := &ir.Component{
		Name: "TaskCard",
		Props: []*ir.Prop{
			{Name: "task", Type: "Task"},
		},
		Content: []*ir.Action{
			{Type: "display", Text: "show the task title in bold"},
			{Type: "display", Text: "show the status as a colored badge"},
			{Type: "display", Text: "show the priority with an icon"},
			{Type: "display", Text: "show the due date in relative format like due in 2 days"},
			{Type: "condition", Text: "if task is overdue, show the due date in red"},
			{Type: "interact", Text: "clicking the card triggers on_click"},
		},
	}

	app := &ir.Application{
		Data: []*ir.DataModel{
			{
				Name: "Task",
				Fields: []*ir.DataField{
					{Name: "title", Type: "text"},
					{Name: "status", Type: "enum"},
					{Name: "priority", Type: "enum"},
					{Name: "due", Type: "date"},
				},
			},
		},
	}

	output := generateComponent(comp, app)

	// Check import for data model type
	if !strings.Contains(output, "import { Task } from '../types/models'") {
		t.Error("missing Task type import")
	}

	// Check props interface
	if !strings.Contains(output, "interface TaskCardProps {") {
		t.Error("missing TaskCardProps interface")
	}
	if !strings.Contains(output, "task: Task;") {
		t.Error("missing task prop with Task type")
	}

	// Check component function
	if !strings.Contains(output, "export default function TaskCard(") {
		t.Error("missing TaskCard function")
	}
	if !strings.Contains(output, `className="task-card"`) {
		t.Error("missing task-card className")
	}

	// Display JSX: title in bold → <strong>{task.title}</strong>
	if !strings.Contains(output, "<strong>{task.title}</strong>") {
		t.Error("title should render in bold with task.title")
	}

	// Display JSX: status as badge → <span className="badge">{task.status}</span>
	if !strings.Contains(output, `<span className="badge">{task.status}</span>`) {
		t.Error("status should render as badge")
	}

	// Display JSX: priority with icon
	if !strings.Contains(output, "task.priority") {
		t.Error("priority should reference task.priority")
	}

	// Display JSX: due date in relative format → <time>{task.due}</time>
	if !strings.Contains(output, "<time>{task.due}</time>") {
		t.Error("due date should render as <time> element")
	}

	// Condition: overdue → text-danger
	if !strings.Contains(output, `className="text-danger"`) {
		t.Error("overdue condition should render with text-danger")
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

	// Verify all expected files exist
	expectedFiles := []string{
		"index.html",
		"src/main.tsx",
		"src/index.css",
		"src/types/models.ts",
		"src/api/client.ts",
		"src/App.tsx",
		"src/pages/HomePage.tsx",
		"src/components/Card.tsx",
	}

	for _, f := range expectedFiles {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}
}

// ── Vite Entry Point ──

func TestGenerateIndexHTML(t *testing.T) {
	app := &ir.Application{Name: "TaskFlow"}
	output := generateIndexHTML(app)

	if !strings.Contains(output, "<!DOCTYPE html>") {
		t.Error("missing DOCTYPE")
	}
	if !strings.Contains(output, "<title>TaskFlow</title>") {
		t.Error("missing app name in <title>")
	}
	if !strings.Contains(output, `<div id="root"></div>`) {
		t.Error("missing root div")
	}
	if !strings.Contains(output, `<script type="module" src="/src/main.tsx"></script>`) {
		t.Error("missing Vite module script tag")
	}
}

func TestGenerateIndexHTMLDefaultTitle(t *testing.T) {
	app := &ir.Application{}
	output := generateIndexHTML(app)

	if !strings.Contains(output, "<title>App</title>") {
		t.Error("missing default title 'App' when no app name set")
	}
}

func TestGenerateMainTsx(t *testing.T) {
	output := generateMainTsx()

	if !strings.Contains(output, "import React from 'react'") {
		t.Error("missing React import")
	}
	if !strings.Contains(output, "import ReactDOM from 'react-dom/client'") {
		t.Error("missing ReactDOM import")
	}
	if !strings.Contains(output, "import App from './App'") {
		t.Error("missing App import")
	}
	if !strings.Contains(output, "import './index.css'") {
		t.Error("missing index.css import")
	}
	if !strings.Contains(output, "ReactDOM.createRoot(") {
		t.Error("missing ReactDOM.createRoot call")
	}
	if !strings.Contains(output, "<React.StrictMode>") {
		t.Error("missing React.StrictMode wrapper")
	}
}

func TestGenerateIndexCSS(t *testing.T) {
	app := &ir.Application{}
	output := generateIndexCSS(app)

	if !strings.Contains(output, "box-sizing: border-box") {
		t.Error("missing CSS reset")
	}
	// No Tailwind directives without a theme
	if strings.Contains(output, "@tailwind") {
		t.Error("should not include Tailwind directives without a Tailwind-based theme")
	}
}

func TestGenerateIndexCSSWithTailwind(t *testing.T) {
	app := &ir.Application{
		Theme: &ir.Theme{DesignSystem: "tailwind"},
	}
	output := generateIndexCSS(app)

	if !strings.Contains(output, "@tailwind base;") {
		t.Error("missing @tailwind base directive")
	}
	if !strings.Contains(output, "@tailwind components;") {
		t.Error("missing @tailwind components directive")
	}
	if !strings.Contains(output, "@tailwind utilities;") {
		t.Error("missing @tailwind utilities directive")
	}
}

func TestGenerateIndexCSSWithShadcn(t *testing.T) {
	app := &ir.Application{
		Theme: &ir.Theme{DesignSystem: "shadcn"},
	}
	output := generateIndexCSS(app)

	// shadcn also uses Tailwind
	if !strings.Contains(output, "@tailwind base;") {
		t.Error("shadcn should include @tailwind directives")
	}
}

// ── Theme Integration ──

func TestGenerateAppWithMaterialTheme(t *testing.T) {
	app := &ir.Application{
		Pages: []*ir.Page{
			{Name: "Home"},
		},
		Theme: &ir.Theme{
			DesignSystem: "material",
		},
	}

	output := generateApp(app)

	if !strings.Contains(output, "ThemeProvider") {
		t.Error("material theme should wrap in ThemeProvider")
	}
	if !strings.Contains(output, "import theme from './theme'") {
		t.Error("material theme should import theme config")
	}
	if !strings.Contains(output, "CssBaseline") {
		t.Error("material theme should include CssBaseline")
	}
}

func TestGenerateAppWithChakraTheme(t *testing.T) {
	app := &ir.Application{
		Pages: []*ir.Page{
			{Name: "Home"},
		},
		Theme: &ir.Theme{
			DesignSystem: "chakra",
		},
	}

	output := generateApp(app)

	if !strings.Contains(output, "ChakraProvider") {
		t.Error("chakra theme should wrap in ChakraProvider")
	}
}

func TestGenerateWritesThemeFiles(t *testing.T) {
	app := &ir.Application{
		Name:     "ThemedApp",
		Platform: "web",
		Pages: []*ir.Page{
			{Name: "Home", Content: []*ir.Action{{Type: "display", Text: "welcome"}}},
		},
		Theme: &ir.Theme{
			DesignSystem: "material",
			Colors:       map[string]string{"primary": "#1976d2"},
		},
	}

	dir := t.TempDir()
	g := Generator{}
	if err := g.Generate(app, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	themeFiles := []string{
		"src/theme.ts",
		"src/styles/global.css",
	}
	for _, f := range themeFiles {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected theme file %s to exist", f)
		}
	}
}

// ── Full Integration Test ──

func TestFullIntegration(t *testing.T) {
	// Locate examples/taskflow/app.human
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	humanFile := filepath.Join(root, "examples", "taskflow", "app.human")

	source, err := os.ReadFile(humanFile)
	if err != nil {
		t.Fatalf("failed to read app.human: %v", err)
	}

	// Parse → IR
	prog, err := parser.Parse(string(source))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	app, err := ir.Build(prog)
	if err != nil {
		t.Fatalf("IR build error: %v", err)
	}

	// Generate to temp directory
	dir := t.TempDir()
	g := Generator{}
	if err := g.Generate(app, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Verify all expected files exist
	expectedFiles := []string{
		"index.html",
		"src/main.tsx",
		"src/index.css",
		"src/types/models.ts",
		"src/api/client.ts",
		"src/App.tsx",
		"src/pages/HomePage.tsx",
		"src/pages/DashboardPage.tsx",
		"src/pages/ProfilePage.tsx",
		"src/components/TaskCard.tsx",
	}
	for _, f := range expectedFiles {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}

	// Verify index.html references the app name and Vite entry point
	htmlContent, err := os.ReadFile(filepath.Join(dir, "index.html"))
	if err != nil {
		t.Fatalf("reading index.html: %v", err)
	}
	html := string(htmlContent)
	if !strings.Contains(html, "<title>TaskFlow</title>") {
		t.Error("index.html: missing TaskFlow title")
	}
	if !strings.Contains(html, `src="/src/main.tsx"`) {
		t.Error("index.html: missing Vite module entry point")
	}

	// Verify main.tsx has React DOM mount
	mainContent, err := os.ReadFile(filepath.Join(dir, "src", "main.tsx"))
	if err != nil {
		t.Fatalf("reading main.tsx: %v", err)
	}
	if !strings.Contains(string(mainContent), "ReactDOM.createRoot") {
		t.Error("main.tsx: missing ReactDOM.createRoot")
	}

	// Verify index.css has base styles
	cssContent, err := os.ReadFile(filepath.Join(dir, "src", "index.css"))
	if err != nil {
		t.Fatalf("reading index.css: %v", err)
	}
	if !strings.Contains(string(cssContent), "box-sizing") {
		t.Error("index.css: missing CSS reset")
	}

	// Verify models.ts has 4 interfaces (User, Task, Tag, TaskTag)
	modelsContent, err := os.ReadFile(filepath.Join(dir, "src", "types", "models.ts"))
	if err != nil {
		t.Fatalf("reading models.ts: %v", err)
	}
	models := string(modelsContent)
	interfaceCount := strings.Count(models, "export interface ")
	if interfaceCount != 4 {
		t.Errorf("models.ts: expected 4 interfaces, got %d", interfaceCount)
	}
	for _, name := range []string{"User", "Task", "Tag", "TaskTag"} {
		if !strings.Contains(models, "export interface "+name+" {") {
			t.Errorf("models.ts: missing interface %s", name)
		}
	}

	// Verify client.ts has 8 functions
	clientContent, err := os.ReadFile(filepath.Join(dir, "src", "api", "client.ts"))
	if err != nil {
		t.Fatalf("reading client.ts: %v", err)
	}
	client := string(clientContent)
	funcCount := strings.Count(client, "export async function ")
	if funcCount != 8 {
		t.Errorf("client.ts: expected 8 functions, got %d", funcCount)
	}

	// Verify App.tsx has 3 routes
	appContent, err := os.ReadFile(filepath.Join(dir, "src", "App.tsx"))
	if err != nil {
		t.Fatalf("reading App.tsx: %v", err)
	}
	appTsx := string(appContent)
	routeCount := strings.Count(appTsx, "<Route ")
	if routeCount != 3 {
		t.Errorf("App.tsx: expected 3 routes, got %d", routeCount)
	}

	// Verify Home → "/"
	if !strings.Contains(appTsx, `path="/"`) {
		t.Error("App.tsx: Home should route to /")
	}

	t.Logf("Generated %d files to %s", len(expectedFiles), dir)
}
