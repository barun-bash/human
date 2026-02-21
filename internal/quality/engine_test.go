package quality

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/ir"
	"github.com/barun-bash/human/internal/parser"
)

// ── Helpers ──

func mustBuild(t *testing.T, source string) *ir.Application {
	t.Helper()
	prog, err := parser.Parse(source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	app, err := ir.Build(prog)
	if err != nil {
		t.Fatalf("build error: %v", err)
	}
	return app
}

func exampleApp(t *testing.T) *ir.Application {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..")
	source, err := os.ReadFile(filepath.Join(root, "examples", "taskflow", "app.human"))
	if err != nil {
		t.Fatalf("reading example: %v", err)
	}
	return mustBuild(t, string(source))
}

// ── Test Generation ──

func TestGenerateTests(t *testing.T) {
	app := exampleApp(t)
	dir := t.TempDir()
	testDir := filepath.Join(dir, "node", "src", "__tests__")

	files, count, err := generateTests(app, testDir)
	if err != nil {
		t.Fatalf("generateTests: %v", err)
	}

	if files == 0 {
		t.Fatal("expected test files, got 0")
	}
	if count == 0 {
		t.Fatal("expected test count > 0")
	}

	// Verify files were created
	entries, err := os.ReadDir(testDir)
	if err != nil {
		t.Fatalf("reading test dir: %v", err)
	}
	if len(entries) != files {
		t.Errorf("expected %d files on disk, got %d", files, len(entries))
	}

	// All files should end with .test.ts
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".test.ts") {
			t.Errorf("unexpected file: %s", e.Name())
		}
	}
}

func TestGenerateEndpointTests_HappyPath(t *testing.T) {
	ep := &ir.Endpoint{
		Name: "CreateTask",
		Params: []*ir.Param{
			{Name: "title"},
			{Name: "description"},
		},
	}
	content, count := generateEndpointTests(ep, &ir.Application{})

	if count < 1 {
		t.Errorf("expected at least 1 test, got %d", count)
	}
	if !strings.Contains(content, "should succeed with valid request") {
		t.Error("missing happy path test")
	}
	if !strings.Contains(content, ".post(") {
		t.Error("expected POST method for Create prefix")
	}
	if !strings.Contains(content, "title") {
		t.Error("expected title param in request body")
	}
}

func TestGenerateEndpointTests_AuthRequired(t *testing.T) {
	ep := &ir.Endpoint{
		Name: "UpdateTask",
		Auth: true,
	}
	content, count := generateEndpointTests(ep, &ir.Application{})

	if count < 2 {
		t.Errorf("expected at least 2 tests (happy + auth), got %d", count)
	}
	if !strings.Contains(content, "should return 401 without auth token") {
		t.Error("missing auth required test")
	}
}

func TestGenerateEndpointTests_Validation(t *testing.T) {
	ep := &ir.Endpoint{
		Name: "SignUp",
		Params: []*ir.Param{
			{Name: "email"},
			{Name: "password"},
		},
		Validation: []*ir.ValidationRule{
			{Field: "email", Rule: "not_empty"},
			{Field: "email", Rule: "valid_email"},
		},
	}
	content, count := generateEndpointTests(ep, &ir.Application{})

	// happy + 2 validation
	if count < 3 {
		t.Errorf("expected at least 3 tests, got %d", count)
	}
	if !strings.Contains(content, "should reject empty email") {
		t.Error("missing not_empty validation test")
	}
	if !strings.Contains(content, "should reject invalid email") {
		t.Error("missing valid_email validation test")
	}
}

func TestGenerateEndpointTests_GetNotFound(t *testing.T) {
	ep := &ir.Endpoint{
		Name: "GetTasks",
	}
	content, count := generateEndpointTests(ep, &ir.Application{})

	if count < 2 {
		t.Errorf("expected at least 2 tests (happy + not-found), got %d", count)
	}
	if !strings.Contains(content, "should handle empty results") {
		t.Error("missing empty results test")
	}
}

// ── HTTP Method + Path Helpers ──

func TestHttpMethod(t *testing.T) {
	tests := []struct {
		name   string
		expect string
	}{
		{"GetTasks", "get"},
		{"CreateTask", "post"},
		{"UpdateTask", "put"},
		{"DeleteTask", "delete"},
		{"SignUp", "post"},
		{"Login", "post"},
	}
	for _, tt := range tests {
		got := httpMethod(tt.name)
		if got != tt.expect {
			t.Errorf("httpMethod(%q) = %q, want %q", tt.name, got, tt.expect)
		}
	}
}

