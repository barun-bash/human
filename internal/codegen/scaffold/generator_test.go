package scaffold

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

func testApp() *ir.Application {
	return &ir.Application{
		Name:     "TaskFlow",
		Platform: "web",
		Config: &ir.BuildConfig{
			Frontend: "React with TypeScript",
			Backend:  "Node with Express",
			Database: "PostgreSQL",
			Deploy:   "Docker",
		},
		Data: []*ir.DataModel{
			{
				Name: "User",
				Fields: []*ir.DataField{
					{Name: "email", Type: "email"},
					{Name: "password", Type: "text", Encrypted: true},
					{Name: "name", Type: "text"},
				},
			},
			{
				Name: "Task",
				Fields: []*ir.DataField{
					{Name: "title", Type: "text"},
					{Name: "status", Type: "enum"},
				},
			},
		},
		APIs: []*ir.Endpoint{
			{Name: "SignUp", Auth: false},
			{Name: "Login", Auth: false},
			{Name: "CreateTask", Auth: true},
			{Name: "GetTasks", Auth: true},
		},
		Database: &ir.DatabaseConfig{Engine: "PostgreSQL"},
		Integrations: []*ir.Integration{
			{Service: "SendGrid", Credentials: map[string]string{"api key": "SENDGRID_API_KEY"}},
		},
	}
}

func testAppVuePython() *ir.Application {
	return &ir.Application{
		Name:     "MyVueApp",
		Platform: "web",
		Config: &ir.BuildConfig{
			Frontend: "Vue",
			Backend:  "Python with FastAPI",
			Database: "PostgreSQL",
			Deploy:   "Docker",
		},
		Data: []*ir.DataModel{
			{Name: "User", Fields: []*ir.DataField{{Name: "email", Type: "email"}}},
		},
		APIs:     []*ir.Endpoint{{Name: "SignUp", Auth: false}},
		Database: &ir.DatabaseConfig{Engine: "PostgreSQL"},
	}
}

func testAppGoBackend() *ir.Application {
	return &ir.Application{
		Name:     "GoService",
		Platform: "api",
		Config: &ir.BuildConfig{
			Frontend: "None",
			Backend:  "Go with Gin",
			Database: "PostgreSQL",
			Deploy:   "Docker",
		},
		Data: []*ir.DataModel{
			{Name: "User", Fields: []*ir.DataField{{Name: "email", Type: "email"}}},
		},
		APIs:     []*ir.Endpoint{{Name: "CreateUser", Auth: true}},
		Database: &ir.DatabaseConfig{Engine: "PostgreSQL"},
	}
}

func testAppAngularNode() *ir.Application {
	return &ir.Application{
		Name:     "AngularApp",
		Platform: "web",
		Config: &ir.BuildConfig{
			Frontend: "Angular",
			Backend:  "Node with Express",
			Database: "PostgreSQL",
			Deploy:   "Docker",
		},
		Data: []*ir.DataModel{
			{Name: "User", Fields: []*ir.DataField{{Name: "email", Type: "email"}}},
		},
		APIs:     []*ir.Endpoint{{Name: "SignUp", Auth: false}},
		Database: &ir.DatabaseConfig{Engine: "PostgreSQL"},
	}
}

// ── Root package.json ──

func TestRootPackageJSON(t *testing.T) {
	app := testApp()
	output := generateRootPackageJSON(app)

	checks := []struct {
		desc    string
		pattern string
	}{
		{"name", `"name": "taskflow"`},
		{"workspaces node", `"node"`},
		{"workspaces react", `"react"`},
		{"dev script", `"dev": "concurrently`},
		{"start script", `"start": "concurrently`},
		{"build script", `"build":`},
		{"test script", `"test":`},
		{"db:migrate", `"db:migrate":`},
		{"db:seed", `"db:seed":`},
		{"db:studio", `"db:studio":`},
		{"docker:dev", `"docker:dev": "docker compose up --build"`},
		{"docker:start", `"docker:start": "docker compose up -d"`},
		{"docker:stop", `"docker:stop": "docker compose down"`},
		{"concurrently dep", `"concurrently": "^9.0.0"`},
	}

	for _, c := range checks {
		if !strings.Contains(output, c.pattern) {
			t.Errorf("root package.json: missing %s (%q)", c.desc, c.pattern)
		}
	}
}

