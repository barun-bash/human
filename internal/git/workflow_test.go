package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// initTestRepo creates a temporary git repo with an initial commit.
// Returns the dir and a cleanup function that restores the original CWD.
func initTestRepo(t *testing.T) (string, func()) {
	t.Helper()
	dir := t.TempDir()

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "checkout", "-b", "main"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("setup %v failed: %s: %v", args, string(out), err)
		}
	}

	// Create initial commit
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test\n"), 0644)
	gitRun(t, dir, "git", "add", ".")
	gitRun(t, dir, "git", "commit", "-m", "initial commit")

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	return dir, func() { os.Chdir(origDir) }
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v failed: %s: %v", args, string(out), err)
	}
}

func TestFeatureCreation(t *testing.T) {
	_, cleanup := initTestRepo(t)
	defer cleanup()

	if err := Feature("user auth"); err != nil {
		t.Fatalf("Feature() failed: %v", err)
	}

	branch, err := CurrentBranch()
	if err != nil {
		t.Fatal(err)
	}
	if branch != "feature/user-auth" {
		t.Errorf("branch = %q, want %q", branch, "feature/user-auth")
	}
}

func TestFeatureEmptyName(t *testing.T) {
	_, cleanup := initTestRepo(t)
	defer cleanup()

	if err := Feature(""); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestFeatureDirtyTree(t *testing.T) {
	dir, cleanup := initTestRepo(t)
	defer cleanup()

	// Create untracked changes
	os.WriteFile(filepath.Join(dir, "dirty.txt"), []byte("dirty"), 0644)
	gitRun(t, dir, "git", "add", "dirty.txt")

	if err := Feature("test"); err == nil {
		t.Fatal("expected error for dirty working tree")
	}
}

func TestFeatureFinishNotOnFeatureBranch(t *testing.T) {
	_, cleanup := initTestRepo(t)
	defer cleanup()

	err := FeatureFinish(false)
	if err == nil {
		t.Fatal("expected error when not on feature branch")
	}
	if !strings.Contains(err.Error(), "not on a feature branch") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFeatureFinishDryRun(t *testing.T) {
	dir, cleanup := initTestRepo(t)
	defer cleanup()

	// Create feature branch
	if err := Feature("test-feature"); err != nil {
		t.Fatalf("Feature() failed: %v", err)
	}

	// Add a commit on the feature branch
	os.WriteFile(filepath.Join(dir, "feature.txt"), []byte("feature"), 0644)
	gitRun(t, dir, "git", "add", ".")
	gitRun(t, dir, "git", "commit", "-m", "feat: add feature")

	err := FeatureFinish(true)
	if err != nil {
		t.Fatalf("dry run failed: %v", err)
	}

	// Should still be on feature branch
	branch, _ := CurrentBranch()
	if branch != "feature/test-feature" {
		t.Errorf("dry run should not change branch, got %q", branch)
	}
}

func TestFeatureFinishMerge(t *testing.T) {
	dir, cleanup := initTestRepo(t)
	defer cleanup()

	if err := Feature("test-merge"); err != nil {
		t.Fatalf("Feature() failed: %v", err)
	}

	os.WriteFile(filepath.Join(dir, "feature.txt"), []byte("data"), 0644)
	gitRun(t, dir, "git", "add", ".")
	gitRun(t, dir, "git", "commit", "-m", "feat: add feature")

	if err := FeatureFinish(false); err != nil {
		t.Fatalf("FeatureFinish failed: %v", err)
	}

	branch, _ := CurrentBranch()
	if branch != "main" {
		t.Errorf("expected to be on main, got %q", branch)
	}

	// Feature branch should be deleted
	out, _ := runOutput("git", "branch")
	if strings.Contains(out, "feature/test-merge") {
		t.Error("feature branch should be deleted after finish")
	}
}

func TestReleaseInvalidVersion(t *testing.T) {
	_, cleanup := initTestRepo(t)
	defer cleanup()

	tests := []string{"abc", "1.2", "v1", "1.2.3.4"}
	for _, v := range tests {
		if err := Release(v, false); err == nil {
			t.Errorf("Release(%q) should fail", v)
		}
	}
}

func TestReleaseValidVersion(t *testing.T) {
	_, cleanup := initTestRepo(t)
	defer cleanup()

	if err := Release("v1.0.0", false); err != nil {
		t.Fatalf("Release failed: %v", err)
	}

	out, err := runOutput("git", "tag")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "v1.0.0") {
		t.Errorf("tag v1.0.0 not found in: %s", out)
	}
}

func TestReleaseWithoutVPrefix(t *testing.T) {
	_, cleanup := initTestRepo(t)
	defer cleanup()

	if err := Release("2.0.0", false); err != nil {
		t.Fatalf("Release failed: %v", err)
	}

	out, _ := runOutput("git", "tag")
	if !strings.Contains(out, "v2.0.0") {
		t.Errorf("tag v2.0.0 not found (auto-prefix): %s", out)
	}
}

func TestReleaseDuplicateTag(t *testing.T) {
	_, cleanup := initTestRepo(t)
	defer cleanup()

	Release("v1.0.0", false)
	if err := Release("v1.0.0", false); err == nil {
		t.Fatal("expected error for duplicate tag")
	}
}

func TestReleaseNotOnDefaultBranch(t *testing.T) {
	_, cleanup := initTestRepo(t)
	defer cleanup()

	if err := Feature("other"); err != nil {
		t.Fatalf("Feature() failed: %v", err)
	}
	if err := Release("v1.0.0", false); err == nil {
		t.Fatal("expected error when not on default branch")
	}
}

func TestReleaseDryRun(t *testing.T) {
	_, cleanup := initTestRepo(t)
	defer cleanup()

	if err := Release("v3.0.0", true); err != nil {
		t.Fatalf("dry run failed: %v", err)
	}

	// Tag should NOT exist
	out, _ := runOutput("git", "tag")
	if strings.Contains(out, "v3.0.0") {
		t.Error("dry run should not create tag")
	}
}

func TestReleaseNotes(t *testing.T) {
	dir, cleanup := initTestRepo(t)
	defer cleanup()

	// Create tag
	gitRun(t, dir, "git", "tag", "-a", "v0.1.0", "-m", "v0.1.0")

	// Add commits
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644)
	gitRun(t, dir, "git", "add", ".")
	gitRun(t, dir, "git", "commit", "-m", "feat: add user auth")

	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0644)
	gitRun(t, dir, "git", "add", ".")
	gitRun(t, dir, "git", "commit", "-m", "fix: resolve login bug")

	os.WriteFile(filepath.Join(dir, "c.txt"), []byte("c"), 0644)
	gitRun(t, dir, "git", "add", ".")
	gitRun(t, dir, "git", "commit", "-m", "docs: update readme")

	notes, err := ReleaseNotes()
	if err != nil {
		t.Fatalf("ReleaseNotes failed: %v", err)
	}

	if !strings.Contains(notes, "Features") {
		t.Error("expected Features section")
	}
	if !strings.Contains(notes, "Bug Fixes") {
		t.Error("expected Bug Fixes section")
	}
	if !strings.Contains(notes, "Documentation") {
		t.Error("expected Documentation section")
	}
	if !strings.Contains(notes, "add user auth") {
		t.Error("expected feat commit message")
	}
}

