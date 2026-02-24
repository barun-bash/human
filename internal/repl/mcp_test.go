package repl

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/cli"
	"github.com/barun-bash/human/internal/config"
)

func TestMCP_ListEmpty(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	cli.ColorEnabled = false

	r, out, _ := newTestREPL("/mcp\n/quit\n")
	r.Run()
	output := out.String()

	if !strings.Contains(output, "No servers configured") {
		t.Errorf("expected 'No servers configured' in output, got: %s", output)
	}
	if !strings.Contains(output, "/mcp add") {
		t.Errorf("expected '/mcp add' hint in output, got: %s", output)
	}
}

func TestMCP_ListWithConfig(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	cli.ColorEnabled = false

	gc := &config.GlobalConfig{
		MCP: []*config.MCPServerConfig{
			{Name: "figma", Command: "npx", Args: []string{"-y", "figma-mcp"}},
		},
	}
	if err := config.SaveGlobalConfig(gc); err != nil {
		t.Fatal(err)
	}

	r, out, _ := newTestREPL("")
	cmdMCP(r, nil)
	output := out.String()

	if !strings.Contains(output, "figma") {
		t.Errorf("expected 'figma' in list output, got: %s", output)
	}
	if !strings.Contains(output, "not connected") {
		t.Errorf("expected 'not connected' status, got: %s", output)
	}
}

func TestMCP_ListSubcommand(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	cli.ColorEnabled = false

	r, out, _ := newTestREPL("")
	cmdMCP(r, []string{"list"})
	output := out.String()

	if !strings.Contains(output, "No servers configured") {
		t.Errorf("expected empty list, got: %s", output)
	}
}

func TestMCP_AddNoArgs(t *testing.T) {
	cli.ColorEnabled = false

	r, _, errOut := newTestREPL("")
	cmdMCP(r, []string{"add"})
	output := errOut.String()

	if !strings.Contains(output, "Usage") {
		t.Errorf("expected usage hint, got: %s", output)
	}
	if !strings.Contains(output, "Known servers") {
		t.Errorf("expected known servers list, got: %s", output)
	}
}

func TestMCP_AddDuplicate(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	cli.ColorEnabled = false

	gc := &config.GlobalConfig{
		MCP: []*config.MCPServerConfig{
			{Name: "figma", Command: "npx"},
		},
	}
	config.SaveGlobalConfig(gc)

	r, _, errOut := newTestREPL("")
	cmdMCP(r, []string{"add", "figma"})
	output := errOut.String()

	if !strings.Contains(output, "already configured") {
		t.Errorf("expected 'already configured' error, got: %s", output)
	}
}

func TestMCP_AddUnknownNoCommand(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	cli.ColorEnabled = false

	r, _, errOut := newTestREPL("")
	cmdMCP(r, []string{"add", "myserver"})
	output := errOut.String()

	if !strings.Contains(output, "Unknown server") {
		t.Errorf("expected 'Unknown server' error, got: %s", output)
	}
	if !strings.Contains(output, "Known servers") {
		t.Errorf("expected known servers hint, got: %s", output)
	}
}

func TestMCP_RemoveNoArgs(t *testing.T) {
	cli.ColorEnabled = false

	r, _, errOut := newTestREPL("")
	cmdMCP(r, []string{"remove"})
	output := errOut.String()

	if !strings.Contains(output, "Usage") {
		t.Errorf("expected usage hint, got: %s", output)
	}
}

func TestMCP_RemoveNotConfigured(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	cli.ColorEnabled = false

	r, _, errOut := newTestREPL("")
	cmdMCP(r, []string{"remove", "nonexistent"})
	output := errOut.String()

	if !strings.Contains(output, "not configured") {
		t.Errorf("expected 'not configured' error, got: %s", output)
	}
}

