package docker

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

func TestDbName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"TaskFlow", "taskflow"},
		{"My App", "my_app"},
		{"", "app"},
	}
	for _, tt := range tests {
		app := &ir.Application{Name: tt.name}
		got := DbName(app)
		if got != tt.want {
			t.Errorf("dbName(%q): got %q, want %q", tt.name, got, tt.want)
		}
	}
}

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
		got := AppNameLower(app)
		if got != tt.want {
			t.Errorf("appNameLower(%q): got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestCollectEnvVars(t *testing.T) {
	app := &ir.Application{
		Name: "TestApp",
		Integrations: []*ir.Integration{
			{
				Service:     "SendGrid",
				Credentials: map[string]string{"api key": "SENDGRID_API_KEY"},
			},
			{
				Service:     "AWS S3",
				Credentials: map[string]string{"api key": "AWS_ACCESS_KEY", "secret": "AWS_SECRET_KEY"},
				Config:      map[string]string{"region": "AWS_REGION", "bucket": "S3_BUCKET"},
			},
		},
	}

	vars := CollectEnvVars(app)

	// Should include core vars + credential vars + config vars
	names := make(map[string]bool)
	for _, v := range vars {
		names[v.Name] = true
	}

	for _, expected := range []string{"DATABASE_URL", "JWT_SECRET", "PORT", "VITE_API_URL", "SENDGRID_API_KEY", "AWS_ACCESS_KEY", "AWS_SECRET_KEY", "AWS_REGION", "S3_BUCKET"} {
		if !names[expected] {
			t.Errorf("missing env var %q", expected)
		}
	}

	// Verify sorted
	for i := 1; i < len(vars); i++ {
		if vars[i-1].Name > vars[i].Name {
			t.Errorf("not sorted: %q > %q", vars[i-1].Name, vars[i].Name)
		}
	}
}

func TestEnvCategory(t *testing.T) {
	tests := []struct {
		name    string
		comment string
		want    string
	}{
		{"DATABASE_URL", "", "Database"},
		{"JWT_SECRET", "", "Authentication"},
		{"PORT", "", "Server"},
		{"VITE_API_URL", "", "Frontend"},
		{"SENDGRID_API_KEY", "SendGrid", "Integration: SendGrid"},
	}
	for _, tt := range tests {
		got := envCategory(EnvVar{Name: tt.name, Comment: tt.comment})
		if got != tt.want {
			t.Errorf("envCategory(%q): got %q, want %q", tt.name, got, tt.want)
		}
	}
}

// ── Backend Dockerfile ──

func TestGenerateBackendDockerfile(t *testing.T) {
	app := &ir.Application{Name: "TestApp"}
	output := generateBackendDockerfile(app)

	checks := []struct {
		desc    string
		pattern string
	}{
		{"Node 20 alpine base", "FROM node:20-alpine"},
		{"multi-stage build", "AS builder"},
		{"npm ci", "RUN npm ci"},
		{"prisma generate", "RUN npx prisma generate"},
		{"copy prisma schema", "COPY prisma ./prisma"},
		{"npm build", "RUN npm run build"},
		{"production stage", "FROM node:20-alpine\n"},
		{"expose 3000", "EXPOSE 3000"},
		{"CMD", "CMD [\"node\", \"dist/server.js\"]"},
	}

	for _, c := range checks {
		if !strings.Contains(output, c.pattern) {
			t.Errorf("backend Dockerfile: missing %s (%q)", c.desc, c.pattern)
		}
	}
}

// ── Frontend Dockerfile ──

func TestGenerateFrontendDockerfile(t *testing.T) {
	app := &ir.Application{Name: "TestApp"}
	output := generateFrontendDockerfile(app)

	checks := []struct {
		desc    string
		pattern string
	}{
		{"Node 20 alpine build", "FROM node:20-alpine AS builder"},
		{"npm ci", "RUN npm ci"},
		{"ARG VITE_API_URL", "ARG VITE_API_URL"},
		{"ENV VITE_API_URL", "ENV VITE_API_URL=$VITE_API_URL"},
		{"npm build", "RUN npm run build"},
		{"nginx serve stage", "FROM nginx:alpine"},
		{"copy dist to nginx", "/usr/share/nginx/html"},
		{"SPA routing", "try_files"},
		{"expose 80", "EXPOSE 80"},
		{"nginx CMD", "daemon off"},
	}

	for _, c := range checks {
		if !strings.Contains(output, c.pattern) {
			t.Errorf("frontend Dockerfile: missing %s (%q)", c.desc, c.pattern)
		}
	}

	// ARG must come before RUN npm run build
	argIdx := strings.Index(output, "ARG VITE_API_URL")
	buildIdx := strings.Index(output, "RUN npm run build")
	if argIdx >= buildIdx {
		t.Error("frontend Dockerfile: ARG VITE_API_URL must appear before RUN npm run build")
	}
}

// ── Docker Compose ──

func TestGenerateDockerCompose(t *testing.T) {
	app := &ir.Application{
		Name: "TaskFlow",
		Integrations: []*ir.Integration{
			{
				Service:     "SendGrid",
				Credentials: map[string]string{"api key": "SENDGRID_API_KEY"},
			},
			{
				Service: "AWS S3",
				Config:  map[string]string{"region": "AWS_REGION", "bucket": "S3_BUCKET"},
			},
		},
	}

	output := generateDockerCompose(app)

	// Services
	if !strings.Contains(output, "services:") {
		t.Error("missing services key")
	}

	// Postgres
	if !strings.Contains(output, "image: postgres:16-alpine") {
		t.Error("missing postgres service")
	}
	if !strings.Contains(output, "POSTGRES_DB: taskflow") {
		t.Error("missing POSTGRES_DB from app name")
	}
	if !strings.Contains(output, "5432:5432") {
		t.Error("missing postgres port mapping")
	}
	if !strings.Contains(output, "taskflow-data:/var/lib/postgresql/data") {
		t.Error("missing postgres volume")
	}

	// Backend
	if !strings.Contains(output, "context: ./node") {
		t.Error("missing backend build context")
	}
	if !strings.Contains(output, "3000:3000") {
		t.Error("missing backend port mapping")
	}
	if !strings.Contains(output, "DATABASE_URL: postgresql://postgres:postgres@db:5432/taskflow?schema=public") {
		t.Error("missing DATABASE_URL in backend env")
	}
	if !strings.Contains(output, "JWT_SECRET: ${JWT_SECRET}") {
		t.Error("missing JWT_SECRET in backend env")
	}
	if !strings.Contains(output, "SENDGRID_API_KEY: ${SENDGRID_API_KEY}") {
		t.Error("missing integration credential env var in backend")
	}
	// Config env vars (from integ.Config)
	if !strings.Contains(output, "AWS_REGION: ${AWS_REGION}") {
		t.Error("missing config env var AWS_REGION in backend")
	}
	if !strings.Contains(output, "S3_BUCKET: ${S3_BUCKET}") {
		t.Error("missing config env var S3_BUCKET in backend")
	}

	// Frontend
	if !strings.Contains(output, "context: ./react") {
		t.Error("missing frontend build context")
	}
	if !strings.Contains(output, "80:80") {
		t.Error("missing frontend port mapping")
	}
	if !strings.Contains(output, "VITE_API_URL: http://localhost:3000") {
		t.Error("missing VITE_API_URL build arg")
	}

	// Depends on
	if !strings.Contains(output, "depends_on:") {
		t.Error("missing depends_on")
	}

	// Volumes
	if !strings.Contains(output, "volumes:") {
		t.Error("missing volumes section")
	}
	if !strings.Contains(output, "taskflow-data:") {
		t.Error("missing named volume definition")
	}
}

// ── .env.example ──

func TestGenerateEnvExample(t *testing.T) {
	app := &ir.Application{
		Name: "TaskFlow",
		Integrations: []*ir.Integration{
			{
				Service:     "SendGrid",
				Credentials: map[string]string{"api key": "SENDGRID_API_KEY"},
			},
			{
				Service:     "Slack",
				Credentials: map[string]string{"api key": "SLACK_WEBHOOK_URL"},
			},
		},
	}

	output := generateEnvExample(app)

	// Core vars
	if !strings.Contains(output, "DATABASE_URL=") {
		t.Error("missing DATABASE_URL")
	}
	if !strings.Contains(output, "JWT_SECRET=") {
		t.Error("missing JWT_SECRET")
	}
	if !strings.Contains(output, "PORT=3000") {
		t.Error("missing PORT")
	}
	if !strings.Contains(output, "VITE_API_URL=") {
		t.Error("missing VITE_API_URL")
	}

	// Integration vars
	if !strings.Contains(output, "SENDGRID_API_KEY=") {
		t.Error("missing SENDGRID_API_KEY")
	}
	if !strings.Contains(output, "SLACK_WEBHOOK_URL=") {
		t.Error("missing SLACK_WEBHOOK_URL")
	}

	// Section headers
	if !strings.Contains(output, "# Database") {
		t.Error("missing Database section header")
	}
	if !strings.Contains(output, "# Authentication") {
		t.Error("missing Authentication section header")
	}
	if !strings.Contains(output, "# Integration: SendGrid") {
		t.Error("missing SendGrid section header")
	}
}

// ── package.json ──

func TestGeneratePackageJSON(t *testing.T) {
	app := &ir.Application{Name: "TaskFlow"}
	output := generatePackageJSON(app)

	checks := []struct {
		desc    string
		pattern string
	}{
		{"name", `"name": "taskflow"`},
		{"version", `"version": "0.1.0"`},
		{"private", `"private": true`},
		{"dev script", `"dev": "docker compose up --build"`},
		{"build script", `"build": "docker compose build"`},
		{"start script", `"start": "docker compose up -d"`},
		{"stop script", `"stop": "docker compose down"`},
		{"db:migrate", `"db:migrate": "cd node && npx prisma migrate deploy"`},
		{"db:seed", `"db:seed": "cd node && npx prisma db seed"`},
	}

	for _, c := range checks {
		if !strings.Contains(output, c.pattern) {
			t.Errorf("package.json: missing %s (%q)", c.desc, c.pattern)
		}
	}
}

// ── Generate to Filesystem ──

func TestGenerateWritesFiles(t *testing.T) {
	app := &ir.Application{
		Name:     "TestApp",
		Platform: "web",
		Database: &ir.DatabaseConfig{Engine: "PostgreSQL"},
		Data: []*ir.DataModel{
			{Name: "User"},
		},
		APIs: []*ir.Endpoint{
			{Name: "SignUp"},
		},
		Integrations: []*ir.Integration{
			{Service: "SendGrid", Credentials: map[string]string{"api key": "SENDGRID_API_KEY"}},
		},
	}

	dir := t.TempDir()
	g := Generator{}
	if err := g.Generate(app, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	expectedFiles := []string{
		"node/Dockerfile",
		"react/Dockerfile",
		"docker-compose.yml",
		".env.example",
		"package.json",
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

	// Verify all 5 files exist
	expectedFiles := []string{
		"node/Dockerfile",
		"react/Dockerfile",
		"docker-compose.yml",
		".env.example",
		"package.json",
	}
	for _, f := range expectedFiles {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}

	// docker-compose.yml: 3 services (db, backend, frontend)
	composeContent, err := os.ReadFile(filepath.Join(dir, "docker-compose.yml"))
	if err != nil {
		t.Fatalf("reading docker-compose.yml: %v", err)
	}
	compose := string(composeContent)
	for _, svc := range []string{"db:", "backend:", "frontend:"} {
		if !strings.Contains(compose, svc) {
			t.Errorf("docker-compose.yml: missing service %s", svc)
		}
	}
	// Should reference taskflow database
	if !strings.Contains(compose, "taskflow") {
		t.Error("docker-compose.yml: missing taskflow database name")
	}

	// All 3 integration env vars in compose
	for _, envVar := range []string{"SENDGRID_API_KEY", "AWS_ACCESS_KEY", "AWS_SECRET_KEY", "SLACK_WEBHOOK_URL"} {
		if !strings.Contains(compose, envVar) {
			t.Errorf("docker-compose.yml: missing integration env var %s", envVar)
		}
	}

	// .env.example: all integration vars present
	envContent, err := os.ReadFile(filepath.Join(dir, ".env.example"))
	if err != nil {
		t.Fatalf("reading .env.example: %v", err)
	}
	env := string(envContent)
	for _, envVar := range []string{"DATABASE_URL", "JWT_SECRET", "PORT", "VITE_API_URL", "SENDGRID_API_KEY", "AWS_ACCESS_KEY", "AWS_SECRET_KEY", "SLACK_WEBHOOK_URL"} {
		if !strings.Contains(env, envVar) {
			t.Errorf(".env.example: missing %s", envVar)
		}
	}

	// package.json: has taskflow name and key scripts
	pkgContent, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatalf("reading package.json: %v", err)
	}
	pkg := string(pkgContent)
	if !strings.Contains(pkg, `"taskflow"`) {
		t.Error("package.json: missing taskflow name")
	}
	if !strings.Contains(pkg, `"dev"`) {
		t.Error("package.json: missing dev script")
	}
	if !strings.Contains(pkg, `"db:migrate"`) {
		t.Error("package.json: missing db:migrate script")
	}

	t.Logf("Generated %d files to %s", len(expectedFiles), dir)
}
