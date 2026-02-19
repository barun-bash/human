package parser

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// helper to parse source and assert no error
func mustParse(t *testing.T, source string) *Program {
	t.Helper()
	prog, err := Parse(source)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	return prog
}

// ── App Declaration ──

func TestParseAppDeclaration(t *testing.T) {
	prog := mustParse(t, "app TaskFlow is a web application")

	if prog.App == nil {
		t.Fatal("expected App declaration")
	}
	if prog.App.Name != "TaskFlow" {
		t.Errorf("expected app name 'TaskFlow', got %q", prog.App.Name)
	}
	if prog.App.Platform != "web" {
		t.Errorf("expected platform 'web', got %q", prog.App.Platform)
	}
}

func TestParseAppMobile(t *testing.T) {
	prog := mustParse(t, "app FitnessPal is a mobile application")
	if prog.App.Platform != "mobile" {
		t.Errorf("expected platform 'mobile', got %q", prog.App.Platform)
	}
}

// ── Data Declarations ──

func TestParseDataSimple(t *testing.T) {
	source := `data User:
  has a name which is text
  has an email which is unique email
  has a password which is encrypted text`
	prog := mustParse(t, source)

	if len(prog.Data) != 1 {
		t.Fatalf("expected 1 data declaration, got %d", len(prog.Data))
	}
	data := prog.Data[0]
	if data.Name != "User" {
		t.Errorf("expected data name 'User', got %q", data.Name)
	}
	if len(data.Fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(data.Fields))
	}

	// name which is text
	f := data.Fields[0]
	if f.Name != "name" {
		t.Errorf("field 0: expected name 'name', got %q", f.Name)
	}
	if f.Type != "text" {
		t.Errorf("field 0: expected type 'text', got %q", f.Type)
	}

	// email which is unique email
	f = data.Fields[1]
	if f.Name != "email" {
		t.Errorf("field 1: expected name 'email', got %q", f.Name)
	}
	if f.Type != "email" {
		t.Errorf("field 1: expected type 'email', got %q", f.Type)
	}
	if !hasModifier(f, "unique") {
		t.Error("field 1: expected 'unique' modifier")
	}

	// password which is encrypted text
	f = data.Fields[2]
	if f.Name != "password" {
		t.Errorf("field 2: expected name 'password', got %q", f.Name)
	}
	if f.Type != "text" {
		t.Errorf("field 2: expected type 'text', got %q", f.Type)
	}
	if !hasModifier(f, "encrypted") {
		t.Error("field 2: expected 'encrypted' modifier")
	}
}

func TestParseDataEnum(t *testing.T) {
	source := `data Task:
  has a status which is either "todo" or "in_progress" or "done"
  has a priority which is either "low" or "medium" or "high"`
	prog := mustParse(t, source)

	data := prog.Data[0]
	if len(data.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(data.Fields))
	}

	f := data.Fields[0]
	if f.Name != "status" {
		t.Errorf("expected name 'status', got %q", f.Name)
	}
	if len(f.EnumValues) != 3 {
		t.Fatalf("expected 3 enum values, got %d", len(f.EnumValues))
	}
	if f.EnumValues[0] != "todo" || f.EnumValues[1] != "in_progress" || f.EnumValues[2] != "done" {
		t.Errorf("unexpected enum values: %v", f.EnumValues)
	}
}

func TestParseDataOptional(t *testing.T) {
	source := `data User:
  has an optional bio which is text`
	prog := mustParse(t, source)

	f := prog.Data[0].Fields[0]
	if f.Name != "bio" {
		t.Errorf("expected name 'bio', got %q", f.Name)
	}
	if f.Type != "text" {
		t.Errorf("expected type 'text', got %q", f.Type)
	}
	if !hasModifier(f, "optional") {
		t.Error("expected 'optional' modifier")
	}
}

