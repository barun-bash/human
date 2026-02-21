package cicd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/ir"
	"github.com/barun-bash/human/internal/parser"
)

// ── Stack Detection ──

func TestStackDetection(t *testing.T) {
	tests := []struct {
		desc    string
		backend string
		isNode  bool
		isPy    bool
		isGo    bool
	}{
		{"empty defaults to node", "", true, false, false},
		{"Node with Express", "Node with Express", true, false, false},
		{"Python with FastAPI", "Python with FastAPI", false, true, false},
		{"Go with Gin", "Go with Gin", false, false, true},
		{"node lowercase", "node", true, false, false},
		{"Python with Django", "Python with Django", false, true, false},
		{"go lowercase", "go", false, false, true},
		{"golang", "golang backend", false, false, true},
	}
	for _, tt := range tests {
		app := &ir.Application{Config: &ir.BuildConfig{Backend: tt.backend}}
		if tt.backend == "" {
			app.Config = nil
		}
		if got := isNodeBackend(app); got != tt.isNode {
			t.Errorf("%s: isNodeBackend = %v, want %v", tt.desc, got, tt.isNode)
		}
		if got := isPythonBackend(app); got != tt.isPy {
			t.Errorf("%s: isPythonBackend = %v, want %v", tt.desc, got, tt.isPy)
		}
		if got := isGoBackend(app); got != tt.isGo {
			t.Errorf("%s: isGoBackend = %v, want %v", tt.desc, got, tt.isGo)
		}
	}
}

func TestIsPostgres(t *testing.T) {
	tests := []struct {
		desc string
		app  *ir.Application
		want bool
	}{
		{"from Config.Database", &ir.Application{Config: &ir.BuildConfig{Database: "PostgreSQL"}}, true},
		{"from Database.Engine", &ir.Application{Database: &ir.DatabaseConfig{Engine: "PostgreSQL"}}, true},
		{"case insensitive", &ir.Application{Config: &ir.BuildConfig{Database: "postgresql"}}, true},
		{"not postgres", &ir.Application{Config: &ir.BuildConfig{Database: "MySQL"}}, false},
		{"nil config", &ir.Application{}, false},
	}
	for _, tt := range tests {
		if got := isPostgres(tt.app); got != tt.want {
			t.Errorf("%s: isPostgres = %v, want %v", tt.desc, got, tt.want)
		}
	}
}

func TestIsMySQL(t *testing.T) {
	tests := []struct {
		desc string
		app  *ir.Application
		want bool
	}{
		{"from Config.Database", &ir.Application{Config: &ir.BuildConfig{Database: "MySQL"}}, true},
		{"from Database.Engine", &ir.Application{Database: &ir.DatabaseConfig{Engine: "MySQL"}}, true},
		{"case insensitive", &ir.Application{Config: &ir.BuildConfig{Database: "mysql"}}, true},
		{"not mysql", &ir.Application{Config: &ir.BuildConfig{Database: "PostgreSQL"}}, false},
		{"nil config", &ir.Application{}, false},
	}
	for _, tt := range tests {
		if got := isMySQL(tt.app); got != tt.want {
			t.Errorf("%s: isMySQL = %v, want %v", tt.desc, got, tt.want)
		}
	}
}

// ── Name Helpers ──

