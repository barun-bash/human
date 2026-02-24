package repl

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/cli"
)

func TestInstructions_ShowNoProject(t *testing.T) {
	cli.ColorEnabled = false
	r, out, _ := newTestREPL("")
	cmdInstructions(r, nil)
	output := out.String()

	if !strings.Contains(output, "No project loaded") {
		t.Errorf("expected 'No project loaded', got: %s", output)
	}
}

func TestInstructions_ShowNoHumanMD(t *testing.T) {
	cli.ColorEnabled = false
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.human")
	os.WriteFile(tmpFile, []byte("app Test is a web application\n"), 0644)

	r, out, _ := newTestREPL("")
	r.setProject(tmpFile)
	// Clear the output from setProject (which may print "Loaded project instructions")
	out.Reset()

	cmdInstructions(r, nil)
	output := out.String()

	if !strings.Contains(output, "No HUMAN.md found") {
		t.Errorf("expected 'No HUMAN.md found', got: %s", output)
	}
	if !strings.Contains(output, "/instructions init") {
		t.Errorf("expected hint about /instructions init, got: %s", output)
	}
}

func TestInstructions_ShowWithHumanMD(t *testing.T) {
	cli.ColorEnabled = false
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.human")
	os.WriteFile(tmpFile, []byte("app Test is a web application\n"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "HUMAN.md"), []byte("Always use React and Go.\n"), 0644)

	r, out, _ := newTestREPL("")
	r.setProject(tmpFile)
	out.Reset()

	cmdInstructions(r, nil)
	output := out.String()

	if !strings.Contains(output, "Always use React and Go") {
		t.Errorf("expected instructions content, got: %s", output)
	}
	if !strings.Contains(output, "Project Instructions") {
		t.Errorf("expected heading, got: %s", output)
	}
}

func TestInstructions_LoadedOnSetProject(t *testing.T) {
	cli.ColorEnabled = false
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.human")
	os.WriteFile(tmpFile, []byte("app Test is a web application\n"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "HUMAN.md"), []byte("Use TypeScript.\n"), 0644)

	r, out, _ := newTestREPL("")
	r.setProject(tmpFile)

	if r.instructions != "Use TypeScript." {
		t.Errorf("instructions = %q, want %q", r.instructions, "Use TypeScript.")
	}
	if !strings.Contains(out.String(), "Loaded project instructions from HUMAN.md") {
		t.Errorf("expected load message, got: %s", out.String())
	}
}

func TestInstructions_ClearedOnNewProject(t *testing.T) {
	cli.ColorEnabled = false
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	// Project 1 has HUMAN.md
	tmpFile1 := filepath.Join(tmpDir1, "a.human")
	os.WriteFile(tmpFile1, []byte("app A is a web application\n"), 0644)
	os.WriteFile(filepath.Join(tmpDir1, "HUMAN.md"), []byte("Use Go.\n"), 0644)

	// Project 2 does NOT have HUMAN.md
	tmpFile2 := filepath.Join(tmpDir2, "b.human")
	os.WriteFile(tmpFile2, []byte("app B is a web application\n"), 0644)

	r, _, _ := newTestREPL("")
	r.setProject(tmpFile1)
	if r.instructions == "" {
		t.Fatal("expected instructions from project 1")
	}

	r.setProject(tmpFile2)
	if r.instructions != "" {
		t.Errorf("expected empty instructions after switching project, got: %q", r.instructions)
	}
}

func TestInstructions_InitNoProject(t *testing.T) {
	cli.ColorEnabled = false
	r, _, errOut := newTestREPL("")
	cmdInstructions(r, []string{"init"})
	output := errOut.String()

	if !strings.Contains(output, "No project loaded") {
		t.Errorf("expected 'No project loaded', got: %s", output)
	}
}

func TestInstructions_InitCreatesFile(t *testing.T) {
	cli.ColorEnabled = false
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.human")
	os.WriteFile(tmpFile, []byte("app Test is a web application\n"), 0644)

	r, out, _ := newTestREPL("")
	r.setProject(tmpFile)
	out.Reset()

	cmdInstructions(r, []string{"init"})
	output := out.String()

	if !strings.Contains(output, "Created") {
		t.Errorf("expected 'Created' message, got: %s", output)
	}

	// Verify file was created.
	path := filepath.Join(tmpDir, "HUMAN.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("HUMAN.md not created: %v", err)
	}
	if !strings.Contains(string(data), "Project Instructions") {
		t.Error("template should contain 'Project Instructions'")
	}
	if !strings.Contains(string(data), "Tech Stack") {
		t.Error("template should contain 'Tech Stack' section")
	}

	// Instructions should be loaded.
	if r.instructions == "" {
		t.Error("expected instructions to be loaded after init")
	}
}

func TestInstructions_InitOverwriteDecline(t *testing.T) {
	cli.ColorEnabled = false
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.human")
	os.WriteFile(tmpFile, []byte("app Test is a web application\n"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "HUMAN.md"), []byte("Original content.\n"), 0644)

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	r := New("0.4.0-test",
		WithInput(strings.NewReader("n\n")),
		WithOutput(out),
		WithErrOutput(errOut),
	)
	r.setProject(tmpFile)
	out.Reset()

	cmdInstructions(r, []string{"init"})

	// Should ask about overwrite and user declines.
	if !strings.Contains(out.String(), "already exists") {
		t.Errorf("expected overwrite prompt, got: %s", out.String())
	}

	// Original content should be preserved.
	data, _ := os.ReadFile(filepath.Join(tmpDir, "HUMAN.md"))
	if !strings.Contains(string(data), "Original content") {
		t.Error("original HUMAN.md should be preserved after declining overwrite")
	}
}

