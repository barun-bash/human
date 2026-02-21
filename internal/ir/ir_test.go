package ir

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/parser"
)

// ── Helpers ──

// mustBuild parses source and builds IR, fataling on error.
func mustBuild(t *testing.T, source string) *Application {
	t.Helper()
	prog, err := parser.Parse(source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	app, err := Build(prog)
	if err != nil {
		t.Fatalf("build error: %v", err)
	}
	return app
}

// ── Build Config ──

func TestBuildConfig(t *testing.T) {
	source := `app MyApp is a web application

build with:
  frontend using React with TypeScript
  backend using Node with Express
  database using PostgreSQL
  deploy to Docker`

	app := mustBuild(t, source)

	if app.Config == nil {
		t.Fatal("expected Config")
	}
	if app.Config.Frontend != "React with TypeScript" {
		t.Errorf("frontend: got %q", app.Config.Frontend)
	}
	if app.Config.Backend != "Node with Express" {
		t.Errorf("backend: got %q", app.Config.Backend)
	}
	if app.Config.Database != "PostgreSQL" {
		t.Errorf("database: got %q", app.Config.Database)
	}
	if app.Config.Deploy != "Docker" {
		t.Errorf("deploy: got %q", app.Config.Deploy)
	}
}

// ── Data Models ──

func TestBuildDataModel(t *testing.T) {
	source := `data User:
  has a name which is text
  has an email which is unique email
  has a password which is encrypted text
  has an optional bio which is text
  has a role which is either "user" or "admin"
  has a created datetime
  has many Task`

	app := mustBuild(t, source)

	if len(app.Data) != 1 {
		t.Fatalf("expected 1 data model, got %d", len(app.Data))
	}
	m := app.Data[0]
	if m.Name != "User" {
		t.Errorf("name: got %q", m.Name)
	}
	if len(m.Fields) != 6 {
		t.Fatalf("expected 6 fields, got %d", len(m.Fields))
	}

	// name - text, required
	f := m.Fields[0]
	if f.Name != "name" || f.Type != "text" || !f.Required {
		t.Errorf("field 0: got %+v", f)
	}

	// email - unique email, required
	f = m.Fields[1]
	if f.Name != "email" || f.Type != "email" || !f.Unique || !f.Required {
		t.Errorf("field 1: got %+v", f)
	}

	// password - encrypted text, required
	f = m.Fields[2]
	if f.Name != "password" || f.Type != "text" || !f.Encrypted || !f.Required {
		t.Errorf("field 2: got %+v", f)
	}

	// bio - optional text
	f = m.Fields[3]
	if f.Name != "bio" || f.Type != "text" || f.Required {
		t.Errorf("field 3: got %+v", f)
	}

	// role - enum
	f = m.Fields[4]
	if f.Name != "role" || f.Type != "enum" || len(f.EnumValues) != 2 {
		t.Errorf("field 4: got %+v", f)
	}
	if f.EnumValues[0] != "user" || f.EnumValues[1] != "admin" {
		t.Errorf("enum values: got %v", f.EnumValues)
	}

	// created - datetime
	f = m.Fields[5]
	if f.Name != "created" || f.Type != "datetime" {
		t.Errorf("field 5: got %+v", f)
	}

	// Relationship: has many Task
	if len(m.Relations) != 1 {
		t.Fatalf("expected 1 relation, got %d", len(m.Relations))
	}
	r := m.Relations[0]
	if r.Kind != "has_many" || r.Target != "Task" {
		t.Errorf("relation: got %+v", r)
	}
}

func TestBuildDataManyThrough(t *testing.T) {
	source := `data Task:
  has a title which is text
  has many Tag through TaskTag`

	app := mustBuild(t, source)

	if len(app.Data[0].Relations) != 1 {
		t.Fatalf("expected 1 relation, got %d", len(app.Data[0].Relations))
	}
	r := app.Data[0].Relations[0]
	if r.Kind != "has_many_through" {
		t.Errorf("kind: got %q, want has_many_through", r.Kind)
	}
	if r.Target != "Tag" {
		t.Errorf("target: got %q", r.Target)
	}
	if r.Through != "TaskTag" {
		t.Errorf("through: got %q", r.Through)
	}
}

func TestBuildDataBelongsTo(t *testing.T) {
	source := `data Task:
  belongs to a User
  has a title which is text`

	app := mustBuild(t, source)

	if len(app.Data[0].Relations) != 1 {
		t.Fatalf("expected 1 relation")
	}
	r := app.Data[0].Relations[0]
	if r.Kind != "belongs_to" || r.Target != "User" {
		t.Errorf("relation: got %+v", r)
	}
}

// ── Pages ──

func TestBuildPage(t *testing.T) {
	source := `page Home:
  show a hero section with the app name
  clicking the button navigates to Dashboard
  if user is not logged in, show login button`

	app := mustBuild(t, source)

	if len(app.Pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(app.Pages))
	}
	p := app.Pages[0]
	if p.Name != "Home" {
		t.Errorf("name: got %q", p.Name)
	}
	if len(p.Content) != 3 {
		t.Fatalf("expected 3 actions, got %d", len(p.Content))
	}
	if p.Content[0].Type != "display" {
		t.Errorf("action 0 type: got %q, want display", p.Content[0].Type)
	}
	if p.Content[1].Type != "interact" {
		t.Errorf("action 1 type: got %q, want interact", p.Content[1].Type)
	}
	if p.Content[2].Type != "condition" {
		t.Errorf("action 2 type: got %q, want condition", p.Content[2].Type)
	}
}

