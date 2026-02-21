package node

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/ir"
	"github.com/barun-bash/human/internal/parser"
)

// ── Helper Utilities ──

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"GetTasks", "getTasks"},
		{"SignUp", "signUp"},
		{"Login", "login"},
		{"", ""},
		{"Sign Up", "signUp"},
	}
	for _, tt := range tests {
		got := toCamelCase(tt.input)
		if got != tt.want {
			t.Errorf("toCamelCase(%q): got %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestToKebabCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"GetTasks", "get-tasks"},
		{"Dashboard", "dashboard"},
		{"SignUp", "sign-up"},
	}
	for _, tt := range tests {
		got := toKebabCase(tt.input)
		if got != tt.want {
			t.Errorf("toKebabCase(%q): got %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestHttpMethod(t *testing.T) {
	tests := []struct {
		name string
		want string
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
		if got != tt.want {
			t.Errorf("httpMethod(%q): got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestRoutePath(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"GetTasks", "/tasks"},
		{"CreateTask", "/task"},
		{"UpdateTask", "/task"},
		{"DeleteTask", "/task"},
		{"SignUp", "/sign-up"},
		{"Login", "/login"},
		{"GetProfile", "/profile"},
	}
	for _, tt := range tests {
		got := routePath(tt.name)
		if got != tt.want {
			t.Errorf("routePath(%q): got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestPrismaType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"text", "String"},
		{"email", "String"},
		{"url", "String"},
		{"file", "String"},
		{"image", "String"},
		{"number", "Int"},
		{"decimal", "Float"},
		{"boolean", "Boolean"},
		{"date", "DateTime"},
		{"datetime", "DateTime"},
		{"json", "Json"},
		{"unknown", "String"},
	}
	for _, tt := range tests {
		got := prismaType(tt.input)
		if got != tt.want {
			t.Errorf("prismaType(%q): got %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestToJwtExpiry(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"7 days", "7d"},
		{"24 hours", "24h"},
		{"30 minutes", "30m"},
		{"60 seconds", "60s"},
		{"1 day", "1d"},
		{"bad", "7d"},
	}
	for _, tt := range tests {
		got := toJwtExpiry(tt.input)
		if got != tt.want {
			t.Errorf("toJwtExpiry(%q): got %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSanitizeParamName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"task_id", "task_id"},
		{"due date", "dueDate"},
		{"name", "name"},
	}
	for _, tt := range tests {
		got := sanitizeParamName(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeParamName(%q): got %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestInferModelFromAction(t *testing.T) {
	tests := []struct {
		text string
		want string
	}{
		{"create a User with the given fields", "User"},
		{"fetch all tasks for the current user", "Task"},
		{"update the Task", "Task"},
		{"delete the Task", "Task"},
	}
	for _, tt := range tests {
		got := inferModelFromAction(tt.text)
		if got != tt.want {
			t.Errorf("inferModelFromAction(%q): got %q, want %q", tt.text, got, tt.want)
		}
	}
}

// ── Prisma Schema Generator ──

func TestGeneratePrismaSchema(t *testing.T) {
	app := &ir.Application{
		Database: &ir.DatabaseConfig{
			Engine: "PostgreSQL",
			Indexes: []*ir.Index{
				{Entity: "User", Fields: []string{"email"}},
				// Use raw IR names to test resolution: "User" is a relation, "due date" is compound
				{Entity: "Task", Fields: []string{"User", "status"}},
				{Entity: "Task", Fields: []string{"User", "due date"}},
			},
		},
		Data: []*ir.DataModel{
			{
				Name: "User",
				Fields: []*ir.DataField{
					{Name: "name", Type: "text", Required: true},
					{Name: "email", Type: "email", Required: true, Unique: true},
					{Name: "password", Type: "text", Required: true, Encrypted: true},
					{Name: "bio", Type: "text", Required: false},
					{Name: "role", Type: "enum", Required: true, EnumValues: []string{"user", "admin"}},
					{Name: "age", Type: "number", Required: true},
					{Name: "active", Type: "boolean", Required: true},
				},
				Relations: []*ir.Relation{
					{Kind: "has_many", Target: "Task"},
				},
			},
			{
				Name: "Task",
				Fields: []*ir.DataField{
					{Name: "title", Type: "text", Required: true},
					{Name: "due", Type: "date", Required: true},
					{Name: "status", Type: "enum", Required: true, EnumValues: []string{"pending", "done"}},
				},
				Relations: []*ir.Relation{
					{Kind: "belongs_to", Target: "User"},
				},
			},
		},
	}

	output := generatePrismaSchema(app)

	// Datasource
	if !strings.Contains(output, `provider = "postgresql"`) {
		t.Error("missing postgresql provider")
	}
	if !strings.Contains(output, `env("DATABASE_URL")`) {
		t.Error("missing DATABASE_URL env reference")
	}

	// Generator
	if !strings.Contains(output, `provider = "prisma-client-js"`) {
		t.Error("missing prisma-client-js generator")
	}

	// Models
	if !strings.Contains(output, "model User {") {
		t.Error("missing User model")
	}
	if !strings.Contains(output, "model Task {") {
		t.Error("missing Task model")
	}

	// ID field
	if !strings.Contains(output, "@id @default(cuid())") {
		t.Error("missing id field with cuid()")
	}

	// Field types
	if !strings.Contains(output, "name      String") {
		t.Error("missing name String field")
	}
	if !strings.Contains(output, "age       Int") {
		t.Error("missing age Int field")
	}
	if !strings.Contains(output, "active    Boolean") {
		t.Error("missing active Boolean field")
	}

	// Unique
	if !strings.Contains(output, "@unique") {
		t.Error("missing @unique on email")
	}

	// Optional
	if !strings.Contains(output, "String?") {
		t.Error("missing optional String? for bio")
	}

	// Enum field reference in model
	if !strings.Contains(output, "UserRole") {
		t.Error("missing UserRole enum type reference in model")
	}

	// Enum blocks must be generated
	if !strings.Contains(output, "enum UserRole {") {
		t.Errorf("missing enum UserRole block\n%s", output)
	}
	if !strings.Contains(output, "enum TaskStatus {") {
		t.Errorf("missing enum TaskStatus block\n%s", output)
	}
	// Verify enum values
	if !strings.Contains(output, "  user\n") || !strings.Contains(output, "  admin\n") {
		t.Error("missing UserRole enum values (user, admin)")
	}
	if !strings.Contains(output, "  pending\n") || !strings.Contains(output, "  done\n") {
		t.Error("missing TaskStatus enum values (pending, done)")
	}

	// Relations
	if !strings.Contains(output, "tasks     Task[]") {
		t.Error("missing has_many Task[] relation on User")
	}
	if !strings.Contains(output, "userId") {
		t.Error("missing userId foreign key on Task")
	}
	if !strings.Contains(output, "@relation(fields: [userId], references: [id])") {
		t.Error("missing @relation directive on Task")
	}

	// Timestamps
	if !strings.Contains(output, "@default(now())") {
		t.Error("missing createdAt default")
	}
	if !strings.Contains(output, "@updatedAt") {
		t.Error("missing @updatedAt")
	}

	// Indexes — must use scalar FK fields, not relation names
	if !strings.Contains(output, "@@index([email])") {
		t.Errorf("missing @@index([email]) on User\n%s", output)
	}
	if !strings.Contains(output, "@@index([userId, status])") {
		t.Errorf("missing @@index([userId, status]) on Task — relation 'User' should resolve to 'userId'\n%s", output)
	}
	// "due date" compound name should resolve to field "due"
	if !strings.Contains(output, "@@index([userId, due])") {
		t.Errorf("missing @@index([userId, due]) on Task — 'due date' should resolve to field 'due'\n%s", output)
	}
	// Must NOT contain raw IR names in indexes
	if strings.Contains(output, "@@index([User,") || strings.Contains(output, "@@index([user,") {
		t.Errorf("@@index should not contain relation name 'user' — must use scalar FK 'userId'\n%s", output)
	}
	if strings.Contains(output, "dueDate") {
		t.Errorf("@@index should not contain 'dueDate' — should resolve to field name 'due'\n%s", output)
	}
}

func TestResolvePrismaFieldName(t *testing.T) {
	model := &ir.DataModel{
		Name: "Task",
		Fields: []*ir.DataField{
			{Name: "title", Type: "text"},
			{Name: "due", Type: "date"},
			{Name: "status", Type: "enum", EnumValues: []string{"pending", "done"}},
		},
		Relations: []*ir.Relation{
			{Kind: "belongs_to", Target: "User"},
		},
	}

	tests := []struct {
		input string
		want  string
	}{
		// Direct field match
		{"title", "title"},
		{"status", "status"},
		// Relation → FK scalar field
		{"User", "userId"},
		{"user", "userId"},
		// Compound name: field name + type
		{"due date", "due"},
		// Fallback
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		got := resolvePrismaFieldName(tt.input, model)
		if got != tt.want {
			t.Errorf("resolvePrismaFieldName(%q): got %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ── Auth Middleware Generator ──

func TestGenerateAuthMiddleware(t *testing.T) {
	app := &ir.Application{
		Auth: &ir.Auth{
			Methods: []*ir.AuthMethod{
				{Type: "jwt", Config: map[string]string{"expiration": "7 days"}},
				{Type: "oauth", Provider: "Google"},
			},
		},
	}

	output := generateAuthMiddleware(app)

	// JWT imports
	if !strings.Contains(output, "import jwt from 'jsonwebtoken'") {
		t.Error("missing jsonwebtoken import")
	}

	// JWT config
	if !strings.Contains(output, "JWT_SECRET") {
		t.Error("missing JWT_SECRET")
	}
	if !strings.Contains(output, "'7d'") {
		t.Error("missing JWT_EXPIRATION = 7d")
	}

	// authenticate function
	if !strings.Contains(output, "export function authenticate(") {
		t.Error("missing authenticate function")
	}
	if !strings.Contains(output, "Bearer ") {
		t.Error("missing Bearer token check")
	}
	if !strings.Contains(output, "jwt.verify(") {
		t.Error("missing jwt.verify call")
	}

	// signToken helper
	if !strings.Contains(output, "export function signToken(") {
		t.Error("missing signToken function")
	}

	// requireRole middleware
	if !strings.Contains(output, "export function requireRole(") {
		t.Error("missing requireRole function")
	}
	if !strings.Contains(output, "403") {
		t.Error("missing 403 status for insufficient permissions")
	}
}

// ── Error Handler Generator ──

func TestGenerateErrorHandler(t *testing.T) {
	app := &ir.Application{
		ErrorHandlers: []*ir.ErrorHandler{
			{
				Condition: "database is unreachable",
				Steps: []*ir.Action{
					{Type: "retry", Text: "retry 3 times with 1 second delay"},
					{Type: "alert", Text: "alert the engineering team via Slack"},
				},
			},
			{
				Condition: "validation fails",
				Steps: []*ir.Action{
					{Type: "respond", Text: "respond with helpful error messages"},
				},
			},
		},
	}

	output := generateErrorHandler(app)

	// Error handler configs
	if !strings.Contains(output, "database is unreachable") {
		t.Error("missing database error handler condition")
	}
	if !strings.Contains(output, "retries: 3") {
		t.Error("missing retries: 3")
	}
	if !strings.Contains(output, "delayMs: 1000") {
		t.Error("missing delayMs: 1000")
	}
	if !strings.Contains(output, "engineering team via Slack") {
		t.Error("missing alert target")
	}
	if !strings.Contains(output, "validation fails") {
		t.Error("missing validation error handler condition")
	}

	// Express error handler
	if !strings.Contains(output, "export function errorHandler(") {
		t.Error("missing errorHandler function")
	}
	if !strings.Contains(output, "503") {
		t.Error("missing 503 status for service unavailable")
	}
	if !strings.Contains(output, "400") {
		t.Error("missing 400 status for validation errors")
	}
	if !strings.Contains(output, "500") {
		t.Error("missing 500 status for unexpected errors")
	}

	// Retry utility
	if !strings.Contains(output, "export async function withRetry<T>(") {
		t.Error("missing withRetry utility")
	}
}

// ── Route Generator ──

func TestGenerateRoute(t *testing.T) {
	ep := &ir.Endpoint{
		Name: "CreateTask",
		Auth: true,
		Params: []*ir.Param{
			{Name: "title"},
			{Name: "description"},
		},
		Validation: []*ir.ValidationRule{
			{Field: "title", Rule: "not_empty"},
			{Field: "title", Rule: "max_length", Value: "200"},
		},
		Steps: []*ir.Action{
			{Type: "create", Text: "create a Task with the given fields"},
			{Type: "respond", Text: "respond with the created task"},
		},
	}

	app := &ir.Application{}
	output := generateRoute(ep, app)

	// Imports
	if !strings.Contains(output, "import { Router") {
		t.Error("missing Router import")
	}
	if !strings.Contains(output, "import { PrismaClient }") {
		t.Error("missing PrismaClient import")
	}
	if !strings.Contains(output, "import { authenticate }") {
		t.Error("missing auth middleware import")
	}

	// HTTP method
	if !strings.Contains(output, "router.post(") {
		t.Error("CreateTask should use POST")
	}

	// Auth middleware
	if !strings.Contains(output, "authenticate") {
		t.Error("missing authenticate middleware")
	}

	// Params extraction
	if !strings.Contains(output, "const { title, description } = req.body") {
		t.Error("missing body destructuring")
	}

	// Validation
	if !strings.Contains(output, "title is required") {
		t.Error("missing not_empty validation for title")
	}
	if !strings.Contains(output, "less than 200 characters") {
		t.Error("missing max_length validation for title")
	}

	// Prisma create
	if !strings.Contains(output, "prisma.task.create(") {
		t.Error("missing prisma.task.create call")
	}

	// Response
	if !strings.Contains(output, "res.json({ data: result })") {
		t.Error("missing JSON response")
	}

	// Error handling
	if !strings.Contains(output, "next(error)") {
		t.Error("missing error forwarding to next()")
	}
}

func TestGenerateRouteNoAuth(t *testing.T) {
	ep := &ir.Endpoint{
		Name: "SignUp",
		Auth: false,
		Params: []*ir.Param{
			{Name: "email"},
			{Name: "password"},
		},
		Steps: []*ir.Action{
			{Type: "create", Text: "create a User with the given fields"},
			{Type: "respond", Text: "respond with the created user"},
		},
	}

	output := generateRoute(ep, &ir.Application{})

	// Should NOT import auth middleware
	if strings.Contains(output, "import { authenticate }") {
		t.Error("SignUp should not import authenticate")
	}

	// Should use POST
	if !strings.Contains(output, "router.post(") {
		t.Error("SignUp should use POST")
	}
}

func TestGenerateRouteGet(t *testing.T) {
	ep := &ir.Endpoint{
		Name: "GetTasks",
		Auth: true,
		Steps: []*ir.Action{
			{Type: "query", Text: "fetch all tasks for the current user"},
			{Type: "respond", Text: "respond with tasks"},
		},
	}

	output := generateRoute(ep, &ir.Application{})

	if !strings.Contains(output, "router.get(") {
		t.Error("GetTasks should use GET")
	}
	if !strings.Contains(output, "prisma.task.findMany()") {
		t.Error("missing prisma.task.findMany call")
	}
}

// ── Route Index Generator ──

func TestGenerateRouteIndex(t *testing.T) {
	app := &ir.Application{
		APIs: []*ir.Endpoint{
			{Name: "SignUp"},
			{Name: "Login"},
			{Name: "GetTasks"},
			{Name: "CreateTask"},
		},
	}

	output := generateRouteIndex(app)

	// Imports
	if !strings.Contains(output, "import { router as signUpRouter } from './sign-up'") {
		t.Error("missing signUp router import")
	}
	if !strings.Contains(output, "import { router as loginRouter } from './login'") {
		t.Error("missing login router import")
	}
	if !strings.Contains(output, "import { router as getTasksRouter } from './get-tasks'") {
		t.Error("missing getTasks router import")
	}

	// Mounting
	if !strings.Contains(output, "router.use('/sign-up', signUpRouter)") {
		t.Error("missing signUp mount")
	}
	if !strings.Contains(output, "router.use('/tasks', getTasksRouter)") {
		t.Error("missing getTasks mount at /tasks")
	}

	// Export
	if !strings.Contains(output, "export { router }") {
		t.Error("missing router export")
	}
}

// ── Server Generator ──

func TestGenerateServer(t *testing.T) {
	app := &ir.Application{
		Name: "TaskFlow",
		Auth: &ir.Auth{
			Rules: []*ir.Action{
				{Type: "configure", Text: "rate limit all endpoints to 100 requests per minute"},
			},
		},
	}

	output := generateServer(app)

	// Imports
	if !strings.Contains(output, "import express from 'express'") {
		t.Error("missing express import")
	}
	if !strings.Contains(output, "import cors from 'cors'") {
		t.Error("missing cors import")
	}
	if !strings.Contains(output, "import { router } from './routes'") {
		t.Error("missing routes import")
	}
	if !strings.Contains(output, "import { errorHandler } from './middleware/errors'") {
		t.Error("missing error handler import")
	}

	// Middleware
	if !strings.Contains(output, "app.use(cors())") {
		t.Error("missing cors middleware")
	}
	if !strings.Contains(output, "app.use(express.json())") {
		t.Error("missing JSON body parser")
	}

	// Rate limiting TODO
	if !strings.Contains(output, "rate limiting") {
		t.Error("missing rate limiting comment")
	}

	// Routes
	if !strings.Contains(output, "app.use('/api', router)") {
		t.Error("missing API route mount")
	}

	// Health check
	if !strings.Contains(output, "/health") {
		t.Error("missing health check endpoint")
	}

	// Error handler
	if !strings.Contains(output, "app.use(errorHandler)") {
		t.Error("missing error handler registration")
	}

	// Listen
	if !strings.Contains(output, "app.listen(PORT") {
		t.Error("missing app.listen")
	}
	if !strings.Contains(output, "TaskFlow server running") {
		t.Error("missing app name in startup log")
	}

	// Export
	if !strings.Contains(output, "export { app }") {
		t.Error("missing app export")
	}
}

// ── Generate to Filesystem ──

func TestGenerateWritesFiles(t *testing.T) {
	app := &ir.Application{
		Name:     "TestApp",
		Platform: "web",
		Database: &ir.DatabaseConfig{Engine: "PostgreSQL"},
		Data: []*ir.DataModel{
			{Name: "User", Fields: []*ir.DataField{{Name: "email", Type: "email", Required: true}}},
		},
		APIs: []*ir.Endpoint{
			{Name: "SignUp", Params: []*ir.Param{{Name: "email"}}},
			{Name: "GetUsers", Auth: true},
		},
		Auth: &ir.Auth{
			Methods: []*ir.AuthMethod{{Type: "jwt", Config: map[string]string{"expiration": "7 days"}}},
		},
		ErrorHandlers: []*ir.ErrorHandler{
			{Condition: "test error", Steps: []*ir.Action{{Type: "retry", Text: "retry 3 times"}}},
		},
	}

	dir := t.TempDir()
	g := Generator{}
	if err := g.Generate(app, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	expectedFiles := []string{
		"prisma/schema.prisma",
		"src/server.ts",
		"src/middleware/auth.ts",
		"src/middleware/errors.ts",
		"src/routes/index.ts",
		"src/routes/sign-up.ts",
		"src/routes/get-users.ts",
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

	// Verify core files exist
	coreFiles := []string{
		"prisma/schema.prisma",
		"src/server.ts",
		"src/middleware/auth.ts",
		"src/middleware/errors.ts",
		"src/routes/index.ts",
	}
	for _, f := range coreFiles {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}

	// Verify route files for all 8 APIs
	expectedRoutes := []string{
		"sign-up.ts", "login.ts", "get-tasks.ts", "create-task.ts",
		"update-task.ts", "delete-task.ts", "get-profile.ts", "update-profile.ts",
	}
	for _, f := range expectedRoutes {
		path := filepath.Join(dir, "src", "routes", f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected route file %s to exist", f)
		}
	}

	// Verify schema.prisma has 4 models
	schemaContent, err := os.ReadFile(filepath.Join(dir, "prisma", "schema.prisma"))
	if err != nil {
		t.Fatalf("reading schema.prisma: %v", err)
	}
	schema := string(schemaContent)
	modelCount := strings.Count(schema, "model ")
	if modelCount != 4 {
		t.Errorf("schema.prisma: expected 4 models, got %d", modelCount)
	}
	for _, name := range []string{"User", "Task", "Tag", "TaskTag"} {
		if !strings.Contains(schema, "model "+name+" {") {
			t.Errorf("schema.prisma: missing model %s", name)
		}
	}

	// Verify postgresql provider
	if !strings.Contains(schema, `provider = "postgresql"`) {
		t.Error("schema.prisma: missing postgresql provider")
	}

	// Verify enum blocks are generated
	for _, enumName := range []string{"UserRole", "TaskStatus", "TaskPriority"} {
		if !strings.Contains(schema, "enum "+enumName+" {") {
			t.Errorf("schema.prisma: missing enum block %s", enumName)
		}
	}

	// Verify indexes use scalar FK fields, not relation names
	if strings.Contains(schema, "@@index([user,") || strings.Contains(schema, "@@index([user]") {
		t.Error("schema.prisma: @@index should use userId, not user (relation name)")
	}
	// "due date" from IR should resolve to field name "due", not "dueDate"
	if strings.Contains(schema, "dueDate") {
		t.Errorf("schema.prisma: should not contain 'dueDate' — should be 'due'\nschema:\n%s", schema)
	}

	// Verify auth middleware has JWT config
	authContent, err := os.ReadFile(filepath.Join(dir, "src", "middleware", "auth.ts"))
	if err != nil {
		t.Fatalf("reading auth.ts: %v", err)
	}
	auth := string(authContent)
	if !strings.Contains(auth, "'7d'") {
		t.Error("auth.ts: missing 7d JWT expiration from IR")
	}

	// Verify server.ts references TaskFlow
	serverContent, err := os.ReadFile(filepath.Join(dir, "src", "server.ts"))
	if err != nil {
		t.Fatalf("reading server.ts: %v", err)
	}
	server := string(serverContent)
	if !strings.Contains(server, "TaskFlow") {
		t.Error("server.ts: missing TaskFlow app name")
	}

	// Verify route index imports all 8 routers
	indexContent, err := os.ReadFile(filepath.Join(dir, "src", "routes", "index.ts"))
	if err != nil {
		t.Fatalf("reading routes/index.ts: %v", err)
	}
	index := string(indexContent)
	importCount := strings.Count(index, "import { router as ")
	if importCount != 8 {
		t.Errorf("routes/index.ts: expected 8 imports, got %d", importCount)
	}

	// Verify error handlers reference database unreachable
	errorsContent, err := os.ReadFile(filepath.Join(dir, "src", "middleware", "errors.ts"))
	if err != nil {
		t.Fatalf("reading errors.ts: %v", err)
	}
	errors := string(errorsContent)
	if !strings.Contains(errors, "database is unreachable") {
		t.Error("errors.ts: missing database error handler from IR")
	}

	// Verify authorization middleware files (TaskFlow has 3 policies)
	policyFiles := []string{
		"src/middleware/policies.ts",
		"src/middleware/authorize.ts",
	}
	for _, f := range policyFiles {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected policy file %s to exist", f)
		}
	}

	// Verify policies.ts contains all 3 policies
	policiesContent, err := os.ReadFile(filepath.Join(dir, "src", "middleware", "policies.ts"))
	if err != nil {
		t.Fatalf("reading policies.ts: %v", err)
	}
	policiesStr := string(policiesContent)
	for _, policyName := range []string{"FreeUser", "ProUser", "Admin"} {
		if !strings.Contains(policiesStr, policyName+":") {
			t.Errorf("policies.ts: missing policy %s", policyName)
		}
	}

	// Verify authorize.ts has the authorize function
	authorizeContent, err := os.ReadFile(filepath.Join(dir, "src", "middleware", "authorize.ts"))
	if err != nil {
		t.Fatalf("reading authorize.ts: %v", err)
	}
	if !strings.Contains(string(authorizeContent), "export function authorize") {
		t.Error("authorize.ts: missing authorize function")
	}

	// Verify routes with auth use authorize middleware
	createTaskContent, err := os.ReadFile(filepath.Join(dir, "src", "routes", "create-task.ts"))
	if err != nil {
		t.Fatalf("reading create-task.ts: %v", err)
	}
	if !strings.Contains(string(createTaskContent), "authorize('create', 'task')") {
		t.Error("create-task.ts: missing authorize middleware")
	}

	totalFiles := len(coreFiles) + len(expectedRoutes) + len(policyFiles)
	t.Logf("Generated %d files to %s", totalFiles, dir)
}