func TestRootPackageJSONVuePython(t *testing.T) {
	app := testAppVuePython()
	output := generateRootPackageJSON(app)

	// Should have vue workspace, no node/react
	if !strings.Contains(output, `"vue"`) {
		t.Error("vue+python root package.json: missing vue workspace")
	}
	if strings.Contains(output, `"node"`) {
		t.Error("vue+python root package.json: should not have node workspace")
	}
	if strings.Contains(output, `"react"`) {
		t.Error("vue+python root package.json: should not have react workspace")
	}
	// Should not have Prisma scripts
	if strings.Contains(output, "prisma") {
		t.Error("vue+python root package.json: should not have Prisma scripts")
	}
	// Should not have concurrently (only 1 workspace)
	if strings.Contains(output, "concurrently") {
		t.Error("vue+python root package.json: should not have concurrently with single workspace")
	}
}

func TestRootPackageJSONGoBackend(t *testing.T) {
	app := testAppGoBackend()
	output := generateRootPackageJSON(app)

	// No workspaces at all (no JS backend, no JS frontend)
	if strings.Contains(output, `"workspaces"`) {
		t.Error("go backend root package.json: should not have workspaces")
	}
	if strings.Contains(output, "prisma") {
		t.Error("go backend root package.json: should not have Prisma scripts")
	}
	if strings.Contains(output, "concurrently") {
		t.Error("go backend root package.json: should not have concurrently")
	}
}

func TestRootPackageJSONAngularNode(t *testing.T) {
	app := testAppAngularNode()
	output := generateRootPackageJSON(app)

	// Should have both node and angular workspaces
	if !strings.Contains(output, `"node"`) {
		t.Error("angular+node root package.json: missing node workspace")
	}
	if !strings.Contains(output, `"angular"`) {
		t.Error("angular+node root package.json: missing angular workspace")
	}
	// Should not have react or vue
	if strings.Contains(output, `"react"`) {
		t.Error("angular+node root package.json: should not have react workspace")
	}
	// Should have concurrently (2 workspaces)
	if !strings.Contains(output, "concurrently") {
		t.Error("angular+node root package.json: missing concurrently")
	}
	// Should have Prisma scripts (Node backend)
	if !strings.Contains(output, "prisma") {
		t.Error("angular+node root package.json: missing Prisma scripts")
	}
	// Scripts should reference angular workspace
	if !strings.Contains(output, "workspace=angular") {
		t.Error("angular+node root package.json: scripts should reference angular workspace")
	}
}

// ── Node package.json ──

func TestNodePackageJSON(t *testing.T) {
	app := testApp()
	output := generateNodePackageJSON(app)

	checks := []struct {
		desc    string
		pattern string
	}{
		{"name", `"taskflow-backend"`},
		{"express", `"express": "^4.21.0"`},
		{"cors", `"cors": "^2.8.5"`},
		{"jsonwebtoken", `"jsonwebtoken": "^9.0.0"`},
		{"bcryptjs", `"bcryptjs": "^2.4.3"`},
		{"prisma client", `"@prisma/client": "^6.0.0"`},
		{"prisma dev", `"prisma": "^6.0.0"`},
		{"typescript", `"typescript": "^5.7.0"`},
		{"ts-node", `"ts-node": "^10.9.0"`},
		{"types/express", `"@types/express": "^5.0.0"`},
		{"types/cors", `"@types/cors": "^2.8.17"`},
		{"types/jsonwebtoken", `"@types/jsonwebtoken": "^9.0.7"`},
		{"types/bcryptjs", `"@types/bcryptjs": "^2.4.6"`},
		{"jest", `"jest": "^29.7.0"`},
		{"ts-jest", `"ts-jest": "^29.2.0"`},
		{"supertest", `"supertest": "^7.0.0"`},
		{"types/jest", `"@types/jest": "^29.5.0"`},
		{"types/supertest", `"@types/supertest": "^6.0.0"`},
		{"start script", `"start": "node dist/server.js"`},
		{"dev script", `"dev": "ts-node src/server.ts"`},
		{"build script", `"build": "tsc"`},
		{"test script", `"test": "jest"`},
	}

	for _, c := range checks {
		if !strings.Contains(output, c.pattern) {
			t.Errorf("node package.json: missing %s (%q)", c.desc, c.pattern)
		}
	}
}