func TestParseDataShorthand(t *testing.T) {
	source := `data User:
  has a created datetime`
	prog := mustParse(t, source)

	f := prog.Data[0].Fields[0]
	if f.Name != "created" {
		t.Errorf("expected name 'created', got %q", f.Name)
	}
	if f.Type != "datetime" {
		t.Errorf("expected type 'datetime', got %q", f.Type)
	}
}

func TestParseDataRelationships(t *testing.T) {
	source := `data Task:
  belongs to a User
  has many Tag through TaskTag
  has many Comment`
	prog := mustParse(t, source)

	rels := prog.Data[0].Relationships
	if len(rels) != 3 {
		t.Fatalf("expected 3 relationships, got %d", len(rels))
	}

	// belongs to a User
	r := rels[0]
	if r.Kind != "belongs_to" {
		t.Errorf("rel 0: expected kind 'belongs_to', got %q", r.Kind)
	}
	if r.Target != "User" {
		t.Errorf("rel 0: expected target 'User', got %q", r.Target)
	}

	// has many Tag through TaskTag
	r = rels[1]
	if r.Kind != "has_many" {
		t.Errorf("rel 1: expected kind 'has_many', got %q", r.Kind)
	}
	if r.Target != "Tag" {
		t.Errorf("rel 1: expected target 'Tag', got %q", r.Target)
	}
	if r.Through != "TaskTag" {
		t.Errorf("rel 1: expected through 'TaskTag', got %q", r.Through)
	}

	// has many Comment
	r = rels[2]
	if r.Kind != "has_many" {
		t.Errorf("rel 2: expected kind 'has_many', got %q", r.Kind)
	}
	if r.Target != "Comment" {
		t.Errorf("rel 2: expected target 'Comment', got %q", r.Target)
	}
	if r.Through != "" {
		t.Errorf("rel 2: expected no through, got %q", r.Through)
	}
}

func TestParseMultipleData(t *testing.T) {
	source := `data User:
  has a name which is text

data Task:
  has a title which is text`
	prog := mustParse(t, source)

	if len(prog.Data) != 2 {
		t.Fatalf("expected 2 data declarations, got %d", len(prog.Data))
	}
	if prog.Data[0].Name != "User" {
		t.Errorf("expected first data 'User', got %q", prog.Data[0].Name)
	}
	if prog.Data[1].Name != "Task" {
		t.Errorf("expected second data 'Task', got %q", prog.Data[1].Name)
	}
}

// ── Page Declarations ──

func TestParsePageDeclaration(t *testing.T) {
	source := `page Dashboard:
  show a greeting with the user's name
  show a list of tasks sorted by due date
  clicking a task navigates to the task detail
  if no tasks match, show "No tasks found"
  while loading, show a spinner`
	prog := mustParse(t, source)

	if len(prog.Pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(prog.Pages))
	}
	page := prog.Pages[0]
	if page.Name != "Dashboard" {
		t.Errorf("expected page name 'Dashboard', got %q", page.Name)
	}
	if len(page.Statements) != 5 {
		t.Fatalf("expected 5 statements, got %d", len(page.Statements))
	}

	// Check statement kinds
	expectedKinds := []string{"show", "show", "clicking", "if", "while"}
	for i, expected := range expectedKinds {
		if page.Statements[i].Kind != expected {
			t.Errorf("statement %d: expected kind %q, got %q",
				i, expected, page.Statements[i].Kind)
		}
	}
}

func TestParsePagePossessive(t *testing.T) {
	source := `page Profile:
  show the user's name`
	prog := mustParse(t, source)

	page := prog.Pages[0]
	if len(page.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(page.Statements))
	}
	// The possessive should be attached: "user's" not "user 's"
	text := page.Statements[0].Text
	if !containsSubstring(text, "user's") {
		t.Errorf("expected possessive 'user's' in text, got %q", text)
	}
}

// ── API Declarations ──

