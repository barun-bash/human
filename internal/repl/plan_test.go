package repl

import (
	"bytes"
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/cli"
)

func TestRenderPlanBox(t *testing.T) {
	cli.ColorEnabled = false
	var buf bytes.Buffer

	plan := &Plan{
		Title: "Build Plan",
		Steps: []string{
			"Parse source file",
			"Generate IR",
			"Run code generators",
		},
		Editable: false,
	}

	renderPlanBox(&buf, plan)
	output := buf.String()

	if !strings.Contains(output, "Build Plan") {
		t.Error("plan box should contain title")
	}
	if !strings.Contains(output, "1. Parse source file") {
		t.Error("plan box should contain step 1")
	}
	if !strings.Contains(output, "2. Generate IR") {
		t.Error("plan box should contain step 2")
	}
	if !strings.Contains(output, "3. Run code generators") {
		t.Error("plan box should contain step 3")
	}
	// Box borders
	if !strings.Contains(output, "\u250c") {
		t.Error("plan box should contain top-left corner")
	}
	if !strings.Contains(output, "\u2514") {
		t.Error("plan box should contain bottom-left corner")
	}
}

func TestShowPlanGo(t *testing.T) {
	cli.ColorEnabled = false
	var buf bytes.Buffer
	in := strings.NewReader("g\n")

	plan := &Plan{
		Title:    "Test Plan",
		Steps:    []string{"Step 1"},
		Editable: false,
	}

	action := ShowPlan(&buf, in, plan)
	if action != PlanGo {
		t.Errorf("expected PlanGo, got %d", action)
	}
}

func TestShowPlanCancel(t *testing.T) {
	cli.ColorEnabled = false
	var buf bytes.Buffer
	in := strings.NewReader("c\n")

	plan := &Plan{
		Title:    "Test Plan",
		Steps:    []string{"Step 1"},
		Editable: false,
	}

	action := ShowPlan(&buf, in, plan)
	if action != PlanCancel {
		t.Errorf("expected PlanCancel, got %d", action)
	}
}

func TestShowPlanEdit(t *testing.T) {
	cli.ColorEnabled = false
	var buf bytes.Buffer
	in := strings.NewReader("e\n")

	plan := &Plan{
		Title:    "Test Plan",
		Steps:    []string{"Step 1"},
		Editable: true,
	}

	action := ShowPlan(&buf, in, plan)
	if action != PlanEdit {
		t.Errorf("expected PlanEdit, got %d", action)
	}
}

func TestShowPlanEditNonEditable(t *testing.T) {
	cli.ColorEnabled = false
	var buf bytes.Buffer
	in := strings.NewReader("e\n")

	plan := &Plan{
		Title:    "Test Plan",
		Steps:    []string{"Step 1"},
		Editable: false,
	}

	// Edit on non-editable plan should cancel.
	action := ShowPlan(&buf, in, plan)
	if action != PlanCancel {
		t.Errorf("expected PlanCancel for edit on non-editable, got %d", action)
	}
}

func TestShowPlanEOF(t *testing.T) {
	cli.ColorEnabled = false
	var buf bytes.Buffer
	in := strings.NewReader("") // EOF

	plan := &Plan{
		Title:    "Test Plan",
		Steps:    []string{"Step 1"},
		Editable: false,
	}

	action := ShowPlan(&buf, in, plan)
	if action != PlanCancel {
		t.Errorf("expected PlanCancel on EOF, got %d", action)
	}
}

func TestShowPlanFullWord(t *testing.T) {
	cli.ColorEnabled = false
	var buf bytes.Buffer
	in := strings.NewReader("go\n")

	plan := &Plan{
		Title:    "Test Plan",
		Steps:    []string{"Step 1"},
		Editable: false,
	}

	action := ShowPlan(&buf, in, plan)
	if action != PlanGo {
		t.Errorf("expected PlanGo for 'go', got %d", action)
	}
}

func TestShowPlanEditableShowsEditOption(t *testing.T) {
	cli.ColorEnabled = false
	var buf bytes.Buffer
	in := strings.NewReader("c\n")

	plan := &Plan{
		Title:    "Test Plan",
		Steps:    []string{"Step 1"},
		Editable: true,
	}

	ShowPlan(&buf, in, plan)
	output := buf.String()
	if !strings.Contains(output, "[e]dit") {
		t.Error("editable plan should show edit option")
	}
}

func TestShowPlanNonEditableHidesEditOption(t *testing.T) {
	cli.ColorEnabled = false
	var buf bytes.Buffer
	in := strings.NewReader("c\n")

	plan := &Plan{
		Title:    "Test Plan",
		Steps:    []string{"Step 1"},
		Editable: false,
	}

	ShowPlan(&buf, in, plan)
	output := buf.String()
	if strings.Contains(output, "[e]dit plan") {
		t.Error("non-editable plan should not show edit option")
	}
}

func TestParseSteps(t *testing.T) {
	input := `# Plan Title

# This is a comment

1. First step
2. Second step
3. Third step
`
	steps := parseSteps(input)
	if len(steps) != 3 {
		t.Fatalf("expected 3 steps, got %d: %v", len(steps), steps)
	}
	if steps[0] != "First step" {
		t.Errorf("step 0 = %q, want %q", steps[0], "First step")
	}
	if steps[1] != "Second step" {
		t.Errorf("step 1 = %q, want %q", steps[1], "Second step")
	}
	if steps[2] != "Third step" {
		t.Errorf("step 2 = %q, want %q", steps[2], "Third step")
	}
}

func TestParseStepsNonNumbered(t *testing.T) {
	input := `Add a login page
Add a signup page`
	steps := parseSteps(input)
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d: %v", len(steps), steps)
	}
	if steps[0] != "Add a login page" {
		t.Errorf("step 0 = %q", steps[0])
	}
}

func TestStripNumberPrefix(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"1. Hello", "Hello"},
		{"10. World", "World"},
		{"Hello", "Hello"},
		{"123. Test", "Test"},
	}
	for _, tt := range tests {
		got := stripNumberPrefix(tt.in)
		if got != tt.want {
			t.Errorf("stripNumberPrefix(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestNumWidth(t *testing.T) {
	if numWidth(1) != 1 {
		t.Error("numWidth(1) should be 1")
	}
	if numWidth(9) != 1 {
		t.Error("numWidth(9) should be 1")
	}
	if numWidth(10) != 2 {
		t.Error("numWidth(10) should be 2")
	}
	if numWidth(99) != 2 {
		t.Error("numWidth(99) should be 2")
	}
	if numWidth(100) != 3 {
		t.Error("numWidth(100) should be 3")
	}
}