func TestNodePackageJSONWithIntegrations(t *testing.T) {
	app := testApp()
	app.Integrations = []*ir.Integration{
		{Service: "SendGrid", Type: "email"},
		{Service: "AWS S3", Type: "storage"},
		{Service: "Stripe", Type: "payment"},
		{Service: "Slack", Type: "messaging"},
		{Service: "Google", Type: "oauth"},
	}
	output := generateNodePackageJSON(app)

	integChecks := []struct {
		desc    string
		pattern string
	}{
		{"sendgrid", `"@sendgrid/mail"`},
		{"s3 client", `"@aws-sdk/client-s3"`},
		{"s3 presigner", `"@aws-sdk/s3-request-presigner"`},
		{"stripe", `"stripe"`},
		{"slack webhook", `"@slack/webhook"`},
		{"passport", `"passport"`},
		{"passport-google", `"passport-google-oauth20"`},
		{"passport-github", `"passport-github2"`},
		{"types/passport", `"@types/passport"`},
	}

	for _, c := range integChecks {
		if !strings.Contains(output, c.pattern) {
			t.Errorf("node package.json with integrations: missing %s (%q)", c.desc, c.pattern)
		}
	}
}

func TestNodePackageJSONNoIntegrations(t *testing.T) {
	app := testApp()
	output := generateNodePackageJSON(app)

	// Without integrations, these should NOT appear
	if strings.Contains(output, "@sendgrid") {
		t.Error("unexpected sendgrid dependency without integration")
	}
	if strings.Contains(output, "stripe") {
		t.Error("unexpected stripe dependency without integration")
	}
}

// ── React package.json ──

func TestReactPackageJSON(t *testing.T) {
	app := testApp()
	output := generateReactPackageJSON(app)

	checks := []struct {
		desc    string
		pattern string
	}{
		{"name", `"taskflow-frontend"`},
		{"type module", `"type": "module"`},
		{"react", `"react": "^19.0.0"`},
		{"react-dom", `"react-dom": "^19.0.0"`},
		{"react-router-dom", `"react-router-dom": "^7.0.0"`},
		{"typescript", `"typescript": "^5.7.0"`},
		{"vite", `"vite": "^6.0.0"`},
		{"vitejs/plugin-react", `"@vitejs/plugin-react": "^4.3.0"`},
		{"types/react", `"@types/react": "^19.0.0"`},
		{"types/react-dom", `"@types/react-dom": "^19.0.0"`},
		{"tailwindcss", `"tailwindcss": "^3.4.0"`},
		{"autoprefixer", `"autoprefixer": "^10.4.0"`},
		{"postcss", `"postcss": "^8.4.0"`},
		{"dev script", `"dev": "vite"`},
		{"build script", `"build": "tsc && vite build"`},
		{"preview script", `"preview": "vite preview"`},
		{"start script", `"start": "vite preview"`},
	}

	for _, c := range checks {
		if !strings.Contains(output, c.pattern) {
			t.Errorf("react package.json: missing %s (%q)", c.desc, c.pattern)
		}
	}
}

// ── Vue package.json ──