func TestAppNameLower(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"TaskFlow", "taskflow"},
		{"My App", "my-app"},
		{"", "app"},
	}
	for _, tt := range tests {
		app := &ir.Application{Name: tt.name}
		if got := appNameLower(app); got != tt.want {
			t.Errorf("appNameLower(%q): got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestDeployTarget(t *testing.T) {
	tests := []struct {
		deploy string
		want   string
	}{
		{"Docker", "docker"},
		{"Vercel", "vercel"},
		{"AWS", "aws"},
		{"GCP", "gcp"},
		{"", "docker"},
	}
	for _, tt := range tests {
		app := &ir.Application{}
		if tt.deploy != "" {
			app.Config = &ir.BuildConfig{Deploy: tt.deploy}
		}
		if got := deployTarget(app); got != tt.want {
			t.Errorf("deployTarget(%q): got %q, want %q", tt.deploy, got, tt.want)
		}
	}
}

// ── CI Workflow ──

func TestCIWorkflowNode(t *testing.T) {
	app := &ir.Application{
		Name:   "TestApp",
		Config: &ir.BuildConfig{Backend: "Node with Express"},
	}
	output := generateCIWorkflow(app)

	checks := []struct {
		desc    string
		pattern string
	}{
		{"workflow name", "name: testapp-ci"},
		{"on key quoted", "\"on\":"},
		{"push trigger", "push:"},
		{"pr trigger", "pull_request:"},
		{"setup-node", "uses: actions/setup-node@v4"},
		{"npm ci", "run: npm ci"},
		{"npm test", "run: npm test"},
		{"npm build", "run: npm run build"},
		{"npm lint", "run: npm run lint"},
	}
	for _, c := range checks {
		if !strings.Contains(output, c.pattern) {
			t.Errorf("CI Node: missing %s (%q)", c.desc, c.pattern)
		}
	}
}

func TestCIWorkflowPython(t *testing.T) {
	app := &ir.Application{
		Name:   "TestApp",
		Config: &ir.BuildConfig{Backend: "Python with FastAPI"},
	}
	output := generateCIWorkflow(app)

	checks := []struct {
		desc    string
		pattern string
	}{
		{"setup-python", "uses: actions/setup-python@v5"},
		{"pip install", "pip install -r requirements.txt"},
		{"flake8", "run: flake8"},
		{"pytest", "run: pytest"},
	}
	for _, c := range checks {
		if !strings.Contains(output, c.pattern) {
			t.Errorf("CI Python: missing %s (%q)", c.desc, c.pattern)
		}
	}

	// Should NOT have node steps
	if strings.Contains(output, "setup-node") {
		t.Error("CI Python: should not contain setup-node")
	}
}

func TestCIWorkflowGo(t *testing.T) {
	app := &ir.Application{
		Name:   "TestApp",
		Config: &ir.BuildConfig{Backend: "Go with Gin"},
	}
	output := generateCIWorkflow(app)

	checks := []struct {
		desc    string
		pattern string
	}{
		{"setup-go", "uses: actions/setup-go@v5"},
		{"go vet", "run: go vet ./..."},
		{"go test", "run: go test ./..."},
		{"go build", "run: go build ./..."},
	}
	for _, c := range checks {
		if !strings.Contains(output, c.pattern) {
			t.Errorf("CI Go: missing %s (%q)", c.desc, c.pattern)
		}
	}
}

func TestCIWorkflowPostgres(t *testing.T) {
	app := &ir.Application{
		Name:     "TestApp",
		Database: &ir.DatabaseConfig{Engine: "PostgreSQL"},
	}
	output := generateCIWorkflow(app)

	checks := []struct {
		desc    string
		pattern string
	}{
		{"postgres service", "image: postgres:16"},
		{"postgres user", "POSTGRES_USER: postgres"},
		{"postgres password", "POSTGRES_PASSWORD: postgres"},
		{"health check", "pg_isready"},
		{"port", "5432:5432"},
	}
	for _, c := range checks {
		if !strings.Contains(output, c.pattern) {
			t.Errorf("CI Postgres: missing %s (%q)", c.desc, c.pattern)
		}
	}
}

func TestCIWorkflowMySQL(t *testing.T) {
	app := &ir.Application{
		Name:     "TestApp",
		Database: &ir.DatabaseConfig{Engine: "MySQL"},
	}
	output := generateCIWorkflow(app)

	checks := []struct {
		desc    string
		pattern string
	}{
		{"mysql service", "image: mysql:8"},
		{"mysql root password", "MYSQL_ROOT_PASSWORD: root"},
		{"mysql database", "MYSQL_DATABASE: testapp_test"},
		{"health check", "mysqladmin ping"},
		{"port", "3306:3306"},
	}
	for _, c := range checks {
		if !strings.Contains(output, c.pattern) {
			t.Errorf("CI MySQL: missing %s (%q)", c.desc, c.pattern)
		}
	}

	// Should NOT have postgres
	if strings.Contains(output, "postgres:16") {
		t.Error("CI MySQL: should not contain postgres service")
	}
}

// ── Deploy Workflow ──

func TestDeployWorkflowDocker(t *testing.T) {
	app := &ir.Application{
		Name:   "TestApp",
		Config: &ir.BuildConfig{Deploy: "Docker"},
	}
	output := generateDeployWorkflow(app)

	checks := []struct {
		desc    string
		pattern string
	}{
		{"workflow name", "name: testapp-deploy"},
		{"push only", "push:"},
		{"docker login", "docker/login-action@v3"},
		{"docker username secret", "${{ secrets.DOCKER_USERNAME }}"},
		{"docker password secret", "${{ secrets.DOCKER_PASSWORD }}"},
		{"build and push", "docker/build-push-action@v5"},
		{"push true", "push: true"},
	}
	for _, c := range checks {
		if !strings.Contains(output, c.pattern) {
			t.Errorf("Deploy Docker: missing %s (%q)", c.desc, c.pattern)
		}
	}

	// Should NOT trigger on PRs
	if strings.Contains(output, "pull_request") {
		t.Error("Deploy: should not trigger on pull_request")
	}
}

func TestDeployWorkflowVercel(t *testing.T) {
	app := &ir.Application{
		Name:   "TestApp",
		Config: &ir.BuildConfig{Deploy: "Vercel"},
	}
	output := generateDeployWorkflow(app)

	checks := []struct {
		desc    string
		pattern string
	}{
		{"vercel install", "npm install -g vercel"},
		{"vercel token", "${{ secrets.VERCEL_TOKEN }}"},
		{"vercel prod", "vercel --prod"},
	}
	for _, c := range checks {
		if !strings.Contains(output, c.pattern) {
			t.Errorf("Deploy Vercel: missing %s (%q)", c.desc, c.pattern)
		}
	}
}

func TestDeployWorkflowAWS(t *testing.T) {
	app := &ir.Application{
		Name:   "TestApp",
		Config: &ir.BuildConfig{Deploy: "AWS"},
	}
	output := generateDeployWorkflow(app)

	checks := []struct {
		desc    string
		pattern string
	}{
		{"aws credentials action", "aws-actions/configure-aws-credentials@v4"},
		{"access key secret", "${{ secrets.AWS_ACCESS_KEY_ID }}"},
		{"secret key secret", "${{ secrets.AWS_SECRET_ACCESS_KEY }}"},
		{"ecr login", "aws-actions/amazon-ecr-login@v2"},
		{"ecs update", "aws ecs update-service"},
	}
	for _, c := range checks {
		if !strings.Contains(output, c.pattern) {
			t.Errorf("Deploy AWS: missing %s (%q)", c.desc, c.pattern)
		}
	}
}

func TestDeployWorkflowGCP(t *testing.T) {
	app := &ir.Application{
		Name:   "TestApp",
		Config: &ir.BuildConfig{Deploy: "GCP"},
	}
	output := generateDeployWorkflow(app)

	checks := []struct {
		desc    string
		pattern string
	}{
		{"gcp auth", "google-github-actions/auth@v2"},
		{"gcp sa key", "${{ secrets.GCP_SA_KEY }}"},
		{"gcr push", "gcloud builds submit"},
		{"cloud run deploy", "gcloud run deploy"},
	}
	for _, c := range checks {
		if !strings.Contains(output, c.pattern) {
			t.Errorf("Deploy GCP: missing %s (%q)", c.desc, c.pattern)
		}
	}
}

// ── Security Workflow ──

func TestSecurityWorkflowNode(t *testing.T) {
	app := &ir.Application{
		Name:   "TestApp",
		Config: &ir.BuildConfig{Backend: "Node with Express"},
	}
	output := generateSecurityWorkflow(app)

	checks := []struct {
		desc    string
		pattern string
	}{
		{"workflow name", "name: testapp-security"},
		{"cron schedule", "cron:"},
		{"pull_request trigger", "pull_request:"},
		{"npm audit", "npm audit --audit-level=high"},
	}
	for _, c := range checks {
		if !strings.Contains(output, c.pattern) {
			t.Errorf("Security Node: missing %s (%q)", c.desc, c.pattern)
		}
	}
}

func TestSecurityWorkflowPython(t *testing.T) {
	app := &ir.Application{
		Name:   "TestApp",
		Config: &ir.BuildConfig{Backend: "Python with FastAPI"},
	}
	output := generateSecurityWorkflow(app)

	if !strings.Contains(output, "pip-audit") {
		t.Error("Security Python: missing pip-audit")
	}
	if strings.Contains(output, "npm audit") {
		t.Error("Security Python: should not contain npm audit")
	}
}

func TestSecurityWorkflowGo(t *testing.T) {
	app := &ir.Application{
		Name:   "TestApp",
		Config: &ir.BuildConfig{Backend: "Go with Gin"},
	}
	output := generateSecurityWorkflow(app)

	if !strings.Contains(output, "govulncheck") {
		t.Error("Security Go: missing govulncheck")
	}
	if !strings.Contains(output, "go vet") {
		t.Error("Security Go: missing go vet")
	}
}

// ── Templates ──

func TestPRTemplate(t *testing.T) {
	app := &ir.Application{Name: "TestApp"}
	output := generatePRTemplate(app)

	checks := []struct {
		desc    string
		pattern string
	}{
		{"description section", "## Description"},
		{"type of change", "## Type of change"},
		{"bug fix checkbox", "- [ ] Bug fix"},
		{"new feature checkbox", "- [ ] New feature"},
		{"checklist section", "## Checklist"},
		{"tests pass", "- [ ] Tests pass"},
		{"security audit", "- [ ] Security audit passes"},
		{"docs updated", "- [ ] Documentation updated"},
	}
	for _, c := range checks {
		if !strings.Contains(output, c.pattern) {
			t.Errorf("PR template: missing %s (%q)", c.desc, c.pattern)
		}
	}
}

func TestBugReportTemplate(t *testing.T) {
	app := &ir.Application{Name: "TestApp"}
	output := generateBugReport(app)

	checks := []struct {
		desc    string
		pattern string
	}{
		{"yaml front matter", "---"},
		{"name field", "name: Bug Report"},
		{"labels", "labels: [bug]"},
		{"describe section", "## Describe the bug"},
		{"reproduce section", "## To reproduce"},
		{"expected section", "## Expected behavior"},
		{"environment section", "## Environment"},
	}
	for _, c := range checks {
		if !strings.Contains(output, c.pattern) {
			t.Errorf("Bug report: missing %s (%q)", c.desc, c.pattern)
		}
	}
}

func TestFeatureRequestTemplate(t *testing.T) {
	app := &ir.Application{Name: "TestApp"}
	output := generateFeatureRequest(app)

	checks := []struct {
		desc    string
		pattern string
	}{
		{"yaml front matter", "---"},
		{"name field", "name: Feature Request"},
		{"labels", "labels: [enhancement]"},
		{"problem section", "## Problem description"},
		{"solution section", "## Proposed solution"},
		{"alternatives section", "## Alternatives considered"},
	}
	for _, c := range checks {
		if !strings.Contains(output, c.pattern) {
			t.Errorf("Feature request: missing %s (%q)", c.desc, c.pattern)
		}
	}
}

// ── Filesystem Test ──

func TestGenerateWritesFiles(t *testing.T) {
	app := &ir.Application{
		Name:     "TestApp",
		Platform: "web",
		Config:   &ir.BuildConfig{Backend: "Node with Express", Database: "PostgreSQL", Deploy: "Docker"},
		Database: &ir.DatabaseConfig{Engine: "PostgreSQL"},
	}

	dir := t.TempDir()
	g := Generator{}
	if err := g.Generate(app, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	expectedFiles := []string{
		".github/workflows/ci.yml",
		".github/workflows/deploy.yml",
		".github/workflows/security.yml",
		".github/PULL_REQUEST_TEMPLATE.md",
		".github/ISSUE_TEMPLATE/bug_report.md",
		".github/ISSUE_TEMPLATE/feature_request.md",
	}

	for _, f := range expectedFiles {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}
}

// ── Full Integration Test ──

func TestFullIntegration(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	humanFile := filepath.Join(root, "examples", "taskflow", "app.human")

	source, err := os.ReadFile(humanFile)
	if err != nil {
		t.Fatalf("failed to read app.human: %v", err)
	}

	prog, err := parser.Parse(string(source))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	app, err := ir.Build(prog)
	if err != nil {
		t.Fatalf("IR build error: %v", err)
	}

	dir := t.TempDir()
	g := Generator{}
	if err := g.Generate(app, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Verify all 6 files exist
	expectedFiles := []string{
		".github/workflows/ci.yml",
		".github/workflows/deploy.yml",
		".github/workflows/security.yml",
		".github/PULL_REQUEST_TEMPLATE.md",
		".github/ISSUE_TEMPLATE/bug_report.md",
		".github/ISSUE_TEMPLATE/feature_request.md",
	}
	for _, f := range expectedFiles {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}

	// CI workflow: TaskFlow uses Node backend
	ciContent, err := os.ReadFile(filepath.Join(dir, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("reading ci.yml: %v", err)
	}
	ci := string(ciContent)
	if !strings.Contains(ci, "setup-node") {
		t.Error("ci.yml: missing setup-node (TaskFlow uses Node backend)")
	}
	// TaskFlow uses PostgreSQL
	if !strings.Contains(ci, "postgres:16") {
		t.Error("ci.yml: missing postgres service (TaskFlow uses PostgreSQL)")
	}

	// Deploy workflow: TaskFlow uses Docker
	deployContent, err := os.ReadFile(filepath.Join(dir, ".github", "workflows", "deploy.yml"))
	if err != nil {
		t.Fatalf("reading deploy.yml: %v", err)
	}
	deploy := string(deployContent)
	if !strings.Contains(deploy, "docker") || !strings.Contains(deploy, "DOCKER_USERNAME") {
		t.Error("deploy.yml: missing Docker deploy steps (TaskFlow uses Docker)")
	}

	// Security workflow: Node → npm audit
	secContent, err := os.ReadFile(filepath.Join(dir, ".github", "workflows", "security.yml"))
	if err != nil {
		t.Fatalf("reading security.yml: %v", err)
	}
	sec := string(secContent)
	if !strings.Contains(sec, "npm audit") {
		t.Error("security.yml: missing npm audit (TaskFlow uses Node backend)")
	}

	t.Logf("Generated %d files to %s", len(expectedFiles), dir)
}
