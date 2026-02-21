package analyzer

import (
	"strings"
	"testing"

	cerr "github.com/barun-bash/human/internal/errors"
	"github.com/barun-bash/human/internal/ir"
)

// helper to build a minimal valid app
func minApp() *ir.Application {
	return &ir.Application{
		Name:     "TestApp",
		Platform: "web",
		Data: []*ir.DataModel{
			{Name: "User", Fields: []*ir.DataField{{Name: "name", Type: "text"}, {Name: "email", Type: "email"}}},
			{Name: "Task", Fields: []*ir.DataField{{Name: "title", Type: "text"}, {Name: "status", Type: "enum"}},
				Relations: []*ir.Relation{{Kind: "belongs_to", Target: "User"}}},
		},
		Pages: []*ir.Page{
			{Name: "Home", Content: []*ir.Action{{Type: "display", Text: "show heading"}}},
			{Name: "Dashboard", Content: []*ir.Action{{Type: "display", Text: "show tasks"}}},
		},
		APIs: []*ir.Endpoint{
			{Name: "CreateTask", Steps: []*ir.Action{{Type: "create", Text: "create a Task with title"}}},
		},
	}
}

// ── Passes cleanly ──

func TestAnalyzeCleanApp(t *testing.T) {
	app := minApp()
	errs := Analyze(app, "test.human")
	if errs.HasErrors() {
		t.Fatalf("expected no errors on clean app, got:\n%s", errs.Format())
	}
	if errs.HasWarnings() {
		t.Fatalf("expected no warnings on clean app, got:\n%s", errs.Format())
	}
}

// ── Duplicate names ──

func TestDuplicateModelName(t *testing.T) {
	app := minApp()
	app.Data = append(app.Data, &ir.DataModel{Name: "User"})
	errs := Analyze(app, "test.human")
	if !errs.HasErrors() {
		t.Fatal("expected error for duplicate model name")
	}
	assertCode(t, errs.Errors(), "E301")
}

func TestDuplicatePageName(t *testing.T) {
	app := minApp()
	app.Pages = append(app.Pages, &ir.Page{Name: "Home"})
	errs := Analyze(app, "test.human")
	assertCode(t, errs.Errors(), "E302")
}

func TestDuplicateAPIName(t *testing.T) {
	app := minApp()
	app.APIs = append(app.APIs, &ir.Endpoint{Name: "CreateTask"})
	errs := Analyze(app, "test.human")
	assertCode(t, errs.Errors(), "E304")
}

// ── Duplicate fields ──

func TestDuplicateFieldName(t *testing.T) {
	app := minApp()
	app.Data[0].Fields = append(app.Data[0].Fields, &ir.DataField{Name: "email", Type: "email"})
	errs := Analyze(app, "test.human")
	assertCode(t, errs.Errors(), "E306")
}

// ── Relation target validation ──

func TestUnknownRelationTarget(t *testing.T) {
	app := minApp()
	app.Data[1].Relations = append(app.Data[1].Relations, &ir.Relation{Kind: "belongs_to", Target: "Userr"})
	errs := Analyze(app, "test.human")
	assertCode(t, errs.Errors(), "E101")
	assertSuggestion(t, errs.Errors(), "User")
}

func TestUnknownThroughModel(t *testing.T) {
	app := minApp()
	app.Data = append(app.Data, &ir.DataModel{Name: "Tag"})
	app.Data[1].Relations = append(app.Data[1].Relations, &ir.Relation{
		Kind:    "has_many_through",
		Target:  "Tag",
		Through: "TasgTag", // typo
	})
	errs := Analyze(app, "test.human")
	assertCode(t, errs.Errors(), "E101")
}

// ── Through-table validation ──