func TestVuePackageJSON(t *testing.T) {
	app := testAppVuePython()
	output := generateVuePackageJSON(app)

	checks := []struct {
		desc    string
		pattern string
	}{
		{"name", `"myvueapp-frontend"`},
		{"type module", `"type": "module"`},
		{"vue", `"vue": "^3.5.0"`},
		{"vue-router", `"vue-router": "^4.4.0"`},
		{"pinia", `"pinia": "^2.2.0"`},
		{"vitejs/plugin-vue", `"@vitejs/plugin-vue": "^5.2.0"`},
		{"typescript", `"typescript": "^5.7.0"`},
		{"vite", `"vite": "^6.0.0"`},
		{"vue-tsc", `"vue-tsc": "^2.1.0"`},
		{"dev script", `"dev": "vite"`},
		{"build script", `"build": "vue-tsc && vite build"`},
	}

	for _, c := range checks {
		if !strings.Contains(output, c.pattern) {
			t.Errorf("vue package.json: missing %s (%q)", c.desc, c.pattern)
		}
	}
}

// ── Node tsconfig ──

func TestNodeTSConfig(t *testing.T) {
	output := generateNodeTSConfig()

	checks := []struct {
		desc    string
		pattern string
	}{
		{"target", `"target": "ES2022"`},
		{"module", `"module": "commonjs"`},
		{"outDir", `"outDir": "./dist"`},
		{"rootDir", `"rootDir": "./src"`},
		{"strict", `"strict": true`},
		{"esModuleInterop", `"esModuleInterop": true`},
		{"sourceMap", `"sourceMap": true`},
	}

	for _, c := range checks {
		if !strings.Contains(output, c.pattern) {
			t.Errorf("node tsconfig: missing %s (%q)", c.desc, c.pattern)
		}
	}
}

// ── React tsconfig ──

func TestReactTSConfig(t *testing.T) {
	output := generateReactTSConfig()

	checks := []struct {
		desc    string
		pattern string
	}{
		{"target", `"target": "ES2020"`},
		{"jsx", `"jsx": "react-jsx"`},
		{"noEmit", `"noEmit": true`},
		{"moduleResolution", `"moduleResolution": "bundler"`},
		{"strict", `"strict": true`},
		{"module", `"module": "ESNext"`},
	}

	for _, c := range checks {
		if !strings.Contains(output, c.pattern) {
			t.Errorf("react tsconfig: missing %s (%q)", c.desc, c.pattern)
		}
	}
}

// ── Vue tsconfig ──

func TestVueTSConfig(t *testing.T) {
	output := generateVueTSConfig()

	checks := []struct {
		desc    string
		pattern string
	}{
		{"target", `"target": "ES2020"`},
		{"noEmit", `"noEmit": true`},
		{"moduleResolution", `"moduleResolution": "bundler"`},
		{"strict", `"strict": true`},
		{"module", `"module": "ESNext"`},
	}

	for _, c := range checks {
		if !strings.Contains(output, c.pattern) {
			t.Errorf("vue tsconfig: missing %s (%q)", c.desc, c.pattern)
		}
	}

	// Should NOT have jsx (that's React-specific)
	if strings.Contains(output, `"jsx"`) {
		t.Error("vue tsconfig: should not contain jsx setting")
	}
}

// ── Vite config ──

func TestViteConfig(t *testing.T) {
	output := generateViteConfig()

	checks := []struct {
		desc    string
		pattern string
	}{
		{"import defineConfig", "import { defineConfig } from 'vite'"},
		{"import react", "import react from '@vitejs/plugin-react'"},
		{"react plugin", "plugins: [react()]"},
		{"api proxy", "'/api'"},
		{"proxy target", "target: 'http://localhost:3000'"},
		{"changeOrigin", "changeOrigin: true"},
	}

	for _, c := range checks {
		if !strings.Contains(output, c.pattern) {
			t.Errorf("vite config: missing %s (%q)", c.desc, c.pattern)
		}
	}
}

// ── README ──