func TestParseAPIDeclaration(t *testing.T) {
	source := `api CreateTask:
  requires authentication
  accepts title, description, and status
  check that title is not empty
  create a Task with the given fields
  respond with the created task`
	prog := mustParse(t, source)

	if len(prog.APIs) != 1 {
		t.Fatalf("expected 1 API, got %d", len(prog.APIs))
	}
	api := prog.APIs[0]
	if api.Name != "CreateTask" {
		t.Errorf("expected api name 'CreateTask', got %q", api.Name)
	}
	if !api.Auth {
		t.Error("expected Auth to be true")
	}
	if len(api.Accepts) != 3 {
		t.Fatalf("expected 3 accepts params, got %d: %v", len(api.Accepts), api.Accepts)
	}
	if api.Accepts[0] != "title" {
		t.Errorf("expected first param 'title', got %q", api.Accepts[0])
	}
	if api.Accepts[1] != "description" {
		t.Errorf("expected second param 'description', got %q", api.Accepts[1])
	}
	if api.Accepts[2] != "status" {
		t.Errorf("expected third param 'status', got %q", api.Accepts[2])
	}
	if len(api.Statements) != 3 {
		t.Fatalf("expected 3 body statements, got %d", len(api.Statements))
	}
}

func TestParseAPINoAuth(t *testing.T) {
	source := `api SignUp:
  accepts name, email, and password
  check that name is not empty
  respond with the created user`
	prog := mustParse(t, source)

	api := prog.APIs[0]
	if api.Auth {
		t.Error("expected Auth to be false")
	}
	if len(api.Accepts) != 3 {
		t.Fatalf("expected 3 params, got %d: %v", len(api.Accepts), api.Accepts)
	}
}

func TestParseAPIMultiWordParam(t *testing.T) {
	source := `api CreateTask:
  accepts title, description, status, priority, and due date`
	prog := mustParse(t, source)

	api := prog.APIs[0]
	if len(api.Accepts) != 5 {
		t.Fatalf("expected 5 params, got %d: %v", len(api.Accepts), api.Accepts)
	}
	// "due date" should be the last parameter
	last := api.Accepts[4]
	if last != "due date" {
		t.Errorf("expected last param 'due date', got %q", last)
	}
}

func TestParseAPISingleParam(t *testing.T) {
	source := `api DeleteTask:
  requires authentication
  accepts task_id
  delete the task`
	prog := mustParse(t, source)

	api := prog.APIs[0]
	if len(api.Accepts) != 1 {
		t.Fatalf("expected 1 param, got %d: %v", len(api.Accepts), api.Accepts)
	}
	if api.Accepts[0] != "task_id" {
		t.Errorf("expected param 'task_id', got %q", api.Accepts[0])
	}
}

// ── Policy Declarations ──

func TestParsePolicyDeclaration(t *testing.T) {
	source := `policy FreeUser:
  can create up to 50 tasks per month
  can view only their own tasks
  cannot delete completed tasks
  cannot export data`
	prog := mustParse(t, source)

	if len(prog.Policies) != 1 {
		t.Fatalf("expected 1 policy, got %d", len(prog.Policies))
	}
	pol := prog.Policies[0]
	if pol.Name != "FreeUser" {
		t.Errorf("expected policy name 'FreeUser', got %q", pol.Name)
	}
	if len(pol.Rules) != 4 {
		t.Fatalf("expected 4 rules, got %d", len(pol.Rules))
	}

	// Check can/cannot
	if !pol.Rules[0].Allowed {
		t.Error("rule 0: expected Allowed=true")
	}
	if !pol.Rules[1].Allowed {
		t.Error("rule 1: expected Allowed=true")
	}
	if pol.Rules[2].Allowed {
		t.Error("rule 2: expected Allowed=false")
	}
	if pol.Rules[3].Allowed {
		t.Error("rule 3: expected Allowed=false")
	}
}

// ── Workflow Declarations ──

