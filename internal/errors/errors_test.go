package errors

import (
	"strings"
	"testing"
)

// ── CompilerErrors ──

func TestAddAndFilter(t *testing.T) {
	ce := New("app.human")
	ce.AddError("E101", "unknown model")
	ce.AddWarning("W101", "unused model")
	ce.AddErrorWithSuggestion("E102", "unknown page", "Did you mean \"Home\"?")

	if len(ce.All()) != 3 {
		t.Fatalf("expected 3 diagnostics, got %d", len(ce.All()))
	}

	if len(ce.Errors()) != 2 {
		t.Fatalf("expected 2 errors, got %d", len(ce.Errors()))
	}

	if len(ce.Warnings()) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(ce.Warnings()))
	}
}

func TestHasErrorsAndWarnings(t *testing.T) {
	ce := New("app.human")

	if ce.HasErrors() {
		t.Fatal("expected no errors initially")
	}
	if ce.HasWarnings() {
		t.Fatal("expected no warnings initially")
	}

	ce.AddWarning("W101", "something unused")
	if ce.HasErrors() {
		t.Fatal("expected no errors after adding only a warning")
	}
	if !ce.HasWarnings() {
		t.Fatal("expected HasWarnings to be true")
	}

	ce.AddError("E101", "bad reference")
	if !ce.HasErrors() {
		t.Fatal("expected HasErrors to be true")
	}
}

func TestDefaultFile(t *testing.T) {
	ce := New("test.human")
	ce.AddError("E101", "test error")

	errs := ce.Errors()
	if errs[0].File != "test.human" {
		t.Fatalf("expected file 'test.human', got %q", errs[0].File)
	}
}

func TestAddWithExplicitFile(t *testing.T) {
	ce := New("default.human")
	ce.Add(&CompilerError{
		Code:     "E101",
		Message:  "specific file error",
		Severity: SeverityError,
		File:     "other.human",
	})

	errs := ce.Errors()
	if errs[0].File != "other.human" {
		t.Fatalf("expected file 'other.human', got %q", errs[0].File)
	}
}

// ── Format ──

func TestCompilerErrorFormat(t *testing.T) {
	e := &CompilerError{
		Code:    "E101",
		Message: "unknown model \"Userr\"",
		File:    "app.human",
	}
	got := e.Format()
	if !strings.Contains(got, "app.human") {
		t.Errorf("expected file in output, got %q", got)
	}
	if !strings.Contains(got, "[E101]") {
		t.Errorf("expected error code in output, got %q", got)
	}
	if !strings.Contains(got, "unknown model") {
		t.Errorf("expected message in output, got %q", got)
	}
}

func TestCompilerErrorsFormat(t *testing.T) {
	ce := New("app.human")
	ce.AddErrorWithSuggestion("E101", `API "CreateTask" references model "Userr" which does not exist`, `Did you mean "User"?`)
	ce.AddWarning("W201", `Data model "Tag" is defined but never referenced`)

	out := ce.Format()

	if !strings.Contains(out, "✗") {
		t.Error("expected ✗ prefix for errors")
	}
	if !strings.Contains(out, "⚠") {
		t.Error("expected ⚠ prefix for warnings")
	}
	if !strings.Contains(out, "suggestion:") {
		t.Error("expected suggestion line")
	}
	if !strings.Contains(out, `Did you mean "User"?`) {
		t.Error("expected suggestion content")
	}
	if !strings.Contains(out, "[E101]") {
		t.Error("expected error code E101")
	}
	if !strings.Contains(out, "[W201]") {
		t.Error("expected warning code W201")
	}
}

// ── Levenshtein ──

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "xyz", 3},
		{"abc", "abc", 0},
		{"kitten", "sitting", 3},
		{"User", "Userr", 1},
		{"Task", "Taks", 1}, // transposition = 1 op (Damerau-Levenshtein)
		{"a", "b", 1},
	}

	for _, tc := range tests {
		got := levenshtein(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

// ── Similarity ──

func TestSimilarity(t *testing.T) {
	// Identical strings
	if s := Similarity("User", "User"); s != 1.0 {
		t.Errorf("expected 1.0 for identical, got %f", s)
	}

	// Case-insensitive identical
	if s := Similarity("User", "user"); s != 1.0 {
		t.Errorf("expected 1.0 for case-insensitive identical, got %f", s)
	}

	// Both empty
	if s := Similarity("", ""); s != 1.0 {
		t.Errorf("expected 1.0 for both empty, got %f", s)
	}

	// Single char off
	s := Similarity("User", "Userr")
	if s < 0.7 {
		t.Errorf("expected high similarity for 'User'/'Userr', got %f", s)
	}

	// Completely different
	s = Similarity("abc", "xyz")
	if s > 0.1 {
		t.Errorf("expected low similarity for 'abc'/'xyz', got %f", s)
	}
}

// ── FindClosest ──

func TestFindClosest(t *testing.T) {
	candidates := []string{"User", "Task", "Tag", "TaskTag"}

	// Typo: "Userr" → "User"
	got := FindClosest("Userr", candidates, 0.6)
	if got != "User" {
		t.Errorf("FindClosest(Userr) = %q, want \"User\"", got)
	}

	// Typo: "Taks" → "Task"
	got = FindClosest("Taks", candidates, 0.6)
	if got != "Task" {
		t.Errorf("FindClosest(Taks) = %q, want \"Task\"", got)
	}

	// Nothing close
	got = FindClosest("Zzzzzzzzz", candidates, 0.6)
	if got != "" {
		t.Errorf("FindClosest(Zzzzzzzzz) = %q, want empty", got)
	}

	// Exact match
	got = FindClosest("Tag", candidates, 0.6)
	if got != "Tag" {
		t.Errorf("FindClosest(Tag) = %q, want \"Tag\"", got)
	}
}

func TestFindClosestEmpty(t *testing.T) {
	got := FindClosest("anything", nil, 0.6)
	if got != "" {
		t.Errorf("FindClosest on empty candidates = %q, want empty", got)
	}
}
