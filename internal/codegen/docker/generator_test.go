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

func TestBackendDir(t *testing.T) {
	tests := []struct {
		backend string
		want    string
	}{
		{"Node with Express", "node"},
		{"Python with FastAPI", "python"},
		{"Go with Gin", "go"},
		{"Go", "go"},
		{"", "node"},
	}
	for _, tt := range tests {
		app := &ir.Application{Config: &ir.BuildConfig{Backend: tt.backend}}
		got := BackendDir(app)
		if got != tt.want {
			t.Errorf("BackendDir(%q): got %q, want %q", tt.backend, got, tt.want)
		}
	}
}

func TestBackendPort(t *testing.T) {
	tests := []struct {
		backend string
		port    int
		want    string
	}{
		{"Node with Express", 0, "3001"},      // default for Node
		{"Node with Express", 3000, "3000"},   // configured port
		{"Python with FastAPI", 0, "8000"},    // default for Python
		{"Go with Gin", 0, "8080"},            // default for Go
		{"", 0, "3001"},                       // default when no backend specified
		{"", 4000, "4000"},                    // configured port overrides default
	}
	for _, tt := range tests {
		config := &ir.BuildConfig{Backend: tt.backend}
		if tt.port > 0 {
			config.Ports = ir.PortConfig{Backend: tt.port}
		}
		app := &ir.Application{Config: config}
		got := BackendPort(app)
		if got != tt.want {
			t.Errorf("BackendPort(%q, port=%d): got %q, want %q", tt.backend, tt.port, got, tt.want)
		}
	}
}

func TestFrontendDir(t *testing.T) {
	tests := []struct {
		frontend string
		want     string
	}{
		{"React with TypeScript", "react"},
		{"Vue with TypeScript", "vue"},
		{"Angular with TypeScript", "angular"},
		{"Svelte with TypeScript", "svelte"},
		{"", "react"},
	}
	for _, tt := range tests {
		app := &ir.Application{Config: &ir.BuildConfig{Frontend: tt.frontend}}
		got := FrontendDir(app)
		if got != tt.want {
			t.Errorf("FrontendDir(%q): got %q, want %q", tt.frontend, got, tt.want)
		}
	}
}

func TestFrontendAPIEnvName(t *testing.T) {
	tests := []struct {
		frontend string
		want     string
	}{
		{"React with TypeScript", "VITE_API_URL"},
		{"Vue with TypeScript", "VITE_API_URL"},
		{"Angular with TypeScript", "NG_APP_API_URL"},
		{"Svelte with TypeScript", "VITE_API_URL"},
	}
	for _, tt := range tests {
		app := &ir.Application{Config: &ir.BuildConfig{Frontend: tt.frontend}}
		got := FrontendAPIEnvName(app)
		if got != tt.want {
			t.Errorf("FrontendAPIEnvName(%q): got %q, want %q", tt.frontend, got, tt.want)
		}
	}
}

func TestCollectEnvVars(t *testing.T) {
	app := &ir.Application{
		Name:   "TestApp",
		Config: &ir.BuildConfig{Frontend: "React with TypeScript"},
		Integrations: []*ir.Integration{
			{
				Service:     "SendGrid",
				Type:        "email",
				Credentials: map[string]string{"api key": "SENDGRID_API_KEY"},
			},
			{
				Service:     "AWS S3",
				Type:        "storage",
				Credentials: map[string]string{"api key": "AWS_ACCESS_KEY", "secret": "AWS_SECRET_KEY"},
				Config:      map[string]string{"region": "us-east-1", "bucket": "user-uploads"},
			},
		},
	}

	vars := CollectEnvVars(app)

	// Should include core vars + VITE_API_URL (has frontend) + credential vars + storage config env vars
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

	// AWS_REGION should have the config value as its example
	for _, v := range vars {
		if v.Name == "AWS_REGION" && v.Example != "us-east-1" {
			t.Errorf("AWS_REGION example: got %q, want %q", v.Example, "us-east-1")
		}
	}
}

func TestCollectEnvVarsPython(t *testing.T) {
	app := &ir.Application{
		Name:   "Blog",
		Config: &ir.BuildConfig{Frontend: "Vue with TypeScript", Backend: "Python with FastAPI"},
	}

	vars := CollectEnvVars(app)
	byName := make(map[string]EnvVar)
	for _, v := range vars {
		byName[v.Name] = v
	}

	// PORT should be 8000 for Python
	if v, ok := byName["PORT"]; !ok || v.Example != "8000" {
		t.Errorf("PORT example: got %q, want %q", byName["PORT"].Example, "8000")
	}

	// VITE_API_URL should reference port 8000
	if v, ok := byName["VITE_API_URL"]; !ok || v.Example != "http://localhost:8000" {
		t.Errorf("VITE_API_URL example: got %q, want %q", byName["VITE_API_URL"].Example, "http://localhost:8000")
	}

	// DATABASE_URL should use sslmode=disable for Python, not schema=public
	if v, ok := byName["DATABASE_URL"]; ok {
		if !strings.Contains(v.Example, "sslmode=disable") {
			t.Errorf("Python DATABASE_URL should use sslmode=disable, got %q", v.Example)
		}
		if strings.Contains(v.Example, "schema=public") {
			t.Errorf("Python DATABASE_URL should not use schema=public, got %q", v.Example)
		}
	}
}