func TestParseWorkflowDeclaration(t *testing.T) {
	source := `when a user signs up:
  create their account
  assign FreeUser policy
  send welcome email with template "welcome"
  after 3 days, send email with template "getting-started"`
	prog := mustParse(t, source)

	if len(prog.Workflows) != 1 {
		t.Fatalf("expected 1 workflow, got %d", len(prog.Workflows))
	}
	wf := prog.Workflows[0]
	if wf.Event != "a user signs up" {
		t.Errorf("expected event 'a user signs up', got %q", wf.Event)
	}
	if len(wf.Statements) != 4 {
		t.Fatalf("expected 4 statements, got %d", len(wf.Statements))
	}
}

func TestParseMultipleWorkflows(t *testing.T) {
	source := `when a user signs up:
  create their account

when a task becomes overdue:
  send notification to the task owner`
	prog := mustParse(t, source)

	if len(prog.Workflows) != 2 {
		t.Fatalf("expected 2 workflows, got %d", len(prog.Workflows))
	}
}

func TestParseWorkflowNestedIndent(t *testing.T) {
	source := `when a task becomes overdue:
  send notification to the task owner
  update task priority to "high"
  if task is still overdue after 3 days,
    send reminder email to the task owner
  alert the admin via Slack`
	prog := mustParse(t, source)

	if len(prog.Workflows) != 1 {
		t.Fatalf("expected 1 workflow, got %d", len(prog.Workflows))
	}
	wf := prog.Workflows[0]
	if len(wf.Statements) != 5 {
		t.Fatalf("expected 5 statements, got %d", len(wf.Statements))
	}
	// "alert the admin via Slack" should be inside the workflow, not leaked to top level
	lastStmt := wf.Statements[4]
	if lastStmt.Kind != "alert" {
		t.Errorf("expected last statement kind 'alert', got %q", lastStmt.Kind)
	}
	if len(prog.Statements) != 0 {
		t.Errorf("expected 0 top-level statements, got %d", len(prog.Statements))
	}
}

// ── Theme Declaration ──

func TestParseThemeDeclaration(t *testing.T) {
	source := `theme:
  primary color is #6C5CE7
  secondary color is #00B894
  danger color is #D63031
  font is Inter for body and Poppins for headings`
	prog := mustParse(t, source)

	if prog.Theme == nil {
		t.Fatal("expected Theme declaration")
	}
	if len(prog.Theme.Properties) != 4 {
		t.Fatalf("expected 4 theme properties, got %d", len(prog.Theme.Properties))
	}
}

// ── Authentication Declaration ──

func TestParseAuthenticationDeclaration(t *testing.T) {
	source := `authentication:
  method JWT tokens that expire in 7 days
  method Google OAuth with redirect to "/auth/google/callback"
  rate limit all endpoints to 100 requests per minute per user`
	prog := mustParse(t, source)

	if prog.Authentication == nil {
		t.Fatal("expected Authentication declaration")
	}
	if len(prog.Authentication.Statements) != 3 {
		t.Fatalf("expected 3 auth statements, got %d", len(prog.Authentication.Statements))
	}
	if prog.Authentication.Statements[0].Kind != "method" {
		t.Errorf("expected first kind 'method', got %q", prog.Authentication.Statements[0].Kind)
	}
}

// ── Database Declaration ──

func TestParseDatabaseDeclaration(t *testing.T) {
	source := `database:
  use PostgreSQL
  index User by email
  index Task by user and due date
  backup daily at 3am
  keep backups for 30 days`
	prog := mustParse(t, source)

	if prog.Database == nil {
		t.Fatal("expected Database declaration")
	}
	if len(prog.Database.Statements) != 5 {
		t.Fatalf("expected 5 database statements, got %d", len(prog.Database.Statements))
	}
}

// ── Integration Declarations ──

func TestParseIntegrationDeclaration(t *testing.T) {
	source := `integrate with SendGrid:
  api key from environment variable SENDGRID_API_KEY
  use for sending transactional emails`
	prog := mustParse(t, source)

	if len(prog.Integrations) != 1 {
		t.Fatalf("expected 1 integration, got %d", len(prog.Integrations))
	}
	integ := prog.Integrations[0]
	if integ.Service != "SendGrid" {
		t.Errorf("expected service 'SendGrid', got %q", integ.Service)
	}
	if len(integ.Statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(integ.Statements))
	}
}