func TestThroughTableMissingBelongsTo(t *testing.T) {
	app := minApp()
	app.Data = append(app.Data,
		&ir.DataModel{Name: "Tag"},
		&ir.DataModel{Name: "TaskTag"}, // missing belongs_to
	)
	app.Data[1].Relations = append(app.Data[1].Relations, &ir.Relation{
		Kind:    "has_many_through",
		Target:  "Tag",
		Through: "TaskTag",
	})
	errs := Analyze(app, "test.human")
	assertCode(t, errs.Errors(), "E105")
	count := 0
	for _, e := range errs.Errors() {
		if e.Code == "E105" {
			count++
		}
	}
	if count != 2 {
		t.Errorf("expected 2 E105 errors, got %d", count)
	}
}

func TestThroughTableValid(t *testing.T) {
	app := minApp()
	app.Data = append(app.Data,
		&ir.DataModel{Name: "Tag"},
		&ir.DataModel{
			Name: "TaskTag",
			Relations: []*ir.Relation{
				{Kind: "belongs_to", Target: "Task"},
				{Kind: "belongs_to", Target: "Tag"},
			},
		},
	)
	app.Data[1].Relations = append(app.Data[1].Relations, &ir.Relation{
		Kind:    "has_many_through",
		Target:  "Tag",
		Through: "TaskTag",
	})
	errs := Analyze(app, "test.human")
	for _, e := range errs.Errors() {
		if e.Code == "E105" {
			t.Errorf("unexpected E105 error: %s", e.Message)
		}
	}
}

// ── Database index validation ──

func TestIndexUnknownModel(t *testing.T) {
	app := minApp()
	app.Database = &ir.DatabaseConfig{
		Indexes: []*ir.Index{{Entity: "Userr", Fields: []string{"email"}}},
	}
	errs := Analyze(app, "test.human")
	assertCode(t, errs.Errors(), "E102")
	assertSuggestion(t, errs.Errors(), "User")
}

func TestIndexUnknownField(t *testing.T) {
	app := minApp()
	app.Database = &ir.DatabaseConfig{
		Indexes: []*ir.Index{{Entity: "User", Fields: []string{"nonexistent"}}},
	}
	errs := Analyze(app, "test.human")
	assertCode(t, errs.Errors(), "E102")
}

func TestIndexValidBelongsToField(t *testing.T) {
	app := minApp()
	app.Database = &ir.DatabaseConfig{
		Indexes: []*ir.Index{{Entity: "Task", Fields: []string{"user"}}},
	}
	errs := Analyze(app, "test.human")
	for _, e := range errs.Errors() {
		if e.Code == "E102" {
			t.Errorf("unexpected E102 — 'user' should resolve via belongs_to: %s", e.Message)
		}
	}
}

func TestIndexValidFieldName(t *testing.T) {
	app := minApp()
	app.Database = &ir.DatabaseConfig{
		Indexes: []*ir.Index{{Entity: "User", Fields: []string{"email"}}},
	}
	errs := Analyze(app, "test.human")
	for _, e := range errs.Errors() {
		if e.Code == "E102" {
			t.Errorf("unexpected E102 — 'email' should match field: %s", e.Message)
		}
	}
}

// ── Page navigation validation ──

func TestPageNavigatesToUnknown(t *testing.T) {
	app := minApp()
	app.Pages[0].Content = append(app.Pages[0].Content, &ir.Action{
		Type: "interact",
		Text: "clicking the button navigates to Settigns",
	})
	errs := Analyze(app, "test.human")
	assertCode(t, errs.Errors(), "E103")
}

func TestPageNavigatesToKnown(t *testing.T) {
	app := minApp()
	app.Pages[0].Content = append(app.Pages[0].Content, &ir.Action{
		Type: "interact",
		Text: "clicking the button navigates to Dashboard",
	})
	errs := Analyze(app, "test.human")
	for _, e := range errs.Errors() {
		if e.Code == "E103" {
			t.Errorf("unexpected E103 — Dashboard exists: %s", e.Message)
		}
	}
}

// ── API model reference validation ──