func TestCollectEnvVarsAngular(t *testing.T) {
	app := &ir.Application{
		Name:   "Shop",
		Config: &ir.BuildConfig{Frontend: "Angular with TypeScript", Backend: "Go with Gin"},
	}

	vars := CollectEnvVars(app)
	names := make(map[string]bool)
	for _, v := range vars {
		names[v.Name] = true
	}

	// Angular should use NG_APP_API_URL, not VITE_API_URL
	if names["VITE_API_URL"] {
		t.Error("Angular app should not have VITE_API_URL")
	}
	if !names["NG_APP_API_URL"] {
		t.Error("Angular app should have NG_APP_API_URL")
	}

	// PORT should be 8080 for Go
	byName := make(map[string]EnvVar)
	for _, v := range vars {
		byName[v.Name] = v
	}
	if v, ok := byName["PORT"]; !ok || v.Example != "8080" {
		t.Errorf("PORT example: got %q, want %q", byName["PORT"].Example, "8080")
	}

	// DATABASE_URL should use sslmode=disable for Go, not schema=public
	if v, ok := byName["DATABASE_URL"]; ok {
		if !strings.Contains(v.Example, "sslmode=disable") {
			t.Errorf("Go DATABASE_URL should use sslmode=disable, got %q", v.Example)
		}
		if strings.Contains(v.Example, "schema=public") {
			t.Errorf("Go DATABASE_URL should not use schema=public, got %q", v.Example)
		}
	}
}

func TestCollectEnvVarsAPIOnly(t *testing.T) {
	app := &ir.Application{
		Name:   "PayGate",
		Config: &ir.BuildConfig{Backend: "Node with Express", Database: "PostgreSQL"},
	}

	vars := CollectEnvVars(app)
	names := make(map[string]bool)
	for _, v := range vars {
		names[v.Name] = true
	}

	// API-only: should NOT have VITE_API_URL
	if names["VITE_API_URL"] {
		t.Error("API-only app should not have VITE_API_URL")
	}

	// Should still have backend vars
	for _, expected := range []string{"DATABASE_URL", "JWT_SECRET", "PORT"} {
		if !names[expected] {
			t.Errorf("missing env var %q", expected)
		}
	}
}

