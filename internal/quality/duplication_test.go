package quality

import (
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/ir"
)

func TestCheckDuplicateAPILogic_SameParamsAndModel(t *testing.T) {
	app := &ir.Application{
		APIs: []*ir.Endpoint{
			{
				Name:   "CreateTask",
				Params: []*ir.Param{{Name: "title"}, {Name: "description"}},
				Steps:  []*ir.Action{{Type: "create", Target: "Task"}},
				Validation: []*ir.ValidationRule{
					{Field: "title", Rule: "not_empty"},
				},
			},
			{
				Name:   "UpdateTask",
				Params: []*ir.Param{{Name: "title"}, {Name: "description"}},
				Steps:  []*ir.Action{{Type: "update", Target: "Task"}},
				Validation: []*ir.ValidationRule{
					{Field: "title", Rule: "not_empty"},
				},
			},
		},
	}

	findings := checkDuplicateAPILogic(app)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Kind != "api-duplicate" {
		t.Errorf("expected api-duplicate kind, got %s", findings[0].Kind)
	}
	if !strings.Contains(findings[0].Message, "CreateTask") || !strings.Contains(findings[0].Message, "UpdateTask") {
		t.Error("expected both endpoint names in message")
	}
}

func TestCheckDuplicateAPILogic_DifferentParams(t *testing.T) {
	app := &ir.Application{
		APIs: []*ir.Endpoint{
			{
				Name:   "CreateTask",
				Params: []*ir.Param{{Name: "title"}},
				Steps:  []*ir.Action{{Type: "create", Target: "Task"}},
			},
			{
				Name:   "CreateUser",
				Params: []*ir.Param{{Name: "email"}, {Name: "name"}},
				Steps:  []*ir.Action{{Type: "create", Target: "User"}},
			},
		},
	}

	findings := checkDuplicateAPILogic(app)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for different params, got %d", len(findings))
	}
}

func TestCheckSimilarPages_IdenticalStructure(t *testing.T) {
	app := &ir.Application{
		Pages: []*ir.Page{
			{
				Name: "TaskList",
				Content: []*ir.Action{
					{Type: "display", Target: "task", Text: "show list of tasks"},
					{Type: "interact", Target: "task", Text: "click to view"},
				},
			},
			{
				Name: "TaskArchive",
				Content: []*ir.Action{
					{Type: "display", Target: "task", Text: "show archived tasks"},
					{Type: "interact", Target: "task", Text: "click to restore"},
				},
			},
		},
	}

	findings := checkSimilarPages(app)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Kind != "similar-pages" {
		t.Errorf("expected similar-pages kind, got %s", findings[0].Kind)
	}
}

func TestCheckSimilarPages_DifferentStructure(t *testing.T) {
	app := &ir.Application{
		Pages: []*ir.Page{
			{
				Name: "TaskList",
				Content: []*ir.Action{
					{Type: "display", Target: "task"},
				},
			},
			{
				Name: "UserProfile",
				Content: []*ir.Action{
					{Type: "display", Target: "user"},
					{Type: "interact", Target: "user"},
				},
			},
		},
	}

	findings := checkSimilarPages(app)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for different page structures, got %d", len(findings))
	}
}

func TestCheckRepeatedValidation_ThreeOrMore(t *testing.T) {
	app := &ir.Application{
		APIs: []*ir.Endpoint{
			{
				Name:       "CreateTask",
				Validation: []*ir.ValidationRule{{Field: "title", Rule: "not_empty"}},
			},
			{
				Name:       "UpdateTask",
				Validation: []*ir.ValidationRule{{Field: "title", Rule: "not_empty"}},
			},
			{
				Name:       "CreateProject",
				Validation: []*ir.ValidationRule{{Field: "title", Rule: "not_empty"}},
			},
		},
	}

	findings := checkRepeatedValidation(app)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Kind != "repeated-validation" {
		t.Errorf("expected repeated-validation kind, got %s", findings[0].Kind)
	}
	if !strings.Contains(findings[0].Message, "3 endpoints") {
		t.Errorf("expected message to mention 3 endpoints, got: %s", findings[0].Message)
	}
}

func TestCheckRepeatedValidation_TwoOnly(t *testing.T) {
	app := &ir.Application{
		APIs: []*ir.Endpoint{
			{
				Name:       "CreateTask",
				Validation: []*ir.ValidationRule{{Field: "title", Rule: "not_empty"}},
			},
			{
				Name:       "UpdateTask",
				Validation: []*ir.ValidationRule{{Field: "title", Rule: "not_empty"}},
			},
		},
	}

	findings := checkRepeatedValidation(app)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for only 2 occurrences, got %d", len(findings))
	}
}

func TestCheckDuplication_Full(t *testing.T) {
	app := &ir.Application{
		APIs: []*ir.Endpoint{
			{Name: "GetTasks"},
			{Name: "CreateTask", Params: []*ir.Param{{Name: "title"}}},
		},
		Pages: []*ir.Page{
			{Name: "Home", Content: []*ir.Action{{Type: "display", Text: "show tasks"}}},
		},
	}

	findings := checkDuplication(app)
	// Clean app — no duplications expected
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for clean app, got %d", len(findings))
	}
}

func TestRenderDuplicationSection(t *testing.T) {
	findings := []DuplicationFinding{
		{Kind: "api-duplicate", Items: []string{"CreateTask", "UpdateTask"}, Message: "duplicate logic"},
	}

	section := renderDuplicationSection(findings)
	if !strings.Contains(section, "## Duplication") {
		t.Error("missing section header")
	}
	if !strings.Contains(section, "1 duplication findings") {
		t.Error("missing finding count")
	}
	if !strings.Contains(section, "api-duplicate") {
		t.Error("missing finding kind")
	}
}

func TestRenderDuplicationSection_Empty(t *testing.T) {
	section := renderDuplicationSection(nil)
	if !strings.Contains(section, "No duplication issues found") {
		t.Error("expected clean message for no findings")
	}
}

func TestSameParamSet(t *testing.T) {
	a := []*ir.Param{{Name: "title"}, {Name: "description"}}
	b := []*ir.Param{{Name: "description"}, {Name: "title"}}
	c := []*ir.Param{{Name: "email"}}

	if !sameParamSet(a, b) {
		t.Error("expected same param set for identical names in different order")
	}
	if sameParamSet(a, c) {
		t.Error("expected different param set for different names")
	}
}

func TestSameValidationRules(t *testing.T) {
	a := []*ir.ValidationRule{
		{Field: "title", Rule: "not_empty"},
		{Field: "email", Rule: "valid_email"},
	}
	b := []*ir.ValidationRule{
		{Field: "email", Rule: "valid_email"},
		{Field: "title", Rule: "not_empty"},
	}
	c := []*ir.ValidationRule{
		{Field: "title", Rule: "min_length"},
	}

	if !sameValidationRules(a, b) {
		t.Error("expected same rules for identical rules in different order")
	}
	if sameValidationRules(a, c) {
		t.Error("expected different rules")
	}
	if !sameValidationRules(nil, nil) {
		t.Error("expected equal for both nil")
	}
}