func TestReadme(t *testing.T) {
	app := testApp()
	output := generateReadme(app)

	checks := []struct {
		desc    string
		pattern string
	}{
		{"title", "# TaskFlow"},
		{"human link", "github.com/barun-bash/human"},
		{"tech stack section", "## Tech Stack"},
		{"frontend config", "React with TypeScript"},
		{"backend config", "Node with Express"},
		{"database config", "PostgreSQL"},
		{"deploy config", "Docker"},
		{"data models section", "## Data Models"},
		{"User model", "| User |"},
		{"Task model", "| Task |"},
		{"api section", "## API Endpoints"},
		{"SignUp endpoint", "| SignUp |"},
		{"CreateTask auth", "| CreateTask | Yes |"},
		{"quick start", "## Quick Start"},
		{"local dev option", "Option 1: Local development"},
		{"docker option", "Option 2: Docker"},
		{"start.sh", "./start.sh"},
		{"ports section", "## Ports"},
		{"backend port", "| Backend | 3000 |"},
		{"frontend port", "| Frontend (dev) | 5173 |"},
		{"postgres port", "| PostgreSQL | 5432 |"},
	}

	for _, c := range checks {
		if !strings.Contains(output, c.pattern) {
			t.Errorf("README: missing %s (%q)", c.desc, c.pattern)
		}
	}
}

func TestReadmeGoBackend(t *testing.T) {
	app := testAppGoBackend()
	output := generateReadme(app)

	// Go backend should have port 8080
	if !strings.Contains(output, "| Backend | 8080 |") {
		t.Error("go backend README: missing port 8080")
	}
	// Should not have frontend dev port
	if strings.Contains(output, "Frontend (dev)") {
		t.Error("go backend README: should not have frontend port with None frontend")
	}
	// Should have go build instructions
	if !strings.Contains(output, "go build") {
		t.Error("go backend README: missing go build instructions")
	}
	// Should not have npm install
	if strings.Contains(output, "npm install") {
		t.Error("go backend README: should not have npm install")
	}
}

func TestReadmePythonBackend(t *testing.T) {
	app := testAppVuePython()
	output := generateReadme(app)

	// Python backend should have port 8000
	if !strings.Contains(output, "| Backend | 8000 |") {
		t.Error("python backend README: missing port 8000")
	}
	// Should have pip install
	if !strings.Contains(output, "pip install") {
		t.Error("python backend README: missing pip install")
	}
	// Should have frontend dev port (Vue)
	if !strings.Contains(output, "| Frontend (dev) | 5173 |") {
		t.Error("vue+python README: missing frontend port 5173")
	}
}

// ── .env.example ──

func TestEnvExample(t *testing.T) {
	app := testApp()
	output := generateEnvExample(app)

	checks := []struct {
		desc    string
		pattern string
	}{
		{"header", "Generated by Human compiler"},
		{"DATABASE_URL", "DATABASE_URL="},
		{"JWT_SECRET", "JWT_SECRET="},
		{"PORT", "PORT=3000"},
		{"VITE_API_URL", "VITE_API_URL="},
		{"SENDGRID_API_KEY", "SENDGRID_API_KEY="},
		{"Database section", "# Database"},
		{"Authentication section", "# Authentication"},
		{"Integration section", "# Integration: SendGrid"},
	}

	for _, c := range checks {
		if !strings.Contains(output, c.pattern) {
			t.Errorf(".env.example: missing %s (%q)", c.desc, c.pattern)
		}
	}
}

// ── start.sh ──

func TestStartScript(t *testing.T) {
	app := testApp()
	output := generateStartScript(app)

	checks := []struct {
		desc    string
		pattern string
	}{
		{"shebang", "#!/usr/bin/env bash"},
		{"set -e", "set -e"},
		{"npm install", "npm install"},
		{"env copy", "cp .env.example .env"},
		{"env check", "[ ! -f .env ]"},
		{"pg_isready check", "pg_isready"},
		{"docker compose suggestion", "docker compose up db -d"},
		{"prisma subshell", "(cd node && npx prisma generate && npx prisma db push)"},
		{"npm run dev", "npm run dev"},
	}

	for _, c := range checks {
		if !strings.Contains(output, c.pattern) {
			t.Errorf("start.sh: missing %s (%q)", c.desc, c.pattern)
		}
	}
}