// ── Components ──

func TestBuildComponent(t *testing.T) {
	source := `component TaskCard:
  accepts task as Task
  show the task title in bold
  if task is overdue, show the due date in red`

	app := mustBuild(t, source)

	if len(app.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(app.Components))
	}
	c := app.Components[0]
	if c.Name != "TaskCard" {
		t.Errorf("name: got %q", c.Name)
	}
	if len(c.Props) != 1 {
		t.Fatalf("expected 1 prop, got %d", len(c.Props))
	}
	if c.Props[0].Name != "task" || c.Props[0].Type != "Task" {
		t.Errorf("prop: got %+v", c.Props[0])
	}
	if len(c.Content) != 2 {
		t.Fatalf("expected 2 content actions, got %d", len(c.Content))
	}
}

// ── API Endpoints ──

func TestBuildEndpointBasic(t *testing.T) {
	source := `api GetTasks:
  requires authentication
  fetch all tasks for the current user
  respond with tasks`

	app := mustBuild(t, source)

	if len(app.APIs) != 1 {
		t.Fatalf("expected 1 API, got %d", len(app.APIs))
	}
	ep := app.APIs[0]
	if ep.Name != "GetTasks" {
		t.Errorf("name: got %q", ep.Name)
	}
	if !ep.Auth {
		t.Error("expected Auth=true")
	}
	if len(ep.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(ep.Steps))
	}
}

func TestBuildEndpointValidation(t *testing.T) {
	source := `api SignUp:
  accepts name, email, and password
  check that name is not empty
  check that email is a valid email
  check that password is at least 8 characters
  check that email is not already taken
  create a User with the given fields
  respond with the created user`

	app := mustBuild(t, source)

	ep := app.APIs[0]
	if len(ep.Params) != 3 {
		t.Fatalf("expected 3 params, got %d", len(ep.Params))
	}
	if ep.Params[0].Name != "name" || ep.Params[1].Name != "email" || ep.Params[2].Name != "password" {
		t.Errorf("params: got %v %v %v", ep.Params[0].Name, ep.Params[1].Name, ep.Params[2].Name)
	}

	if len(ep.Validation) != 4 {
		t.Fatalf("expected 4 validation rules, got %d", len(ep.Validation))
	}

	// check that name is not empty
	v := ep.Validation[0]
	if v.Field != "name" || v.Rule != "not_empty" {
		t.Errorf("validation 0: got %+v", v)
	}

	// check that email is a valid email
	v = ep.Validation[1]
	if v.Field != "email" || v.Rule != "valid_email" {
		t.Errorf("validation 1: got %+v", v)
	}

	// check that password is at least 8 characters
	v = ep.Validation[2]
	if v.Field != "password" || v.Rule != "min_length" || v.Value != "8" {
		t.Errorf("validation 2: got %+v", v)
	}

	// check that email is not already taken
	v = ep.Validation[3]
	if v.Field != "email" || v.Rule != "unique" {
		t.Errorf("validation 3: got %+v", v)
	}

	// Non-validation steps
	if len(ep.Steps) != 2 {
		t.Errorf("expected 2 steps after validation extraction, got %d", len(ep.Steps))
	}
}

func TestBuildEndpointMaxLength(t *testing.T) {
	source := `api CreateTask:
  requires authentication
  accepts title
  check that title is less than 200 characters
  respond with the task`

	app := mustBuild(t, source)
	ep := app.APIs[0]

	if len(ep.Validation) != 1 {
		t.Fatalf("expected 1 validation rule, got %d", len(ep.Validation))
	}
	v := ep.Validation[0]
	if v.Rule != "max_length" || v.Value != "200" {
		t.Errorf("got rule=%q value=%q", v.Rule, v.Value)
	}
}

func TestBuildEndpointFutureDate(t *testing.T) {
	source := `api CreateTask:
  requires authentication
  accepts due_date
  check that due date is in the future
  respond with the task`

	app := mustBuild(t, source)
	ep := app.APIs[0]

	if len(ep.Validation) != 1 {
		t.Fatalf("expected 1 validation rule, got %d", len(ep.Validation))
	}
	if ep.Validation[0].Rule != "future_date" {
		t.Errorf("got rule=%q", ep.Validation[0].Rule)
	}
}

func TestBuildEndpointAuthorizationCheck(t *testing.T) {
	source := `api UpdateTask:
  requires authentication
  accepts task_id
  check that current user is the owner or an admin
  update the task
  respond with the task`

	app := mustBuild(t, source)
	ep := app.APIs[0]

	if len(ep.Validation) != 1 {
		t.Fatalf("expected 1 validation rule, got %d", len(ep.Validation))
	}
	v := ep.Validation[0]
	if v.Field != "current_user" || v.Rule != "authorization" {
		t.Errorf("got %+v", v)
	}
}