func TestApiPath(t *testing.T) {
	tests := []struct {
		name   string
		expect string
	}{
		{"GetTasks", "/api/tasks"},
		{"CreateTask", "/api/task"},
		{"DeleteTask", "/api/task"},
		{"SignUp", "/api/sign-up"},
	}
	for _, tt := range tests {
		got := apiPath(tt.name)
		if got != tt.expect {
			t.Errorf("apiPath(%q) = %q, want %q", tt.name, got, tt.expect)
		}
	}
}

// ── Security Checks ──

func TestCheckMissingAuth(t *testing.T) {
	app := &ir.Application{
		APIs: []*ir.Endpoint{
			{Name: "GetTasks"},                // GET without auth — OK
			{Name: "CreateTask"},              // POST without auth — flagged
			{Name: "SignUp"},                  // signup without auth — OK
			{Name: "Login"},                   // login without auth — OK
			{Name: "UpdateTask", Auth: true},  // PUT with auth — OK
		},
	}

	findings := checkMissingAuth(app)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Target != "CreateTask" {
		t.Errorf("expected CreateTask flagged, got %s", findings[0].Target)
	}
	if findings[0].Severity != "critical" {
		t.Errorf("expected critical severity, got %s", findings[0].Severity)
	}
}

func TestCheckMissingValidation(t *testing.T) {
	app := &ir.Application{
		APIs: []*ir.Endpoint{
			{Name: "GetTasks"}, // no params — OK
			{Name: "CreateTask", Params: []*ir.Param{{Name: "title"}}}, // params, no validation — flagged
			{
				Name:       "SignUp",
				Params:     []*ir.Param{{Name: "email"}},
				Validation: []*ir.ValidationRule{{Field: "email", Rule: "valid_email"}},
			}, // has validation — OK
		},
	}

	findings := checkMissingValidation(app)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Target != "CreateTask" {
		t.Errorf("expected CreateTask, got %s", findings[0].Target)
	}
}

func TestCheckHardcodedSecrets(t *testing.T) {
	app := &ir.Application{
		Auth: &ir.Auth{
			Methods: []*ir.AuthMethod{
				{
					Type: "jwt",
					Config: map[string]string{
						"secret":     "super-long-hardcoded-secret-value",
						"expiration": "7 days",
					},
				},
			},
		},
		Integrations: []*ir.Integration{
			{
				Service: "SendGrid",
				Credentials: map[string]string{
					"API key": "SENDGRID_API_KEY",
				},
			},
			{
				Service: "Stripe",
				Credentials: map[string]string{
					"secret key": "sk_live_hardcoded123456",
				},
			},
		},
	}

	findings := checkHardcodedSecrets(app)
	// Should flag: jwt secret (hardcoded), stripe credential (not env var)
	if len(findings) < 2 {
		t.Fatalf("expected at least 2 findings, got %d", len(findings))
	}

	found := map[string]bool{}
	for _, f := range findings {
		found[f.Target] = true
	}
	if !found["authentication"] {
		t.Error("expected auth secret finding")
	}
	if !found["Stripe"] {
		t.Error("expected Stripe credential finding")
	}
}

func TestCheckRateLimiting_Missing(t *testing.T) {
	app := &ir.Application{
		Auth: &ir.Auth{
			Methods: []*ir.AuthMethod{{Type: "jwt"}},
		},
	}

	findings := checkRateLimiting(app)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Category != "rate-limiting" {
		t.Errorf("expected rate-limiting category, got %s", findings[0].Category)
	}
}

func TestCheckRateLimiting_Present(t *testing.T) {
	app := &ir.Application{
		Auth: &ir.Auth{
			Rules: []*ir.Action{
				{Type: "configure", Text: "rate limit 100 requests per minute per user"},
			},
		},
	}

	findings := checkRateLimiting(app)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings, got %d", len(findings))
	}
}

func TestRenderSecurityReport(t *testing.T) {
	app := &ir.Application{Name: "TestApp"}
	findings := []Finding{
		{Severity: "critical", Category: "auth", Message: "Missing auth", Target: "CreateTask"},
		{Severity: "warning", Category: "validation", Message: "No validation", Target: "Login"},
	}

	report := renderSecurityReport(app, findings)
	if !strings.Contains(report, "# Security Report") {
		t.Error("missing report header")
	}
	if !strings.Contains(report, "1 critical") {
		t.Error("missing critical count")
	}
	if !strings.Contains(report, "1 warnings") {
		t.Error("missing warning count")
	}
	if !strings.Contains(report, "Missing auth") {
		t.Error("missing finding row")
	}
}

