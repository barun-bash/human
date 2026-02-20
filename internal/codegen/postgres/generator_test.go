package postgres

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

func TestPgType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"text", "TEXT"},
		{"email", "TEXT"},
		{"url", "TEXT"},
		{"file", "TEXT"},
		{"image", "TEXT"},
		{"number", "INTEGER"},
		{"decimal", "NUMERIC"},
		{"boolean", "BOOLEAN"},
		{"date", "DATE"},
		{"datetime", "TIMESTAMPTZ"},
		{"json", "JSONB"},
		{"unknown", "TEXT"},
	}
	for _, tt := range tests {
		got := pgType(tt.input)
		if got != tt.want {
			t.Errorf("pgType(%q): got %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"TaskTag", "task_tag"},
		{"User", "user"},
		{"createdAt", "created_at"},
		{"GetTasks", "get_tasks"},
	}
	for _, tt := range tests {
		got := toSnakeCase(tt.input)
		if got != tt.want {
			t.Errorf("toSnakeCase(%q): got %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestToTableName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"User", "users"},
		{"Task", "tasks"},
		{"TaskTag", "task_tags"},
		{"Tag", "tags"},
	}
	for _, tt := range tests {
		got := toTableName(tt.input)
		if got != tt.want {
			t.Errorf("toTableName(%q): got %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestEnumTypeName(t *testing.T) {
	tests := []struct {
		model string
		field string
		want  string
	}{
		{"User", "role", "user_role"},
		{"Task", "status", "task_status"},
		{"Task", "priority", "task_priority"},
	}
	for _, tt := range tests {
		got := enumTypeName(tt.model, tt.field)
		if got != tt.want {
			t.Errorf("enumTypeName(%q, %q): got %q, want %q", tt.model, tt.field, got, tt.want)
		}
	}
}

func TestSanitizeIdentifier(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"due date", "due_date"},
		{"name", "name"},
		{"task_id", "task_id"},
	}
	for _, tt := range tests {
		got := sanitizeIdentifier(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeIdentifier(%q): got %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIsJoinTable(t *testing.T) {
	join := &ir.DataModel{
		Name: "TaskTag",
		Relations: []*ir.Relation{
			{Kind: "belongs_to", Target: "Task"},
			{Kind: "belongs_to", Target: "Tag"},
		},
	}
	if !isJoinTable(join) {
		t.Error("TaskTag should be a join table")
	}

	notJoin := &ir.DataModel{
		Name: "User",
		Fields: []*ir.DataField{
			{Name: "email", Type: "email"},
		},
	}
	if isJoinTable(notJoin) {
		t.Error("User should not be a join table")
	}
}

func TestSortModelsForCreation(t *testing.T) {
	models := []*ir.DataModel{
		{Name: "TaskTag", Relations: []*ir.Relation{{Kind: "belongs_to", Target: "Task"}, {Kind: "belongs_to", Target: "Tag"}}},
		{Name: "User"},
		{Name: "Task", Relations: []*ir.Relation{{Kind: "belongs_to", Target: "User"}}},
		{Name: "Tag"},
	}

	sorted := sortModelsForCreation(models)

	// Independent models first
	if sorted[0].Name != "User" || sorted[1].Name != "Tag" {
		t.Errorf("expected independent models first, got %s, %s", sorted[0].Name, sorted[1].Name)
	}

	// Dependent models after
	depNames := []string{sorted[2].Name, sorted[3].Name}
	if !contains(depNames, "Task") || !contains(depNames, "TaskTag") {
		t.Errorf("expected dependent models after, got %v", depNames)
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// ── Enum Collection ──

func TestCollectEnums(t *testing.T) {
	app := &ir.Application{
		Data: []*ir.DataModel{
			{
				Name: "User",
				Fields: []*ir.DataField{
					{Name: "role", Type: "enum", EnumValues: []string{"user", "admin"}},
				},
			},
			{
				Name: "Task",
				Fields: []*ir.DataField{
					{Name: "status", Type: "enum", EnumValues: []string{"pending", "done"}},
					{Name: "title", Type: "text"},
				},
			},
		},
	}

	enums := collectEnums(app)
	if len(enums) != 2 {
		t.Fatalf("expected 2 enums, got %d", len(enums))
	}
	if enums[0].TypeName != "user_role" {
		t.Errorf("enum 0: got %q", enums[0].TypeName)
	}
	if enums[1].TypeName != "task_status" {
		t.Errorf("enum 1: got %q", enums[1].TypeName)
	}
}

// ── Migration Generation ──

func TestGenerateMigration(t *testing.T) {
	app := &ir.Application{
		Database: &ir.DatabaseConfig{
			Engine: "PostgreSQL",
			Indexes: []*ir.Index{
				{Entity: "User", Fields: []string{"email"}},
				{Entity: "Task", Fields: []string{"user_id", "status"}},
			},
		},
		Data: []*ir.DataModel{
			{
				Name: "User",
				Fields: []*ir.DataField{
					{Name: "name", Type: "text", Required: true},
					{Name: "email", Type: "email", Required: true, Unique: true},
					{Name: "password", Type: "text", Required: true, Encrypted: true},
					{Name: "role", Type: "enum", Required: true, EnumValues: []string{"user", "admin"}},
					{Name: "bio", Type: "text", Required: false},
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

	output := generateMigration(app)

	// Transaction wrapping
	if !strings.Contains(output, "BEGIN;") {
		t.Error("missing BEGIN")
	}
	if !strings.Contains(output, "COMMIT;") {
		t.Error("missing COMMIT")
	}

	// Enum types
	if !strings.Contains(output, "CREATE TYPE user_role AS ENUM ('user', 'admin')") {
		t.Error("missing user_role enum")
	}
	if !strings.Contains(output, "CREATE TYPE task_status AS ENUM ('pending', 'done')") {
		t.Error("missing task_status enum")
	}

	// Tables
	if !strings.Contains(output, "CREATE TABLE users (") {
		t.Error("missing users table")
	}
	if !strings.Contains(output, "CREATE TABLE tasks (") {
		t.Error("missing tasks table")
	}

	// Primary key
	if !strings.Contains(output, "id UUID PRIMARY KEY DEFAULT gen_random_uuid()") {
		t.Error("missing UUID primary key")
	}

	// Column types
	if !strings.Contains(output, "name TEXT NOT NULL") {
		t.Error("missing name TEXT NOT NULL")
	}
	if !strings.Contains(output, "email TEXT NOT NULL UNIQUE") {
		t.Error("missing email TEXT NOT NULL UNIQUE")
	}
	if !strings.Contains(output, "age INTEGER NOT NULL") {
		t.Error("missing age INTEGER")
	}
	if !strings.Contains(output, "active BOOLEAN NOT NULL") {
		t.Error("missing active BOOLEAN")
	}
	if !strings.Contains(output, "due DATE NOT NULL") {
		t.Error("missing due DATE")
	}

	// Optional field (no NOT NULL)
	if !strings.Contains(output, "bio TEXT,") {
		t.Error("bio should be optional (no NOT NULL)")
	}

	// Enum column type
	if !strings.Contains(output, "role user_role NOT NULL") {
		t.Error("missing role using user_role enum type")
	}
	if !strings.Contains(output, "status task_status NOT NULL") {
		t.Error("missing status using task_status enum type")
	}

	// Foreign key inline
	if !strings.Contains(output, "user_id UUID NOT NULL REFERENCES users(id)") {
		t.Error("missing user_id FK in tasks table")
	}

	// Timestamps
	if !strings.Contains(output, "created_at TIMESTAMPTZ NOT NULL DEFAULT now()") {
		t.Error("missing created_at")
	}
	if !strings.Contains(output, "updated_at TIMESTAMPTZ NOT NULL DEFAULT now()") {
		t.Error("missing updated_at")
	}

	// Indexes
	if !strings.Contains(output, "CREATE INDEX idx_users_email ON users (email)") {
		t.Error("missing email index on users")
	}
	if !strings.Contains(output, "CREATE INDEX idx_tasks_user_id_status ON tasks (user_id, status)") {
		t.Error("missing composite index on tasks")
	}

	// Foreign key constraints
	if !strings.Contains(output, "ALTER TABLE tasks ADD CONSTRAINT fk_tasks_user_id FOREIGN KEY (user_id) REFERENCES users(id)") {
		t.Error("missing FK constraint")
	}

	// Section ordering: enums before tables before indexes before FKs
	enumIdx := strings.Index(output, "CREATE TYPE")
	tableIdx := strings.Index(output, "CREATE TABLE")
	indexIdx := strings.Index(output, "CREATE INDEX")
	fkIdx := strings.Index(output, "ALTER TABLE")

	if enumIdx >= tableIdx {
		t.Error("enums should come before tables")
	}
	if tableIdx >= indexIdx {
		t.Error("tables should come before indexes")
	}
	if indexIdx >= fkIdx {
		t.Error("indexes should come before foreign keys")
	}
}

func TestGenerateMigrationJoinTable(t *testing.T) {
	app := &ir.Application{
		Data: []*ir.DataModel{
			{Name: "Task", Fields: []*ir.DataField{{Name: "title", Type: "text", Required: true}}},
			{Name: "Tag", Fields: []*ir.DataField{{Name: "name", Type: "text", Required: true}}},
			{
				Name: "TaskTag",
				Relations: []*ir.Relation{
					{Kind: "belongs_to", Target: "Task"},
					{Kind: "belongs_to", Target: "Tag"},
				},
			},
		},
	}

	output := generateMigration(app)

	if !strings.Contains(output, "CREATE TABLE task_tags (") {
		t.Error("missing task_tags join table")
	}
	if !strings.Contains(output, "task_id UUID NOT NULL REFERENCES tasks(id)") {
		t.Error("missing task_id FK in join table")
	}
	if !strings.Contains(output, "tag_id UUID NOT NULL REFERENCES tags(id)") {
		t.Error("missing tag_id FK in join table")
	}
}

// ── Seed Generation ──

func TestGenerateSeed(t *testing.T) {
	app := &ir.Application{
		Data: []*ir.DataModel{
			{
				Name: "User",
				Fields: []*ir.DataField{
					{Name: "name", Type: "text", Required: true},
					{Name: "email", Type: "email", Required: true},
					{Name: "role", Type: "enum", Required: true, EnumValues: []string{"admin", "user"}},
					{Name: "active", Type: "boolean", Required: true},
				},
			},
			{
				Name: "Task",
				Fields: []*ir.DataField{
					{Name: "title", Type: "text", Required: true},
					{Name: "due", Type: "date", Required: true},
				},
				Relations: []*ir.Relation{
					{Kind: "belongs_to", Target: "User"},
				},
			},
			{
				Name: "TaskTag",
				Relations: []*ir.Relation{
					{Kind: "belongs_to", Target: "Task"},
					{Kind: "belongs_to", Target: "Tag"},
				},
			},
		},
	}

	output := generateSeed(app)

	// Transaction
	if !strings.Contains(output, "BEGIN;") {
		t.Error("missing BEGIN")
	}
	if !strings.Contains(output, "COMMIT;") {
		t.Error("missing COMMIT")
	}

	// Insert for User
	if !strings.Contains(output, "INSERT INTO users") {
		t.Error("missing users insert")
	}

	// Insert for Task
	if !strings.Contains(output, "INSERT INTO tasks") {
		t.Error("missing tasks insert")
	}

	// Insert for join table
	if !strings.Contains(output, "INSERT INTO task_tags") {
		t.Error("missing task_tags insert")
	}

	// Sample values
	if !strings.Contains(output, "@example.com") {
		t.Error("missing sample email")
	}
	if !strings.Contains(output, "'admin'") {
		t.Error("missing enum sample value")
	}
	if !strings.Contains(output, "user_id") {
		t.Error("missing FK reference in task seed")
	}
}

// ── Generate to Filesystem ──

func TestGenerateWritesFiles(t *testing.T) {
	app := &ir.Application{
		Name:     "TestApp",
		Database: &ir.DatabaseConfig{Engine: "PostgreSQL"},
		Data: []*ir.DataModel{
			{Name: "User", Fields: []*ir.DataField{{Name: "email", Type: "email", Required: true}}},
		},
	}

	dir := t.TempDir()
	g := Generator{}
	if err := g.Generate(app, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	expectedFiles := []string{
		"migrations/001_initial.sql",
		"seed.sql",
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

	// Verify files exist
	for _, f := range []string{"migrations/001_initial.sql", "seed.sql"} {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}

	// ── Migration checks ──
	migContent, err := os.ReadFile(filepath.Join(dir, "migrations", "001_initial.sql"))
	if err != nil {
		t.Fatalf("reading migration: %v", err)
	}
	mig := string(migContent)

	// 4 tables (users, tasks, tags, task_tags)
	tableCount := strings.Count(mig, "CREATE TABLE ")
	if tableCount != 4 {
		t.Errorf("migration: expected 4 CREATE TABLE, got %d", tableCount)
	}

	// 3 enum types (user_role, task_status, task_priority)
	enumCount := strings.Count(mig, "CREATE TYPE ")
	if enumCount != 3 {
		t.Errorf("migration: expected 3 CREATE TYPE, got %d", enumCount)
	}
	for _, enumName := range []string{"user_role", "task_status", "task_priority"} {
		if !strings.Contains(mig, "CREATE TYPE "+enumName) {
			t.Errorf("migration: missing enum %s", enumName)
		}
	}

	// Indexes from database config (4 in the IR)
	indexCount := strings.Count(mig, "CREATE INDEX ")
	if indexCount < 3 {
		t.Errorf("migration: expected at least 3 CREATE INDEX, got %d", indexCount)
	}

	// Foreign keys
	fkCount := strings.Count(mig, "ALTER TABLE")
	if fkCount < 3 {
		t.Errorf("migration: expected at least 3 FK constraints, got %d", fkCount)
	}

	// ── Seed checks ──
	seedContent, err := os.ReadFile(filepath.Join(dir, "seed.sql"))
	if err != nil {
		t.Fatalf("reading seed: %v", err)
	}
	seed := string(seedContent)

	// Inserts for main tables + join table
	insertCount := strings.Count(seed, "INSERT INTO ")
	if insertCount < 4 {
		t.Errorf("seed: expected at least 4 INSERT INTO, got %d", insertCount)
	}

	t.Logf("Migration: %d bytes, Seed: %d bytes", len(mig), len(seed))
}