func TestBuildEndpointMatchesValidation(t *testing.T) {
	source := `api ChangePassword:
  accepts old_password, new_password, and confirm_password
  check that new_password matches confirm_password
  respond with success`

	app := mustBuild(t, source)
	ep := app.APIs[0]

	if len(ep.Validation) != 1 {
		t.Fatalf("expected 1 validation rule, got %d", len(ep.Validation))
	}
	if ep.Validation[0].Rule != "matches" {
		t.Errorf("got rule=%q", ep.Validation[0].Rule)
	}
}

// ── Policies ──

func TestBuildPolicy(t *testing.T) {
	source := `policy FreeUser:
  can create up to 50 tasks per month
  can view only their own tasks
  cannot delete completed tasks
  cannot export data`

	app := mustBuild(t, source)

	if len(app.Policies) != 1 {
		t.Fatalf("expected 1 policy, got %d", len(app.Policies))
	}
	pol := app.Policies[0]
	if pol.Name != "FreeUser" {
		t.Errorf("name: got %q", pol.Name)
	}
	if len(pol.Permissions) != 2 {
		t.Errorf("expected 2 permissions, got %d", len(pol.Permissions))
	}
	if len(pol.Restrictions) != 2 {
		t.Errorf("expected 2 restrictions, got %d", len(pol.Restrictions))
	}
}

// ── Workflows & Pipelines ──

func TestBuildWorkflow(t *testing.T) {
	source := `when a user signs up:
  create their account
  send welcome email`

	app := mustBuild(t, source)

	if len(app.Workflows) != 1 {
		t.Fatalf("expected 1 workflow, got %d", len(app.Workflows))
	}
	wf := app.Workflows[0]
	if wf.Trigger != "a user signs up" {
		t.Errorf("trigger: got %q", wf.Trigger)
	}
	if len(wf.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(wf.Steps))
	}
	if wf.Steps[0].Type != "create" {
		t.Errorf("step 0 type: got %q", wf.Steps[0].Type)
	}
	if wf.Steps[1].Type != "send" {
		t.Errorf("step 1 type: got %q", wf.Steps[1].Type)
	}
}

func TestBuildPipeline(t *testing.T) {
	source := `when code is pushed to a feature branch:
  run all tests
  check code formatting
  report results back to the pull request`

	app := mustBuild(t, source)

	if len(app.Pipelines) != 1 {
		t.Fatalf("expected 1 pipeline, got %d", len(app.Pipelines))
	}
	if len(app.Workflows) != 0 {
		t.Errorf("expected 0 workflows, got %d", len(app.Workflows))
	}
	p := app.Pipelines[0]
	if !strings.Contains(p.Trigger, "code is pushed") {
		t.Errorf("trigger: got %q", p.Trigger)
	}
	if len(p.Steps) != 3 {
		t.Errorf("expected 3 steps, got %d", len(p.Steps))
	}
}

func TestPipelineMergedTrigger(t *testing.T) {
	source := `when code is merged to main:
  run all tests
  deploy to production`

	app := mustBuild(t, source)

	if len(app.Pipelines) != 1 {
		t.Fatalf("expected 1 pipeline, got %d", len(app.Pipelines))
	}
	if len(app.Workflows) != 0 {
		t.Errorf("should not create workflow for pipeline trigger")
	}
}

// ── Theme ──

func TestBuildTheme(t *testing.T) {
	source := `theme:
  primary color is #6C5CE7
  secondary color is #00B894
  font is Inter for body and Poppins for headings
  dark mode is supported`

	app := mustBuild(t, source)

	if app.Theme == nil {
		t.Fatal("expected Theme")
	}
	if app.Theme.Colors["primary"] != "#6c5ce7" {
		t.Errorf("primary color: got %q", app.Theme.Colors["primary"])
	}
	if app.Theme.Colors["secondary"] != "#00b894" {
		t.Errorf("secondary color: got %q", app.Theme.Colors["secondary"])
	}
	if app.Theme.Fonts["body"] != "Inter" {
		t.Errorf("body font: got %q", app.Theme.Fonts["body"])
	}
	if app.Theme.Fonts["headings"] != "Poppins" {
		t.Errorf("headings font: got %q", app.Theme.Fonts["headings"])
	}
	if _, ok := app.Theme.Options["dark mode"]; !ok {
		t.Error("expected 'dark mode' in options")
	}
}

func TestBuildTheme_DesignSystem(t *testing.T) {
	source := `theme:
  design system is Material UI
  primary color is #1976d2`

	app := mustBuild(t, source)

	if app.Theme == nil {
		t.Fatal("expected Theme")
	}
	if app.Theme.DesignSystem != "material" {
		t.Errorf("design system: got %q, want \"material\"", app.Theme.DesignSystem)
	}
	if app.Theme.Colors["primary"] != "#1976d2" {
		t.Errorf("primary color: got %q", app.Theme.Colors["primary"])
	}
}