func TestAPIReferencesUnknownModel(t *testing.T) {
	app := minApp()
	app.APIs = append(app.APIs, &ir.Endpoint{
		Name: "BadAPI",
		Steps: []*ir.Action{
			{Type: "create", Text: "create a Userr with name"},
		},
	})
	errs := Analyze(app, "test.human")
	assertCode(t, errs.Errors(), "E104")
	assertSuggestion(t, errs.Errors(), "User")
}

func TestAPIReferencesKnownModel(t *testing.T) {
	app := minApp()
	// "create a Task" should be fine
	errs := Analyze(app, "test.human")
	for _, e := range errs.Errors() {
		if e.Code == "E104" {
			t.Errorf("unexpected E104 — Task exists: %s", e.Message)
		}
	}
}

// ── Completeness ──

func TestAuthRequiredButMissing(t *testing.T) {
	app := minApp()
	app.APIs[0].Auth = true
	errs := Analyze(app, "test.human")
	assertCode(t, errs.Errors(), "E201")
}

func TestAuthRequiredAndPresent(t *testing.T) {
	app := minApp()
	app.APIs[0].Auth = true
	app.Auth = &ir.Auth{Methods: []*ir.AuthMethod{{Type: "jwt"}}}
	errs := Analyze(app, "test.human")
	for _, e := range errs.Errors() {
		if e.Code == "E201" {
			t.Errorf("unexpected E201 — auth is configured: %s", e.Message)
		}
	}
}

func TestDatabaseWithoutModels(t *testing.T) {
	app := &ir.Application{
		Config: &ir.BuildConfig{Database: "PostgreSQL"},
	}
	errs := Analyze(app, "test.human")
	assertCode(t, errs.Errors(), "E202")
}

func TestFrontendWithoutPages(t *testing.T) {
	app := &ir.Application{
		Config: &ir.BuildConfig{Frontend: "React"},
	}
	errs := Analyze(app, "test.human")
	assertCode(t, errs.Errors(), "E203")
}

// ── Design system validation ──

func TestUnknownDesignSystem(t *testing.T) {
	app := minApp()
	app.Theme = &ir.Theme{DesignSystem: "materail"}
	errs := Analyze(app, "test.human")
	assertWarningCode(t, errs.Warnings(), "W301")
}

func TestValidDesignSystem(t *testing.T) {
	app := minApp()
	app.Theme = &ir.Theme{DesignSystem: "material"}
	errs := Analyze(app, "test.human")
	for _, w := range errs.Warnings() {
		if w.Code == "W301" {
			t.Errorf("unexpected W301 — material is valid: %s", w.Message)
		}
	}
}

func TestDesignSystemFrameworkIncompatibility(t *testing.T) {
	app := minApp()
	app.Theme = &ir.Theme{DesignSystem: "chakra"}
	app.Config = &ir.BuildConfig{Frontend: "Vue with TypeScript"}
	errs := Analyze(app, "test.human")
	assertWarningCode(t, errs.Warnings(), "W302")
}

func TestInvalidSpacing(t *testing.T) {
	app := minApp()
	app.Theme = &ir.Theme{DesignSystem: "material", Spacing: "huge"}
	errs := Analyze(app, "test.human")
	assertWarningCode(t, errs.Warnings(), "W303")
}

func TestValidSpacing(t *testing.T) {
	app := minApp()
	app.Theme = &ir.Theme{DesignSystem: "material", Spacing: "compact"}
	errs := Analyze(app, "test.human")
	for _, w := range errs.Warnings() {
		if w.Code == "W303" {
			t.Errorf("unexpected W303 — compact is valid: %s", w.Message)
		}
	}
}

func TestInvalidBorderRadius(t *testing.T) {
	app := minApp()
	app.Theme = &ir.Theme{DesignSystem: "material", BorderRadius: "extreme"}
	errs := Analyze(app, "test.human")
	assertWarningCode(t, errs.Warnings(), "W304")
}

