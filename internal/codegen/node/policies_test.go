package node

import (
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/ir"
)

func TestParseRuleText(t *testing.T) {
	tests := []struct {
		text      string
		action    string
		model     string
		scope     string
		limit     int
		period    string
		condition string
	}{
		{
			text:   "create up to 50 tasks per month",
			action: "create", model: "task", limit: 50, period: "month",
		},
		{
			text:   "view only their own tasks",
			action: "view", model: "task", scope: "own",
		},
		{
			text:   "edit only their own tasks",
			action: "edit", model: "task", scope: "own",
		},
		{
			text:      "delete completed tasks",
			action:    "delete", model: "task",
			condition: "completed",
		},
		{
			text:   "export data",
			action: "export", model: "data",
		},
		{
			text:   "create unlimited tasks",
			action: "create", model: "task",
		},
		{
			text:   "delete any of their own tasks",
			action: "delete", model: "task", scope: "own",
		},
		{
			text:   "edit any task",
			action: "edit", model: "task", scope: "any",
		},
		{
			text:   "delete any task",
			action: "delete", model: "task", scope: "any",
		},
		{
			text:   "view all users and their data",
			action: "view", model: "user", scope: "all",
		},
		{
			text:   "view system analytics",
			action: "view", model: "analytics",
		},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			r := parseRuleText(tt.text)
			if r.Action != tt.action {
				t.Errorf("action: got %q, want %q", r.Action, tt.action)
			}
			if r.Model != tt.model {
				t.Errorf("model: got %q, want %q", r.Model, tt.model)
			}
			if r.Scope != tt.scope {
				t.Errorf("scope: got %q, want %q", r.Scope, tt.scope)
			}
			if r.Limit != tt.limit {
				t.Errorf("limit: got %d, want %d", r.Limit, tt.limit)
			}
			if r.Period != tt.period {
				t.Errorf("period: got %q, want %q", r.Period, tt.period)
			}
			if r.Condition != tt.condition {
				t.Errorf("condition: got %q, want %q", r.Condition, tt.condition)
			}
		})
	}
}

func TestParseRuleTextEdgeCases(t *testing.T) {
	tests := []struct {
		text   string
		action string
		model  string
	}{
		{"", "", ""},                              // empty text
		{"create", "create", ""},                  // action only
		{"create up to 50 per month", "create", ""}, // "up to" without model
		{"delete tasks that are completed", "delete", "task"}, // "that" in rule
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			r := parseRuleText(tt.text)
			if r.Action != tt.action {
				t.Errorf("action: got %q, want %q", r.Action, tt.action)
			}
			if r.Model != tt.model {
				t.Errorf("model: got %q, want %q", r.Model, tt.model)
			}
		})
	}
}