// ── Lint Checks ──

func TestCheckEmptyPages(t *testing.T) {
	app := &ir.Application{
		Pages: []*ir.Page{
			{Name: "Home", Content: []*ir.Action{{Type: "display", Text: "show tasks"}}},
			{Name: "Empty"},
		},
	}

	warnings := checkEmptyPages(app)
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
	if warnings[0].Target != "Empty" {
		t.Errorf("expected Empty page flagged, got %s", warnings[0].Target)
	}
	if warnings[0].Category != "empty" {
		t.Errorf("expected empty category, got %s", warnings[0].Category)
	}
}

func TestCheckAPIsWithoutValidation(t *testing.T) {
	app := &ir.Application{
		APIs: []*ir.Endpoint{
			{Name: "GetTasks", Params: []*ir.Param{{Name: "filter"}}}, // GET — skipped
			{Name: "CreateTask", Params: []*ir.Param{{Name: "title"}}}, // POST, no validation — flagged
			{
				Name:       "UpdateTask",
				Params:     []*ir.Param{{Name: "title"}},
				Validation: []*ir.ValidationRule{{Field: "title", Rule: "not_empty"}},
			}, // has validation — OK
		},
	}

	warnings := checkAPIsWithoutValidation(app)
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
	if warnings[0].Target != "CreateTask" {
		t.Errorf("expected CreateTask, got %s", warnings[0].Target)
	}
}

func TestCheckEmptyWorkflows(t *testing.T) {
	app := &ir.Application{
		Workflows: []*ir.Workflow{
			{Trigger: "user signs up", Steps: []*ir.Action{{Type: "send", Text: "welcome email"}}},
			{Trigger: "task overdue"},
		},
	}

	warnings := checkEmptyWorkflows(app)
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
	if warnings[0].Target != "task overdue" {
		t.Errorf("expected 'task overdue' target, got %s", warnings[0].Target)
	}
}

func TestCheckUnusedModels(t *testing.T) {
	app := &ir.Application{
		Data: []*ir.DataModel{
			{Name: "User"},
			{Name: "AuditLog"}, // never referenced
		},
		APIs: []*ir.Endpoint{
			{
				Name: "GetUser",
				Steps: []*ir.Action{
					{Type: "query", Text: "find user by id"},
				},
			},
		},
	}

	warnings := checkUnusedModels(app)
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
	if warnings[0].Target != "AuditLog" {
		t.Errorf("expected AuditLog, got %s", warnings[0].Target)
	}
}

func TestRenderLintReport(t *testing.T) {
	app := &ir.Application{Name: "TestApp"}
	warnings := []Warning{
		{Category: "empty", Message: "Page Empty has no content", Target: "Empty"},
	}

	report := renderLintReport(app, warnings)
	if !strings.Contains(report, "# Lint Report") {
		t.Error("missing report header")
	}
	if !strings.Contains(report, "1 warnings") {
		t.Error("missing warning count")
	}
	if !strings.Contains(report, "Page Empty has no content") {
		t.Error("missing warning row")
	}
}

func TestRenderLintReport_Empty(t *testing.T) {
	app := &ir.Application{Name: "TestApp"}
	report := renderLintReport(app, nil)
	if !strings.Contains(report, "No lint issues found") {
		t.Error("expected clean lint report")
	}
}

// ── Build Summary ──

