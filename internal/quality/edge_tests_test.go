package quality

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/ir"
)

func TestGenerateEdgeTests(t *testing.T) {
	app := exampleApp(t)
	dir := t.TempDir()

	files, count, err := generateEdgeTests(app, dir)
	if err != nil {
		t.Fatalf("generateEdgeTests: %v", err)
	}

	if files == 0 {
		t.Fatal("expected edge test files, got 0")
	}
	if count == 0 {
		t.Fatal("expected edge test count > 0")
	}

	// Verify .edge.test.ts files
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading dir: %v", err)
	}

	edgeCount := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".edge.test.ts") {
			edgeCount++
		}
	}
	if edgeCount != files {
		t.Errorf("expected %d .edge.test.ts files, got %d", files, edgeCount)
	}
}

func TestGenerateModelEdgeTests_TextFields(t *testing.T) {
	model := &ir.DataModel{
		Name: "Post",
		Fields: []*ir.DataField{
			{Name: "title", Type: "text", Required: true},
		},
	}
	app := &ir.Application{
		APIs: []*ir.Endpoint{{Name: "CreatePost"}},
	}

	content, count := generateModelEdgeTests(model, app)

	if count == 0 {
		t.Fatal("expected edge tests for text field")
	}
	if !strings.Contains(content, "XSS payload") {
		t.Error("missing XSS test for text field")
	}
	if !strings.Contains(content, "SQL injection") {
		t.Error("missing SQL injection test for text field")
	}
	if !strings.Contains(content, "empty string") {
		t.Error("missing empty string test")
	}
	if !strings.Contains(content, "very long string") {
		t.Error("missing long string test")
	}
}

func TestGenerateModelEdgeTests_EmailField(t *testing.T) {
	model := &ir.DataModel{
		Name: "User",
		Fields: []*ir.DataField{
			{Name: "email", Type: "email"},
		},
	}
	app := &ir.Application{
		APIs: []*ir.Endpoint{{Name: "CreateUser"}},
	}

	content, count := generateModelEdgeTests(model, app)

	if count == 0 {
		t.Fatal("expected edge tests for email field")
	}
	if !strings.Contains(content, "missing @") {
		t.Error("missing @ test for email field")
	}
	if !strings.Contains(content, "missing domain") {
		t.Error("missing domain test for email field")
	}
	if !strings.Contains(content, "double @") {
		t.Error("missing double @ test for email field")
	}
}

func TestGenerateModelEdgeTests_EnumField(t *testing.T) {
	model := &ir.DataModel{
		Name: "Task",
		Fields: []*ir.DataField{
			{Name: "status", Type: "enum", EnumValues: []string{"open", "closed"}},
		},
	}
	app := &ir.Application{
		APIs: []*ir.Endpoint{{Name: "CreateTask"}},
	}

	content, count := generateModelEdgeTests(model, app)

	if count == 0 {
		t.Fatal("expected edge tests for enum field")
	}
	// Should have tests for valid enum values
	if !strings.Contains(content, "accept valid enum status = open") {
		t.Error("missing valid enum 'open' test")
	}
	if !strings.Contains(content, "accept valid enum status = closed") {
		t.Error("missing valid enum 'closed' test")
	}
	// Should have test for invalid enum value
	if !strings.Contains(content, "reject invalid enum status") {
		t.Error("missing invalid enum test")
	}
	if !strings.Contains(content, "INVALID_ENUM_VALUE") {
		t.Error("missing INVALID_ENUM_VALUE in rejection test")
	}
}

func TestGenerateModelEdgeTests_RequiredField(t *testing.T) {
	model := &ir.DataModel{
		Name: "Item",
		Fields: []*ir.DataField{
			{Name: "name", Type: "text", Required: true},
		},
	}
	app := &ir.Application{
		APIs: []*ir.Endpoint{{Name: "CreateItem"}},
	}

	content, count := generateModelEdgeTests(model, app)

	if count == 0 {
		t.Fatal("expected edge tests for required field")
	}
	if !strings.Contains(content, "reject null name") {
		t.Error("missing null test for required field")
	}
	if !strings.Contains(content, "reject undefined name") {
		t.Error("missing undefined test for required field")
	}
	if !strings.Contains(content, "reject missing name") {
		t.Error("missing omitted field test for required field")
	}
}

func TestGenerateModelEdgeTests_DateField(t *testing.T) {
	model := &ir.DataModel{
		Name: "Event",
		Fields: []*ir.DataField{
			{Name: "startDate", Type: "date"},
		},
	}
	app := &ir.Application{
		APIs: []*ir.Endpoint{{Name: "CreateEvent"}},
	}

	content, count := generateModelEdgeTests(model, app)

	if count == 0 {
		t.Fatal("expected edge tests for date field")
	}
	if !strings.Contains(content, "invalid format") {
		t.Error("missing invalid format test for date field")
	}
	if !strings.Contains(content, "epoch zero") {
		t.Error("missing epoch zero test for date field")
	}
}

func TestEdgePayloadsForType(t *testing.T) {
	tests := []struct {
		fieldType string
		minCount  int
	}{
		{"text", 5},
		{"email", 5},
		{"date", 4},
		{"datetime", 4},
		{"number", 4},
		{"decimal", 3},
		{"boolean", 2},
		{"url", 3},
		{"unknown_type", 0},
	}
	for _, tt := range tests {
		payloads := edgePayloadsForType(tt.fieldType)
		if len(payloads) < tt.minCount {
			t.Errorf("edgePayloadsForType(%q) returned %d payloads, want at least %d", tt.fieldType, len(payloads), tt.minCount)
		}
	}
}

func TestGenerateModelEdgeTests_NoEndpoint(t *testing.T) {
	// Model has no matching Create endpoint — generateEdgeTests should skip it
	app := &ir.Application{
		Data: []*ir.DataModel{
			{Name: "OrphanModel", Fields: []*ir.DataField{
				{Name: "name", Type: "text"},
			}},
		},
		APIs: []*ir.Endpoint{{Name: "CreateOther"}},
	}

	dir := t.TempDir()
	files, count, err := generateEdgeTests(app, dir)
	if err != nil {
		t.Fatalf("generateEdgeTests: %v", err)
	}

	if files != 0 {
		t.Errorf("expected 0 edge test files for orphan model, got %d", files)
	}
	if count != 0 {
		t.Errorf("expected 0 edge tests for orphan model, got %d", count)
	}
}

func TestGenerateEdgeTests_OutputPath(t *testing.T) {
	app := &ir.Application{
		Data: []*ir.DataModel{
			{Name: "User", Fields: []*ir.DataField{{Name: "name", Type: "text"}}},
		},
		APIs: []*ir.Endpoint{{Name: "CreateUser"}},
	}
	dir := t.TempDir()

	_, _, err := generateEdgeTests(app, dir)
	if err != nil {
		t.Fatalf("generateEdgeTests: %v", err)
	}

	path := filepath.Join(dir, "user.edge.test.ts")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist", path)
	}
}
