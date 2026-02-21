package gobackend

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/ir"
	"github.com/barun-bash/human/internal/parser"
)

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"GetTasks", "GetTasks"},
		{"signUp", "SignUp"},
		{"login", "Login"},
		{"", ""},
		{"Sign Up", "SignUp"},
		{"user_role", "UserRole"},
	}
	for _, tt := range tests {
		got := toPascalCase(tt.input)
		if got != tt.want {
			t.Errorf("toPascalCase(%q): got %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"GetTasks", "getTasks"},
		{"Dashboard", "dashboard"},
		{"SignUp", "signUp"},
		{"userRole", "userRole"},
		{"Sign Up", "signUp"},
		{"user-role", "userRole"},
	}
	for _, tt := range tests {
		got := toCamelCase(tt.input)
		if got != tt.want {
			t.Errorf("toCamelCase(%q): got %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestHttpMethod(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"GetTasks", "GET"},
		{"CreateTask", "POST"},
		{"UpdateTask", "PUT"},
		{"DeleteTask", "DELETE"},
		{"SignUp", "POST"},
		{"Login", "POST"},
	}
	for _, tt := range tests {
		got := httpMethod(tt.name)
		if got != tt.want {
			t.Errorf("httpMethod(%q): got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestRoutePath(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"GetTasks", "/tasks"},
		{"CreateTask", "/task"},
		{"UpdateTask", "/task"},
		{"DeleteTask", "/task"},
		{"SignUp", "/sign-up"},
		{"Login", "/login"},
		{"GetProfile", "/profile"},
	}
	for _, tt := range tests {
		got := routePath(tt.name)
		if got != tt.want {
			t.Errorf("routePath(%q): got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestGoType(t *testing.T) {
	tests := []struct {
		input    string
		required bool
		want     string
	}{
		{"text", true, "string"},
		{"email", true, "string"},
		{"number", true, "int"},
		{"decimal", true, "float64"},
		{"boolean", true, "bool"},
		{"date", true, "time.Time"},
		{"datetime", true, "time.Time"},
		{"json", true, "map[string]any"},
		{"enum", true, "string"},
		{"text", false, "*string"},
		{"number", false, "*int"},
	}
	for _, tt := range tests {
		got := goType(tt.input, tt.required)
		if got != tt.want {
			t.Errorf("goType(%q, %v): got %q, want %q", tt.input, tt.required, got, tt.want)
		}
	}
}

func TestGenerateWritesFiles(t *testing.T) {
	app := &ir.Application{
		Name:     "TestApp",
		Platform: "web",
		Database: &ir.DatabaseConfig{Engine: "PostgreSQL"},
		Data: []*ir.DataModel{
			{Name: "User", Fields: []*ir.DataField{{Name: "email", Type: "email", Required: true}}},
		},
		APIs: []*ir.Endpoint{
			{Name: "SignUp", Params: []*ir.Param{{Name: "email"}}},
			{Name: "GetUsers", Auth: true},
		},
		Auth: &ir.Auth{
			Methods: []*ir.AuthMethod{{Type: "jwt", Config: map[string]string{"expiration": "7 days"}}},
		},
		ErrorHandlers: []*ir.ErrorHandler{
			{Condition: "test error", Steps: []*ir.Action{{Type: "retry", Text: "retry 3 times"}}},
		},
	}

	dir := t.TempDir()
	g := Generator{}
	if err := g.Generate(app, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	expectedFiles := []string{
		"go.mod",
		"main.go",
		"config/config.go",
		"database/database.go",
		"models/models.go",
		"dto/dto.go",
		"middleware/auth.go",
		"handlers/handlers.go",
		"routes/routes.go",
		"migrations/initial.sql",
	}

	for _, f := range expectedFiles {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
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

	coreFiles := []string{
		"go.mod",
		"main.go",
		"config/config.go",
		"database/database.go",
		"models/models.go",
		"dto/dto.go",
		"middleware/auth.go",
		"handlers/handlers.go",
		"routes/routes.go",
	}
	for _, f := range coreFiles {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}

	modContent, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatalf("reading go.mod: %v", err)
	}
	modStr := string(modContent)
	if !strings.Contains(modStr, "module taskflow") {
		t.Error("go.mod: missing taskflow module name")
	}
	if !strings.Contains(modStr, "gin-gonic") || !strings.Contains(modStr, "gorm.io") {
		t.Error("go.mod: missing core dependencies")
	}

	modelsContent, err := os.ReadFile(filepath.Join(dir, "models", "models.go"))
	if err != nil {
		t.Fatalf("reading models.go: %v", err)
	}
	modelsStr := string(modelsContent)
	modelCount := strings.Count(modelsStr, "type ")
	if modelCount != 4 {
		t.Errorf("models.go: expected 4 models, got %d", modelCount)
	}
	for _, name := range []string{"User", "Task", "Tag", "TaskTag"} {
		if !strings.Contains(modelsStr, "type "+name+" struct {") {
			t.Errorf("models.go: missing model %s", name)
		}
	}

	handlersContent, err := os.ReadFile(filepath.Join(dir, "handlers", "handlers.go"))
	if err != nil {
		t.Fatalf("reading handlers.go: %v", err)
	}
	handlersStr := string(handlersContent)
	if !strings.Contains(handlersStr, "func SignUp(") || !strings.Contains(handlersStr, "func GetTasks(") {
		t.Error("handlers.go: missing expected handler functions")
	}

	routesContent, err := os.ReadFile(filepath.Join(dir, "routes", "routes.go"))
	if err != nil {
		t.Fatalf("reading routes.go: %v", err)
	}
	routesStr := string(routesContent)
	if !strings.Contains(routesStr, "api.POST(\"/sign-up\", handlers.SignUp(db, cfg))") {
		t.Error("routes.go: missing sign-up route without auth")
	}
	if !strings.Contains(routesStr, "api.GET(\"/tasks\", middleware.RequireAuth(db, cfg), handlers.GetTasks(db, cfg))") {
		t.Error("routes.go: missing tasks route with auth")
	}
}