func TestRenderBuildSummary(t *testing.T) {
	app := &ir.Application{
		Name:     "TaskFlow",
		Platform: "web",
		Config: &ir.BuildConfig{
			Frontend: "React with TypeScript",
			Backend:  "Node with Express",
			Database: "PostgreSQL",
			Deploy:   "Docker",
		},
		Data:  []*ir.DataModel{{Name: "User"}, {Name: "Task"}},
		Pages: []*ir.Page{{Name: "Home"}},
		APIs:  []*ir.Endpoint{{Name: "GetTasks"}, {Name: "CreateTask"}},
	}

	result := &Result{
		TestFiles:            2,
		TestCount:            5,
		ComponentTestFiles:   2,
		ComponentTestCount:   4,
		EdgeTestFiles:        1,
		EdgeTestCount:        10,
		IntegrationTestCount: 3,
		Coverage: &CoverageReport{
			EndpointsTested: 2, EndpointsTotal: 2,
			PagesTested: 1, PagesTotal: 1,
			FieldsTested: 3, FieldsTotal: 3,
			Overall: 100,
		},
		SecurityFindings: []Finding{{Severity: "critical"}, {Severity: "warning"}},
		LintWarnings:     []Warning{{Category: "empty"}},
	}

	summary := renderBuildSummary(app, ".human/output", result)
	if !strings.Contains(summary, "# Build Report") {
		t.Error("missing report header")
	}
	if !strings.Contains(summary, "TaskFlow") {
		t.Error("missing app name")
	}
	if !strings.Contains(summary, "React with TypeScript") {
		t.Error("missing frontend config")
	}
	if !strings.Contains(summary, "| API Tests | 5 |") {
		t.Error("missing API test count")
	}
	if !strings.Contains(summary, "| Component Tests | 4 |") {
		t.Error("missing component test count")
	}
	if !strings.Contains(summary, "| Edge Case Tests | 10 |") {
		t.Error("missing edge case test count")
	}
	if !strings.Contains(summary, "| Integration Tests | 3 |") {
		t.Error("missing integration test count")
	}
	if !strings.Contains(summary, "**Total Tests**") {
		t.Error("missing total tests row")
	}
	if !strings.Contains(summary, "| Security Critical | 1 |") {
		t.Error("missing security critical count")
	}
	if !strings.Contains(summary, "| Lint Warnings | 1 |") {
		t.Error("missing lint warning count")
	}
	if !strings.Contains(summary, "## Test Coverage") {
		t.Error("missing test coverage section")
	}
}

// ── Validation Test Descriptions ──

func TestValidationTestDesc(t *testing.T) {
	tests := []struct {
		rule   *ir.ValidationRule
		expect string
	}{
		{&ir.ValidationRule{Field: "email", Rule: "not_empty"}, "should reject empty email"},
		{&ir.ValidationRule{Field: "email", Rule: "valid_email"}, "should reject invalid email"},
		{&ir.ValidationRule{Field: "password", Rule: "min_length", Value: "8"}, "should reject password shorter than 8 characters"},
		{&ir.ValidationRule{Field: "bio", Rule: "max_length", Value: "500"}, "should reject bio longer than 500 characters"},
		{&ir.ValidationRule{Field: "email", Rule: "unique"}, "should reject duplicate email"},
		{&ir.ValidationRule{Field: "due", Rule: "future_date"}, "should reject past due"},
		{&ir.ValidationRule{Field: "confirm", Rule: "matches"}, "should reject mismatched confirm"},
		{&ir.ValidationRule{Field: "name", Rule: "custom_rule"}, "should validate name"},
	}

	for _, tt := range tests {
		got := validationTestDesc(tt.rule)
		if got != tt.expect {
			t.Errorf("validationTestDesc(%s/%s) = %q, want %q", tt.rule.Field, tt.rule.Rule, got, tt.expect)
		}
	}
}

func TestInvalidValue(t *testing.T) {
	tests := []struct {
		rule   string
		expect string
	}{
		{"not_empty", "''"},
		{"valid_email", "'not-an-email'"},
		{"min_length", "'x'"},
		{"unique", "'existing@example.com'"},
		{"future_date", "'2020-01-01'"},
		{"unknown", "'invalid'"},
	}

	for _, tt := range tests {
		got := invalidValue(&ir.ValidationRule{Rule: tt.rule})
		if got != tt.expect {
			t.Errorf("invalidValue(%s) = %q, want %q", tt.rule, got, tt.expect)
		}
	}
}

// ── IsEnvVarName ──

func TestIsEnvVarName(t *testing.T) {
	tests := []struct {
		input  string
		expect bool
	}{
		{"SENDGRID_API_KEY", true},
		{"AWS_SECRET", true},
		{"JWT_SECRET", true},
		{"", false},
		{"lowercase", false},
		{"Mixed_Case", false},
		{"sk_live_123", false},
	}

	for _, tt := range tests {
		got := isEnvVarName(tt.input)
		if got != tt.expect {
			t.Errorf("isEnvVarName(%q) = %v, want %v", tt.input, got, tt.expect)
		}
	}
}

// ── PrintSummary ──

func TestPrintSummary(t *testing.T) {
	result := &Result{
		TestCount:            10,
		ComponentTestCount:   5,
		EdgeTestCount:        8,
		IntegrationTestCount: 3,
		Coverage:             &CoverageReport{Overall: 85},
		SecurityFindings:     []Finding{{Severity: "critical"}, {Severity: "warning"}},
		LintWarnings:         []Warning{{Category: "empty"}},
	}

	// Just verify it doesn't panic
	PrintSummary(result)
}

