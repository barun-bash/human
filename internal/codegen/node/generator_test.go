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
	app := &ir.Application{
		Data: []*ir.DataModel{
			{Name: "User", Fields: []*ir.DataField{{Name: "email", Type: "email"}}},
			{Name: "Task", Fields: []*ir.DataField{{Name: "title", Type: "text"}}},
			{Name: "Tag", Fields: []*ir.DataField{{Name: "name", Type: "text"}}},
		},
	}

	tests := []struct {
		text string
		want string
	}{
		{"create a User with the given fields", "User"},
		{"fetch all tasks for the current user", "Task"},
		{"update the Task", "Task"},
		{"delete the Task", "Task"},
		// These previously returned "Record" — now resolved via app.Data
		{"fetch the user by email", "User"},
		{"update the task with the given fields", "Task"},
	}
	for _, tt := range tests {
		got := inferModelFromAction(tt.text, app)
		if got != tt.want {
			t.Errorf("inferModelFromAction(%q): got %q, want %q", tt.text, got, tt.want)
		}
	}
}

func TestInferModelFromActionNilApp(t *testing.T) {
	// With nil app, capitalized words still work
	got := inferModelFromAction("create a User", nil)
	if got != "User" {
		t.Errorf("expected User, got %q", got)
	}
	// Without capitalized words and no app, falls back to "Record"
	got = inferModelFromAction("sort by due date", nil)
	if got != "Record" {
		t.Errorf("expected Record, got %q", got)
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

	// No duplicate const result
	count := strings.Count(output, "const result ")
	if count > 1 {
		t.Errorf("expected at most 1 'const result', got %d\n%s", count, output)
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

	// Should have bcrypt import (SignUp endpoint)
	if !strings.Contains(output, "import bcrypt from 'bcryptjs'") {
		t.Error("SignUp should import bcrypt")
	}

	// Should have signToken import
	if !strings.Contains(output, "import { signToken }") {
		t.Error("SignUp should import signToken")
	}

	// Should hash password
	if !strings.Contains(output, "bcrypt.hash(password, 12)") {
		t.Error("SignUp should hash password with bcrypt")
	}

	// Should use hashedPassword in create
	if !strings.Contains(output, "password: hashedPassword") {
		t.Error("SignUp should use hashedPassword in create data")
	}

	// Should return token in response
	if !strings.Contains(output, "signToken(result.id, result.role)") {
		t.Error("SignUp response should include signToken call")
	}
	if !strings.Contains(output, "{ data: result, token }") {
		t.Error("SignUp response should include token")
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

	app := &ir.Application{
		Data: []*ir.DataModel{
			{Name: "User"},
			{
				Name: "Task",
				Relations: []*ir.Relation{
					{Kind: "belongs_to", Target: "User"},
				},
			},
		},
	}

	output := generateRoute(ep, app)

	if !strings.Contains(output, "router.get(") {
		t.Error("GetTasks should use GET")
	}
	if !strings.Contains(output, "prisma.task.findMany(") {
		t.Error("missing prisma.task.findMany call")
	}
	// Should have userId scoping since Task belongs_to User and endpoint has Auth
	if !strings.Contains(output, "userId: req.userId") {
		t.Errorf("GetTasks should scope by userId when auth=true and Task belongs_to User\n%s", output)
	}
}

// ── SignUp Route Tests ──

func TestGenerateRouteSignUp(t *testing.T) {
	ep := &ir.Endpoint{
		Name: "SignUp",
		Auth: false,
		Params: []*ir.Param{
			{Name: "name"},
			{Name: "email"},
			{Name: "password"},
		},
		Validation: []*ir.ValidationRule{
			{Field: "email", Rule: "valid_email"},
			{Field: "email", Rule: "unique"},
			{Field: "password", Rule: "min_length", Value: "8"},
		},
		Steps: []*ir.Action{
			{Type: "create", Text: "create a User with the given fields"},
			{Type: "respond", Text: "respond with the created user and auth token"},
		},
	}

	app := &ir.Application{
		Data: []*ir.DataModel{
			{
				Name: "User",
				Fields: []*ir.DataField{
					{Name: "name", Type: "text"},
					{Name: "email", Type: "email", Unique: true},
					{Name: "password", Type: "text", Encrypted: true},
				},
			},
		},
	}

	output := generateRoute(ep, app)

	// bcrypt import
	if !strings.Contains(output, "import bcrypt from 'bcryptjs'") {
		t.Error("SignUp: missing bcrypt import")
	}
	// signToken import
	if !strings.Contains(output, "import { signToken } from '../middleware/auth'") {
		t.Error("SignUp: missing signToken import")
	}
	// Password hashing
	if !strings.Contains(output, "bcrypt.hash(password, 12)") {
		t.Errorf("SignUp: missing bcrypt.hash call\n%s", output)
	}
	// hashedPassword in create data
	if !strings.Contains(output, "password: hashedPassword") {
		t.Errorf("SignUp: should use hashedPassword in create data\n%s", output)
	}
	// Prisma model should be "user" not "record"
	if strings.Contains(output, "prisma.record") {
		t.Errorf("SignUp: should not contain prisma.record\n%s", output)
	}
	if !strings.Contains(output, "prisma.user.create(") {
		t.Errorf("SignUp: should create on prisma.user\n%s", output)
	}
	// Token in response
	if !strings.Contains(output, "signToken(") {
		t.Errorf("SignUp: missing signToken in response\n%s", output)
	}
	if !strings.Contains(output, "token") {
		t.Error("SignUp: response should include token")
	}
	// Unique validation should use "user" model (from app.Data)
	if !strings.Contains(output, "prisma.user.findUnique") {
		t.Errorf("SignUp: unique validation should use prisma.user.findUnique\n%s", output)
	}
}

// ── Login Route Tests ──

func TestGenerateRouteLogin(t *testing.T) {
	ep := &ir.Endpoint{
		Name: "Login",
		Auth: false,
		Params: []*ir.Param{
			{Name: "email"},
			{Name: "password"},
		},
		Steps: []*ir.Action{
			{Type: "query", Text: "fetch the user by email"},
			{Type: "condition", Text: "if user does not exist, respond with invalid credentials"},
			{Type: "condition", Text: "if password does not match, respond with error"},
			{Type: "respond", Text: "respond with the user and auth token"},
		},
	}

	app := &ir.Application{
		Data: []*ir.DataModel{
			{
				Name: "User",
				Fields: []*ir.DataField{
					{Name: "email", Type: "email"},
					{Name: "password", Type: "text", Encrypted: true},
				},
			},
		},
	}

	output := generateRoute(ep, app)

	// bcrypt import
	if !strings.Contains(output, "import bcrypt from 'bcryptjs'") {
		t.Errorf("Login: missing bcrypt import\n%s", output)
	}
	// signToken import
	if !strings.Contains(output, "import { signToken }") {
		t.Errorf("Login: missing signToken import\n%s", output)
	}
	// Should use findUnique, not findMany
	if !strings.Contains(output, "findUnique") {
		t.Errorf("Login: should use findUnique\n%s", output)
	}
	if strings.Contains(output, "findMany") {
		t.Errorf("Login: should not use findMany\n%s", output)
	}
	// Should use prisma.user, not prisma.record
	if strings.Contains(output, "prisma.record") {
		t.Errorf("Login: should not contain prisma.record\n%s", output)
	}
	if !strings.Contains(output, "prisma.user.findUnique") {
		t.Errorf("Login: should query prisma.user\n%s", output)
	}
	// Password comparison
	if !strings.Contains(output, "bcrypt.compare") {
		t.Errorf("Login: missing bcrypt.compare\n%s", output)
	}
	// Token generation
	if !strings.Contains(output, "signToken(") {
		t.Errorf("Login: missing signToken call\n%s", output)
	}
	// Error response for invalid credentials
	if !strings.Contains(output, "401") {
		t.Errorf("Login: missing 401 status for invalid credentials\n%s", output)
	}
	if !strings.Contains(output, "Invalid credentials") {
		t.Errorf("Login: missing 'Invalid credentials' error message\n%s", output)
	}
}

// ── Query Modifier Tests ──

func TestQueryModifierSkipped(t *testing.T) {
	app := &ir.Application{
		Data: []*ir.DataModel{
			{
				Name: "Task",
				Relations: []*ir.Relation{
					{Kind: "belongs_to", Target: "User"},
				},
			},
		},
	}

	ep := &ir.Endpoint{
		Name: "GetTasks",
		Auth: true,
		Steps: []*ir.Action{
			{Type: "query", Text: "fetch all tasks for the current user"},
			{Type: "query", Text: "sort by due date"},
			{Type: "query", Text: "support filtering by status"},
			{Type: "query", Text: "paginate with 20 per page"},
			{Type: "respond", Text: "respond with tasks"},
		},
	}

	output := generateRoute(ep, app)

	// The main query should exist
	if !strings.Contains(output, "prisma.task.findMany") {
		t.Errorf("missing main findMany query\n%s", output)
	}

	// Modifiers should be TODO comments, not additional Prisma queries
	if strings.Count(output, "prisma.task.findMany") > 1 {
		t.Errorf("query modifiers should not generate additional findMany calls, got %d\n%s",
			strings.Count(output, "prisma.task.findMany"), output)
	}

	// Modifiers should appear as TODO comments
	if !strings.Contains(output, "// TODO: sort by due date") {
		t.Errorf("missing TODO comment for sort modifier\n%s", output)
	}
	if !strings.Contains(output, "// TODO: support filtering by status") {
		t.Errorf("missing TODO comment for filter modifier\n%s", output)
	}
	if !strings.Contains(output, "// TODO: paginate with 20 per page") {
		t.Errorf("missing TODO comment for paginate modifier\n%s", output)
	}

	// No duplicate const result
	count := strings.Count(output, "const result ")
	if count > 1 {
		t.Errorf("expected at most 1 'const result', got %d\n%s", count, output)
	}
}

// ── Default Assignment Tests ──

func TestDefaultAssignment(t *testing.T) {
	ep := &ir.Endpoint{
		Name: "CreateTask",
		Auth: true,
		Params: []*ir.Param{
			{Name: "title"},
			{Name: "status"},
		},
		Steps: []*ir.Action{
			{Type: "update", Text: "set status to pending if not provided"},
			{Type: "create", Text: "create a Task with the given fields"},
			{Type: "respond", Text: "respond with the created task"},
		},
	}

	app := &ir.Application{
		Data: []*ir.DataModel{
			{Name: "Task", Fields: []*ir.DataField{{Name: "title", Type: "text"}, {Name: "status", Type: "text"}}},
		},
	}

	output := generateRoute(ep, app)

	// Should emit default assignment logic, not a Prisma update
	if !strings.Contains(output, "if (!status)") {
		t.Errorf("default assignment: missing status default check\n%s", output)
	}
	if !strings.Contains(output, "= 'pending'") {
		t.Errorf("default assignment: missing default value 'pending'\n%s", output)
	}
	// Should NOT have prisma.X.update for the default step
	if strings.Contains(output, "prisma.task.update(") || strings.Contains(output, "prisma.record.update(") {
		t.Errorf("default assignment should not emit prisma.update\n%s", output)
	}
}

// ── Field Name Mapping Tests ──

func TestFieldNameMapping(t *testing.T) {
	model := &ir.DataModel{
		Name: "Task",
		Fields: []*ir.DataField{
			{Name: "title", Type: "text"},
			{Name: "due", Type: "date"},
			{Name: "status", Type: "enum"},
		},
	}

	tests := []struct {
		paramName string
		wantField string
		wantParam string
	}{
		{"title", "title", "title"},
		{"due date", "due", "dueDate"}, // compound name maps to Prisma field "due"
		{"status", "status", "status"},
		{"unknown", "unknown", "unknown"},
	}

	for _, tt := range tests {
		gotField, gotParam := mapParamToPrismaField(tt.paramName, model)
		if gotField != tt.wantField || gotParam != tt.wantParam {
			t.Errorf("mapParamToPrismaField(%q): got (%q, %q), want (%q, %q)",
				tt.paramName, gotField, gotParam, tt.wantField, tt.wantParam)
		}
	}
}

func TestFieldNameMappingNilModel(t *testing.T) {
	// With nil model, should just return sanitized name for both
	gotField, gotParam := mapParamToPrismaField("due date", nil)
	if gotField != "dueDate" || gotParam != "dueDate" {
		t.Errorf("nil model: got (%q, %q), want (dueDate, dueDate)", gotField, gotParam)
	}
}

// ── ID Parameter Resolution Tests ──

func TestFindIdParam(t *testing.T) {
	tests := []struct {
		name   string
		params []*ir.Param
		want   string
	}{
		{"has task_id", []*ir.Param{{Name: "task_id"}, {Name: "title"}}, "task_id"},
		{"no id param", []*ir.Param{{Name: "title"}, {Name: "status"}}, ""},
		{"has id", []*ir.Param{{Name: "id"}, {Name: "title"}}, "id"},
	}

	for _, tt := range tests {
		ep := &ir.Endpoint{Params: tt.params}
		got := findIdParam(ep)
		if got != tt.want {
			t.Errorf("findIdParam(%s): got %q, want %q", tt.name, got, tt.want)
		}
	}
}

// ── Update and Delete ID Resolution Tests ──

func TestUpdateUsesIdParam(t *testing.T) {
	ep := &ir.Endpoint{
		Name: "UpdateTask",
		Auth: true,
		Params: []*ir.Param{
			{Name: "task_id"},
			{Name: "title"},
			{Name: "status"},
		},
		Steps: []*ir.Action{
			{Type: "update", Text: "update the Task with the given fields"},
			{Type: "respond", Text: "respond with the updated task"},
		},
	}

	app := &ir.Application{
		Data: []*ir.DataModel{
			{Name: "Task", Fields: []*ir.DataField{{Name: "title", Type: "text"}, {Name: "status", Type: "text"}}},
		},
	}

	output := generateRoute(ep, app)

	// Should use task_id, not req.params.id
	if strings.Contains(output, "req.params.id") {
		t.Errorf("UpdateTask should not use req.params.id\n%s", output)
	}
	if !strings.Contains(output, "where: { id: task_id }") {
		t.Errorf("UpdateTask should use task_id from params\n%s", output)
	}
}

func TestDeleteUsesIdParam(t *testing.T) {
	ep := &ir.Endpoint{
		Name: "DeleteTask",
		Auth: true,
		Params: []*ir.Param{
			{Name: "task_id"},
		},
		Steps: []*ir.Action{
			{Type: "delete", Text: "delete the Task"},
			{Type: "respond", Text: "respond with success"},
		},
	}

	app := &ir.Application{
		Data: []*ir.DataModel{
			{Name: "Task"},
		},
	}

	output := generateRoute(ep, app)

	if strings.Contains(output, "req.params.id") {
		t.Errorf("DeleteTask should not use req.params.id\n%s", output)
	}
	if !strings.Contains(output, "where: { id: task_id }") {
		t.Errorf("DeleteTask should use task_id from params\n%s", output)
	}
}

// ── userId Scoping Tests ──

func TestUserIdScopingOnCreate(t *testing.T) {
	ep := &ir.Endpoint{
		Name: "CreateTask",
		Auth: true,
		Params: []*ir.Param{
			{Name: "title"},
		},
		Steps: []*ir.Action{
			{Type: "create", Text: "create a Task with the given fields"},
			{Type: "respond", Text: "respond with the created task"},
		},
	}

	app := &ir.Application{
		Data: []*ir.DataModel{
			{Name: "User"},
			{
				Name: "Task",
				Fields: []*ir.DataField{
					{Name: "title", Type: "text"},
				},
				Relations: []*ir.Relation{
					{Kind: "belongs_to", Target: "User"},
				},
			},
		},
	}

	output := generateRoute(ep, app)

	if !strings.Contains(output, "userId: req.userId") {
		t.Errorf("CreateTask should include userId in create data when Task belongs_to User\n%s", output)
	}
}

// ── Condition Step Tests ──

func TestConditionStepNotFound(t *testing.T) {
	ep := &ir.Endpoint{
		Name: "GetProfile",
		Auth: true,
		Steps: []*ir.Action{
			{Type: "query", Text: "fetch the User by user_id"},
			{Type: "condition", Text: "if user does not exist, respond with user not found"},
			{Type: "respond", Text: "respond with the user"},
		},
	}

	app := &ir.Application{
		Data: []*ir.DataModel{
			{Name: "User", Fields: []*ir.DataField{{Name: "name", Type: "text"}}},
		},
	}

	output := generateRoute(ep, app)

	// Should generate actual not-found check
	if !strings.Contains(output, "if (!result)") {
		t.Errorf("condition step should generate not-found check\n%s", output)
	}
	if !strings.Contains(output, "404") {
		t.Errorf("condition step should return 404\n%s", output)
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
	createTaskStr := string(createTaskContent)
	if !strings.Contains(createTaskStr, "authorize('create', 'task')") {
		t.Error("create-task.ts: missing authorize middleware")
	}

	// ── Runtime Correctness Checks ──

	// sign-up.ts should contain bcrypt.hash
	signUpContent, err := os.ReadFile(filepath.Join(dir, "src", "routes", "sign-up.ts"))
	if err != nil {
		t.Fatalf("reading sign-up.ts: %v", err)
	}
	signUpStr := string(signUpContent)
	if !strings.Contains(signUpStr, "bcrypt.hash") {
		t.Errorf("sign-up.ts: missing bcrypt.hash\n%s", signUpStr)
	}
	if !strings.Contains(signUpStr, "signToken") {
		t.Errorf("sign-up.ts: missing signToken\n%s", signUpStr)
	}

	// login.ts should contain bcrypt.compare and signToken
	loginContent, err := os.ReadFile(filepath.Join(dir, "src", "routes", "login.ts"))
	if err != nil {
		t.Fatalf("reading login.ts: %v", err)
	}
	loginStr := string(loginContent)
	if !strings.Contains(loginStr, "bcrypt.compare") {
		t.Errorf("login.ts: missing bcrypt.compare\n%s", loginStr)
	}
	if !strings.Contains(loginStr, "signToken") {
		t.Errorf("login.ts: missing signToken\n%s", loginStr)
	}
	if !strings.Contains(loginStr, "findUnique") {
		t.Errorf("login.ts: should use findUnique, not findMany\n%s", loginStr)
	}

	// No route file should contain "prisma.record."
	routesDir := filepath.Join(dir, "src", "routes")
	routeFiles, _ := os.ReadDir(routesDir)
	for _, rf := range routeFiles {
		if rf.IsDir() || !strings.HasSuffix(rf.Name(), ".ts") || rf.Name() == "index.ts" {
			continue
		}
		content, err := os.ReadFile(filepath.Join(routesDir, rf.Name()))
		if err != nil {
			continue
		}
		if strings.Contains(string(content), "prisma.record.") {
			t.Errorf("%s: contains prisma.record. — model inference failed", rf.Name())
		}
	}

	// get-tasks.ts should have userId scoping
	getTasksContent, err := os.ReadFile(filepath.Join(dir, "src", "routes", "get-tasks.ts"))
	if err != nil {
		t.Fatalf("reading get-tasks.ts: %v", err)
	}
	getTasksStr := string(getTasksContent)
	if !strings.Contains(getTasksStr, "userId: req.userId") {
		t.Errorf("get-tasks.ts: missing userId scoping\n%s", getTasksStr)
	}

	// create-task.ts should not have duplicate const result
	resultCount := strings.Count(createTaskStr, "const result ")
	if resultCount > 1 {
		t.Errorf("create-task.ts: has %d 'const result' declarations (expected at most 1)\n%s", resultCount, createTaskStr)
	}

	// update-task.ts should use task_id, not req.params.id
	updateTaskContent, err := os.ReadFile(filepath.Join(dir, "src", "routes", "update-task.ts"))
	if err != nil {
		t.Fatalf("reading update-task.ts: %v", err)
	}
	updateTaskStr := string(updateTaskContent)
	if strings.Contains(updateTaskStr, "req.params.id") {
		t.Errorf("update-task.ts: should not use req.params.id\n%s", updateTaskStr)
	}

	totalFiles := len(coreFiles) + len(expectedRoutes) + len(policyFiles)
	t.Logf("Generated %d files to %s", totalFiles, dir)
}