func TestMCP_RemoveConfigured(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	cli.ColorEnabled = false

	gc := &config.GlobalConfig{
		MCP: []*config.MCPServerConfig{
			{Name: "figma", Command: "npx"},
			{Name: "github", Command: "npx"},
		},
	}
	config.SaveGlobalConfig(gc)

	r, out, _ := newTestREPL("")
	cmdMCP(r, []string{"remove", "figma"})
	output := out.String()

	if !strings.Contains(output, "Removed figma") {
		t.Errorf("expected 'Removed figma', got: %s", output)
	}

	// Verify only github remains.
	loaded, err := config.LoadGlobalConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.MCP) != 1 {
		t.Fatalf("expected 1 MCP server after remove, got %d", len(loaded.MCP))
	}
	if loaded.MCP[0].Name != "github" {
		t.Errorf("expected github to remain, got %s", loaded.MCP[0].Name)
	}
}

func TestMCP_RemovePreservesLLM(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	cli.ColorEnabled = false

	gc := &config.GlobalConfig{
		LLM: &config.GlobalLLMConfig{
			Provider: "anthropic",
			Model:    "claude-sonnet-4-20250514",
			APIKey:   "sk-test-key",
		},
		MCP: []*config.MCPServerConfig{
			{Name: "figma", Command: "npx"},
		},
	}
	config.SaveGlobalConfig(gc)

	r, _, _ := newTestREPL("")
	cmdMCP(r, []string{"remove", "figma"})

	loaded, err := config.LoadGlobalConfig()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.LLM == nil {
		t.Fatal("expected LLM config to be preserved after MCP remove")
	}
	if loaded.LLM.Provider != "anthropic" {
		t.Errorf("LLM provider = %q, want anthropic", loaded.LLM.Provider)
	}
}

func TestMCP_StatusEmpty(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	cli.ColorEnabled = false

	r, out, _ := newTestREPL("")
	cmdMCP(r, []string{"status"})
	output := out.String()

	if !strings.Contains(output, "No MCP servers configured") {
		t.Errorf("expected 'No MCP servers configured', got: %s", output)
	}
}

func TestMCP_StatusWithConfig(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	cli.ColorEnabled = false

	gc := &config.GlobalConfig{
		MCP: []*config.MCPServerConfig{
			{Name: "figma", Command: "npx", Args: []string{"-y", "figma-mcp"}},
		},
	}
	config.SaveGlobalConfig(gc)

	r, out, _ := newTestREPL("")
	cmdMCP(r, []string{"status"})
	output := out.String()

	if !strings.Contains(output, "figma") {
		t.Errorf("expected 'figma' in status, got: %s", output)
	}
	if !strings.Contains(output, "not connected") {
		t.Errorf("expected 'not connected', got: %s", output)
	}
	if !strings.Contains(output, "npx") {
		t.Errorf("expected command in status, got: %s", output)
	}
}

func TestMCP_UnknownSubcommand(t *testing.T) {
	cli.ColorEnabled = false

	r, _, errOut := newTestREPL("")
	cmdMCP(r, []string{"foobar"})
	output := errOut.String()

	if !strings.Contains(output, "Unknown /mcp subcommand") {
		t.Errorf("expected 'Unknown /mcp subcommand', got: %s", output)
	}
}

func TestMCP_HelpListing(t *testing.T) {
	cli.ColorEnabled = false

	r, out, _ := newTestREPL("/help\n/quit\n")
	r.Run()
	output := out.String()

	if !strings.Contains(output, "/mcp") {
		t.Error("expected /help to list /mcp")
	}
}

func TestMCP_HelpOrder(t *testing.T) {
	cli.ColorEnabled = false

	r, out, _ := newTestREPL("/help\n/quit\n")
	r.Run()
	output := out.String()

	// Search within the help listing section only (after "Available Commands")
	// to avoid false matches in the banner/tips area.
	helpStart := strings.Index(output, "Available Commands")
	if helpStart < 0 {
		t.Fatal("expected 'Available Commands' heading in output")
	}
	helpSection := output[helpStart:]

	mcpIdx := strings.Index(helpSection, "/mcp")
	connectIdx := strings.Index(helpSection, "/connect")
	themeIdx := strings.Index(helpSection, "/theme")

	if mcpIdx < 0 || connectIdx < 0 || themeIdx < 0 {
		t.Fatal("expected /mcp, /connect, and /theme in help output")
	}

	if mcpIdx < connectIdx {
		t.Error("expected /mcp to appear after /connect in help")
	}
	if mcpIdx > themeIdx {
		t.Error("expected /mcp to appear before /theme in help")
	}
}

