package python

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

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"GetTasks", "get_tasks"},
		{"Dashboard", "dashboard"},
		{"SignUp", "sign_up"},
		{"userRole", "user_role"},
		{"Sign Up", "sign_up"},
		{"user-role", "user_role"},
	}
	for _, tt := range tests {
		got := toSnakeCase(tt.input)
		if got != tt.want {
			t.Errorf("toSnakeCase(%q): got %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestHttpMethod(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"GetTasks", "get"},
		{"CreateTask", "post"},
		{"UpdateTask", "put"},
		{"DeleteTask", "delete"},
		{"SignUp", "post"},
		{"Login", "post"},
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

func TestPythonType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"text", "str"},
		{"email", "str"},
		{"url", "str"},
		{"file", "str"},
		{"image", "str"},
		{"number", "int"},
		{"decimal", "float"},
		{"boolean", "bool"},
		{"date", "datetime.date"},
		{"datetime", "datetime.datetime"},
		{"json", "dict"},
		{"enum", "str"},
		{"unknown", "str"},
	}
	for _, tt := range tests {
		got := pythonType(tt.input)
		if got != tt.want {
			t.Errorf("pythonType(%q): got %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSqlAlchemyType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"text", "String"},
		{"number", "Integer"},
		{"decimal", "Float"},
		{"boolean", "Boolean"},
		{"date", "Date"},
		{"datetime", "DateTime"},
		{"json", "JSON"},
		{"enum", "String"},
		{"unknown", "String"},
	}
	for _, tt := range tests {
		got := sqlAlchemyType(tt.input)
		if got != tt.want {
			t.Errorf("sqlAlchemyType(%q): got %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestInferModelFromAction(t *testing.T) {
	tests := []struct {
		text string
		want string
	}{
		{"create a User with the given fields", "User"},
		{"fetch all tasks for the current user", "Task"},
		{"update the Task", "Task"},
		{"delete the Task", "Task"},
	}
	for _, tt := range tests {
		got := inferModelFromAction(tt.text)
		if got != tt.want {
			t.Errorf("inferModelFromAction(%q): got %q, want %q", tt.text, got, tt.want)
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
		"requirements.txt",
		"main.py",
		"models.py",
		"schemas.py",
		"routes.py",
		"auth.py",
		"database.py",
		"alembic.ini",
		filepath.Join("alembic", "env.py"),
		filepath.Join("alembic", "script.py.mako"),
		filepath.Join("alembic", "versions", "initial.py"),
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
		"requirements.txt",
		"main.py",
		"models.py",
		"schemas.py",
		"routes.py",
		"auth.py",
		"database.py",
		"alembic.ini",
		filepath.Join("alembic", "env.py"),
		filepath.Join("alembic", "script.py.mako"),
		filepath.Join("alembic", "versions", "initial.py"),
	}
	for _, f := range coreFiles {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}

	// Verify main.py
	mainContent, err := os.ReadFile(filepath.Join(dir, "main.py"))
	if err != nil {
		t.Fatalf("reading main.py: %v", err)
	}
	mainStr := string(mainContent)
	if !strings.Contains(mainStr, "FastAPI(title=\"TaskFlow\")") {
		t.Error("main.py: missing TaskFlow app name")
	}

	// Verify models.py has 4 models
	modelsContent, err := os.ReadFile(filepath.Join(dir, "models.py"))
	if err != nil {
		t.Fatalf("reading models.py: %v", err)
	}
	modelsStr := string(modelsContent)
	modelCount := strings.Count(modelsStr, "(Base):")
	if modelCount != 4 {
		t.Errorf("models.py: expected 4 models, got %d", modelCount)
	}
	for _, name := range []string{"User", "Task", "Tag", "TaskTag"} {
		if !strings.Contains(modelsStr, "class "+name+"(Base):") {
			t.Errorf("models.py: missing model %s", name)
		}
	}

	// Verify requirements.txt
	reqsContent, err := os.ReadFile(filepath.Join(dir, "requirements.txt"))
	if err != nil {
		t.Fatalf("reading requirements.txt: %v", err)
	}
	reqsStr := string(reqsContent)
	if !strings.Contains(reqsStr, "fastapi") || !strings.Contains(reqsStr, "sqlalchemy") || !strings.Contains(reqsStr, "alembic") {
		t.Error("requirements.txt: missing core dependencies")
	}

	// Verify routes.py
	routesContent, err := os.ReadFile(filepath.Join(dir, "routes.py"))
	if err != nil {
		t.Fatalf("reading routes.py: %v", err)
	}
	routesStr := string(routesContent)
	if !strings.Contains(routesStr, "def sign_up(") || !strings.Contains(routesStr, "def get_tasks(") {
		t.Error("routes.py: missing expected route functions")
	}
}
