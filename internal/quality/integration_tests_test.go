package quality

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/ir"
)

func TestGenerateIntegrationTests(t *testing.T) {
	app := exampleApp(t)
	dir := t.TempDir()

	count, err := generateIntegrationTests(app, dir)
	if err != nil {
		t.Fatalf("generateIntegrationTests: %v", err)
	}

	if count == 0 {
		t.Fatal("expected integration tests, got 0")
	}

	// Verify integration.test.ts exists
	path := filepath.Join(dir, "integration.test.ts")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("expected integration.test.ts to exist")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}

	if !strings.Contains(string(content), "supertest") {
		t.Error("missing supertest import")
	}
}

func TestGenerateIntegrationTests_AuthFlow(t *testing.T) {
	app := &ir.Application{
		APIs: []*ir.Endpoint{
			{Name: "SignUp", Params: []*ir.Param{{Name: "email"}, {Name: "password"}}},
			{Name: "Login", Params: []*ir.Param{{Name: "email"}, {Name: "password"}}},
			{Name: "GetTasks", Auth: true},
		},
	}
	dir := t.TempDir()

	count, err := generateIntegrationTests(app, dir)
	if err != nil {
		t.Fatalf("generateIntegrationTests: %v", err)
	}

	if count < 3 {
		t.Errorf("expected at least 3 auth flow tests, got %d", count)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "integration.test.ts"))
	s := string(content)

	if !strings.Contains(s, "sign up a new user") {
		t.Error("missing signup test")
	}
	if !strings.Contains(s, "login and receive token") {
		t.Error("missing login test")
	}
	if !strings.Contains(s, "authToken") {
		t.Error("missing authToken variable")
	}
}

func TestGenerateIntegrationTests_CRUDCycle(t *testing.T) {
	app := &ir.Application{
		Data: []*ir.DataModel{
			{Name: "Task"},
		},
		APIs: []*ir.Endpoint{
			{Name: "CreateTask", Auth: true, Params: []*ir.Param{{Name: "title"}}},
			{Name: "GetTask", Auth: true},
			{Name: "UpdateTask", Auth: true, Params: []*ir.Param{{Name: "title"}}},
			{Name: "DeleteTask", Auth: true},
		},
	}
	dir := t.TempDir()

	count, err := generateIntegrationTests(app, dir)
	if err != nil {
		t.Fatalf("generateIntegrationTests: %v", err)
	}

	if count < 4 {
		t.Errorf("expected at least 4 CRUD tests, got %d", count)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "integration.test.ts"))
	s := string(content)

	if !strings.Contains(s, "create a Task") {
		t.Error("missing create test")
	}
	if !strings.Contains(s, "read Task") {
		t.Error("missing read test")
	}
	if !strings.Contains(s, "update Task") {
		t.Error("missing update test")
	}
	if !strings.Contains(s, "delete Task") {
		t.Error("missing delete test")
	}
}

func TestGenerateIntegrationTests_AuthRejection(t *testing.T) {
	app := &ir.Application{
		APIs: []*ir.Endpoint{
			{Name: "CreateTask", Auth: true},
			{Name: "GetTasks", Auth: true},
		},
	}
	dir := t.TempDir()

	count, err := generateIntegrationTests(app, dir)
	if err != nil {
		t.Fatalf("generateIntegrationTests: %v", err)
	}

	if count < 2 {
		t.Errorf("expected at least 2 auth rejection tests, got %d", count)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "integration.test.ts"))
	s := string(content)

	if !strings.Contains(s, "reject unauthenticated") {
		t.Error("missing auth rejection test")
	}
	if !strings.Contains(s, "401") {
		t.Error("missing 401 status check")
	}
}

func TestFindEndpoint(t *testing.T) {
	app := &ir.Application{
		APIs: []*ir.Endpoint{
			{Name: "SignUp"},
			{Name: "CreateTask"},
			{Name: "GetTasks"},
		},
	}

	tests := []struct {
		name   string
		found  bool
	}{
		{"SignUp", true},
		{"signup", true},       // case-insensitive
		{"SIGNUP", true},       // case-insensitive
		{"CreateTask", true},
		{"createtask", true},
		{"NonExistent", false},
	}

	for _, tt := range tests {
		ep := findEndpoint(app, tt.name)
		if tt.found && ep == nil {
			t.Errorf("findEndpoint(%q) = nil, want found", tt.name)
		}
		if !tt.found && ep != nil {
			t.Errorf("findEndpoint(%q) = %v, want nil", tt.name, ep.Name)
		}
	}
}

func TestGenerateIntegrationTests_NoEndpoints(t *testing.T) {
	app := &ir.Application{}
	dir := t.TempDir()

	count, err := generateIntegrationTests(app, dir)
	if err != nil {
		t.Fatalf("generateIntegrationTests: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0 tests for empty app, got %d", count)
	}
}

func TestGenerateIntegrationTests_CRUDNeedsMultipleEndpoints(t *testing.T) {
	// Model with only Create endpoint — no CRUD cycle
	app := &ir.Application{
		Data: []*ir.DataModel{
			{Name: "Orphan"},
		},
		APIs: []*ir.Endpoint{
			{Name: "CreateOrphan", Auth: true, Params: []*ir.Param{{Name: "name"}}},
		},
	}
	dir := t.TempDir()

	count, err := generateIntegrationTests(app, dir)
	if err != nil {
		t.Fatalf("generateIntegrationTests: %v", err)
	}

	// Should have auth rejection tests but no CRUD cycle (only Create, no Get/Update/Delete)
	content, _ := os.ReadFile(filepath.Join(dir, "integration.test.ts"))
	s := string(content)

	if strings.Contains(s, "CRUD cycle") {
		t.Error("should not generate CRUD cycle for model with only Create endpoint")
	}
	_ = count
}