func TestPrintSummary_NoIssues(t *testing.T) {
	result := &Result{
		TestCount: 5,
	}

	// Should contain "no issues"
	PrintSummary(result)
}

// ── Full Integration ──

func TestRunIntegration(t *testing.T) {
	app := exampleApp(t)
	dir := t.TempDir()

	result, err := Run(app, dir)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result.TestFiles == 0 {
		t.Error("expected test files generated")
	}
	if result.TestCount == 0 {
		t.Error("expected tests generated")
	}

	// New result fields
	if result.ComponentTestFiles == 0 {
		t.Error("expected component test files generated")
	}
	if result.ComponentTestCount == 0 {
		t.Error("expected component tests generated")
	}
	if result.EdgeTestFiles == 0 {
		t.Error("expected edge test files generated")
	}
	if result.EdgeTestCount == 0 {
		t.Error("expected edge tests generated")
	}
	if result.IntegrationTestCount == 0 {
		t.Error("expected integration tests generated")
	}
	if result.Coverage == nil {
		t.Fatal("expected coverage report")
	}
	if result.Coverage.Overall <= 0 {
		t.Error("expected non-zero coverage")
	}

	// Should produce report files
	reportFiles := []string{
		filepath.Join(dir, "security-report.md"),
		filepath.Join(dir, "lint-report.md"),
		filepath.Join(dir, "build-report.md"),
	}
	for _, rf := range reportFiles {
		if _, err := os.Stat(rf); os.IsNotExist(err) {
			t.Errorf("missing report: %s", rf)
		}
	}

	// Security report should have content
	secReport, err := os.ReadFile(reportFiles[0])
	if err != nil {
		t.Fatalf("reading security report: %v", err)
	}
	if !strings.Contains(string(secReport), "# Security Report") {
		t.Error("security report missing header")
	}

	// Build report should have content
	buildReport, err := os.ReadFile(reportFiles[2])
	if err != nil {
		t.Fatalf("reading build report: %v", err)
	}
	if !strings.Contains(string(buildReport), "TaskFlow") {
		t.Error("build report missing app name")
	}
	if !strings.Contains(string(buildReport), "Test Coverage") {
		t.Error("build report missing Test Coverage section")
	}
	if !strings.Contains(string(buildReport), "Component Tests") {
		t.Error("build report missing Component Tests row")
	}
	if !strings.Contains(string(buildReport), "Edge Case Tests") {
		t.Error("build report missing Edge Case Tests row")
	}
	if !strings.Contains(string(buildReport), "Integration Tests") {
		t.Error("build report missing Integration Tests row")
	}
}

// ── New Security Checks ──

func TestCheckInputSanitization(t *testing.T) {
	app := &ir.Application{
		Data: []*ir.DataModel{
			{Name: "Task", Fields: []*ir.DataField{
				{Name: "title", Type: "text"},
				{Name: "count", Type: "number"},
			}},
		},
		APIs: []*ir.Endpoint{
			{Name: "CreateTask", Params: []*ir.Param{{Name: "title"}, {Name: "count"}}},
			{
				Name:       "UpdateTask",
				Params:     []*ir.Param{{Name: "title"}},
				Validation: []*ir.ValidationRule{{Field: "title", Rule: "not_empty"}},
			},
		},
	}

	findings := checkInputSanitization(app)

	// CreateTask has 'title' (text) without validation — should be flagged
	// CreateTask has 'count' which is number type, not text — should not be flagged via model
	// UpdateTask has 'title' with validation — should not be flagged
	found := false
	for _, f := range findings {
		if f.Target == "CreateTask" && strings.Contains(f.Message, "title") {
			found = true
		}
	}
	if !found {
		t.Error("expected sanitization warning for CreateTask title")
	}

	// UpdateTask should not be flagged (has validation)
	for _, f := range findings {
		if f.Target == "UpdateTask" {
			t.Error("UpdateTask should not be flagged — it has validation")
		}
	}
}