func TestValidBorderRadius(t *testing.T) {
	app := minApp()
	app.Theme = &ir.Theme{DesignSystem: "material", BorderRadius: "rounded"}
	errs := Analyze(app, "test.human")
	for _, w := range errs.Warnings() {
		if w.Code == "W304" {
			t.Errorf("unexpected W304 — rounded is valid: %s", w.Message)
		}
	}
}

// ── Integration: taskflow-like example ──

func TestAnalyzeTaskflowIR(t *testing.T) {
	app := &ir.Application{
		Name:     "TaskFlow",
		Platform: "web",
		Data: []*ir.DataModel{
			{
				Name: "User",
				Fields: []*ir.DataField{
					{Name: "name", Type: "text"},
					{Name: "email", Type: "email"},
					{Name: "password", Type: "text"},
					{Name: "avatar", Type: "image"},
					{Name: "bio", Type: "text"},
					{Name: "role", Type: "enum", EnumValues: []string{"admin", "member", "viewer"}},
				},
			},
			{
				Name: "Task",
				Fields: []*ir.DataField{
					{Name: "title", Type: "text"},
					{Name: "description", Type: "text"},
					{Name: "status", Type: "enum"},
					{Name: "priority", Type: "enum"},
					{Name: "due", Type: "datetime"},
				},
				Relations: []*ir.Relation{
					{Kind: "belongs_to", Target: "User"},
					{Kind: "has_many_through", Target: "Tag", Through: "TaskTag"},
				},
			},
			{
				Name:   "Tag",
				Fields: []*ir.DataField{{Name: "name", Type: "text"}},
			},
			{
				Name: "TaskTag",
				Relations: []*ir.Relation{
					{Kind: "belongs_to", Target: "Task"},
					{Kind: "belongs_to", Target: "Tag"},
				},
			},
		},
		Pages: []*ir.Page{
			{Name: "Home", Content: []*ir.Action{
				{Type: "interact", Text: "clicking the \"Get Started\" button navigates to Dashboard"},
			}},
			{Name: "Dashboard", Content: []*ir.Action{{Type: "display", Text: "show tasks"}}},
			{Name: "Profile", Content: []*ir.Action{{Type: "display", Text: "show user info"}}},
		},
		APIs: []*ir.Endpoint{
			{Name: "SignUp", Steps: []*ir.Action{{Type: "create", Text: "create a User with name, email, password"}}},
			{Name: "Login"},
			{Name: "CreateTask", Auth: true, Steps: []*ir.Action{{Type: "create", Text: "create a Task with title, description"}}},
			{Name: "GetTasks", Auth: true, Steps: []*ir.Action{{Type: "query", Text: "fetch the Task list"}}},
			{Name: "UpdateTask", Auth: true, Steps: []*ir.Action{{Type: "update", Text: "update the Task with provided fields"}}},
			{Name: "DeleteTask", Auth: true, Steps: []*ir.Action{{Type: "delete", Text: "delete the Task"}}},
		},
		Auth: &ir.Auth{
			Methods: []*ir.AuthMethod{{Type: "jwt"}},
		},
		Database: &ir.DatabaseConfig{
			Engine: "PostgreSQL",
			Indexes: []*ir.Index{
				{Entity: "User", Fields: []string{"email"}},
				{Entity: "Task", Fields: []string{"user", "status"}},
				{Entity: "Task", Fields: []string{"due date"}},
			},
		},
		Config: &ir.BuildConfig{
			Frontend: "React with TypeScript",
			Backend:  "Node with Express",
			Database: "PostgreSQL",
			Deploy:   "Docker",
		},
	}

	errs := Analyze(app, "app.human")
	if errs.HasErrors() {
		t.Fatalf("expected no errors on taskflow-like app, got:\n%s", errs.Format())
	}
}

// ── Architecture validation ──

func TestUnknownArchitectureStyle(t *testing.T) {
	app := minApp()
	app.Architecture = &ir.Architecture{Style: "microservise"} // typo
	errs := Analyze(app, "test.human")
	assertWarningCode(t, errs.Warnings(), "W401")
}

