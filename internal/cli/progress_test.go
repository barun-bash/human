package cli

import (
	"bytes"
	"errors"
	"testing"
)

func TestWithStepsSuccess(t *testing.T) {
	var buf bytes.Buffer
	steps := []Step{
		{Name: "step1", Fn: func() error { return nil }},
		{Name: "step2", Fn: func() error { return nil }},
	}
	err := WithSteps(&buf, "Test", steps)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestWithStepsError(t *testing.T) {
	var buf bytes.Buffer
	want := errors.New("step2 failed")
	steps := []Step{
		{Name: "step1", Fn: func() error { return nil }},
		{Name: "step2", Fn: func() error { return want }},
		{Name: "step3", Fn: func() error { return nil }},
	}
	err := WithSteps(&buf, "Test", steps)
	if err != want {
		t.Fatalf("expected %v, got %v", want, err)
	}
}

func TestProgressBoxFailStage(t *testing.T) {
	var buf bytes.Buffer
	box := NewProgressBox(&buf, "Test", []string{"a", "b", "c"})
	// Non-TTY: FailStage should set the failed index.
	box.Update("a")
	box.Update("b")
	box.FailStage("b")
	if box.failed != 1 {
		t.Fatalf("expected failed=1, got %d", box.failed)
	}
}

func TestProgressBoxFailedInitValue(t *testing.T) {
	var buf bytes.Buffer
	box := NewProgressBox(&buf, "Test", []string{"a"})
	if box.failed != -1 {
		t.Fatalf("expected failed=-1, got %d", box.failed)
	}
}
