package quality

import (
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/ir"
)

func TestGenerateTraceabilityMatrix_ReactNode(t *testing.T) {
	app := &ir.Application{
		Name: "TestApp",
		Config: &ir.BuildConfig{
			Frontend: "React with TypeScript",
			Backend:  "Node with Express",
		},
		Data: []*ir.DataModel{
			{Name: "Task", Fields: []*ir.DataField{{Name: "title", Type: "text"}}},
		},
		Pages: []*ir.Page{
			{Name: "Dashboard", Content: []*ir.Action{{Type: "display"}}},
		},
		APIs: []*ir.Endpoint{
			{Name: "CreateTask"},
		},
		Auth: &ir.Auth{Methods: []*ir.AuthMethod{{Type: "jwt"}}},
		Workflows: []*ir.Workflow{
			{Trigger: "user signs up", Steps: []*ir.Action{{Type: "send"}}},
		},
	}

	matrix := generateTraceabilityMatrix(app, app.Config)

	if !strings.Contains(matrix, "# Traceability Matrix") {
		t.Error("missing matrix header")
	}
	if !strings.Contains(matrix, "## Summary") {
		t.Error("missing summary section")
	}
	if !strings.Contains(matrix, "## Details") {
		t.Error("missing details section")
	}

	// Check React file paths
	if !strings.Contains(matrix, "node/src/pages/Dashboard.tsx") {
		t.Error("missing React page file path")
	}
	// Check Node route paths
	if !strings.Contains(matrix, "node/src/routes/create-task.ts") {
		t.Error("missing Node route file path")
	}
	// Check auth middleware
	if !strings.Contains(matrix, "node/src/middleware/auth.ts") {
		t.Error("missing auth middleware path")
	}
	// Check workflow event handler
	if !strings.Contains(matrix, "node/src/events/user-signs-up.ts") {
		t.Error("missing workflow event handler path")
	}
}

func TestGenerateTraceabilityMatrix_GoPython(t *testing.T) {
	app := &ir.Application{
		Name: "TestApp",
		Config: &ir.BuildConfig{
			Backend: "Go with Gin",
		},
		APIs: []*ir.Endpoint{
			{Name: "GetTasks"},
		},
	}

	matrix := generateTraceabilityMatrix(app, app.Config)

	if !strings.Contains(matrix, "go/handlers/gettasks.go") {
		t.Error("missing Go handler file path")
	}
}

func TestGenerateTraceabilityMatrix_WithPolicies(t *testing.T) {
	app := &ir.Application{
		Name: "TestApp",
		Config: &ir.BuildConfig{
			Backend: "Node with Express",
		},
		Policies: []*ir.Policy{
			{Name: "Admin", Permissions: []*ir.PolicyRule{{Text: "can manage all"}}},
		},
	}

	matrix := generateTraceabilityMatrix(app, app.Config)
	if !strings.Contains(matrix, "admin-policy.ts") {
		t.Error("missing policy middleware file path")
	}
}

func TestBuildTraceEntries_StatusCovered(t *testing.T) {
	app := &ir.Application{
		Config: &ir.BuildConfig{
			Frontend: "React with TypeScript",
			Backend:  "Node with Express",
		},
		Data: []*ir.DataModel{
			{Name: "Task", Fields: []*ir.DataField{{Name: "title"}}},
		},
	}

	entries := buildTraceEntries(app, app.Config)

	if len(entries) == 0 {
		t.Fatal("expected at least one trace entry")
	}

	// Data models have test files → should be "covered"
	for _, e := range entries {
		if e.Category == "data" && e.Status != "covered" {
			t.Errorf("expected 'covered' status for data entry, got %s", e.Status)
		}
	}
}

func TestBuildTraceEntries_WorkflowPartial(t *testing.T) {
	app := &ir.Application{
		Config: &ir.BuildConfig{
			Backend: "Node with Express",
		},
		Workflows: []*ir.Workflow{
			{Trigger: "task overdue"},
		},
	}

	entries := buildTraceEntries(app, app.Config)

	for _, e := range entries {
		if e.Category == "workflow" && e.Status != "partial" {
			t.Errorf("expected 'partial' status for workflow (no test file), got %s", e.Status)
		}
	}
}

func TestTraceStatus(t *testing.T) {
	tests := []struct {
		generated []string
		testFiles []string
		expect    string
	}{
		{[]string{"file.ts"}, []string{"file.test.ts"}, "covered"},
		{[]string{"file.ts"}, nil, "partial"},
		{nil, nil, "untested"},
	}

	for _, tt := range tests {
		got := traceStatus(tt.generated, tt.testFiles)
		if got != tt.expect {
			t.Errorf("traceStatus(%v, %v) = %q, want %q", tt.generated, tt.testFiles, got, tt.expect)
		}
	}
}

func TestRenderTraceabilitySection(t *testing.T) {
	entries := []TraceEntry{
		{Category: "data", Status: "covered"},
		{Category: "page", Status: "covered"},
		{Category: "api", Status: "partial"},
		{Category: "workflow", Status: "untested"},
	}

	section := renderTraceabilitySection(entries)
	if !strings.Contains(section, "## Traceability") {
		t.Error("missing section header")
	}
	if !strings.Contains(section, "4 statements") {
		t.Error("missing total count")
	}
	if !strings.Contains(section, "2 covered") {
		t.Error("missing covered count")
	}
	if !strings.Contains(section, "1 partial") {
		t.Error("missing partial count")
	}
	if !strings.Contains(section, "1 untested") {
		t.Error("missing untested count")
	}
}

func TestGenerateTraceabilityMatrix_NoConfig(t *testing.T) {
	app := &ir.Application{
		Name: "TestApp",
		Data: []*ir.DataModel{
			{Name: "Task"},
		},
	}

	// Should not panic with nil config
	matrix := generateTraceabilityMatrix(app, nil)
	if !strings.Contains(matrix, "# Traceability Matrix") {
		t.Error("missing matrix header with nil config")
	}
}
