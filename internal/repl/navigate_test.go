package repl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/cli"
)

func TestPwd(t *testing.T) {
	cli.ColorEnabled = false
	r, out, _ := newTestREPL("/pwd\n/quit\n")
	r.Run()

	cwd, _ := os.Getwd()
	if !strings.Contains(out.String(), cwd) {
		t.Errorf("expected %q in output, got: %s", cwd, out.String())
	}
}

func TestCd_ToDirectory(t *testing.T) {
	cli.ColorEnabled = false
	tmpDir := t.TempDir()

	// Save and restore CWD.
	old, _ := os.Getwd()
	defer os.Chdir(old)

	r, out, errOut := newTestREPL("/cd " + tmpDir + "\n/quit\n")
	r.Run()

	if errOut.Len() > 0 {
		t.Errorf("unexpected error: %s", errOut.String())
	}
	if !strings.Contains(out.String(), tmpDir) {
		t.Errorf("expected %q in output, got: %s", tmpDir, out.String())
	}

	// Verify CWD actually changed (resolve symlinks for macOS /private/var).
	cwd, _ := os.Getwd()
	cwdReal, _ := filepath.EvalSymlinks(cwd)
	tmpReal, _ := filepath.EvalSymlinks(tmpDir)
	if cwdReal != tmpReal {
		t.Errorf("CWD = %q, want %q", cwdReal, tmpReal)
	}
}

func TestCd_NoArgs_GoesHome(t *testing.T) {
	cli.ColorEnabled = false
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home dir")
	}

	old, _ := os.Getwd()
	defer os.Chdir(old)

	r, out, _ := newTestREPL("/cd\n/quit\n")
	r.Run()

	if !strings.Contains(out.String(), home) {
		t.Errorf("expected home dir %q in output, got: %s", home, out.String())
	}
}

func TestCd_NonExistent(t *testing.T) {
	cli.ColorEnabled = false
	old, _ := os.Getwd()
	defer os.Chdir(old)

	r, _, errOut := newTestREPL("/cd /nonexistent/path/xyz\n/quit\n")
	r.Run()

	if !strings.Contains(errOut.String(), "No such directory") {
		t.Errorf("expected 'No such directory' error, got: %s", errOut.String())
	}
}

func TestCd_NotADirectory(t *testing.T) {
	cli.ColorEnabled = false
	old, _ := os.Getwd()
	defer os.Chdir(old)

	tmpFile := filepath.Join(t.TempDir(), "file.txt")
	os.WriteFile(tmpFile, []byte("hello"), 0644)

	r, _, errOut := newTestREPL("/cd " + tmpFile + "\n/quit\n")
	r.Run()

	if !strings.Contains(errOut.String(), "Not a directory") {
		t.Errorf("expected 'Not a directory' error, got: %s", errOut.String())
	}
}

func TestCd_AutoDetectsProject(t *testing.T) {
	cli.ColorEnabled = false
	old, _ := os.Getwd()
	defer os.Chdir(old)

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "myapp.human"), []byte("app MyApp is a web application\n"), 0644)

	r, out, _ := newTestREPL("/cd " + tmpDir + "\n/status\n/quit\n")
	r.Run()

	if !strings.Contains(out.String(), "myapp") {
		t.Errorf("expected project 'myapp' auto-detected, got: %s", out.String())
	}
}

func TestCd_ClearsOldProject(t *testing.T) {
	cli.ColorEnabled = false
	old, _ := os.Getwd()
	defer os.Chdir(old)

	// Start with a project loaded.
	tmpDir1 := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir1, "proj1.human"), []byte("app Proj1 is a web application\n"), 0644)

	// Second dir has no .human files.
	tmpDir2 := t.TempDir()

	r, out, _ := newTestREPL("/cd " + tmpDir1 + "\n/cd " + tmpDir2 + "\n/status\n/quit\n")
	r.Run()

	if !strings.Contains(out.String(), "No project loaded") {
		t.Errorf("expected 'No project loaded' after cd to dir without .human files, got: %s", out.String())
	}
}

func TestCompleteDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	os.Mkdir(filepath.Join(tmpDir, "subdir1"), 0755)
	os.Mkdir(filepath.Join(tmpDir, "subdir2"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("text"), 0644)

	old, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(old)

	results := completeDirectories("")
	found := map[string]bool{}
	for _, r := range results {
		found[r] = true
	}

	if !found["subdir1/"] {
		t.Errorf("expected subdir1/ in results, got %v", results)
	}
	if !found["subdir2/"] {
		t.Errorf("expected subdir2/ in results, got %v", results)
	}
	if found["file.txt"] {
		t.Error("should not include files in directory completion")
	}
}

func TestCompleteDirectories_WithPrefix(t *testing.T) {
	tmpDir := t.TempDir()
	os.Mkdir(filepath.Join(tmpDir, "src"), 0755)
	os.Mkdir(filepath.Join(tmpDir, "scripts"), 0755)
	os.Mkdir(filepath.Join(tmpDir, "build"), 0755)

	old, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(old)

	results := completeDirectories("s")
	if len(results) != 2 {
		t.Errorf("expected 2 matches for 's', got %v", results)
	}
	for _, r := range results {
		if !strings.HasPrefix(filepath.Base(strings.TrimSuffix(r, "/")), "s") {
			t.Errorf("result %q doesn't start with 's'", r)
		}
	}
}

func TestCd_HelpOrder(t *testing.T) {
	cli.ColorEnabled = false
	r, out, _ := newTestREPL("/help\n/quit\n")
	r.Run()
	output := out.String()

	helpStart := strings.Index(output, "Available Commands")
	if helpStart < 0 {
		t.Fatal("expected 'Available Commands' heading")
	}
	helpSection := output[helpStart:]

	cdIdx := strings.Index(helpSection, "/cd")
	pwdIdx := strings.Index(helpSection, "/pwd")
	clearIdx := strings.Index(helpSection, "/clear")

	if cdIdx < 0 || pwdIdx < 0 {
		t.Fatal("expected /cd and /pwd in help")
	}
	if cdIdx > pwdIdx {
		t.Error("expected /cd before /pwd")
	}
	if pwdIdx > clearIdx {
		t.Error("expected /pwd before /clear")
	}
}