func TestBuildTheme_BorderRadius(t *testing.T) {
	source := `theme:
  border radius is smooth`

	app := mustBuild(t, source)

	if app.Theme == nil {
		t.Fatal("expected Theme")
	}
	if app.Theme.BorderRadius != "smooth" {
		t.Errorf("border radius: got %q, want \"smooth\"", app.Theme.BorderRadius)
	}
}

func TestBuildTheme_Spacing(t *testing.T) {
	source := `theme:
  spacing is compact`

	app := mustBuild(t, source)

	if app.Theme == nil {
		t.Fatal("expected Theme")
	}
	if app.Theme.Spacing != "compact" {
		t.Errorf("spacing: got %q, want \"compact\"", app.Theme.Spacing)
	}
}

func TestBuildTheme_DarkMode(t *testing.T) {
	source := `theme:
  dark mode is supported and toggles from the header`

	app := mustBuild(t, source)

	if app.Theme == nil {
		t.Fatal("expected Theme")
	}
	if !app.Theme.DarkMode {
		t.Error("expected DarkMode=true")
	}
	// Should also be in Options
	if _, ok := app.Theme.Options["dark mode"]; !ok {
		t.Error("expected 'dark mode' in Options")
	}
}

func TestBuildTheme_NormalizeDesignSystem(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Material UI", "material"},
		{"MUI", "material"},
		{"shadcn/ui", "shadcn"},
		{"Tailwind CSS", "tailwind"},
		{"Ant Design", "ant"},
		{"Chakra UI", "chakra"},
		{"Bootstrap", "bootstrap"},
		{"Untitled UI", "untitled"},
	}

	for _, tt := range tests {
		got := normalizeDesignSystem(tt.input)
		if got != tt.want {
			t.Errorf("normalizeDesignSystem(%q): got %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ── Authentication ──

func TestBuildAuth(t *testing.T) {
	source := `authentication:
  method JWT tokens that expire in 7 days
  method Google OAuth with redirect to /auth/google/callback
  rate limit all endpoints to 100 requests per minute per user`

	app := mustBuild(t, source)

	if app.Auth == nil {
		t.Fatal("expected Auth")
	}
	if len(app.Auth.Methods) != 2 {
		t.Fatalf("expected 2 methods, got %d", len(app.Auth.Methods))
	}

	jwt := app.Auth.Methods[0]
	if jwt.Type != "jwt" {
		t.Errorf("method 0 type: got %q", jwt.Type)
	}
	if jwt.Config["expiration"] != "7 days" {
		t.Errorf("jwt expiration: got %q", jwt.Config["expiration"])
	}

	oauth := app.Auth.Methods[1]
	if oauth.Type != "oauth" {
		t.Errorf("method 1 type: got %q", oauth.Type)
	}
	if oauth.Provider != "Google" {
		t.Errorf("oauth provider: got %q", oauth.Provider)
	}
	// Note: parser strips slashes and dots from tokens, so the callback URL
	// becomes "auth google callback" after tokenization/reconstruction.
	if oauth.Config["callback_url"] != "auth google callback" {
		t.Errorf("callback_url: got %q", oauth.Config["callback_url"])
	}

	// Non-method statements become rules
	if len(app.Auth.Rules) != 1 {
		t.Errorf("expected 1 rule, got %d", len(app.Auth.Rules))
	}
}

// ── Database ──

func TestBuildDatabase(t *testing.T) {
	source := `database:
  use PostgreSQL
  index User by email
  index Task by user and status
  backup daily at 3am`

	app := mustBuild(t, source)

	if app.Database == nil {
		t.Fatal("expected Database")
	}
	if app.Database.Engine != "PostgreSQL" {
		t.Errorf("engine: got %q", app.Database.Engine)
	}
	if len(app.Database.Indexes) != 2 {
		t.Fatalf("expected 2 indexes, got %d", len(app.Database.Indexes))
	}

	idx0 := app.Database.Indexes[0]
	if idx0.Entity != "User" || len(idx0.Fields) != 1 || idx0.Fields[0] != "email" {
		t.Errorf("index 0: got %+v", idx0)
	}

	idx1 := app.Database.Indexes[1]
	if idx1.Entity != "Task" || len(idx1.Fields) != 2 {
		t.Errorf("index 1: got %+v", idx1)
	}
	if idx1.Fields[0] != "user" || idx1.Fields[1] != "status" {
		t.Errorf("index 1 fields: got %v", idx1.Fields)
	}

	// backup rule
	if len(app.Database.Rules) != 1 {
		t.Errorf("expected 1 rule, got %d", len(app.Database.Rules))
	}
}

// ── Integrations ──

func TestBuildIntegration(t *testing.T) {
	source := `integrate with SendGrid:
  api key from environment variable SENDGRID_API_KEY
  use for sending transactional emails`

	app := mustBuild(t, source)

	if len(app.Integrations) != 1 {
		t.Fatalf("expected 1 integration, got %d", len(app.Integrations))
	}
	integ := app.Integrations[0]
	if integ.Service != "SendGrid" {
		t.Errorf("service: got %q", integ.Service)
	}
	if integ.Type != "email" {
		t.Errorf("type: got %q, want %q", integ.Type, "email")
	}
	if integ.Credentials["api key"] != "SENDGRID_API_KEY" {
		t.Errorf("credentials: got %v", integ.Credentials)
	}
	if integ.Purpose != "sending transactional emails" {
		t.Errorf("purpose: got %q", integ.Purpose)
	}
}

func TestBuildIntegrationMultipleCredentials(t *testing.T) {
	source := `integrate with AWS S3:
  api key from environment variable AWS_ACCESS_KEY
  secret from environment variable AWS_SECRET_KEY
  use for storing files`

	app := mustBuild(t, source)

	integ := app.Integrations[0]
	if integ.Type != "storage" {
		t.Errorf("type: got %q, want %q", integ.Type, "storage")
	}
	if integ.Credentials["api key"] != "AWS_ACCESS_KEY" {
		t.Errorf("api key cred: got %v", integ.Credentials)
	}
	if integ.Credentials["secret"] != "AWS_SECRET_KEY" {
		t.Errorf("secret cred: got %v", integ.Credentials)
	}
}

func TestBuildIntegrationRichConfig(t *testing.T) {
	source := `integrate with AWS S3:
  api key from environment variable AWS_ACCESS_KEY
  secret from environment variable AWS_SECRET_KEY
  region is "us-east-1"
  bucket is "user-uploads"
  use for storing user avatars`

	app := mustBuild(t, source)

	integ := app.Integrations[0]
	if integ.Type != "storage" {
		t.Errorf("type: got %q, want %q", integ.Type, "storage")
	}
	if integ.Config["region"] != "us-east-1" {
		t.Errorf("region: got %q", integ.Config["region"])
	}
	if integ.Config["bucket"] != "user-uploads" {
		t.Errorf("bucket: got %q", integ.Config["bucket"])
	}
}

func TestBuildIntegrationWebhookAndTemplates(t *testing.T) {
	source := `integrate with Stripe:
  api key from environment variable STRIPE_SECRET_KEY
  webhook endpoint is "/webhooks/stripe"
  use for processing payments`

	app := mustBuild(t, source)

	integ := app.Integrations[0]
	if integ.Type != "payment" {
		t.Errorf("type: got %q, want %q", integ.Type, "payment")
	}
	if integ.Config["webhook_endpoint"] != "/webhooks/stripe" {
		t.Errorf("webhook_endpoint: got %q", integ.Config["webhook_endpoint"])
	}
}

func TestBuildIntegrationSlackChannel(t *testing.T) {
	source := `integrate with Slack:
  api key from environment variable SLACK_WEBHOOK_URL
  channel is "#engineering"
  use for team notifications and alerts`

	app := mustBuild(t, source)

	integ := app.Integrations[0]
	if integ.Type != "messaging" {
		t.Errorf("type: got %q, want %q", integ.Type, "messaging")
	}
	if integ.Config["channel"] != "#engineering" {
		t.Errorf("channel: got %q", integ.Config["channel"])
	}
}

func TestBuildIntegrationEmailConfig(t *testing.T) {
	source := `integrate with SendGrid:
  api key from environment variable SENDGRID_API_KEY
  sender email is "noreply@example.com"
  template "welcome"
  template "password-reset"
  use for sending transactional emails`

	app := mustBuild(t, source)

	integ := app.Integrations[0]
	if integ.Config["sender_email"] != "noreply@example.com" {
		t.Errorf("sender_email: got %q", integ.Config["sender_email"])
	}
	if len(integ.Templates) != 2 {
		t.Fatalf("templates: got %d, want 2", len(integ.Templates))
	}
	if integ.Templates[0] != "welcome" {
		t.Errorf("template 0: got %q", integ.Templates[0])
	}
	if integ.Templates[1] != "password-reset" {
		t.Errorf("template 1: got %q", integ.Templates[1])
	}
}

func TestInferIntegrationType(t *testing.T) {
	tests := []struct {
		service string
		want    string
	}{
		{"SendGrid", "email"},
		{"Mailgun", "email"},
		{"AWS S3", "storage"},
		{"GCS", "storage"},
		{"Cloudinary", "storage"},
		{"Stripe", "payment"},
		{"PayPal", "payment"},
		{"Slack", "messaging"},
		{"Discord", "messaging"},
		{"Twilio", "messaging"},
		{"Google", "oauth"},
		{"GitHub", "oauth"},
		{"Auth0", "oauth"},
		{"CustomService", ""},
	}

	for _, tt := range tests {
		t.Run(tt.service, func(t *testing.T) {
			got := InferIntegrationType(tt.service)
			if got != tt.want {
				t.Errorf("InferIntegrationType(%q) = %q, want %q", tt.service, got, tt.want)
			}
		})
	}
}

// ── Environments ──

func TestBuildEnvironment(t *testing.T) {
	source := `environment staging:
  url is staging example com
  uses staging database`

	// Note: parser strips dots from tokens, so "staging.example.com"
	// becomes "staging example com" after tokenization/reconstruction.
	app := mustBuild(t, source)

	if len(app.Environments) != 1 {
		t.Fatalf("expected 1 environment, got %d", len(app.Environments))
	}
	env := app.Environments[0]
	if env.Name != "staging" {
		t.Errorf("name: got %q", env.Name)
	}
	if env.Config["url"] != "staging example com" {
		t.Errorf("url config: got %q", env.Config["url"])
	}
}

// ── Error Handlers ──

func TestBuildErrorHandler(t *testing.T) {
	source := `if database is unreachable:
  retry 3 times with 1 second delay
  alert the engineering team via Slack`

	app := mustBuild(t, source)

	if len(app.ErrorHandlers) != 1 {
		t.Fatalf("expected 1 error handler, got %d", len(app.ErrorHandlers))
	}
	eh := app.ErrorHandlers[0]
	if eh.Condition != "database is unreachable" {
		t.Errorf("condition: got %q", eh.Condition)
	}
	if len(eh.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(eh.Steps))
	}
	if eh.Steps[0].Type != "retry" {
		t.Errorf("step 0 type: got %q", eh.Steps[0].Type)
	}
	if eh.Steps[1].Type != "alert" {
		t.Errorf("step 1 type: got %q", eh.Steps[1].Type)
	}
}

// ── Action Classification ──

func TestClassifyAction(t *testing.T) {
	tests := []struct {
		kind     string
		wantType string
	}{
		{"show", "display"},
		{"display", "display"},
		{"render", "display"},
		{"clicking", "interact"},
		{"dragging", "interact"},
		{"there", "input"},
		{"navigate", "navigate"},
		{"if", "condition"},
		{"when", "condition"},
		{"unless", "condition"},
		{"each", "loop"},
		{"every", "loop"},
		{"fetch", "query"},
		{"get", "query"},
		{"find", "query"},
		{"paginate", "query"},
		{"sort", "query"},
		{"create", "create"},
		{"update", "update"},
		{"set", "update"},
		{"delete", "delete"},
		{"remove", "delete"},
		{"check", "validate"},
		{"validate", "validate"},
		{"respond", "respond"},
		{"send", "send"},
		{"notify", "send"},
		{"assign", "assign"},
		{"alert", "alert"},
		{"log", "log"},
		{"track", "log"},
		{"after", "delay"},
		{"retry", "retry"},
		{"run", "configure"},
		{"deploy", "configure"},
	}

	for _, tt := range tests {
		stmt := &parser.Statement{Kind: tt.kind, Text: "test"}
		action := classifyAction(stmt)
		if action.Type != tt.wantType {
			t.Errorf("classifyAction(%q): got %q, want %q", tt.kind, action.Type, tt.wantType)
		}
	}
}

// ── Serialization ──

func TestToJSONRoundTrip(t *testing.T) {
	app := &Application{
		Name:     "TestApp",
		Platform: "web",
		Config:   &BuildConfig{Frontend: "React"},
		Data: []*DataModel{
			{
				Name: "User",
				Fields: []*DataField{
					{Name: "email", Type: "email", Required: true, Unique: true},
				},
			},
		},
	}

	data, err := ToJSON(app)
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	app2, err := FromJSON(data)
	if err != nil {
		t.Fatalf("FromJSON: %v", err)
	}

	if app2.Name != app.Name {
		t.Errorf("name: got %q, want %q", app2.Name, app.Name)
	}
	if app2.Platform != app.Platform {
		t.Errorf("platform: got %q", app2.Platform)
	}
	if app2.Config.Frontend != "React" {
		t.Errorf("config frontend: got %q", app2.Config.Frontend)
	}
	if len(app2.Data) != 1 || app2.Data[0].Name != "User" {
		t.Errorf("data: got %+v", app2.Data)
	}
}

func TestFromJSONInvalid(t *testing.T) {
	_, err := FromJSON([]byte("{invalid"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestToYAMLBasic(t *testing.T) {
	app := &Application{
		Name:     "TestApp",
		Platform: "web",
		Data: []*DataModel{
			{
				Name: "User",
				Fields: []*DataField{
					{Name: "email", Type: "email", Required: true},
				},
			},
		},
	}

	yaml, err := ToYAML(app)
	if err != nil {
		t.Fatalf("ToYAML: %v", err)
	}

	// Check key content is present
	if !strings.Contains(yaml, "name: TestApp") {
		t.Errorf("expected 'name: TestApp' in YAML, got:\n%s", yaml)
	}
	if !strings.Contains(yaml, "platform: web") {
		t.Errorf("expected 'platform: web' in YAML")
	}
	if !strings.Contains(yaml, "type: email") {
		t.Errorf("expected 'type: email' in YAML")
	}
}

func TestToYAMLEmptyCollections(t *testing.T) {
	app := &Application{
		Name:     "Empty",
		Platform: "web",
	}

	yaml, err := ToYAML(app)
	if err != nil {
		t.Fatalf("ToYAML: %v", err)
	}
	if !strings.Contains(yaml, "name: Empty") {
		t.Errorf("expected 'name: Empty' in YAML, got:\n%s", yaml)
	}
}

func TestYAMLKeyOrdering(t *testing.T) {
	app := &Application{
		Name:     "OrderTest",
		Platform: "web",
		Config:   &BuildConfig{Frontend: "React"},
	}

	yaml, err := ToYAML(app)
	if err != nil {
		t.Fatalf("ToYAML: %v", err)
	}

	nameIdx := strings.Index(yaml, "name:")
	platformIdx := strings.Index(yaml, "platform:")
	configIdx := strings.Index(yaml, "config:")

	if nameIdx == -1 || platformIdx == -1 || configIdx == -1 {
		t.Fatalf("missing keys in YAML:\n%s", yaml)
	}

	// name should come before platform, platform before config
	if nameIdx >= platformIdx {
		t.Error("name should appear before platform")
	}
	if platformIdx >= configIdx {
		t.Error("platform should appear before config")
	}
}

func TestYAMLStringQuoting(t *testing.T) {
	tests := []struct {
		input string
		want  bool // needs quoting?
	}{
		{"", true},
		{"true", true},
		{"false", true},
		{"null", true},
		{"~", true},
		{"hello", false},
		{"hello world", false},
		{"#comment", true},
		{"key: value", true},
		{"123", true},
		{"-42", true},
		{"3.14", true},
		{"normal text", false},
	}

	for _, tt := range tests {
		got := needsYAMLQuoting(tt.input)
		if got != tt.want {
			t.Errorf("needsYAMLQuoting(%q): got %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestLooksLikeNumber(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"123", true},
		{"-42", true},
		{"+7", true},
		{"3.14", true},
		{"0", true},
		{"", false},
		{"abc", false},
		{"-", false},
		{"+", false},
		{"12.34.56", false},
	}

	for _, tt := range tests {
		got := looksLikeNumber(tt.input)
		if got != tt.want {
			t.Errorf("looksLikeNumber(%q): got %v, want %v", tt.input, got, tt.want)
		}
	}
}

// ── String Helpers ──

func TestExtractAfter(t *testing.T) {
	got := extractAfter("is a valid email", "is a valid ")
	if got != "email" {
		t.Errorf("got %q", got)
	}
	got = extractAfter("no match here", "xyz ")
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestExtractBetween(t *testing.T) {
	got := extractBetween("is at least 8 characters long", "is at least ", " characters")
	if got != "8" {
		t.Errorf("got %q", got)
	}
	got = extractBetween("no start marker", "xxx ", " yyy")
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

// ── Pipeline vs Workflow Trigger Detection ──

func TestIsPipelineTrigger(t *testing.T) {
	tests := []struct {
		event string
		want  bool
	}{
		{"code is pushed to main", true},
		{"code is merged to staging", true},
		{"Code is pushed to a feature branch", true},
		{"a user signs up", false},
		{"a task becomes overdue", false},
	}

	for _, tt := range tests {
		got := isPipelineTrigger(tt.event)
		if got != tt.want {
			t.Errorf("isPipelineTrigger(%q): got %v, want %v", tt.event, got, tt.want)
		}
	}
}

// ── Full Integration Test ──

func TestFullIntegration(t *testing.T) {
	// Locate examples/taskflow/app.human relative to this test file
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..")
	humanFile := filepath.Join(root, "examples", "taskflow", "app.human")

	source, err := os.ReadFile(humanFile)
	if err != nil {
		t.Fatalf("failed to read app.human: %v", err)
	}

	// Parse
	prog, err := parser.Parse(string(source))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	// Build IR
	app, err := Build(prog)
	if err != nil {
		t.Fatalf("build error: %v", err)
	}

	// ── Verify top-level structure ──
	if app.Name != "TaskFlow" {
		t.Errorf("app name: got %q, want TaskFlow", app.Name)
	}
	if app.Platform != "web" {
		t.Errorf("platform: got %q, want web", app.Platform)
	}

	// Config
	if app.Config == nil {
		t.Fatal("expected Config")
	}
	if app.Config.Frontend != "React with TypeScript" {
		t.Errorf("frontend: got %q", app.Config.Frontend)
	}
	if app.Config.Backend != "Node with Express" {
		t.Errorf("backend: got %q", app.Config.Backend)
	}

	// Data models
	if len(app.Data) != 4 {
		t.Errorf("expected 4 data models, got %d", len(app.Data))
	} else {
		names := make(map[string]bool)
		for _, d := range app.Data {
			names[d.Name] = true
		}
		for _, want := range []string{"User", "Task", "Tag", "TaskTag"} {
			if !names[want] {
				t.Errorf("missing data model %q", want)
			}
		}
	}

	// Pages
	if len(app.Pages) != 3 {
		t.Errorf("expected 3 pages, got %d", len(app.Pages))
	} else {
		pageNames := []string{app.Pages[0].Name, app.Pages[1].Name, app.Pages[2].Name}
		expected := []string{"Home", "Dashboard", "Profile"}
		for i, want := range expected {
			if pageNames[i] != want {
				t.Errorf("page %d: got %q, want %q", i, pageNames[i], want)
			}
		}
	}

	// Components
	if len(app.Components) != 1 {
		t.Errorf("expected 1 component, got %d", len(app.Components))
	} else if app.Components[0].Name != "TaskCard" {
		t.Errorf("component name: got %q", app.Components[0].Name)
	}

	// APIs
	if len(app.APIs) != 8 {
		t.Errorf("expected 8 APIs, got %d", len(app.APIs))
	} else {
		expectedAPIs := []string{"SignUp", "Login", "GetTasks", "CreateTask", "UpdateTask", "DeleteTask", "GetProfile", "UpdateProfile"}
		for i, want := range expectedAPIs {
			if app.APIs[i].Name != want {
				t.Errorf("API %d: got %q, want %q", i, app.APIs[i].Name, want)
			}
		}
		// SignUp should have 4 validation rules
		if len(app.APIs[0].Validation) != 4 {
			t.Errorf("SignUp validation rules: got %d, want 4", len(app.APIs[0].Validation))
		}
		// GetTasks should require auth
		if !app.APIs[2].Auth {
			t.Error("GetTasks should require auth")
		}
		// SignUp should not require auth
		if app.APIs[0].Auth {
			t.Error("SignUp should not require auth")
		}
	}

	// Policies
	if len(app.Policies) != 3 {
		t.Errorf("expected 3 policies, got %d", len(app.Policies))
	} else {
		for _, want := range []string{"FreeUser", "ProUser", "Admin"} {
			found := false
			for _, p := range app.Policies {
				if p.Name == want {
					found = true
				}
			}
			if !found {
				t.Errorf("missing policy %q", want)
			}
		}
		// FreeUser: 3 permissions, 2 restrictions
		free := app.Policies[0]
		if free.Name == "FreeUser" {
			if len(free.Permissions) != 3 {
				t.Errorf("FreeUser permissions: got %d, want 3", len(free.Permissions))
			}
			if len(free.Restrictions) != 2 {
				t.Errorf("FreeUser restrictions: got %d, want 2", len(free.Restrictions))
			}
		}
	}

	// Workflows (business workflows, not pipelines)
	if len(app.Workflows) != 3 {
		t.Errorf("expected 3 workflows, got %d", len(app.Workflows))
	}

	// Pipelines (CI/CD triggers)
	if len(app.Pipelines) != 3 {
		t.Errorf("expected 3 pipelines, got %d", len(app.Pipelines))
	}

	// Theme
	if app.Theme == nil {
		t.Fatal("expected Theme")
	}
	if len(app.Theme.Colors) < 3 {
		t.Errorf("expected at least 3 colors, got %d", len(app.Theme.Colors))
	}
	if len(app.Theme.Fonts) < 2 {
		t.Errorf("expected at least 2 fonts, got %d", len(app.Theme.Fonts))
	}

	// Auth
	if app.Auth == nil {
		t.Fatal("expected Auth")
	}
	if len(app.Auth.Methods) != 2 {
		t.Errorf("expected 2 auth methods, got %d", len(app.Auth.Methods))
	}

	// Database
	if app.Database == nil {
		t.Fatal("expected Database")
	}
	if app.Database.Engine != "PostgreSQL" {
		t.Errorf("database engine: got %q", app.Database.Engine)
	}
	if len(app.Database.Indexes) != 4 {
		t.Errorf("expected 4 indexes, got %d", len(app.Database.Indexes))
	}

	// Integrations
	if len(app.Integrations) != 3 {
		t.Errorf("expected 3 integrations, got %d", len(app.Integrations))
	} else {
		services := []string{app.Integrations[0].Service, app.Integrations[1].Service, app.Integrations[2].Service}
		for _, want := range []string{"SendGrid", "AWS S3", "Slack"} {
			found := false
			for _, s := range services {
				if s == want {
					found = true
				}
			}
			if !found {
				t.Errorf("missing integration %q", want)
			}
		}
	}

	// Environments
	if len(app.Environments) != 2 {
		t.Errorf("expected 2 environments, got %d", len(app.Environments))
	}

	// Error handlers
	if len(app.ErrorHandlers) != 2 {
		t.Errorf("expected 2 error handlers, got %d", len(app.ErrorHandlers))
	}

	// ── Serialize to JSON and verify round-trip ──
	jsonBytes, err := ToJSON(app)
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	// Verify it's valid JSON
	var raw map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &raw); err != nil {
		t.Fatalf("JSON output is not valid: %v", err)
	}

	// Round-trip
	app2, err := FromJSON(jsonBytes)
	if err != nil {
		t.Fatalf("FromJSON round-trip: %v", err)
	}
	if app2.Name != "TaskFlow" {
		t.Errorf("round-trip name: got %q", app2.Name)
	}
	if len(app2.APIs) != 8 {
		t.Errorf("round-trip APIs: got %d", len(app2.APIs))
	}

	// ── Serialize to YAML ──
	yaml, err := ToYAML(app)
	if err != nil {
		t.Fatalf("ToYAML: %v", err)
	}

	// Verify YAML contains expected content
	if !strings.Contains(yaml, "name: TaskFlow") {
		t.Error("YAML missing 'name: TaskFlow'")
	}
	if !strings.Contains(yaml, "platform: web") {
		t.Error("YAML missing 'platform: web'")
	}
	if !strings.Contains(yaml, "frontend: React with TypeScript") {
		t.Error("YAML missing frontend config")
	}
	if !strings.Contains(yaml, "engine: PostgreSQL") {
		t.Error("YAML missing database engine")
	}

	// Log the YAML output for manual inspection
	t.Logf("YAML output length: %d bytes", len(yaml))
}
