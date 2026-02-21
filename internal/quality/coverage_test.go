package quality

import (
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/ir"
)

func TestCalculateCoverage_Full(t *testing.T) {
	app := &ir.Application{
		APIs: []*ir.Endpoint{
			{Name: "CreateUser"},
			{Name: "GetTasks"},
		},
		Pages: []*ir.Page{
			{Name: "Home"},
		},
		Data: []*ir.DataModel{
			{Name: "User", Fields: []*ir.DataField{
				{Name: "name", Type: "text"},
				{Name: "email", Type: "email"},
			}},
		},
	}

	result := &Result{
		TestFiles:          2, // covers both endpoints
		ComponentTestFiles: 1, // covers the page
	}

	cov := calculateCoverage(app, result)

	// Endpoints: 2/2, Pages: 1/1, Fields: 2/2 (User has CreateUser)
	if cov.EndpointsTested != 2 || cov.EndpointsTotal != 2 {
		t.Errorf("endpoints: got %d/%d, want 2/2", cov.EndpointsTested, cov.EndpointsTotal)
	}
	if cov.PagesTested != 1 || cov.PagesTotal != 1 {
		t.Errorf("pages: got %d/%d, want 1/1", cov.PagesTested, cov.PagesTotal)
	}
	if cov.FieldsTested != 2 || cov.FieldsTotal != 2 {
		t.Errorf("fields: got %d/%d, want 2/2", cov.FieldsTested, cov.FieldsTotal)
	}
	if cov.Overall != 100 {
		t.Errorf("overall: got %.0f%%, want 100%%", cov.Overall)
	}
}

func TestCalculateCoverage_Partial(t *testing.T) {
	app := &ir.Application{
		APIs: []*ir.Endpoint{
			{Name: "CreateUser"},
			{Name: "GetTasks"},
			{Name: "DeleteTask"},
		},
		Pages: []*ir.Page{
			{Name: "Home"},
			{Name: "Dashboard"},
		},
		Data: []*ir.DataModel{
			{Name: "User", Fields: []*ir.DataField{
				{Name: "name", Type: "text"},
			}},
			{Name: "Task", Fields: []*ir.DataField{
				{Name: "title", Type: "text"},
			}},
		},
	}

	result := &Result{
		TestFiles:          2, // only 2 of 3 endpoints
		ComponentTestFiles: 1, // only 1 of 2 pages
	}

	cov := calculateCoverage(app, result)

	if cov.Overall >= 100 {
		t.Errorf("expected < 100%% coverage, got %.0f%%", cov.Overall)
	}
	if cov.EndpointsTested != 2 {
		t.Errorf("endpoints tested: got %d, want 2", cov.EndpointsTested)
	}
}

func TestCalculateCoverage_Empty(t *testing.T) {
	app := &ir.Application{}
	result := &Result{}

	cov := calculateCoverage(app, result)

	if cov.Overall != 0 {
		t.Errorf("expected 0%% coverage for empty app, got %.0f%%", cov.Overall)
	}
	// Should not panic
	if cov.EndpointsTotal != 0 {
		t.Errorf("expected 0 endpoints total, got %d", cov.EndpointsTotal)
	}
}

func TestRenderCoverageSection_Below90(t *testing.T) {
	cov := &CoverageReport{
		EndpointsTested: 1,
		EndpointsTotal:  3,
		PagesTested:     1,
		PagesTotal:      2,
		FieldsTested:    2,
		FieldsTotal:     5,
		Overall:         40,
	}

	section := renderCoverageSection(cov)

	if !strings.Contains(section, "## Test Coverage") {
		t.Error("missing coverage header")
	}
	if !strings.Contains(section, "Warning") {
		t.Error("missing warning for below-90% coverage")
	}
	if !strings.Contains(section, "below 90%") {
		t.Error("missing specific 90% threshold message")
	}
}

func TestRenderCoverageSection_Above90(t *testing.T) {
	cov := &CoverageReport{
		EndpointsTested: 5,
		EndpointsTotal:  5,
		PagesTested:     3,
		PagesTotal:      3,
		FieldsTested:    10,
		FieldsTotal:     10,
		Overall:         100,
	}

	section := renderCoverageSection(cov)

	if !strings.Contains(section, "## Test Coverage") {
		t.Error("missing coverage header")
	}
	if strings.Contains(section, "Warning") {
		t.Error("should not have warning for 100% coverage")
	}
}

func TestHasCreateEndpoint(t *testing.T) {
	app := &ir.Application{
		APIs: []*ir.Endpoint{
			{Name: "CreateUser"},
			{Name: "CreateTask"},
			{Name: "GetTasks"},
		},
	}

	tests := []struct {
		model  string
		expect bool
	}{
		{"User", true},
		{"Task", true},
		{"Comment", false},
		{"user", true}, // case-insensitive
	}

	for _, tt := range tests {
		got := hasCreateEndpoint(app, tt.model)
		if got != tt.expect {
			t.Errorf("hasCreateEndpoint(%q) = %v, want %v", tt.model, got, tt.expect)
		}
	}
}

func TestCountTestedFields(t *testing.T) {
	app := &ir.Application{
		Data: []*ir.DataModel{
			{Name: "User", Fields: []*ir.DataField{
				{Name: "name", Type: "text"},
				{Name: "email", Type: "email"},
			}},
			{Name: "Task", Fields: []*ir.DataField{
				{Name: "title", Type: "text"},
			}},
			{Name: "AuditLog", Fields: []*ir.DataField{
				{Name: "action", Type: "text"},
				{Name: "timestamp", Type: "datetime"},
			}},
		},
		APIs: []*ir.Endpoint{
			{Name: "CreateUser"},
			{Name: "CreateTask"},
			// No CreateAuditLog
		},
	}

	got := countTestedFields(app)
	// User (2 fields) + Task (1 field) = 3
	if got != 3 {
		t.Errorf("countTestedFields = %d, want 3", got)
	}
}

func TestCountAllFields(t *testing.T) {
	app := &ir.Application{
		Data: []*ir.DataModel{
			{Name: "User", Fields: []*ir.DataField{
				{Name: "name", Type: "text"},
				{Name: "email", Type: "email"},
			}},
			{Name: "Task", Fields: []*ir.DataField{
				{Name: "title", Type: "text"},
			}},
		},
	}

	got := countAllFields(app)
	if got != 3 {
		t.Errorf("countAllFields = %d, want 3", got)
	}
}

func TestPct(t *testing.T) {
	tests := []struct {
		tested int
		total  int
		expect float64
	}{
		{5, 10, 50},
		{10, 10, 100},
		{0, 10, 0},
		{0, 0, 0},
	}
	for _, tt := range tests {
		got := pct(tt.tested, tt.total)
		if got != tt.expect {
			t.Errorf("pct(%d, %d) = %f, want %f", tt.tested, tt.total, got, tt.expect)
		}
	}
}