func TestStartScriptGoBackend(t *testing.T) {
	app := testAppGoBackend()
	output := generateStartScript(app)

	// Should have Go build
	if !strings.Contains(output, "go build") {
		t.Error("go start.sh: missing go build")
	}
	// Should have postgres check
	if !strings.Contains(output, "pg_isready") {
		t.Error("go start.sh: missing pg_isready (config has PostgreSQL)")
	}
	// Should NOT have npm install
	if strings.Contains(output, "npm install") {
		t.Error("go start.sh: should not have npm install")
	}
	// Should NOT have Prisma
	if strings.Contains(output, "prisma") {
		t.Error("go start.sh: should not have prisma")
	}
	// Should start go binary
	if !strings.Contains(output, "./bin/server") {
		t.Error("go start.sh: missing ./bin/server")
	}
}

func TestStartScriptPythonBackend(t *testing.T) {
	app := testAppVuePython()
	output := generateStartScript(app)

	// Should have pip install
	if !strings.Contains(output, "pip install") {
		t.Error("python start.sh: missing pip install")
	}
	// Should have npm install (for Vue frontend)
	if !strings.Contains(output, "npm install") {
		t.Error("vue+python start.sh: missing npm install for Vue frontend")
	}
	// Should NOT have Prisma
	if strings.Contains(output, "prisma") {
		t.Error("python start.sh: should not have prisma")
	}
	// Should have npm run dev (Vue uses npm)
	if !strings.Contains(output, "npm run dev") {
		t.Error("vue+python start.sh: missing npm run dev")
	}
}

// ── matchesGoBackend ──

