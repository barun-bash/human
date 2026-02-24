package repl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/cli"
)

func TestEdit_RequiresProject(t *testing.T) {
	cli.ColorEnabled = false
	r, _, errOut := newTestREPL("/edit add dark mode\n/quit\n")
	r.Run()
	output := errOut.String()

	if !strings.Contains(output, "No project loaded") {
		t.Errorf("expected 'No project loaded' error, got: %s", output)
	}
}

func TestEdit_RequiresLLM(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	cli.ColorEnabled = false

	// Create a temp .human file to load as project.
	tmpFile := filepath.Join(t.TempDir(), "test.human")
	if err := os.WriteFile(tmpFile, []byte("app Test is a web application\n"), 0644); err != nil {
		t.Fatal(err)
	}

	r, _, errOut := newTestREPL("/open " + tmpFile + "\n/edit add dark mode\n/quit\n")
	r.Run()
	output := errOut.String()

	if !strings.Contains(output, "/connect") {
		t.Errorf("expected error pointing to /connect, got: %s", output)
	}
}

func TestEdit_NoArgsPromptsForInstruction(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	cli.ColorEnabled = false

	tmpFile := filepath.Join(t.TempDir(), "test.human")
	if err := os.WriteFile(tmpFile, []byte("app Test is a web application\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// /edit with no args will try to load LLM and fail before the interactive prompt.
	// That's expected — the key test is that it doesn't panic.
	r, _, errOut := newTestREPL("/open " + tmpFile + "\n/edit\n/quit\n")
	r.Run()
	output := errOut.String()

	// Should fail at LLM loading, not panic.
	if !strings.Contains(output, "/connect") {
		t.Errorf("expected LLM error, got: %s", output)
	}
}

func TestEdit_HelpListing(t *testing.T) {
	cli.ColorEnabled = false
	r, out, _ := newTestREPL("/help\n/quit\n")
	r.Run()
	output := out.String()

	if !strings.Contains(output, "/edit") {
		t.Error("expected /help output to list /edit")
	}
	if !strings.Contains(output, "/undo") {
		t.Error("expected /help output to list /undo")
	}
}

func TestEdit_HelpOrder(t *testing.T) {
	cli.ColorEnabled = false
	r, out, _ := newTestREPL("/help\n/quit\n")
	r.Run()
	output := out.String()

	editIdx := strings.Index(output, "/edit")
	undoIdx := strings.Index(output, "/undo")
	buildIdx := strings.Index(output, "/build")

	if editIdx < 0 || undoIdx < 0 || buildIdx < 0 {
		t.Fatal("expected /edit, /undo, and /build in help output")
	}

	// /edit and /undo should appear before /build.
	if editIdx > buildIdx {
		t.Error("expected /edit to appear before /build in help")
	}
	if undoIdx > buildIdx {
		t.Error("expected /undo to appear before /build in help")
	}
	// /edit should appear before /undo.
	if editIdx > undoIdx {
		t.Error("expected /edit to appear before /undo in help")
	}
}

func TestEdit_AliasE(t *testing.T) {
	cli.ColorEnabled = false
	// /e is an alias for /edit; without a project it should show an error.
	r, _, errOut := newTestREPL("/e\n/quit\n")
	r.Run()
	output := errOut.String()

	if !strings.Contains(output, "No project loaded") {
		t.Errorf("expected 'No project loaded' for /e alias, got: %s", output)
	}
}

// ── Undo tests ──

func TestUndo_RequiresProject(t *testing.T) {
	cli.ColorEnabled = false
	r, _, errOut := newTestREPL("/undo\n/quit\n")
	r.Run()
	output := errOut.String()

	if !strings.Contains(output, "No project loaded") {
		t.Errorf("expected 'No project loaded' error, got: %s", output)
	}
}

func TestUndo_NothingToUndo(t *testing.T) {
	cli.ColorEnabled = false

	tmpFile := filepath.Join(t.TempDir(), "test.human")
	if err := os.WriteFile(tmpFile, []byte("app Test is a web application\n"), 0644); err != nil {
		t.Fatal(err)
	}

	r, _, errOut := newTestREPL("/open " + tmpFile + "\n/undo\n/quit\n")
	r.Run()
	output := errOut.String()

	if !strings.Contains(output, "Nothing to undo") {
		t.Errorf("expected 'Nothing to undo' error, got: %s", output)
	}
}

func TestUndo_RestoresBackup(t *testing.T) {
	cli.ColorEnabled = false

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.human")
	origContent := "app Test is a web application\n"

	if err := os.WriteFile(tmpFile, []byte("app Modified is a web application\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a backup manually (simulating what /edit would do).
	backupDirPath := filepath.Join(tmpDir, ".human", "backup")
	if err := os.MkdirAll(backupDirPath, 0755); err != nil {
		t.Fatal(err)
	}
	backupFilePath := filepath.Join(backupDirPath, "test.human.bak")
	if err := os.WriteFile(backupFilePath, []byte(origContent), 0644); err != nil {
		t.Fatal(err)
	}

	// We need /undo to look in the right place. backupPath() uses relative
	// ".human/backup", so we must run from tmpDir.
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	r, out, _ := newTestREPL("/open " + tmpFile + "\n/undo\n/quit\n")
	r.Run()
	output := out.String()

	if !strings.Contains(output, "Reverted") {
		t.Errorf("expected 'Reverted' message, got: %s", output)
	}

	// Verify file was restored.
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != origContent {
		t.Errorf("file content = %q, want %q", string(data), origContent)
	}

	// Verify backup was removed (single-level undo).
	if _, err := os.Stat(backupFilePath); !os.IsNotExist(err) {
		t.Error("backup file should be removed after undo")
	}
}

func TestUndo_IdenticalBackup(t *testing.T) {
	cli.ColorEnabled = false

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.human")
	content := "app Test is a web application\n"

	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a backup with identical content.
	backupDirPath := filepath.Join(tmpDir, ".human", "backup")
	if err := os.MkdirAll(backupDirPath, 0755); err != nil {
		t.Fatal(err)
	}
	backupFilePath := filepath.Join(backupDirPath, "test.human.bak")
	if err := os.WriteFile(backupFilePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	r, out, _ := newTestREPL("/open " + tmpFile + "\n/undo\n/quit\n")
	r.Run()
	output := out.String()

	if !strings.Contains(output, "identical") {
		t.Errorf("expected 'identical' message when backup matches current, got: %s", output)
	}
}

// ── Backup helper tests ──

func TestBackupPath(t *testing.T) {
	got := backupPath("my-app.human")
	want := filepath.Join(".human", "backup", "my-app.human.bak")
	if got != want {
		t.Errorf("backupPath() = %q, want %q", got, want)
	}
}

func TestBackupPath_NestedFile(t *testing.T) {
	got := backupPath("/some/path/project.human")
	want := filepath.Join(".human", "backup", "project.human.bak")
	if got != want {
		t.Errorf("backupPath() = %q, want %q", got, want)
	}
}

func TestBackupFile(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Create a file.
	content := "app MyApp is a web application\n"
	if err := os.WriteFile("my-app.human", []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Backup it.
	backupFile("my-app.human")

	// Verify backup exists.
	bp := backupPath("my-app.human")
	data, err := os.ReadFile(bp)
	if err != nil {
		t.Fatalf("backup not created: %v", err)
	}
	if string(data) != content {
		t.Errorf("backup content = %q, want %q", string(data), content)
	}
}

// ── Diff helper tests ──

func TestDiffSummary_NoChanges(t *testing.T) {
	lines := []string{"a", "b", "c"}
	added, removed := diffSummary(lines, lines)
	if added != 0 || removed != 0 {
		t.Errorf("diffSummary(same, same) = +%d/-%d, want +0/-0", added, removed)
	}
}

func TestDiffSummary_AddedLines(t *testing.T) {
	old := []string{"a", "b"}
	new := []string{"a", "b", "c", "d"}
	added, removed := diffSummary(old, new)
	if added != 2 {
		t.Errorf("added = %d, want 2", added)
	}
	if removed != 0 {
		t.Errorf("removed = %d, want 0", removed)
	}
}

func TestDiffSummary_RemovedLines(t *testing.T) {
	old := []string{"a", "b", "c"}
	new := []string{"a"}
	added, removed := diffSummary(old, new)
	if added != 0 {
		t.Errorf("added = %d, want 0", added)
	}
	if removed != 2 {
		t.Errorf("removed = %d, want 2", removed)
	}
}

func TestDiffSummary_MixedChanges(t *testing.T) {
	old := []string{"a", "b", "c"}
	new := []string{"a", "d", "e"}
	added, removed := diffSummary(old, new)
	if added != 2 {
		t.Errorf("added = %d, want 2", added)
	}
	if removed != 2 {
		t.Errorf("removed = %d, want 2", removed)
	}
}

func TestShowDiff_NoChanges(t *testing.T) {
	cli.ColorEnabled = false
	r, out, _ := newTestREPL("")
	showDiff(r, "same", "same")
	if !strings.Contains(out.String(), "No changes") {
		t.Errorf("expected 'No changes', got: %s", out.String())
	}
}