func TestInstructions_InitOverwriteAccept(t *testing.T) {
	cli.ColorEnabled = false
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.human")
	os.WriteFile(tmpFile, []byte("app Test is a web application\n"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "HUMAN.md"), []byte("Old content.\n"), 0644)

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	r := New("0.4.0-test",
		WithInput(strings.NewReader("y\n")),
		WithOutput(out),
		WithErrOutput(errOut),
	)
	r.setProject(tmpFile)
	out.Reset()

	cmdInstructions(r, []string{"init"})

	// File should now contain template content.
	data, _ := os.ReadFile(filepath.Join(tmpDir, "HUMAN.md"))
	if strings.Contains(string(data), "Old content") {
		t.Error("HUMAN.md should be overwritten with template")
	}
	if !strings.Contains(string(data), "Project Instructions") {
		t.Error("HUMAN.md should contain template")
	}
}

func TestInstructions_EditNoProject(t *testing.T) {
	cli.ColorEnabled = false
	r, _, errOut := newTestREPL("")
	cmdInstructions(r, []string{"edit"})
	output := errOut.String()

	if !strings.Contains(output, "No project loaded") {
		t.Errorf("expected 'No project loaded', got: %s", output)
	}
}

func TestInstructions_UnknownSubcommand(t *testing.T) {
	cli.ColorEnabled = false
	r, _, errOut := newTestREPL("")
	cmdInstructions(r, []string{"foobar"})
	output := errOut.String()

	if !strings.Contains(output, "Unknown /instructions subcommand") {
		t.Errorf("expected error message, got: %s", output)
	}
}

func TestInstructions_HelpListing(t *testing.T) {
	cli.ColorEnabled = false
	r, out, _ := newTestREPL("/help\n/quit\n")
	r.Run()
	output := out.String()

	if !strings.Contains(output, "/instructions") {
		t.Error("expected /help to list /instructions")
	}
}

func TestInstructions_HelpOrder(t *testing.T) {
	cli.ColorEnabled = false
	r, out, _ := newTestREPL("/help\n/quit\n")
	r.Run()
	output := out.String()

	// Search within the help listing section only.
	helpStart := strings.Index(output, "Available Commands")
	if helpStart < 0 {
		t.Fatal("expected 'Available Commands' heading in output")
	}
	helpSection := output[helpStart:]

	instrIdx := strings.Index(helpSection, "/instructions")
	connectIdx := strings.Index(helpSection, "/connect")
	examplesIdx := strings.Index(helpSection, "/examples")

	if instrIdx < 0 || connectIdx < 0 || examplesIdx < 0 {
		t.Fatal("expected /instructions, /connect, and /examples in help output")
	}

	if instrIdx < examplesIdx {
		t.Error("expected /instructions to appear after /examples in help")
	}
	if instrIdx > connectIdx {
		t.Error("expected /instructions to appear before /connect in help")
	}
}

func TestInstructions_EmptyHumanMD(t *testing.T) {
	cli.ColorEnabled = false
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.human")
	os.WriteFile(tmpFile, []byte("app Test is a web application\n"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "HUMAN.md"), []byte("   \n  \n"), 0644)

	r, _, _ := newTestREPL("")
	r.setProject(tmpFile)

	// Empty/whitespace-only HUMAN.md should not be loaded.
	if r.instructions != "" {
		t.Errorf("expected empty instructions for whitespace-only HUMAN.md, got: %q", r.instructions)
	}
}

func TestInstructions_InstructionsPath(t *testing.T) {
	cli.ColorEnabled = false
	r, _, _ := newTestREPL("")

	// No project — empty path.
	if r.instructionsPath() != "" {
		t.Error("expected empty path with no project")
	}

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.human")
	os.WriteFile(tmpFile, []byte("app Test is a web application\n"), 0644)
	r.setProject(tmpFile)

	expected := filepath.Join(tmpDir, "HUMAN.md")
	if r.instructionsPath() != expected {
		t.Errorf("instructionsPath() = %q, want %q", r.instructionsPath(), expected)
	}
}

func TestInstructions_OpenClearsAndReloads(t *testing.T) {
	cli.ColorEnabled = false
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	// Project 1 with instructions.
	f1 := filepath.Join(tmpDir1, "a.human")
	os.WriteFile(f1, []byte("app A is a web application\n"), 0644)
	os.WriteFile(filepath.Join(tmpDir1, "HUMAN.md"), []byte("Prefer Go.\n"), 0644)

	// Project 2 with different instructions.
	f2 := filepath.Join(tmpDir2, "b.human")
	os.WriteFile(f2, []byte("app B is a web application\n"), 0644)
	os.WriteFile(filepath.Join(tmpDir2, "HUMAN.md"), []byte("Prefer Rust.\n"), 0644)

	r, _, _ := newTestREPL("")

	// Load project 1.
	cmdOpen(r, []string{f1})
	if r.instructions != "Prefer Go." {
		t.Errorf("after /open a: instructions = %q, want %q", r.instructions, "Prefer Go.")
	}

	// Load project 2.
	cmdOpen(r, []string{f2})
	if r.instructions != "Prefer Rust." {
		t.Errorf("after /open b: instructions = %q, want %q", r.instructions, "Prefer Rust.")
	}
}

func TestInstructions_TemplateContent(t *testing.T) {
	// Verify the template has the expected sections.
	sections := []string{
		"Project Description",
		"Tech Stack",
		"Design System",
		"Coding Conventions",
		"Deployment Target",
	}
	for _, section := range sections {
		if !strings.Contains(humanMDTemplate, section) {
			t.Errorf("template missing section: %q", section)
		}
	}
}
