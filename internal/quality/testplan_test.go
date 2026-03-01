package quality

import (
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/ir"
)

func TestGenerateTestPlan_WithPages(t *testing.T) {
	app := &ir.Application{
		Name: "TestApp",
		Pages: []*ir.Page{
			{
				Name: "Dashboard",
				Content: []*ir.Action{
					{Type: "display", Text: "show task list"},
					{Type: "interact", Text: "click task to view details"},
					{Type: "navigate", Text: "go to TaskDetail", Value: "TaskDetail"},
				},
			},
		},
	}

	plan := generateTestPlan(app)

	if !strings.Contains(plan, "# QA Test Plan") {
		t.Error("missing plan header")
	}
	if !strings.Contains(plan, "## Page Tests") {
		t.Error("missing page tests section")
	}
	if !strings.Contains(plan, "### Dashboard") {
		t.Error("missing Dashboard page section")
	}
	if !strings.Contains(plan, "Page Dashboard loads without errors") {
		t.Error("missing page load test")
	}
	if !strings.Contains(plan, "show task list") {
		t.Error("missing display action test")
	}
	if !strings.Contains(plan, "click task to view details") {
		t.Error("missing interaction test")
	}
	if !strings.Contains(plan, "Desktop") {
		t.Error("missing responsive test")
	}
	if !strings.Contains(plan, "Empty state") {
		t.Error("missing state test")
	}
}

func TestGenerateTestPlan_WithAPIs(t *testing.T) {
	app := &ir.Application{
		Name: "TestApp",
		APIs: []*ir.Endpoint{
			{
				Name: "CreateTask",
				Auth: true,
				Params: []*ir.Param{{Name: "title"}},
				Validation: []*ir.ValidationRule{
					{Field: "title", Rule: "not_empty"},
					{Field: "title", Rule: "max_length", Value: "100"},
				},
			},
		},
	}

	plan := generateTestPlan(app)

	if !strings.Contains(plan, "## API Tests") {
		t.Error("missing API tests section")
	}
	if !strings.Contains(plan, "### CreateTask") {
		t.Error("missing endpoint section")
	}
	if !strings.Contains(plan, "succeeds with valid request") {
		t.Error("missing happy path test")
	}
	if !strings.Contains(plan, "title fails not_empty") {
		t.Error("missing validation test for not_empty")
	}
	if !strings.Contains(plan, "title fails max_length") {
		t.Error("missing validation test for max_length")
	}
	if !strings.Contains(plan, "Returns 401 without auth token") {
		t.Error("missing auth test")
	}
}

func TestGenerateTestPlan_WithPolicies(t *testing.T) {
	app := &ir.Application{
		Name: "TestApp",
		APIs: []*ir.Endpoint{
			{
				Name: "DeleteTask",
				Auth: true,
				Steps: []*ir.Action{
					{Type: "delete", Target: "Task"},
				},
			},
		},
		Policies: []*ir.Policy{
			{
				Name: "Admin",
				Permissions: []*ir.PolicyRule{
					{Text: "can delete all Tasks"},
				},
			},
			{
				Name: "Member",
				Restrictions: []*ir.PolicyRule{
					{Text: "cannot delete other users' Tasks"},
				},
			},
		},
	}

	plan := generateTestPlan(app)

	if !strings.Contains(plan, "Authorization Tests") {
		t.Error("missing authorization tests section")
	}
	if !strings.Contains(plan, "Admin role: allowed") {
		t.Error("missing admin permission test")
	}
	if !strings.Contains(plan, "Member role: denied") {
		t.Error("missing member restriction test")
	}
}

func TestGenerateTestPlan_WithWorkflows(t *testing.T) {
	app := &ir.Application{
		Name: "TestApp",
		Workflows: []*ir.Workflow{
			{
				Trigger: "user signs up",
				Steps: []*ir.Action{
					{Type: "send", Text: "send welcome email"},
					{Type: "create", Text: "create default workspace"},
				},
			},
		},
	}

	plan := generateTestPlan(app)

	if !strings.Contains(plan, "## Workflow Tests") {
		t.Error("missing workflow tests section")
	}
	if !strings.Contains(plan, "user signs up") {
		t.Error("missing workflow trigger")
	}
	if !strings.Contains(plan, "send welcome email") {
		t.Error("missing workflow step test")
	}
}

func TestGenerateTestPlan_CrossCuttingAuth(t *testing.T) {
	app := &ir.Application{
		Name: "TestApp",
		Auth: &ir.Auth{
			Methods: []*ir.AuthMethod{{Type: "jwt"}},
		},
		Pages: []*ir.Page{
			{
				Name: "Home",
				Content: []*ir.Action{
					{Type: "navigate", Text: "go to Profile", Value: "Profile"},
				},
			},
			{
				Name:    "Profile",
				Content: []*ir.Action{{Type: "display", Text: "show user info"}},
			},
		},
	}

	plan := generateTestPlan(app)

	if !strings.Contains(plan, "Authentication Flow") {
		t.Error("missing auth flow section")
	}
	if !strings.Contains(plan, "User can log in") {
		t.Error("missing login test")
	}
	if !strings.Contains(plan, "Navigation") {
		t.Error("missing navigation section")
	}
}

func TestGenerateTestPlan_EmptyApp(t *testing.T) {
	app := &ir.Application{Name: "EmptyApp"}

	plan := generateTestPlan(app)

	if !strings.Contains(plan, "# QA Test Plan") {
		t.Error("missing plan header for empty app")
	}
	if !strings.Contains(plan, "EmptyApp") {
		t.Error("missing app name")
	}
}