func TestSingularize(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"tasks", "task"},
		{"users", "user"},
		{"policies", "policy"},
		{"categories", "category"},
		{"data", "data"},
		{"analytics", "analytics"},
		{"task", "task"},
		{"boxes", "box"},
		{"classes", "class"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := singularize(tt.input)
			if got != tt.want {
				t.Errorf("singularize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestInferRouteAction(t *testing.T) {
	tests := []struct {
		name, want string
	}{
		{"CreateTask", "create"},
		{"GetTasks", "view"},
		{"ListUsers", "view"},
		{"FetchProfile", "view"},
		{"UpdateTask", "edit"},
		{"DeleteTask", "delete"},
		{"RemoveTag", "delete"},
		{"SignUp", ""},
		{"Login", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferRouteAction(tt.name)
			if got != tt.want {
				t.Errorf("inferRouteAction(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestInferRouteModel(t *testing.T) {
	tests := []struct {
		name, want string
	}{
		{"CreateTask", "task"},
		{"GetTasks", "task"},
		{"UpdateTask", "task"},
		{"DeleteTask", "task"},
		{"GetProfile", "profile"},
		{"ListUsers", "user"},
		{"SignUp", ""},
		{"Login", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferRouteModel(tt.name)
			if got != tt.want {
				t.Errorf("inferRouteModel(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestGeneratePolicies(t *testing.T) {
	app := &ir.Application{
		Policies: []*ir.Policy{
			{
				Name: "FreeUser",
				Permissions: []*ir.PolicyRule{
					{Text: "create up to 50 tasks per month"},
					{Text: "view only their own tasks"},
				},
				Restrictions: []*ir.PolicyRule{
					{Text: "delete completed tasks"},
					{Text: "export data"},
				},
			},
			{
				Name: "Admin",
				Permissions: []*ir.PolicyRule{
					{Text: "edit any task"},
					{Text: "delete any task"},
				},
			},
		},
	}

	output := generatePolicies(app)

	// Check structure
	if !strings.Contains(output, "export const policies") {
		t.Error("missing policies export")
	}
	if !strings.Contains(output, "FreeUser:") {
		t.Error("missing FreeUser policy")
	}
	if !strings.Contains(output, "Admin:") {
		t.Error("missing Admin policy")
	}

	// Check parsed rule values
	if !strings.Contains(output, "action: 'create', model: 'task', scope: '', limit: 50, period: 'month'") {
		t.Error("FreeUser create rule not correctly parsed")
	}
	if !strings.Contains(output, "action: 'view', model: 'task', scope: 'own'") {
		t.Error("FreeUser view rule not correctly parsed")
	}
	if !strings.Contains(output, "action: 'delete', model: 'task', scope: ''") {
		t.Error("FreeUser delete restriction not found")
	}
	if !strings.Contains(output, "action: 'edit', model: 'task', scope: 'any'") {
		t.Error("Admin edit rule not correctly parsed")
	}
}

func TestGenerateAuthorize(t *testing.T) {
	app := &ir.Application{
		Policies: []*ir.Policy{
			{Name: "FreeUser"},
		},
	}

	output := generateAuthorize(app)

	if !strings.Contains(output, "export function authorize") {
		t.Error("missing authorize function export")
	}
	if !strings.Contains(output, "import { policies }") {
		t.Error("missing policies import")
	}
	if !strings.Contains(output, "req.authzScope") {
		t.Error("missing authzScope assignment")
	}
	if !strings.Contains(output, "restrictions.find") {
		t.Error("missing restriction check logic")
	}
}

func TestGenerateRouteWithAuthorize(t *testing.T) {
	app := &ir.Application{
		Policies: []*ir.Policy{
			{Name: "FreeUser", Permissions: []*ir.PolicyRule{{Text: "create task"}}},
		},
		APIs: []*ir.Endpoint{
			{
				Name: "CreateTask",
				Auth: true,
			},
		},
	}

	output := generateRoute(app.APIs[0], app)

	if !strings.Contains(output, "import { authorize }") {
		t.Error("missing authorize import for authenticated endpoint with policies")
	}
	if !strings.Contains(output, "authorize('create', 'task')") {
		t.Error("missing authorize middleware in chain")
	}
}

func TestGenerateRouteWithoutAuthorize(t *testing.T) {
	// No policies defined — should not import authorize
	app := &ir.Application{
		APIs: []*ir.Endpoint{
			{
				Name: "CreateTask",
				Auth: true,
			},
		},
	}

	output := generateRoute(app.APIs[0], app)

	if strings.Contains(output, "authorize") {
		t.Error("should not include authorize when no policies are defined")
	}
}

func TestGenerateRouteNoAuthNoAuthorize(t *testing.T) {
	// Auth is false — should not import authorize even with policies
	app := &ir.Application{
		Policies: []*ir.Policy{
			{Name: "FreeUser"},
		},
		APIs: []*ir.Endpoint{
			{
				Name: "GetPublicData",
				Auth: false,
			},
		},
	}

	output := generateRoute(app.APIs[0], app)

	if strings.Contains(output, "authorize") {
		t.Error("should not include authorize for unauthenticated endpoint")
	}
}

func TestAuthorizationValidationReplacesTodo(t *testing.T) {
	app := &ir.Application{
		Policies: []*ir.Policy{
			{Name: "FreeUser"},
		},
		APIs: []*ir.Endpoint{
			{
				Name: "UpdateTask",
				Auth: true,
				Validation: []*ir.ValidationRule{
					{Field: "current_user", Rule: "authorization", Value: "the owner or an admin"},
				},
			},
		},
	}

	output := generateRoute(app.APIs[0], app)

	if strings.Contains(output, "// TODO: verify user authorization") {
		t.Error("authorization TODO should be replaced with actual code")
	}
	if !strings.Contains(output, "Ownership check") {
		t.Error("should contain ownership check for 'owner' rule")
	}
	if !strings.Contains(output, "req.authzScope") {
		t.Error("should reference authzScope for ownership check")
	}
	if !strings.Contains(output, "prisma.task.findUnique") {
		t.Error("ownership check should query the correct model via prisma (task for UpdateTask)")
	}
}