func TestValidArchitectureStyle(t *testing.T) {
	app := minApp()
	app.Architecture = &ir.Architecture{Style: "microservices",
		Services: []*ir.ServiceDef{{Name: "Svc1"}}}
	errs := Analyze(app, "test.human")
	for _, w := range errs.Warnings() {
		if w.Code == "W401" {
			t.Errorf("unexpected W401 — microservices is valid: %s", w.Message)
		}
	}
}

func TestMicroservicesWithoutServices(t *testing.T) {
	app := minApp()
	app.Architecture = &ir.Architecture{Style: "microservices"}
	errs := Analyze(app, "test.human")
	assertCode(t, errs.Errors(), "E401")
}

func TestMicroservicesWithServices(t *testing.T) {
	app := minApp()
	app.Architecture = &ir.Architecture{
		Style:    "microservices",
		Services: []*ir.ServiceDef{{Name: "UserService"}},
	}
	errs := Analyze(app, "test.human")
	for _, e := range errs.Errors() {
		if e.Code == "E401" {
			t.Errorf("unexpected E401 — services are defined: %s", e.Message)
		}
	}
}

func TestServiceReferencesUnknownModel(t *testing.T) {
	app := minApp()
	app.Architecture = &ir.Architecture{
		Style: "microservices",
		Services: []*ir.ServiceDef{
			{Name: "Svc1", Models: []string{"Userr"}}, // typo
		},
	}
	errs := Analyze(app, "test.human")
	assertWarningCode(t, errs.Warnings(), "W402")
}

func TestServiceTalksToUnknown(t *testing.T) {
	app := minApp()
	app.Architecture = &ir.Architecture{
		Style: "microservices",
		Services: []*ir.ServiceDef{
			{Name: "Svc1", TalksTo: []string{"Svc2"}},
		},
	}
	errs := Analyze(app, "test.human")
	assertWarningCode(t, errs.Warnings(), "W403")
}

func TestServiceTalksToValid(t *testing.T) {
	app := minApp()
	app.Architecture = &ir.Architecture{
		Style: "microservices",
		Services: []*ir.ServiceDef{
			{Name: "Svc1", TalksTo: []string{"Svc2"}},
			{Name: "Svc2"},
		},
	}
	errs := Analyze(app, "test.human")
	for _, w := range errs.Warnings() {
		if w.Code == "W403" {
			t.Errorf("unexpected W403 — Svc2 exists: %s", w.Message)
		}
	}
}

func TestServerlessWithoutAPIs(t *testing.T) {
	app := &ir.Application{
		Name:         "TestApp",
		Architecture: &ir.Architecture{Style: "serverless"},
	}
	errs := Analyze(app, "test.human")
	assertCode(t, errs.Errors(), "E402")
}

func TestServerlessWithAPIs(t *testing.T) {
	app := minApp()
	app.Architecture = &ir.Architecture{Style: "serverless"}
	errs := Analyze(app, "test.human")
	for _, e := range errs.Errors() {
		if e.Code == "E402" {
			t.Errorf("unexpected E402 — APIs are defined: %s", e.Message)
		}
	}
}

// ── Integration validation ──

func TestDuplicateIntegration(t *testing.T) {
	app := minApp()
	app.Integrations = []*ir.Integration{
		{Service: "SendGrid", Type: "email"},
		{Service: "SendGrid", Type: "email"},
	}
	errs := Analyze(app, "test.human")
	assertCode(t, errs.Errors(), "E501")
}

func TestIntegrationNoDuplicateClean(t *testing.T) {
	app := minApp()
	app.Integrations = []*ir.Integration{
		{Service: "SendGrid", Type: "email", Credentials: map[string]string{"api key": "SG_KEY"}},
		{Service: "Stripe", Type: "payment", Credentials: map[string]string{"api key": "STRIPE_KEY"}},
	}
	errs := Analyze(app, "test.human")
	for _, e := range errs.Errors() {
		if e.Code == "E501" {
			t.Errorf("unexpected E501 — services are distinct: %s", e.Message)
		}
	}
}