func TestParseIntegrationMultiWordService(t *testing.T) {
	source := `integrate with AWS S3:
  use for file storage`
	prog := mustParse(t, source)

	integ := prog.Integrations[0]
	if integ.Service != "AWS S3" {
		t.Errorf("expected service 'AWS S3', got %q", integ.Service)
	}
}

// ── Environment Declarations ──

func TestParseEnvironmentDeclaration(t *testing.T) {
	source := `environment staging:
  url is "staging.taskflow.app"
  uses staging database`
	prog := mustParse(t, source)

	if len(prog.Environments) != 1 {
		t.Fatalf("expected 1 environment, got %d", len(prog.Environments))
	}
	env := prog.Environments[0]
	if env.Name != "staging" {
		t.Errorf("expected name 'staging', got %q", env.Name)
	}
	if len(env.Statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(env.Statements))
	}
}

// ── Error Handler ──

func TestParseErrorHandler(t *testing.T) {
	source := `if database is unreachable:
  retry 3 times with 1 second delay
  alert the engineering team via Slack`
	prog := mustParse(t, source)

	if len(prog.ErrorHandlers) != 1 {
		t.Fatalf("expected 1 error handler, got %d", len(prog.ErrorHandlers))
	}
	eh := prog.ErrorHandlers[0]
	if eh.Condition != "database is unreachable" {
		t.Errorf("expected condition 'database is unreachable', got %q", eh.Condition)
	}
	if len(eh.Statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(eh.Statements))
	}
}

// ── Build Declaration ──

func TestParseBuildDeclaration(t *testing.T) {
	source := `build with:
  frontend using React with TypeScript
  backend using Node with Express
  database using PostgreSQL
  deploy to Vercel`
	prog := mustParse(t, source)

	if prog.Build == nil {
		t.Fatal("expected Build declaration")
	}
	if len(prog.Build.Statements) != 4 {
		t.Fatalf("expected 4 build statements, got %d", len(prog.Build.Statements))
	}
}

// ── Section Headers ──

func TestParseSectionHeaders(t *testing.T) {
	source := `── frontend ──

── backend ──

── security ──`
	prog := mustParse(t, source)

	if len(prog.Sections) != 3 {
		t.Fatalf("expected 3 sections, got %d", len(prog.Sections))
	}
	expected := []string{"frontend", "backend", "security"}
	for i, name := range expected {
		if prog.Sections[i] != name {
			t.Errorf("section %d: expected %q, got %q", i, name, prog.Sections[i])
		}
	}
}

func TestParseSectionHeadersWithBlocks(t *testing.T) {
	source := `── security ──

authentication:
  method JWT tokens that expire in 7 days
  rate limit all endpoints to 100 requests per minute

── policies ──

policy FreeUser:
  can create up to 50 tasks`
	prog := mustParse(t, source)

	if len(prog.Sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(prog.Sections))
	}
	if prog.Sections[0] != "security" {
		t.Errorf("section 0: expected 'security', got %q", prog.Sections[0])
	}
	if prog.Sections[1] != "policies" {
		t.Errorf("section 1: expected 'policies', got %q", prog.Sections[1])
	}
	if prog.Authentication == nil {
		t.Error("missing Authentication declaration")
	}
	if len(prog.Policies) != 1 {
		t.Errorf("expected 1 policy, got %d", len(prog.Policies))
	}
}

// ── Top-Level Statements ──

func TestParseTopLevelStatements(t *testing.T) {
	source := `source control using Git on GitHub
track response time for all api endpoints
alert on Slack if response time exceeds 500ms`
	prog := mustParse(t, source)

	if len(prog.Statements) < 3 {
		t.Fatalf("expected at least 3 top-level statements, got %d", len(prog.Statements))
	}
}

