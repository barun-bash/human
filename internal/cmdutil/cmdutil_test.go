package cmdutil

import (
	"path/filepath"
	"runtime"
	"testing"
)

// projectRoot returns the path to the project root.
func projectRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..")
}

func TestParseAndAnalyze_ValidFile(t *testing.T) {
	file := filepath.Join(projectRoot(), "examples", "taskflow", "app.human")
	result, err := ParseAndAnalyze(file)
	if err != nil {
		t.Fatalf("ParseAndAnalyze failed: %v", err)
	}
	if result.Prog == nil {
		t.Fatal("expected non-nil Program")
	}
	if result.App == nil {
		t.Fatal("expected non-nil Application")
	}
	if result.Errs == nil {
		t.Fatal("expected non-nil CompilerErrors")
	}
	if result.App.Name != "TaskFlow" {
		t.Errorf("expected app name TaskFlow, got %s", result.App.Name)
	}
}

func TestParseAndAnalyze_MissingFile(t *testing.T) {
	_, err := ParseAndAnalyze("nonexistent.human")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestCheckSummary(t *testing.T) {
	file := filepath.Join(projectRoot(), "examples", "taskflow", "app.human")
	result, err := ParseAndAnalyze(file)
	if err != nil {
		t.Fatalf("ParseAndAnalyze failed: %v", err)
	}
	summary := CheckSummary(result.Prog, file)
	if summary == "" {
		t.Fatal("expected non-empty summary")
	}
	// Should contain "is valid"
	if !contains(summary, "is valid") {
		t.Errorf("expected summary to contain 'is valid', got: %s", summary)
	}
}

func TestPlural(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "s"},
		{1, ""},
		{2, "s"},
		{100, "s"},
	}
	for _, tt := range tests {
		got := Plural(tt.n)
		if got != tt.want {
			t.Errorf("Plural(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func TestPluralY(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "ies"},
		{1, "y"},
		{2, "ies"},
		{100, "ies"},
	}
	for _, tt := range tests {
		got := PluralY(tt.n)
		if got != tt.want {
			t.Errorf("PluralY(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