func TestMCP_KnownServerNames(t *testing.T) {
	names := knownServerNames()
	if len(names) < 2 {
		t.Fatalf("expected at least 2 known servers, got %d", len(names))
	}

	// Should be sorted.
	for i := 1; i < len(names); i++ {
		if names[i] < names[i-1] {
			t.Errorf("known server names not sorted: %v", names)
			break
		}
	}

	// Should include figma and github.
	found := map[string]bool{}
	for _, n := range names {
		found[n] = true
	}
	if !found["figma"] {
		t.Error("expected 'figma' in known servers")
	}
	if !found["github"] {
		t.Error("expected 'github' in known servers")
	}
}

func TestMCP_BannerStatusEmpty(t *testing.T) {
	cli.ColorEnabled = false
	r, _, _ := newTestREPL("")
	status := r.mcpBannerStatus()
	if status != "" {
		t.Errorf("expected empty banner status, got: %q", status)
	}
}

func TestMCP_BannerShows(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	cli.ColorEnabled = false

	r, out, _ := newTestREPL("/quit\n")
	r.Run()
	output := out.String()

	// With no MCP clients, banner should show "0 servers connected"
	if !strings.Contains(output, "0 servers connected") {
		t.Errorf("expected '0 servers connected' in banner, got: %s", output)
	}
}

func TestMCP_AddKnownEmptyInput(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	cli.ColorEnabled = false

	// Simulate /mcp add figma with empty token input.
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	r := New("0.4.0-test",
		WithInput(strings.NewReader("\n")),
		WithOutput(out),
		WithErrOutput(errOut),
	)

	cmdMCP(r, []string{"add", "figma"})
	output := errOut.String()

	if !strings.Contains(output, "No value provided") {
		t.Errorf("expected 'No value provided' for empty env var, got: %s", output)
	}
}

func TestMCP_CloseMCPClients(t *testing.T) {
	cli.ColorEnabled = false
	r, _, _ := newTestREPL("")

	// closeMCPClients with empty map should not panic.
	r.closeMCPClients()

	if len(r.mcpClients) != 0 {
		t.Error("expected empty mcpClients after close")
	}
}

func TestMCP_AddKnownEnvPrompt(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	cli.ColorEnabled = false

	// The add command should prompt for FIGMA_ACCESS_TOKEN.
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	r := New("0.4.0-test",
		WithInput(strings.NewReader("my-token-value\n")),
		WithOutput(out),
		WithErrOutput(errOut),
	)

	// This will fail at the Connect step (no real npx/server), but we can
	// verify it prompted for the env var.
	cmdMCP(r, []string{"add", "figma"})
	outStr := out.String()

	if !strings.Contains(outStr, "FIGMA_ACCESS_TOKEN") {
		t.Errorf("expected prompt for FIGMA_ACCESS_TOKEN, got: %s", outStr)
	}
	// Should show "Connecting..." before failing.
	if !strings.Contains(outStr, "Connecting") {
		t.Errorf("expected 'Connecting...' message, got: %s", outStr)
	}
	// Connection should fail (no npx/server available in test).
	errStr := errOut.String()
	if !strings.Contains(errStr, "Connection failed") && !strings.Contains(errStr, "not added") {
		// On CI or machines without npx, we get "Connection failed".
		// Either way the server should NOT be saved.
		loaded, _ := config.LoadGlobalConfig()
		if len(loaded.MCP) > 0 {
			t.Error("failed connection should not save MCP config")
		}
	}
}

func TestMCP_GlobalConfigFilePermissions(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	gc := &config.GlobalConfig{
		MCP: []*config.MCPServerConfig{
			{
				Name:    "test",
				Command: "echo",
				Env:     map[string]string{"SECRET": "value"},
			},
		},
	}
	config.SaveGlobalConfig(gc)

	path := tmpHome + "/.human/config.json"
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	// Should be 0600 (owner read/write only) since it may contain secrets.
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("global config permissions = %o, want 0600", perm)
	}
}