func TestReleaseNotesNoTags(t *testing.T) {
	_, cleanup := initTestRepo(t)
	defer cleanup()

	notes, err := ReleaseNotes()
	if err != nil {
		t.Fatalf("ReleaseNotes failed: %v", err)
	}
	// Should still produce output (all commits)
	if !strings.Contains(notes, "Changelog") {
		t.Error("expected Changelog header")
	}
}

func TestIsValidVersion(t *testing.T) {
	valid := []string{"v1.0.0", "v0.5.1", "1.2.3", "v10.20.30"}
	for _, v := range valid {
		if !IsValidVersion(v) {
			t.Errorf("IsValidVersion(%q) = false, want true", v)
		}
	}

	invalid := []string{"abc", "1.2", "v1", "1.2.3.4", "va.b.c"}
	for _, v := range invalid {
		if IsValidVersion(v) {
			t.Errorf("IsValidVersion(%q) = true, want false", v)
		}
	}
}

func TestDefaultBranch(t *testing.T) {
	_, cleanup := initTestRepo(t)
	defer cleanup()

	branch, err := DefaultBranch()
	if err != nil {
		t.Fatal(err)
	}
	if branch != "main" {
		t.Errorf("DefaultBranch() = %q, want %q", branch, "main")
	}
}

func TestToKebabCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"user auth", "user-auth"},
		{"UserAuth", "user-auth"},
		{"add login page", "add-login-page"},
		{"fix_bug", "fix-bug"},
		{"  spaces  ", "spaces"},
	}

	for _, tt := range tests {
		got := toKebabCase(tt.input)
		if got != tt.expected {
			t.Errorf("toKebabCase(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestCleanCommitMsg(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"feat: add login", "add login"},
		{"fix(auth): resolve bug", "resolve bug"},
		{"no prefix here", "no prefix here"},
	}

	for _, tt := range tests {
		got := cleanCommitMsg(tt.input)
		if got != tt.expected {
			t.Errorf("cleanCommitMsg(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
