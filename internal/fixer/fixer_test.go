package fixer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAnalyzeNoFiles(t *testing.T) {
	_, err := Analyze(nil)
	if err == nil {
		t.Fatal("expected error for no files")
	}
}

func TestAnalyzeNonexistentFile(t *testing.T) {
	_, err := Analyze([]string{"nonexistent.human"})
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestAnalyzeSimpleFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "app.human")
	content := `app TestApp is a web application

data User:
  has a name which is text
  has an email which is email

page Dashboard:
  show a list of users
  show a greeting with the user's name

build with:
  frontend using React with TypeScript
  backend using Node with Express
  database using PostgreSQL
`
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := Analyze([]string{file})
	if err != nil {
		t.Fatal(err)
	}

	// Should detect W601 (no loading state) and W602 (no empty state).
	hasW601 := false
	hasW602 := false
	for _, w := range result.Warnings {
		if w.Code == "W601" {
			hasW601 = true
		}
		if w.Code == "W602" {
			hasW602 = true
		}
	}
	if !hasW601 {
		t.Error("expected W601 warning for missing loading state")
	}
	if !hasW602 {
		t.Error("expected W602 warning for missing empty state")
	}

	// Should have fixes for these warnings.
	if len(result.Fixes) < 2 {
		t.Errorf("expected at least 2 fixes, got %d", len(result.Fixes))
	}
}

func TestAnalyzeWithLoadingState(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "app.human")
	content := `app TestApp is a web application

data User:
  has a name which is text

page Dashboard:
  show a list of users
  while loading, show a spinner
  if no users match, show "No users found"

build with:
  frontend using React with TypeScript
  backend using Node with Express
  database using PostgreSQL
`
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := Analyze([]string{file})
	if err != nil {
		t.Fatal(err)
	}

	// Should NOT detect W601 or W602.
	for _, w := range result.Warnings {
		if w.Code == "W601" {
			t.Error("should not detect W601 when loading state exists")
		}
		if w.Code == "W602" {
			t.Error("should not detect W602 when empty state exists")
		}
	}
}

func TestApplyFix(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "app.human")
	content := "app Test is a web application\n"
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	fix := Fix{
		File:       file,
		Code:       "W601",
		Suggestion: "  while loading, show a spinner",
		Kind:       "append",
	}

	if err := Apply(fix); err != nil {
		t.Fatal(err)
	}

	// Check backup exists.
	backup := file + ".bak"
	if _, err := os.Stat(backup); os.IsNotExist(err) {
		t.Error("expected backup file")
	}

	// Check fix was applied.
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !contains(got, "while loading, show a spinner") {
		t.Error("expected fix to be applied to file")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsSub(s, sub))
}

func containsSub(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestAnalyzeAPIWithoutAuth(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "app.human")
	content := `app TestApp is a web application

data Post:
  has a title which is text

api CreatePost:
  accepts title
  create a Post with the given fields
  respond with the created post

build with:
  frontend using React with TypeScript
  backend using Node with Express
  database using PostgreSQL
`
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := Analyze([]string{file})
	if err != nil {
		t.Fatal(err)
	}

	hasW604 := false
	for _, w := range result.Warnings {
		if w.Code == "W604" {
			hasW604 = true
		}
	}
	if !hasW604 {
		t.Error("expected W604 warning for API without auth that modifies data")
	}
}
