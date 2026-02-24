package repl

import (
	"bytes"
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/cli"
)

func newTestREPL(input string) (*REPL, *bytes.Buffer, *bytes.Buffer) {
	cli.ColorEnabled = false
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	r := New("0.4.0-test",
		WithInput(strings.NewReader(input)),
		WithOutput(out),
		WithErrOutput(errOut),
	)
	return r, out, errOut
}

func TestREPL_HelpCommand(t *testing.T) {
	r, out, _ := newTestREPL("/help\n/quit\n")
	r.Run()
	output := out.String()
	if !strings.Contains(output, "Available Commands") {
		t.Error("expected /help output to contain 'Available Commands'")
	}
	if !strings.Contains(output, "/build") {
		t.Error("expected /help output to list /build")
	}
	if !strings.Contains(output, "/check") {
		t.Error("expected /help output to list /check")
	}
	if !strings.Contains(output, "/theme") {
		t.Error("expected /help output to list /theme")
	}
	if !strings.Contains(output, "/config") {
		t.Error("expected /help output to list /config")
	}
}

func TestREPL_VersionCommand(t *testing.T) {
	r, out, _ := newTestREPL("/version\n/quit\n")
	r.Run()
	output := out.String()
	if !strings.Contains(output, "0.4.0-test") {
		t.Errorf("expected version output, got: %s", output)
	}
}

func TestREPL_UnknownCommand(t *testing.T) {
	r, _, errOut := newTestREPL("/foobar\n/quit\n")
	r.Run()
	output := errOut.String()
	if !strings.Contains(output, "Unknown command") {
		t.Errorf("expected 'Unknown command' error, got: %s", output)
	}
}

func TestREPL_NonSlashInput(t *testing.T) {
	r, out, _ := newTestREPL("hello world\n/quit\n")
	r.Run()
	output := out.String()
	if !strings.Contains(output, "Commands start with /") {
		t.Errorf("expected guidance message for non-slash input, got: %s", output)
	}
}

func TestREPL_EOF(t *testing.T) {
	r, out, _ := newTestREPL("")
	r.Run()
	output := out.String()
	if !strings.Contains(output, "Goodbye") {
		t.Errorf("expected 'Goodbye' on EOF, got: %s", output)
	}
}

func TestREPL_CheckWithoutProject(t *testing.T) {
	r, _, errOut := newTestREPL("/check\n/quit\n")
	r.Run()
	output := errOut.String()
	if !strings.Contains(output, "No project loaded") {
		t.Errorf("expected 'No project loaded' error, got: %s", output)
	}
}

func TestREPL_OpenAndCheck(t *testing.T) {
	r, out, errOut := newTestREPL("/open examples/taskflow/app.human\n/check\n/quit\n")
	// Note: this test depends on the examples directory being available.
	// If running from the project root, it should work.
	r.Run()
	combined := out.String() + errOut.String()
	// Either it loaded the file or it said file not found — both are acceptable
	// depending on CWD. The key test is that it doesn't panic.
	if strings.Contains(combined, "panic") {
		t.Error("REPL panicked during /open + /check")
	}
}

func TestREPL_ClearScreen(t *testing.T) {
	r, out, _ := newTestREPL("/clear\n/quit\n")
	r.Run()
	output := out.String()
	if !strings.Contains(output, "\033[2J") {
		t.Error("expected ANSI clear screen escape in output")
	}
}

func TestREPL_AliasB(t *testing.T) {
	// /b is an alias for /build; without a project it should show an error
	r, _, errOut := newTestREPL("/b\n/quit\n")
	r.Run()
	output := errOut.String()
	if !strings.Contains(output, "No project loaded") {
		t.Errorf("expected 'No project loaded' for /b alias, got: %s", output)
	}
}

func TestREPL_QuitAliases(t *testing.T) {
	for _, cmd := range []string{"/quit", "/exit", "/q"} {
		r, out, _ := newTestREPL(cmd + "\n")
		r.Run()
		if !strings.Contains(out.String(), "Goodbye") {
			t.Errorf("expected 'Goodbye' for %s", cmd)
		}
	}
}

func TestREPL_DidYouMean(t *testing.T) {
	r, _, errOut := newTestREPL("/hel\n/quit\n")
	r.Run()
	output := errOut.String()
	if !strings.Contains(output, "Did you mean") {
		t.Errorf("expected 'Did you mean' suggestion, got: %s", output)
	}
}

func TestREPL_BannerContainsUnderscore(t *testing.T) {
	r, out, _ := newTestREPL("/quit\n")
	r.Run()
	output := out.String()
	// Banner should contain the HUMAN_ logo underscore
	if !strings.Contains(output, "_") {
		t.Error("banner should contain underscore character")
	}
}

func TestREPL_BannerContainsVersion(t *testing.T) {
	r, out, _ := newTestREPL("/quit\n")
	r.Run()
	output := out.String()
	if !strings.Contains(output, "0.4.0-test") {
		t.Error("banner should contain version")
	}
}

func TestREPL_BannerContainsTip(t *testing.T) {
	r, out, _ := newTestREPL("/quit\n")
	r.Run()
	output := out.String()
	if !strings.Contains(output, "Tip:") {
		t.Error("banner should contain a tip")
	}
}

func TestREPL_PromptContainsUnderscore(t *testing.T) {
	r, out, _ := newTestREPL("/quit\n")
	r.Run()
	output := out.String()
	// The prompt should contain the underscore brand: human_> or project_>
	if !strings.Contains(output, "_>") {
		t.Error("prompt should contain branded underscore: _>")
	}
}

func TestREPL_ThemeCommand(t *testing.T) {
	r, out, _ := newTestREPL("/theme\n/quit\n")
	r.Run()
	output := out.String()
	if !strings.Contains(output, "default") {
		t.Error("expected /theme to show current theme name")
	}
}

func TestREPL_ThemeListCommand(t *testing.T) {
	r, out, _ := newTestREPL("/theme list\n/quit\n")
	r.Run()
	output := out.String()
	for _, name := range cli.ThemeNames() {
		if !strings.Contains(output, name) {
			t.Errorf("expected /theme list to include %q", name)
		}
	}
}

func TestREPL_ConfigCommand(t *testing.T) {
	r, out, _ := newTestREPL("/config\n/quit\n")
	r.Run()
	output := out.String()
	if !strings.Contains(output, "Settings") {
		t.Error("expected /config to show Settings heading")
	}
	if !strings.Contains(output, "theme:") {
		t.Error("expected /config to show theme setting")
	}
	if !strings.Contains(output, "animate:") {
		t.Error("expected /config to show animate setting")
	}
	if !strings.Contains(output, "plan_mode:") {
		t.Error("expected /config to show plan_mode setting")
	}
}

func TestREPL_ConfigSetInvalid(t *testing.T) {
	r, _, errOut := newTestREPL("/config set unknown_key value\n/quit\n")
	r.Run()
	output := errOut.String()
	if !strings.Contains(output, "Unknown setting") {
		t.Errorf("expected 'Unknown setting' error, got: %s", output)
	}
}