func TestCollectEnvVarsCloudinaryNoAWS(t *testing.T) {
	app := &ir.Application{
		Name: "TestApp",
		Integrations: []*ir.Integration{
			{
				Service:     "Cloudinary",
				Type:        "storage",
				Credentials: map[string]string{"api key": "CLOUDINARY_API_KEY", "secret": "CLOUDINARY_SECRET"},
			},
		},
	}

	vars := CollectEnvVars(app)
	names := make(map[string]bool)
	for _, v := range vars {
		names[v.Name] = true
	}

	// Cloudinary should NOT get AWS-specific env vars
	for _, unexpected := range []string{"AWS_REGION", "S3_BUCKET"} {
		if names[unexpected] {
			t.Errorf("Cloudinary should not have %q env var", unexpected)
		}
	}

	// Should have Cloudinary credentials
	for _, expected := range []string{"CLOUDINARY_API_KEY", "CLOUDINARY_SECRET"} {
		if !names[expected] {
			t.Errorf("missing env var %q", expected)
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

// ── Backend Dockerfiles ──

func TestGenerateBackendDockerfileNode(t *testing.T) {
	app := &ir.Application{Name: "TestApp", Config: &ir.BuildConfig{Backend: "Node with Express"}}
	output := generateBackendDockerfile(app)

	checks := []struct {
		desc    string
		pattern string
	}{
		{"Node 20 alpine base", "FROM node:20-alpine"},
		{"multi-stage build", "AS builder"},
		{"npm install", "RUN npm install"},
		{"prisma generate", "RUN npx prisma generate"},
		{"copy prisma schema", "COPY prisma ./prisma"},
		{"npm build", "RUN npm run build"},
		{"production stage", "FROM node:20-alpine\n"},
		{"expose 3001", "EXPOSE 3001"},
		{"CMD", "CMD [\"./start.sh\"]"},
	}

	for _, c := range checks {
		if !strings.Contains(output, c.pattern) {
			t.Errorf("backend Dockerfile: missing %s (%q)", c.desc, c.pattern)
		}
	}
}

func TestGenerateBackendDockerfilePython(t *testing.T) {
	app := &ir.Application{Name: "TestApp", Config: &ir.BuildConfig{Backend: "Python with FastAPI"}}
	output := generateBackendDockerfile(app)

	checks := []struct {
		desc    string
		pattern string
	}{
		{"Python base image", "FROM python:3.12-slim"},
		{"multi-stage build", "AS builder"},
		{"requirements.txt", "COPY requirements.txt"},
		{"pip install", "pip install"},
		{"expose 8000", "EXPOSE 8000"},
		{"uvicorn CMD", "uvicorn"},
	}

	for _, c := range checks {
		if !strings.Contains(output, c.pattern) {
			t.Errorf("Python backend Dockerfile: missing %s (%q)", c.desc, c.pattern)
		}
	}

	// Should NOT contain Node-specific things
	if strings.Contains(output, "npm") {
		t.Error("Python Dockerfile should not contain npm")
	}
	if strings.Contains(output, "prisma") {
		t.Error("Python Dockerfile should not contain prisma")
	}
}

func TestGenerateBackendDockerfileGo(t *testing.T) {
	app := &ir.Application{Name: "TestApp", Config: &ir.BuildConfig{Backend: "Go with Gin"}}
	output := generateBackendDockerfile(app)

	checks := []struct {
		desc    string
		pattern string
	}{
		{"Go base image", "FROM golang:1.23-alpine"},
		{"multi-stage build", "AS builder"},
		{"git installed", "apk add --no-cache git"},
		{"copy go.mod first", "COPY go.mod go.sum*"},
		{"go mod tidy", "go mod tidy"},
		{"go build", "go build"},
		{"CGO disabled", "CGO_ENABLED=0"},
		{"alpine production", "FROM alpine:"},
		{"expose 8080", "EXPOSE 8080"},
		{"binary name", "testapp"},
	}

	for _, c := range checks {
		if !strings.Contains(output, c.pattern) {
			t.Errorf("Go backend Dockerfile: missing %s (%q)", c.desc, c.pattern)
		}
	}

	// Should NOT contain Node-specific things
	if strings.Contains(output, "npm") {
		t.Error("Go Dockerfile should not contain npm")
	}
}

// ── Frontend Dockerfiles ──

func TestGenerateFrontendDockerfileVite(t *testing.T) {
	app := &ir.Application{Name: "TestApp", Config: &ir.BuildConfig{Frontend: "React with TypeScript"}}
	output := generateFrontendDockerfile(app)

	checks := []struct {
		desc    string
		pattern string
	}{
		{"Node 20 alpine build", "FROM node:20-alpine AS builder"},
		{"npm install", "RUN npm install"},
		{"ARG VITE_API_URL", "ARG VITE_API_URL"},
		{"ENV VITE_API_URL", "ENV VITE_API_URL=$VITE_API_URL"},
		{"npm build", "RUN npm run build"},
		{"nginx serve stage", "FROM nginx:alpine"},
		{"copy dist to nginx", "/usr/share/nginx/html"},
		{"SPA routing", "try_files"},
		{"expose 80", "EXPOSE 80"},
		{"nginx CMD", "daemon off"},
		{"api proxy", "proxy_pass http://backend:"},
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

func TestGenerateFrontendDockerfileVue(t *testing.T) {
	app := &ir.Application{Name: "TestApp", Config: &ir.BuildConfig{Frontend: "Vue with TypeScript"}}
	output := generateFrontendDockerfile(app)

	// Vue also uses Vite, so should have VITE_API_URL
	if !strings.Contains(output, "ARG VITE_API_URL") {
		t.Error("Vue frontend Dockerfile should have ARG VITE_API_URL")
	}
	if !strings.Contains(output, "COPY --from=builder /app/dist") {
		t.Error("Vue frontend Dockerfile should copy dist/")
	}
}

func TestGenerateFrontendDockerfileAngular(t *testing.T) {
	app := &ir.Application{Name: "TestApp", Config: &ir.BuildConfig{Frontend: "Angular with TypeScript"}}
	output := generateFrontendDockerfile(app)

	checks := []struct {
		desc    string
		pattern string
	}{
		{"NG_APP_API_URL ARG", "ARG NG_APP_API_URL"},
		{"NG_APP_API_URL ENV", "ENV NG_APP_API_URL=$NG_APP_API_URL"},
		{"Angular dist path", "dist/app/browser"},
		{"nginx", "FROM nginx:alpine"},
		{"SPA routing", "try_files"},
	}

	for _, c := range checks {
		if !strings.Contains(output, c.pattern) {
			t.Errorf("Angular frontend Dockerfile: missing %s (%q)", c.desc, c.pattern)
		}
	}

	// Should NOT have VITE references
	if strings.Contains(output, "VITE_API_URL") {
		t.Error("Angular Dockerfile should not reference VITE_API_URL")
	}
}

// ── Docker Compose ──

func TestGenerateDockerCompose(t *testing.T) {
	app := &ir.Application{
		Name:   "TaskFlow",
		Config: &ir.BuildConfig{Frontend: "React with TypeScript", Backend: "Node with Express"},
		Integrations: []*ir.Integration{
			{
				Service:     "SendGrid",
				Type:        "email",
				Credentials: map[string]string{"api key": "SENDGRID_API_KEY"},
			},
			{
				Service:     "AWS S3",
				Type:        "storage",
				Credentials: map[string]string{"api key": "AWS_ACCESS_KEY", "secret": "AWS_SECRET_KEY"},
				Config:      map[string]string{"region": "us-east-1", "bucket": "user-uploads"},
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
	if !strings.Contains(output, "3001:3001") {
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
	if !strings.Contains(output, "VITE_API_URL: http://localhost:3001") {
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

func TestGenerateDockerComposePython(t *testing.T) {
	app := &ir.Application{
		Name:   "Blog",
		Config: &ir.BuildConfig{Frontend: "Vue with TypeScript", Backend: "Python with FastAPI"},
	}

	output := generateDockerCompose(app)

	// Backend should reference python directory and port 8000
	if !strings.Contains(output, "context: ./python") {
		t.Error("Python backend should have context: ./python")
	}
	if !strings.Contains(output, "8000:8000") {
		t.Error("Python backend should map port 8000")
	}
	if !strings.Contains(output, `PORT: "8000"`) {
		t.Error("Python backend should set PORT to 8000")
	}

	// Frontend should reference vue directory
	if !strings.Contains(output, "context: ./vue") {
		t.Error("Vue frontend should have context: ./vue")
	}
	if !strings.Contains(output, "VITE_API_URL: http://localhost:8000") {
		t.Error("VITE_API_URL should reference port 8000")
	}
}

func TestGenerateDockerComposeGo(t *testing.T) {
	app := &ir.Application{
		Name:   "Shop",
		Config: &ir.BuildConfig{Frontend: "Angular with TypeScript", Backend: "Go with Gin"},
	}

	output := generateDockerCompose(app)

	// Backend should reference go directory and port 8080
	if !strings.Contains(output, "context: ./go") {
		t.Error("Go backend should have context: ./go")
	}
	if !strings.Contains(output, "8080:8080") {
		t.Error("Go backend should map port 8080")
	}
	if !strings.Contains(output, `PORT: "8080"`) {
		t.Error("Go backend should set PORT to 8080")
	}
	if !strings.Contains(output, "sslmode=disable") {
		t.Error("Go backend DATABASE_URL should use sslmode=disable, not schema=public")
	}

	// Frontend should reference angular directory
	if !strings.Contains(output, "context: ./angular") {
		t.Error("Angular frontend should have context: ./angular")
	}
	if !strings.Contains(output, "NG_APP_API_URL: http://localhost:8080") {
		t.Error("NG_APP_API_URL should reference port 8080")
	}
	// Should NOT have VITE references
	if strings.Contains(output, "VITE_API_URL") {
		t.Error("Angular compose should not have VITE_API_URL")
	}
}

func TestGenerateDockerComposeAPIOnly(t *testing.T) {
	app := &ir.Application{
		Name:   "PayGate",
		Config: &ir.BuildConfig{Backend: "Node with Express", Database: "PostgreSQL"},
	}

	output := generateDockerCompose(app)

	// Should have db + backend
	if !strings.Contains(output, "  db:") {
		t.Error("missing db service")
	}
	if !strings.Contains(output, "  backend:") {
		t.Error("missing backend service")
	}

	// Should NOT have frontend service
	if strings.Contains(output, "  frontend:") {
		t.Error("API-only app should not have frontend service")
	}
	if strings.Contains(output, "context: ./react") {
		t.Error("API-only app should not have react build context")
	}
	if strings.Contains(output, "VITE_API_URL") {
		t.Error("API-only app should not have VITE_API_URL")
	}
	if strings.Contains(output, "80:80") {
		t.Error("API-only app should not have port 80 mapping")
	}
}

// ── .env.example ──

func TestGenerateEnvExample(t *testing.T) {
	app := &ir.Application{
		Name:   "TaskFlow",
		Config: &ir.BuildConfig{Frontend: "React with TypeScript"},
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
	if !strings.Contains(output, "PORT=3001") {
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

func TestGenerateEnvExampleAPIOnly(t *testing.T) {
	app := &ir.Application{
		Name:   "PayGate",
		Config: &ir.BuildConfig{Backend: "Node with Express"},
	}

	output := generateEnvExample(app)

	// Should have backend vars
	if !strings.Contains(output, "DATABASE_URL=") {
		t.Error("missing DATABASE_URL")
	}
	if !strings.Contains(output, "JWT_SECRET=") {
		t.Error("missing JWT_SECRET")
	}

	// Should NOT have VITE_API_URL
	if strings.Contains(output, "VITE_API_URL") {
		t.Error("API-only app should not have VITE_API_URL")
	}
	// Should NOT have Frontend section header
	if strings.Contains(output, "# Frontend") {
		t.Error("API-only app should not have Frontend section")
	}
}

// ── .env ──

func TestGenerateEnvFile(t *testing.T) {
	app := &ir.Application{
		Name:   "TaskFlow",
		Config: &ir.BuildConfig{Frontend: "React with TypeScript"},
	}

	output := generateEnvFile(app)

	// Should have local dev defaults comment
	if !strings.Contains(output, "local development defaults") {
		t.Error("missing local development header")
	}

	// Core vars should have values (not empty)
	if !strings.Contains(output, "DATABASE_URL=postgresql://") {
		t.Error("missing DATABASE_URL with value")
	}
	if !strings.Contains(output, "JWT_SECRET=change-me-to-a-random-secret") {
		t.Error("missing JWT_SECRET with default value")
	}
	if !strings.Contains(output, "PORT=3001") {
		t.Error("missing PORT with value")
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

func TestGeneratePackageJSONPython(t *testing.T) {
	app := &ir.Application{
		Name:   "Blog",
		Config: &ir.BuildConfig{Backend: "Python with FastAPI"},
	}
	output := generatePackageJSON(app)

	if !strings.Contains(output, "cd python && alembic upgrade head") {
		t.Error("Python package.json should use alembic for db:migrate")
	}
	// Should NOT contain prisma or node references
	if strings.Contains(output, "prisma") {
		t.Error("Python package.json should not reference prisma")
	}
	if strings.Contains(output, "cd node") {
		t.Error("Python package.json should not reference cd node")
	}
}

func TestGeneratePackageJSONGo(t *testing.T) {
	app := &ir.Application{
		Name:   "Shop",
		Config: &ir.BuildConfig{Backend: "Go with Gin"},
	}
	output := generatePackageJSON(app)

	if !strings.Contains(output, "cd go && go run") {
		t.Error("Go package.json should use go run for db tasks")
	}
	// Should NOT contain prisma
	if strings.Contains(output, "prisma") {
		t.Error("Go package.json should not reference prisma")
	}
}

// ── .dockerignore ──

func TestGenerateBackendDockerignore(t *testing.T) {
	tests := []struct {
		backend  string
		contains string
	}{
		{"Node with Express", "node_modules"},
		{"Python with FastAPI", "__pycache__"},
		{"Go with Gin", ".env"},
	}
	for _, tt := range tests {
		app := &ir.Application{Config: &ir.BuildConfig{Backend: tt.backend}}
		output := generateBackendDockerignore(app)
		if !strings.Contains(output, tt.contains) {
			t.Errorf("%s .dockerignore should contain %q", tt.backend, tt.contains)
		}
	}
}

func TestGenerateFrontendDockerignore(t *testing.T) {
	app := &ir.Application{}
	output := generateFrontendDockerignore(app)
	if !strings.Contains(output, "node_modules") {
		t.Error("frontend .dockerignore should contain node_modules")
	}
}

// ── Generate to Filesystem ──

func TestGenerateWritesFiles(t *testing.T) {
	app := &ir.Application{
		Name:     "TestApp",
		Platform: "web",
		Config:   &ir.BuildConfig{Frontend: "React with TypeScript", Backend: "Node with Express"},
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
		"node/.dockerignore",
		"react/Dockerfile",
		"react/.dockerignore",
		"docker-compose.yml",
		".env.example",
		".env",
		"package.json",
	}

	for _, f := range expectedFiles {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}
}

func TestGenerateWritesFilesPython(t *testing.T) {
	app := &ir.Application{
		Name:     "Blog",
		Platform: "web",
		Config:   &ir.BuildConfig{Frontend: "Vue with TypeScript", Backend: "Python with FastAPI"},
	}

	dir := t.TempDir()
	g := Generator{}
	if err := g.Generate(app, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Should have python and vue directories, not node and react
	for _, f := range []string{"python/Dockerfile", "python/.dockerignore", "vue/Dockerfile", "vue/.dockerignore"} {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}

	// Should NOT have node/Dockerfile or react/Dockerfile
	for _, f := range []string{"node/Dockerfile", "react/Dockerfile"} {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); err == nil {
			t.Errorf("file %s should not exist for Python+Vue app", f)
		}
	}
}

func TestGenerateWritesFilesGo(t *testing.T) {
	app := &ir.Application{
		Name:     "Shop",
		Platform: "web",
		Config:   &ir.BuildConfig{Frontend: "Angular with TypeScript", Backend: "Go with Gin"},
	}

	dir := t.TempDir()
	g := Generator{}
	if err := g.Generate(app, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Should have go and angular directories
	for _, f := range []string{"go/Dockerfile", "go/.dockerignore", "angular/Dockerfile", "angular/.dockerignore"} {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}
}

func TestGenerateWritesFilesAPIOnly(t *testing.T) {
	app := &ir.Application{
		Name:     "PayGate",
		Platform: "api",
		Config:   &ir.BuildConfig{Backend: "Node with Express", Database: "PostgreSQL"},
		Database: &ir.DatabaseConfig{Engine: "PostgreSQL"},
		APIs: []*ir.Endpoint{
			{Name: "CreateCharge"},
		},
	}

	dir := t.TempDir()
	g := Generator{}
	if err := g.Generate(app, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Should have backend files but NOT frontend Dockerfile
	for _, f := range []string{"node/Dockerfile", "node/.dockerignore", "docker-compose.yml", ".env.example", ".env", "package.json"} {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}

	// react/Dockerfile should NOT exist
	reactDF := filepath.Join(dir, "react", "Dockerfile")
	if _, err := os.Stat(reactDF); err == nil {
		t.Error("API-only app should not have react/Dockerfile")
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

	// Verify all files exist (including .dockerignore)
	expectedFiles := []string{
		"node/Dockerfile",
		"node/.dockerignore",
		"react/Dockerfile",
		"react/.dockerignore",
		"docker-compose.yml",
		".env.example",
		".env",
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

// ── Full Integration Tests for Blog and Ecommerce ──

func TestFullIntegrationBlog(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	humanFile := filepath.Join(root, "examples", "blog", "app.human")

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

	// Blog uses Python + Vue
	expectedFiles := []string{
		"python/Dockerfile",
		"python/.dockerignore",
		"vue/Dockerfile",
		"vue/.dockerignore",
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

	// Verify compose references python and vue
	composeContent, err := os.ReadFile(filepath.Join(dir, "docker-compose.yml"))
	if err != nil {
		t.Fatalf("reading docker-compose.yml: %v", err)
	}
	compose := string(composeContent)
	if !strings.Contains(compose, "context: ./python") {
		t.Error("docker-compose.yml: should reference ./python")
	}
	if !strings.Contains(compose, "context: ./vue") {
		t.Error("docker-compose.yml: should reference ./vue")
	}
	if !strings.Contains(compose, "8000:8000") {
		t.Error("docker-compose.yml: should use port 8000 for Python")
	}

	// Verify Python Dockerfile
	pyDF, err := os.ReadFile(filepath.Join(dir, "python", "Dockerfile"))
	if err != nil {
		t.Fatalf("reading python/Dockerfile: %v", err)
	}
	if !strings.Contains(string(pyDF), "python:3.12-slim") {
		t.Error("python/Dockerfile: should use python:3.12-slim base image")
	}
	if !strings.Contains(string(pyDF), "uvicorn") {
		t.Error("python/Dockerfile: should use uvicorn")
	}
}

func TestFullIntegrationEcommerce(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	humanFile := filepath.Join(root, "examples", "ecommerce", "app.human")

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

	// Ecommerce uses Go + Angular
	expectedFiles := []string{
		"go/Dockerfile",
		"go/.dockerignore",
		"angular/Dockerfile",
		"angular/.dockerignore",
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

	// Verify compose references go and angular
	composeContent, err := os.ReadFile(filepath.Join(dir, "docker-compose.yml"))
	if err != nil {
		t.Fatalf("reading docker-compose.yml: %v", err)
	}
	compose := string(composeContent)
	if !strings.Contains(compose, "context: ./go") {
		t.Error("docker-compose.yml: should reference ./go")
	}
	if !strings.Contains(compose, "context: ./angular") {
		t.Error("docker-compose.yml: should reference ./angular")
	}
	if !strings.Contains(compose, "8080:8080") {
		t.Error("docker-compose.yml: should use port 8080 for Go")
	}
	if !strings.Contains(compose, "NG_APP_API_URL") {
		t.Error("docker-compose.yml: should use NG_APP_API_URL for Angular")
	}
	if strings.Contains(compose, "VITE_API_URL") {
		t.Error("docker-compose.yml: should not use VITE_API_URL for Angular")
	}

	// Verify Go Dockerfile
	goDF, err := os.ReadFile(filepath.Join(dir, "go", "Dockerfile"))
	if err != nil {
		t.Fatalf("reading go/Dockerfile: %v", err)
	}
	if !strings.Contains(string(goDF), "golang:1.23-alpine") {
		t.Error("go/Dockerfile: should use golang base image")
	}
	if !strings.Contains(string(goDF), "go build") {
		t.Error("go/Dockerfile: should use go build")
	}

	// Verify Angular Dockerfile
	angDF, err := os.ReadFile(filepath.Join(dir, "angular", "Dockerfile"))
	if err != nil {
		t.Fatalf("reading angular/Dockerfile: %v", err)
	}
	if !strings.Contains(string(angDF), "NG_APP_API_URL") {
		t.Error("angular/Dockerfile: should use NG_APP_API_URL")
	}
	if !strings.Contains(string(angDF), "dist/app/browser") {
		t.Error("angular/Dockerfile: should copy from dist/app/browser")
	}
}

// ── Priority 2 Stack Integration Tests ──

func TestFullIntegrationEvents(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	humanFile := filepath.Join(root, "examples", "events", "app.human")

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

	// Events uses Angular + Node
	for _, f := range []string{"node/Dockerfile", "angular/Dockerfile", "docker-compose.yml", ".env.example", ".env"} {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}

	compose, err := os.ReadFile(filepath.Join(dir, "docker-compose.yml"))
	if err != nil {
		t.Fatalf("reading docker-compose.yml: %v", err)
	}
	cs := string(compose)
	if !strings.Contains(cs, "context: ./node") {
		t.Error("docker-compose.yml: should reference ./node")
	}
	if !strings.Contains(cs, "context: ./angular") {
		t.Error("docker-compose.yml: should reference ./angular")
	}
	if !strings.Contains(cs, "3001:3001") {
		t.Error("docker-compose.yml: should use port 3001 for Node")
	}
	if !strings.Contains(cs, "NG_APP_API_URL") {
		t.Error("docker-compose.yml: should use NG_APP_API_URL for Angular")
	}
	// Node backend uses schema=public (Prisma)
	if !strings.Contains(cs, "schema=public") {
		t.Error("docker-compose.yml: Node backend should use schema=public")
	}

	// .env should also use schema=public for Node
	envContent, err := os.ReadFile(filepath.Join(dir, ".env"))
	if err != nil {
		t.Fatalf("reading .env: %v", err)
	}
	if !strings.Contains(string(envContent), "schema=public") {
		t.Error(".env: Node backend should use schema=public")
	}
}

func TestFullIntegrationInventory(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	humanFile := filepath.Join(root, "examples", "inventory", "app.human")

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

	// Inventory uses React + Go
	for _, f := range []string{"go/Dockerfile", "react/Dockerfile", "docker-compose.yml", ".env.example", ".env"} {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}

	compose, err := os.ReadFile(filepath.Join(dir, "docker-compose.yml"))
	if err != nil {
		t.Fatalf("reading docker-compose.yml: %v", err)
	}
	cs := string(compose)
	if !strings.Contains(cs, "context: ./go") {
		t.Error("docker-compose.yml: should reference ./go")
	}
	if !strings.Contains(cs, "context: ./react") {
		t.Error("docker-compose.yml: should reference ./react")
	}
	if !strings.Contains(cs, "8080:8080") {
		t.Error("docker-compose.yml: should use port 8080 for Go")
	}
	if !strings.Contains(cs, "VITE_API_URL") {
		t.Error("docker-compose.yml: should use VITE_API_URL for React")
	}
	// Go backend uses sslmode=disable
	if !strings.Contains(cs, "sslmode=disable") {
		t.Error("docker-compose.yml: Go backend should use sslmode=disable")
	}

	// Verify Go Dockerfile has git installed
	goDF, err := os.ReadFile(filepath.Join(dir, "go", "Dockerfile"))
	if err != nil {
		t.Fatalf("reading go/Dockerfile: %v", err)
	}
	if !strings.Contains(string(goDF), "apk add --no-cache git") {
		t.Error("go/Dockerfile: should install git")
	}

	// .env should also use sslmode=disable for Go
	envContent, err := os.ReadFile(filepath.Join(dir, ".env"))
	if err != nil {
		t.Fatalf("reading .env: %v", err)
	}
	if !strings.Contains(string(envContent), "sslmode=disable") {
		t.Error(".env: Go backend should use sslmode=disable")
	}
}

func TestFullIntegrationFigmaDemo(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	humanFile := filepath.Join(root, "examples", "figma-demo", "app.human")

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

	// Figma-demo uses React + Python
	for _, f := range []string{"python/Dockerfile", "react/Dockerfile", "docker-compose.yml", ".env.example", ".env"} {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}

	compose, err := os.ReadFile(filepath.Join(dir, "docker-compose.yml"))
	if err != nil {
		t.Fatalf("reading docker-compose.yml: %v", err)
	}
	cs := string(compose)
	if !strings.Contains(cs, "context: ./python") {
		t.Error("docker-compose.yml: should reference ./python")
	}
	if !strings.Contains(cs, "context: ./react") {
		t.Error("docker-compose.yml: should reference ./react")
	}
	if !strings.Contains(cs, "8000:8000") {
		t.Error("docker-compose.yml: should use port 8000 for Python")
	}
	if !strings.Contains(cs, "VITE_API_URL: http://localhost:8000") {
		t.Error("docker-compose.yml: VITE_API_URL should reference port 8000")
	}
	// Python backend uses sslmode=disable
	if !strings.Contains(cs, "sslmode=disable") {
		t.Error("docker-compose.yml: Python backend should use sslmode=disable")
	}

	// .env should also use sslmode=disable for Python
	envContent, err := os.ReadFile(filepath.Join(dir, ".env"))
	if err != nil {
		t.Fatalf("reading .env: %v", err)
	}
	if !strings.Contains(string(envContent), "sslmode=disable") {
		t.Error(".env: Python backend should use sslmode=disable")
	}
}

func TestFullIntegrationAPIOnly(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	humanFile := filepath.Join(root, "examples", "api-only", "app.human")

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

	// API-only uses Node with no frontend
	for _, f := range []string{"node/Dockerfile", "docker-compose.yml", ".env.example", ".env", "package.json"} {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}

	// Should NOT have any frontend directory
	for _, feDir := range []string{"react", "vue", "angular", "svelte"} {
		path := filepath.Join(dir, feDir, "Dockerfile")
		if _, err := os.Stat(path); err == nil {
			t.Errorf("API-only app should not have %s/Dockerfile", feDir)
		}
	}

	compose, err := os.ReadFile(filepath.Join(dir, "docker-compose.yml"))
	if err != nil {
		t.Fatalf("reading docker-compose.yml: %v", err)
	}
	cs := string(compose)
	if strings.Contains(cs, "frontend:") {
		t.Error("docker-compose.yml: API-only should not have frontend service")
	}
	if strings.Contains(cs, "VITE_API_URL") || strings.Contains(cs, "NG_APP_API_URL") {
		t.Error("docker-compose.yml: API-only should not have frontend env vars")
	}
	if !strings.Contains(cs, "context: ./node") {
		t.Error("docker-compose.yml: should reference ./node")
	}

	// .env should NOT have VITE_API_URL
	envContent, err := os.ReadFile(filepath.Join(dir, ".env"))
	if err != nil {
		t.Fatalf("reading .env: %v", err)
	}
	if strings.Contains(string(envContent), "VITE_API_URL") {
		t.Error(".env: API-only should not have VITE_API_URL")
	}
}

func TestFullIntegrationSaas(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	humanFile := filepath.Join(root, "examples", "saas", "app.human")

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

	// SaaS uses Svelte + Node
	for _, f := range []string{"node/Dockerfile", "svelte/Dockerfile", "docker-compose.yml", ".env.example", ".env"} {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}

	compose, err := os.ReadFile(filepath.Join(dir, "docker-compose.yml"))
	if err != nil {
		t.Fatalf("reading docker-compose.yml: %v", err)
	}
	cs := string(compose)
	if !strings.Contains(cs, "context: ./node") {
		t.Error("docker-compose.yml: should reference ./node")
	}
	if !strings.Contains(cs, "context: ./svelte") {
		t.Error("docker-compose.yml: should reference ./svelte")
	}
	if !strings.Contains(cs, "VITE_API_URL") {
		t.Error("docker-compose.yml: Svelte should use VITE_API_URL")
	}

	// Verify Svelte Dockerfile uses Vite (not Angular)
	svelteDF, err := os.ReadFile(filepath.Join(dir, "svelte", "Dockerfile"))
	if err != nil {
		t.Fatalf("reading svelte/Dockerfile: %v", err)
	}
	if !strings.Contains(string(svelteDF), "VITE_API_URL") {
		t.Error("svelte/Dockerfile: should use VITE_API_URL")
	}
	if strings.Contains(string(svelteDF), "NG_APP") {
		t.Error("svelte/Dockerfile: should not use NG_APP_API_URL")
	}
	// Svelte Vite build outputs to dist/, not dist/app/browser
	if !strings.Contains(string(svelteDF), "COPY --from=builder /app/dist") {
		t.Error("svelte/Dockerfile: should copy from /app/dist")
	}
}

// ── Port Configuration Tests ──

func TestFrontendPort(t *testing.T) {
	tests := []struct {
		port int
		want string
	}{
		{0, "80"},       // default (Nginx container port)
		{3000, "3000"},  // configured
		{8080, "8080"},  // custom
	}
	for _, tt := range tests {
		config := &ir.BuildConfig{}
		if tt.port > 0 {
			config.Ports = ir.PortConfig{Frontend: tt.port}
		}
		app := &ir.Application{Config: config}
		got := FrontendPort(app)
		if got != tt.want {
			t.Errorf("FrontendPort(%d): got %q, want %q", tt.port, got, tt.want)
		}
	}
}

func TestDatabasePort(t *testing.T) {
	tests := []struct {
		port int
		want string
	}{
		{0, "5432"},     // default
		{5432, "5432"},  // configured
		{3306, "3306"},  // custom
	}
	for _, tt := range tests {
		config := &ir.BuildConfig{}
		if tt.port > 0 {
			config.Ports = ir.PortConfig{Database: tt.port}
		}
		app := &ir.Application{Config: config}
		got := DatabasePort(app)
		if got != tt.want {
			t.Errorf("DatabasePort(%d): got %q, want %q", tt.port, got, tt.want)
		}
	}
}