func TestIntegrationNoCredentials(t *testing.T) {
	app := minApp()
	app.Integrations = []*ir.Integration{
		{Service: "SendGrid", Type: "email"}, // no credentials
	}
	errs := Analyze(app, "test.human")
	assertWarningCode(t, errs.Warnings(), "W501")
}

func TestIntegrationNoCredentialsLocalService(t *testing.T) {
	app := minApp()
	app.Integrations = []*ir.Integration{
		{Service: "Ollama"}, // local service, no credentials needed
	}
	errs := Analyze(app, "test.human")
	for _, w := range errs.Warnings() {
		if w.Code == "W501" {
			t.Errorf("unexpected W501 — Ollama is local: %s", w.Message)
		}
	}
}

func TestWorkflowEmailNoIntegration(t *testing.T) {
	app := minApp()
	app.Workflows = []*ir.Workflow{
		{Trigger: "user signs up", Steps: []*ir.Action{
			{Type: "action", Text: "send welcome email with template welcome"},
		}},
	}
	errs := Analyze(app, "test.human")
	assertWarningCode(t, errs.Warnings(), "W502")
}

func TestWorkflowEmailWithIntegration(t *testing.T) {
	app := minApp()
	app.Integrations = []*ir.Integration{
		{Service: "SendGrid", Type: "email", Credentials: map[string]string{"api key": "SG_KEY"}},
	}
	app.Workflows = []*ir.Workflow{
		{Trigger: "user signs up", Steps: []*ir.Action{
			{Type: "action", Text: "send welcome email with template welcome"},
		}},
	}
	errs := Analyze(app, "test.human")
	for _, w := range errs.Warnings() {
		if w.Code == "W502" {
			t.Errorf("unexpected W502 — email integration exists: %s", w.Message)
		}
	}
}

func TestWorkflowSlackNoIntegration(t *testing.T) {
	app := minApp()
	app.Workflows = []*ir.Workflow{
		{Trigger: "task becomes overdue", Steps: []*ir.Action{
			{Type: "action", Text: "alert the admin via Slack"},
		}},
	}
	errs := Analyze(app, "test.human")
	assertWarningCode(t, errs.Warnings(), "W503")
}

func TestWorkflowSlackWithIntegration(t *testing.T) {
	app := minApp()
	app.Integrations = []*ir.Integration{
		{Service: "Slack", Type: "messaging", Credentials: map[string]string{"api key": "SLACK_URL"}},
	}
	app.Workflows = []*ir.Workflow{
		{Trigger: "task becomes overdue", Steps: []*ir.Action{
			{Type: "action", Text: "alert the admin via Slack"},
		}},
	}
	errs := Analyze(app, "test.human")
	for _, w := range errs.Warnings() {
		if w.Code == "W503" {
			t.Errorf("unexpected W503 — Slack integration exists: %s", w.Message)
		}
	}
}

// ── Test helpers ──

func assertCode(t *testing.T, errs []*cerr.CompilerError, code string) {
	t.Helper()
	for _, e := range errs {
		if e.Code == code {
			return
		}
	}
	t.Errorf("expected at least one error with code %s, found none", code)
}

func assertWarningCode(t *testing.T, warnings []*cerr.CompilerError, code string) {
	t.Helper()
	for _, w := range warnings {
		if w.Code == code {
			return
		}
	}
	t.Errorf("expected at least one warning with code %s, found none", code)
}

func assertSuggestion(t *testing.T, errs []*cerr.CompilerError, contains string) {
	t.Helper()
	for _, e := range errs {
		if strings.Contains(e.Suggestion, contains) {
			return
		}
	}
	t.Errorf("expected a suggestion containing %q, found none", contains)
}