func TestMatchesGoBackend(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"Go", true},
		{"go", true},
		{"Go with Gin", true},
		{"go with fiber", true},
		{"golang", true},
		{"Gin", true},
		{"Fiber", true},
		{"Node", false},
		{"Python", false},
		{"Django", false},
		{"MongoDB", false},
		{"", false},
	}
	for _, tt := range tests {
		got := matchesGoBackend(tt.input)
		if got != tt.want {
			t.Errorf("matchesGoBackend(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// ── Generate to Filesystem ──

func TestGenerateWritesFiles(t *testing.T) {
	app := testApp()

	dir := t.TempDir()
	g := Generator{}
	if err := g.Generate(app, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	expectedFiles := []string{
		"package.json",
		"node/package.json",
		"react/package.json",
		"node/tsconfig.json",
		"react/tsconfig.json",
		"react/vite.config.ts",
		"README.md",
		".env.example",
		"start.sh",
	}

	for _, f := range expectedFiles {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}
}

func TestGenerateVuePythonWritesFiles(t *testing.T) {
	app := testAppVuePython()

	dir := t.TempDir()
	g := Generator{}
	if err := g.Generate(app, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Should exist
	expectedFiles := []string{
		"package.json",
		"vue/package.json",
		"vue/tsconfig.json",
		"README.md",
		".env.example",
		"start.sh",
	}
	for _, f := range expectedFiles {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("vue+python: expected file %s to exist", f)
		}
	}

	// Should NOT exist (no react, no node)
	unexpectedFiles := []string{
		"react/package.json",
		"react/tsconfig.json",
		"react/vite.config.ts",
		"node/package.json",
		"node/tsconfig.json",
	}
	for _, f := range unexpectedFiles {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); err == nil {
			t.Errorf("vue+python: file %s should not exist", f)
		}
	}
}

func TestGenerateGoBackendWritesFiles(t *testing.T) {
	app := testAppGoBackend()

	dir := t.TempDir()
	g := Generator{}
	if err := g.Generate(app, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Should exist (always generated)
	expectedFiles := []string{
		"package.json",
		"README.md",
		".env.example",
		"start.sh",
	}
	for _, f := range expectedFiles {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("go backend: expected file %s to exist", f)
		}
	}

	// Should NOT exist (no react, no node, no vue)
	unexpectedFiles := []string{
		"react/package.json",
		"react/tsconfig.json",
		"react/vite.config.ts",
		"node/package.json",
		"node/tsconfig.json",
		"vue/package.json",
		"vue/tsconfig.json",
	}
	for _, f := range unexpectedFiles {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); err == nil {
			t.Errorf("go backend: file %s should not exist", f)
		}
	}
}

func TestGenerateAngularNodeWritesFiles(t *testing.T) {
	app := testAppAngularNode()

	dir := t.TempDir()
	g := Generator{}
	if err := g.Generate(app, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Should exist (Node backend scaffold files + common files)
	expectedFiles := []string{
		"package.json",
		"node/package.json",
		"node/tsconfig.json",
		"README.md",
		".env.example",
		"start.sh",
	}
	for _, f := range expectedFiles {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("angular+node: expected file %s to exist", f)
		}
	}

	// Should NOT exist (Angular writes its own config, no react/vue)
	unexpectedFiles := []string{
		"react/package.json",
		"react/tsconfig.json",
		"react/vite.config.ts",
		"vue/package.json",
		"vue/tsconfig.json",
	}
	for _, f := range unexpectedFiles {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); err == nil {
			t.Errorf("angular+node: file %s should not exist", f)
		}
	}
}

// ── File Permissions ──

func TestStartScriptIsExecutable(t *testing.T) {
	app := testApp()

	dir := t.TempDir()
	g := Generator{}
	if err := g.Generate(app, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	info, err := os.Stat(filepath.Join(dir, "start.sh"))
	if err != nil {
		t.Fatalf("stat start.sh: %v", err)
	}

	mode := info.Mode().Perm()
	if mode&0111 == 0 {
		t.Errorf("start.sh is not executable: %o", mode)
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

	// Verify React+Node files exist (taskflow uses React+Node)
	expectedFiles := []string{
		"package.json",
		"node/package.json",
		"react/package.json",
		"node/tsconfig.json",
		"react/tsconfig.json",
		"react/vite.config.ts",
		"README.md",
		".env.example",
		"start.sh",
	}
	for _, f := range expectedFiles {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}

	// Root package.json: workspaces + concurrently
	rootPkg, _ := os.ReadFile(filepath.Join(dir, "package.json"))
	rootPkgStr := string(rootPkg)
	if !strings.Contains(rootPkgStr, `"taskflow"`) {
		t.Error("root package.json: missing taskflow name")
	}
	if !strings.Contains(rootPkgStr, `"workspaces"`) {
		t.Error("root package.json: missing workspaces")
	}
	if !strings.Contains(rootPkgStr, `"concurrently"`) {
		t.Error("root package.json: missing concurrently")
	}

	// Node package.json: express, prisma
	nodePkg, _ := os.ReadFile(filepath.Join(dir, "node/package.json"))
	nodePkgStr := string(nodePkg)
	if !strings.Contains(nodePkgStr, `"express"`) {
		t.Error("node package.json: missing express")
	}
	if !strings.Contains(nodePkgStr, `"@prisma/client"`) {
		t.Error("node package.json: missing prisma client")
	}

	// React package.json: react, vite
	reactPkg, _ := os.ReadFile(filepath.Join(dir, "react/package.json"))
	reactPkgStr := string(reactPkg)
	if !strings.Contains(reactPkgStr, `"react"`) {
		t.Error("react package.json: missing react")
	}
	if !strings.Contains(reactPkgStr, `"vite"`) {
		t.Error("react package.json: missing vite")
	}

	// .env.example: integration vars from taskflow app
	envContent, _ := os.ReadFile(filepath.Join(dir, ".env.example"))
	envStr := string(envContent)
	for _, v := range []string{"DATABASE_URL", "JWT_SECRET", "PORT", "VITE_API_URL"} {
		if !strings.Contains(envStr, v) {
			t.Errorf(".env.example: missing %s", v)
		}
	}

	// README: app name and sections
	readme, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	readmeStr := string(readme)
	if !strings.Contains(readmeStr, "# TaskFlow") {
		t.Error("README: missing app title")
	}
	if !strings.Contains(readmeStr, "## Data Models") {
		t.Error("README: missing data models section")
	}
	if !strings.Contains(readmeStr, "## API Endpoints") {
		t.Error("README: missing API endpoints section")
	}

	// start.sh: executable
	info, _ := os.Stat(filepath.Join(dir, "start.sh"))
	if info.Mode().Perm()&0111 == 0 {
		t.Error("start.sh: not executable")
	}

	t.Logf("Generated %d scaffold files to %s", len(expectedFiles), dir)
}