func TestIsTextField(t *testing.T) {
	app := &ir.Application{
		Data: []*ir.DataModel{
			{Name: "User", Fields: []*ir.DataField{
				{Name: "name", Type: "text"},
				{Name: "age", Type: "number"},
				{Name: "email", Type: "email"},
				{Name: "website", Type: "url"},
				{Name: "active", Type: "boolean"},
			}},
		},
	}

	tests := []struct {
		param  string
		expect bool
	}{
		{"name", true},     // text field in model
		{"age", false},     // number field in model
		{"email", true},    // email field in model
		{"website", true},  // url field in model
		{"active", false},  // boolean field in model
		{"title", true},    // common text name fallback
		{"description", true}, // common text name fallback
		{"bio", true},      // common text name fallback
		{"quantity", false}, // not in model, not a common text name
	}

	for _, tt := range tests {
		got := isTextField(app, tt.param)
		if got != tt.expect {
			t.Errorf("isTextField(%q) = %v, want %v", tt.param, got, tt.expect)
		}
	}
}

func TestCheckCORSConfig(t *testing.T) {
	// CORS mentioned but not enabled
	app := &ir.Application{
		Auth: &ir.Auth{
			Rules: []*ir.Action{
				{Type: "configure", Text: "CORS policy for frontend"},
			},
		},
		APIs: []*ir.Endpoint{{Name: "GetTasks"}},
	}

	findings := checkCORSConfig(app)
	if len(findings) == 0 {
		t.Fatal("expected CORS warning")
	}
	if findings[0].Category != "cors" {
		t.Errorf("expected cors category, got %s", findings[0].Category)
	}
}

func TestCheckCORSConfig_NoMention(t *testing.T) {
	// No CORS mentioned, APIs exist
	app := &ir.Application{
		APIs: []*ir.Endpoint{{Name: "GetTasks"}},
	}

	findings := checkCORSConfig(app)
	if len(findings) == 0 {
		t.Fatal("expected CORS info suggestion")
	}
	if findings[0].Severity != "info" {
		t.Errorf("expected info severity, got %s", findings[0].Severity)
	}
}

func TestCheckCORSConfig_Enabled(t *testing.T) {
	// CORS properly enabled
	app := &ir.Application{
		Auth: &ir.Auth{
			Rules: []*ir.Action{
				{Type: "configure", Text: "enable cors for trusted origins"},
			},
		},
	}

	findings := checkCORSConfig(app)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for properly configured CORS, got %d", len(findings))
	}
}

func TestCheckSecretPatterns(t *testing.T) {
	app := &ir.Application{
		Environments: []*ir.Environment{
			{
				Name: "production",
				Config: map[string]string{
					"url":     "https://example.com",
					"api_key": "sk_live_abc123def456ghi789",
				},
			},
		},
	}

	findings := checkSecretPatterns(app)
	if len(findings) == 0 {
		t.Fatal("expected secret pattern finding")
	}
	if findings[0].Severity != "critical" {
		t.Errorf("expected critical severity, got %s", findings[0].Severity)
	}
	if findings[0].Category != "secrets" {
		t.Errorf("expected secrets category, got %s", findings[0].Category)
	}
}

func TestCheckSecretPatterns_Clean(t *testing.T) {
	app := &ir.Application{
		Environments: []*ir.Environment{
			{
				Name: "production",
				Config: map[string]string{
					"url":      "https://example.com",
					"database": "postgres://localhost/app",
				},
			},
		},
	}

	findings := checkSecretPatterns(app)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for clean config, got %d", len(findings))
	}
}

func TestLooksLikeSecret(t *testing.T) {
	tests := []struct {
		input  string
		expect bool
	}{
		{"sk_live_abc123", true},
		{"sk_test_abc123", true},
		{"AKIAIOSFODNN7EXAMPLE", true},
		{"ghp_xxxxxxxxxxxxxxxxxxxx", true},
		{"xoxb-token-value", true},
		{"abcdefghijklmnopqrstuvwxyz123456", true},  // 32 chars alphanumeric
		{"short", false},
		{"https://example.com", false},              // contains special chars
		{"SENDGRID_API_KEY", false},                  // env var name
		{"normal value", false},
	}

	for _, tt := range tests {
		got := looksLikeSecret(tt.input)
		if got != tt.expect {
			t.Errorf("looksLikeSecret(%q) = %v, want %v", tt.input, got, tt.expect)
		}
	}
}

func TestIsAlphanumeric(t *testing.T) {
	tests := []struct {
		input  string
		expect bool
	}{
		{"abc123", true},
		{"ABC_DEF", true},
		{"test-value", true},
		{"hello world", false},
		{"has@symbol", false},
		{"has.dot", false},
		{"", false},
	}

	for _, tt := range tests {
		got := isAlphanumeric(tt.input)
		if got != tt.expect {
			t.Errorf("isAlphanumeric(%q) = %v, want %v", tt.input, got, tt.expect)
		}
	}
}