func TestParseRepository(t *testing.T) {
	source := `repository: "https://github.com/taskflow/taskflow"`
	prog := mustParse(t, source)

	if len(prog.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(prog.Statements))
	}
	if prog.Statements[0].Kind != "repository" {
		t.Errorf("expected kind 'repository', got %q", prog.Statements[0].Kind)
	}
}

// ── Full Integration Test ──

func TestParseAppHuman(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	projectRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	appPath := filepath.Join(projectRoot, "examples", "taskflow", "app.human")

	source, err := os.ReadFile(appPath)
	if err != nil {
		t.Fatalf("could not read app.human: %v", err)
	}

	prog, err := Parse(string(source))
	if err != nil {
		t.Fatalf("failed to parse app.human: %v", err)
	}

	// ── App declaration ──
	if prog.App == nil {
		t.Fatal("missing App declaration")
	}
	if prog.App.Name != "TaskFlow" {
		t.Errorf("app name: expected 'TaskFlow', got %q", prog.App.Name)
	}
	if prog.App.Platform != "web" {
		t.Errorf("app platform: expected 'web', got %q", prog.App.Platform)
	}

	// ── Data models ──
	if len(prog.Data) != 4 {
		t.Errorf("expected 4 data declarations (User, Task, Tag, TaskTag), got %d", len(prog.Data))
	} else {
		names := []string{"User", "Task", "Tag", "TaskTag"}
		for i, name := range names {
			if prog.Data[i].Name != name {
				t.Errorf("data[%d]: expected %q, got %q", i, name, prog.Data[i].Name)
			}
		}
		// User should have fields and relationships
		user := prog.Data[0]
		if len(user.Fields) < 5 {
			t.Errorf("User: expected at least 5 fields, got %d", len(user.Fields))
		}
		if len(user.Relationships) < 1 {
			t.Errorf("User: expected at least 1 relationship, got %d", len(user.Relationships))
		}

		// Task should have belongs_to and has_many
		task := prog.Data[1]
		belongsCount := 0
		hasManyCount := 0
		for _, r := range task.Relationships {
			if r.Kind == "belongs_to" {
				belongsCount++
			}
			if r.Kind == "has_many" {
				hasManyCount++
			}
		}
		if belongsCount < 1 {
			t.Error("Task: expected at least 1 belongs_to relationship")
		}
		if hasManyCount < 1 {
			t.Error("Task: expected at least 1 has_many relationship")
		}
	}

	// ── Pages ──
	if len(prog.Pages) != 3 {
		t.Errorf("expected 3 pages (Home, Dashboard, Profile), got %d", len(prog.Pages))
	} else {
		pageNames := []string{"Home", "Dashboard", "Profile"}
		for i, name := range pageNames {
			if prog.Pages[i].Name != name {
				t.Errorf("page[%d]: expected %q, got %q", i, name, prog.Pages[i].Name)
			}
		}
		// Dashboard should have multiple statements
		dash := prog.Pages[1]
		if len(dash.Statements) < 5 {
			t.Errorf("Dashboard: expected at least 5 statements, got %d", len(dash.Statements))
		}
	}

	// ── APIs ──
	if len(prog.APIs) != 8 {
		t.Errorf("expected 8 APIs, got %d", len(prog.APIs))
	} else {
		apiNames := []string{"SignUp", "Login", "GetTasks", "CreateTask",
			"UpdateTask", "DeleteTask", "GetProfile", "UpdateProfile"}
		for i, name := range apiNames {
			if prog.APIs[i].Name != name {
				t.Errorf("api[%d]: expected %q, got %q", i, name, prog.APIs[i].Name)
			}
		}
		// CreateTask should require auth
		for _, api := range prog.APIs {
			if api.Name == "CreateTask" && !api.Auth {
				t.Error("CreateTask should require authentication")
			}
			if api.Name == "SignUp" && api.Auth {
				t.Error("SignUp should not require authentication")
			}
		}
	}

	// ── Policies ──
	if len(prog.Policies) != 3 {
		t.Errorf("expected 3 policies, got %d", len(prog.Policies))
	} else {
		polNames := []string{"FreeUser", "ProUser", "Admin"}
		for i, name := range polNames {
			if prog.Policies[i].Name != name {
				t.Errorf("policy[%d]: expected %q, got %q", i, name, prog.Policies[i].Name)
			}
		}
		// FreeUser should have both can and cannot rules
		freeUser := prog.Policies[0]
		hasCan := false
		hasCannot := false
		for _, r := range freeUser.Rules {
			if r.Allowed {
				hasCan = true
			} else {
				hasCannot = true
			}
		}
		if !hasCan {
			t.Error("FreeUser: expected at least one 'can' rule")
		}
		if !hasCannot {
			t.Error("FreeUser: expected at least one 'cannot' rule")
		}
	}

	// ── Workflows ──
	if len(prog.Workflows) != 6 {
		t.Errorf("expected 6 workflows, got %d", len(prog.Workflows))
	}

	// ── Theme ──
	if prog.Theme == nil {
		t.Error("missing Theme declaration")
	} else if len(prog.Theme.Properties) < 3 {
		t.Errorf("Theme: expected at least 3 properties, got %d", len(prog.Theme.Properties))
	}

	// ── Authentication ──
	if prog.Authentication == nil {
		t.Error("missing Authentication declaration")
	} else if len(prog.Authentication.Statements) != 7 {
		t.Errorf("Authentication: expected 7 statements, got %d",
			len(prog.Authentication.Statements))
	}

	// ── Database ──
	if prog.Database == nil {
		t.Error("missing Database declaration")
	} else if len(prog.Database.Statements) != 8 {
		t.Errorf("Database: expected 8 statements, got %d",
			len(prog.Database.Statements))
	}

	// ── Integrations ──
	if len(prog.Integrations) != 3 {
		t.Errorf("expected 3 integrations, got %d", len(prog.Integrations))
	}

	// ── Environments ──
	if len(prog.Environments) != 2 {
		t.Errorf("expected 2 environments, got %d", len(prog.Environments))
	}

	// ── Error handlers ──
	if len(prog.ErrorHandlers) != 2 {
		t.Errorf("expected 2 error handlers, got %d", len(prog.ErrorHandlers))
	}

	// ── Build ──
	if prog.Build == nil {
		t.Error("missing Build declaration")
	} else if len(prog.Build.Statements) != 4 {
		t.Errorf("Build: expected 4 statements, got %d", len(prog.Build.Statements))
	}

	// ── Section headers ──
	if len(prog.Sections) != 11 {
		t.Errorf("expected 11 section headers, got %d", len(prog.Sections))
	} else {
		expectedSections := []string{"theme", "frontend", "backend", "security",
			"policies", "workflows", "logic", "database", "integrations", "devops", "build"}
		for i, name := range expectedSections {
			if prog.Sections[i] != name {
				t.Errorf("section[%d]: expected %q, got %q", i, name, prog.Sections[i])
			}
		}
	}

	t.Logf("Successfully parsed app.human:")
	t.Logf("  App: %s (%s)", prog.App.Name, prog.App.Platform)
	t.Logf("  Data models: %d", len(prog.Data))
	t.Logf("  Pages: %d", len(prog.Pages))
	t.Logf("  APIs: %d", len(prog.APIs))
	t.Logf("  Policies: %d", len(prog.Policies))
	t.Logf("  Workflows: %d", len(prog.Workflows))
	t.Logf("  Integrations: %d", len(prog.Integrations))
	t.Logf("  Environments: %d", len(prog.Environments))
	t.Logf("  Error handlers: %d", len(prog.ErrorHandlers))
	t.Logf("  Sections: %d", len(prog.Sections))
	t.Logf("  Top-level statements: %d", len(prog.Statements))
}

// ── Helpers ──

func hasModifier(f *Field, mod string) bool {
	for _, m := range f.Modifiers {
		if m == mod {
			return true
		}
	}
	return false
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && searchSubstring(s, sub)
}

func searchSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
