package quality

import (
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/ir"
)

func TestCheckNPlusOne_Detected(t *testing.T) {
	app := &ir.Application{
		Data: []*ir.DataModel{
			{
				Name: "Task",
				Relations: []*ir.Relation{
					{Kind: "belongs_to", Target: "User"},
				},
			},
		},
		Pages: []*ir.Page{
			{
				Name: "TaskList",
				Content: []*ir.Action{
					{Type: "loop", Target: "Task", Text: "for each task"},
					{Type: "display", Target: "User", Text: "show task user name"},
				},
			},
		},
	}

	findings := checkNPlusOne(app)
	if len(findings) != 1 {
		t.Fatalf("expected 1 N+1 finding, got %d", len(findings))
	}
	if findings[0].Kind != "n-plus-one" {
		t.Errorf("expected n-plus-one kind, got %s", findings[0].Kind)
	}
	if findings[0].Severity != "warning" {
		t.Errorf("expected warning severity, got %s", findings[0].Severity)
	}
}

func TestCheckNPlusOne_NoRelation(t *testing.T) {
	app := &ir.Application{
		Data: []*ir.DataModel{
			{Name: "Task"},
		},
		Pages: []*ir.Page{
			{
				Name: "TaskList",
				Content: []*ir.Action{
					{Type: "loop", Target: "Task", Text: "for each task"},
					{Type: "display", Text: "show task title"},
				},
			},
		},
	}

	findings := checkNPlusOne(app)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for no relations, got %d", len(findings))
	}
}

func TestCheckMissingPagination_Detected(t *testing.T) {
	app := &ir.Application{
		APIs: []*ir.Endpoint{
			{
				Name: "GetTasks",
				Steps: []*ir.Action{
					{Type: "query", Text: "fetch all tasks for current user"},
					{Type: "respond", Text: "respond with tasks"},
				},
			},
		},
	}

	findings := checkMissingPagination(app)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Kind != "missing-pagination" {
		t.Errorf("expected missing-pagination kind, got %s", findings[0].Kind)
	}
}

func TestCheckMissingPagination_HasPagination(t *testing.T) {
	app := &ir.Application{
		APIs: []*ir.Endpoint{
			{
				Name: "GetTasks",
				Params: []*ir.Param{{Name: "page"}, {Name: "limit"}},
				Steps: []*ir.Action{
					{Type: "query", Text: "fetch all tasks"},
				},
			},
		},
	}

	findings := checkMissingPagination(app)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings when pagination params exist, got %d", len(findings))
	}
}

func TestCheckMissingPagination_NoBulkFetch(t *testing.T) {
	app := &ir.Application{
		APIs: []*ir.Endpoint{
			{
				Name: "GetTask",
				Steps: []*ir.Action{
					{Type: "query", Text: "find task by id"},
				},
			},
		},
	}

	findings := checkMissingPagination(app)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for single-item fetch, got %d", len(findings))
	}
}

func TestCheckLargePayloads_Detected(t *testing.T) {
	fields := make([]*ir.DataField, 15)
	for i := range fields {
		fields[i] = &ir.DataField{Name: "field" + string(rune('a'+i)), Type: "text"}
	}

	app := &ir.Application{
		Data: []*ir.DataModel{
			{Name: "User", Fields: fields},
		},
		APIs: []*ir.Endpoint{
			{
				Name: "GetUser",
				Steps: []*ir.Action{
					{Type: "query", Text: "find user by id"},
					{Type: "respond", Text: "respond with the user", Target: "User"},
				},
			},
		},
	}

	findings := checkLargePayloads(app)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Kind != "large-payload" {
		t.Errorf("expected large-payload kind, got %s", findings[0].Kind)
	}
	if !strings.Contains(findings[0].Message, "15 fields") {
		t.Errorf("expected message to mention 15 fields, got: %s", findings[0].Message)
	}
}

func TestCheckLargePayloads_SmallModel(t *testing.T) {
	app := &ir.Application{
		Data: []*ir.DataModel{
			{
				Name: "Task",
				Fields: []*ir.DataField{
					{Name: "title", Type: "text"},
					{Name: "status", Type: "text"},
				},
			},
		},
		APIs: []*ir.Endpoint{
			{
				Name: "GetTask",
				Steps: []*ir.Action{
					{Type: "respond", Target: "Task"},
				},
			},
		},
	}

	findings := checkLargePayloads(app)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for small model, got %d", len(findings))
	}
}

func TestCheckPerformance_CleanApp(t *testing.T) {
	app := &ir.Application{
		Data: []*ir.DataModel{
			{Name: "Task", Fields: []*ir.DataField{{Name: "title", Type: "text"}}},
		},
		Pages: []*ir.Page{
			{Name: "Home", Content: []*ir.Action{{Type: "display", Text: "show tasks"}}},
		},
		APIs: []*ir.Endpoint{
			{
				Name: "GetTask",
				Steps: []*ir.Action{
					{Type: "query", Text: "find task by id"},
					{Type: "respond", Target: "Task"},
				},
			},
		},
	}

	findings := checkPerformance(app)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for clean app, got %d", len(findings))
	}
}

func TestRenderPerformanceReport(t *testing.T) {
	findings := []PerformanceFinding{
		{Kind: "n-plus-one", Severity: "warning", Target: "TaskList", Message: "N+1 query", Fix: "include User"},
		{Kind: "large-payload", Severity: "info", Target: "GetUser", Message: "Large payload", Fix: "select fields"},
	}

	report := renderPerformanceReport(findings)
	if !strings.Contains(report, "# Performance Report") {
		t.Error("missing report header")
	}
	if !strings.Contains(report, "1 warnings, 1 info") {
		t.Error("missing summary counts")
	}
	if !strings.Contains(report, "n-plus-one") {
		t.Error("missing finding row")
	}
}

func TestRenderPerformanceReport_Empty(t *testing.T) {
	report := renderPerformanceReport(nil)
	if !strings.Contains(report, "No performance issues found") {
		t.Error("expected clean message for no findings")
	}
}

func TestRenderPerformanceSection(t *testing.T) {
	findings := []PerformanceFinding{
		{Kind: "missing-pagination", Severity: "warning", Target: "GetTasks", Message: "No pagination"},
	}

	section := renderPerformanceSection(findings)
	if !strings.Contains(section, "## Performance") {
		t.Error("missing section header")
	}
	if !strings.Contains(section, "1 findings") {
		t.Error("missing finding count")
	}
}

func TestCheckMissingIndexes(t *testing.T) {
	app := &ir.Application{
		Data: []*ir.DataModel{
			{
				Name: "Task",
				Fields: []*ir.DataField{
					{Name: "status", Type: "text"},
					{Name: "title", Type: "text"},
				},
			},
		},
		Pages: []*ir.Page{
			{
				Name: "TaskList",
				Content: []*ir.Action{
					{Type: "display", Target: "task", Text: "filter Task by status"},
				},
			},
		},
		Database: &ir.DatabaseConfig{
			Indexes: []*ir.Index{
				{Entity: "Task", Fields: []string{"title"}},
			},
		},
	}

	findings := checkMissingIndexes(app)
	// Should flag status (no index) but not title (has index)
	found := false
	for _, f := range findings {
		if f.Kind == "missing-index" && strings.Contains(f.Message, "status") {
			found = true
		}
	}
	if !found {
		t.Error("expected missing-index finding for status field")
	}
}
