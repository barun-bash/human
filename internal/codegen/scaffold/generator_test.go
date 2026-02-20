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
		{"npm option", "Option 1: npm"},
		{"docker option", "Option 2: Docker"},
		{"start.sh", "./start.sh"},
		{"ports section", "## Ports"},
	}

	for _, c := range checks {
		if !strings.Contains(output, c.pattern) {
			t.Errorf("README: missing %s (%q)", c.desc, c.pattern)
		}
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
		{"prisma generate", "npx prisma generate"},
		{"prisma db push", "npx prisma db push"},
		{"npm run dev", "npm run dev"},
	}

	for _, c := range checks {
		if !strings.Contains(output, c.pattern) {
			t.Errorf("start.sh: missing %s (%q)", c.desc, c.pattern)
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

	// Verify all 9 files exist
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
