package repl

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/barun-bash/human/internal/cli"
	"github.com/barun-bash/human/internal/config"
	"github.com/barun-bash/human/internal/mcp"
)

// knownServer is a pre-configured MCP server that users can add by name.
type knownServer struct {
	DisplayName string
	Command     string
	Args        []string
	EnvKeys     []string // required env vars the user must provide
}

// knownServers is the registry of well-known MCP servers.
var knownServers = map[string]knownServer{
	"figma": {
		DisplayName: "Figma",
		Command:     "npx",
		Args:        []string{"-y", "@anthropic/figma-mcp-server@latest"},
		EnvKeys:     []string{"FIGMA_ACCESS_TOKEN"},
	},
	"github": {
		DisplayName: "GitHub",
		Command:     "npx",
		Args:        []string{"-y", "@anthropic/github-mcp-server@latest"},
		EnvKeys:     []string{"GITHUB_TOKEN"},
	},
}

// cmdMCP dispatches the /mcp subcommand.
func cmdMCP(r *REPL, args []string) {
	sub := ""
	if len(args) > 0 {
		sub = strings.ToLower(args[0])
	}

	switch sub {
	case "":
		mcpList(r)
	case "list":
		mcpList(r)
	case "add":
		mcpAdd(r, args[1:])
	case "remove":
		mcpRemove(r, args[1:])
	case "status":
		mcpStatus(r)
	default:
		fmt.Fprintf(r.errOut, "Unknown /mcp subcommand: %s\n", sub)
		fmt.Fprintln(r.errOut, "Usage: /mcp [list|add|remove|status]")
	}
}

// mcpList shows configured MCP servers and their connection state.
func mcpList(r *REPL) {
	gc, err := config.LoadGlobalConfig()
	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Could not load config: %v", err)))
		return
	}

	fmt.Fprintln(r.out)
	fmt.Fprintln(r.out, cli.Heading("MCP Servers"))
	fmt.Fprintln(r.out, strings.Repeat("\u2500", 40))

	if len(gc.MCP) == 0 {
		fmt.Fprintln(r.out, "  No servers configured.")
		fmt.Fprintln(r.out)
		fmt.Fprintln(r.out, cli.Muted("  Run /mcp add <name> to add a server."))
		fmt.Fprintf(r.out, "  %s %s\n", cli.Muted("Known servers:"), strings.Join(knownServerNames(), ", "))
		fmt.Fprintln(r.out)
		return
	}

	for _, srv := range gc.MCP {
		status := "not connected"
		if client, ok := r.mcpClients[srv.Name]; ok && client.Alive() {
			tools := client.Tools()
			status = fmt.Sprintf("connected (%d tools)", len(tools))
		}
		fmt.Fprintf(r.out, "  %-12s %s\n", srv.Name, cli.Muted(status))
	}
	fmt.Fprintln(r.out)
}

// mcpAdd adds an MCP server by name (known) or custom command.
func mcpAdd(r *REPL, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(r.errOut, "Usage: /mcp add <name> [command args...]")
		fmt.Fprintf(r.errOut, "Known servers: %s\n", strings.Join(knownServerNames(), ", "))
		return
	}

	name := strings.ToLower(args[0])

	// Check if already configured.
	gc, err := config.LoadGlobalConfig()
	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Could not load config: %v", err)))
		return
	}
	for _, srv := range gc.MCP {
		if srv.Name == name {
			fmt.Fprintf(r.errOut, "Server %q is already configured. Use /mcp remove %s first.\n", name, name)
			return
		}
	}

	var mcpCfg *config.MCPServerConfig

	if known, ok := knownServers[name]; ok {
		// Known server — prompt for required env vars.
		mcpCfg = &config.MCPServerConfig{
			Name:    name,
			Command: known.Command,
			Args:    known.Args,
			Env:     make(map[string]string),
		}

		for _, envKey := range known.EnvKeys {
			fmt.Fprintf(r.out, "Enter %s: ", envKey)
			val, ok := r.scanLine()
			if !ok || val == "" {
				fmt.Fprintf(r.errOut, "No value provided for %s. Aborting.\n", envKey)
				return
			}
			mcpCfg.Env[envKey] = val
		}
	} else if len(args) >= 2 {
		// Custom server: /mcp add <name> <command> [args...]
		mcpCfg = &config.MCPServerConfig{
			Name:    name,
			Command: args[1],
			Args:    args[2:],
		}
	} else {
		fmt.Fprintf(r.errOut, "Unknown server %q. Provide a command: /mcp add %s <command> [args...]\n", name, name)
		fmt.Fprintf(r.errOut, "Known servers: %s\n", strings.Join(knownServerNames(), ", "))
		return
	}

	// Attempt to connect.
	fmt.Fprintln(r.out, cli.Muted("  Connecting..."))

	client := mcp.NewClient(mcpCfg.Name, mcpCfg.Command, mcpCfg.Args, mcpCfg.Env)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Connection failed: %v", err)))
		fmt.Fprintln(r.errOut, cli.Muted("  Server was not added. Check the command and credentials."))
		return
	}

	// Save to global config (preserve existing entries).
	gc.MCP = append(gc.MCP, mcpCfg)
	if err := config.SaveGlobalConfig(gc); err != nil {
		client.Close()
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Could not save config: %v", err)))
		return
	}

	// Track the live client.
	r.mcpClients[mcpCfg.Name] = client

	tools := client.Tools()
	fmt.Fprintln(r.out, cli.Success(fmt.Sprintf("Connected to %s (%d tools)", mcpCfg.Name, len(tools))))
}

// mcpRemove removes an MCP server by name.
func mcpRemove(r *REPL, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(r.errOut, "Usage: /mcp remove <name>")
		return
	}

	name := strings.ToLower(args[0])

	gc, err := config.LoadGlobalConfig()
	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Could not load config: %v", err)))
		return
	}

	found := false
	filtered := make([]*config.MCPServerConfig, 0, len(gc.MCP))
	for _, srv := range gc.MCP {
		if srv.Name == name {
			found = true
			continue
		}
		filtered = append(filtered, srv)
	}

	if !found {
		fmt.Fprintf(r.errOut, "Server %q is not configured.\n", name)
		return
	}

	gc.MCP = filtered
	if err := config.SaveGlobalConfig(gc); err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Could not save config: %v", err)))
		return
	}

	// Close live client if running.
	if client, ok := r.mcpClients[name]; ok {
		client.Close()
		delete(r.mcpClients, name)
	}

	fmt.Fprintln(r.out, cli.Success(fmt.Sprintf("Removed %s", name)))
}

// mcpStatus shows detailed info about connected MCP servers and their tools.
func mcpStatus(r *REPL) {
	gc, err := config.LoadGlobalConfig()
	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Could not load config: %v", err)))
		return
	}

	if len(gc.MCP) == 0 {
		fmt.Fprintln(r.out, "No MCP servers configured. Run /mcp add <name> to add one.")
		return
	}

	fmt.Fprintln(r.out)
	fmt.Fprintln(r.out, cli.Heading("MCP Server Status"))
	fmt.Fprintln(r.out, strings.Repeat("\u2500", 50))

	for _, srv := range gc.MCP {
		client, ok := r.mcpClients[srv.Name]
		if !ok || !client.Alive() {
			fmt.Fprintf(r.out, "  %s: %s\n", srv.Name, cli.Muted("not connected"))
			fmt.Fprintf(r.out, "    Command: %s %s\n", srv.Command, strings.Join(srv.Args, " "))
			continue
		}

		tools := client.Tools()
		fmt.Fprintf(r.out, "  %s: connected (%d tools)\n", srv.Name, len(tools))
		fmt.Fprintf(r.out, "    Command: %s %s\n", srv.Command, strings.Join(srv.Args, " "))
		if len(tools) > 0 {
			fmt.Fprintln(r.out, "    Tools:")
			for _, t := range tools {
				desc := t.Description
				if len(desc) > 60 {
					desc = desc[:57] + "..."
				}
				fmt.Fprintf(r.out, "      - %s: %s\n", t.Name, cli.Muted(desc))
			}
		}
	}
	fmt.Fprintln(r.out)
}

// closeMCPClients shuts down all live MCP client connections.
func (r *REPL) closeMCPClients() {
	for name, client := range r.mcpClients {
		client.Close()
		delete(r.mcpClients, name)
	}
}

// autoConnectMCP connects to any MCP servers saved in global config.
// Called during REPL startup. Failures are silent — users can reconnect via /mcp add.
func (r *REPL) autoConnectMCP() {
	gc, err := config.LoadGlobalConfig()
	if err != nil || len(gc.MCP) == 0 {
		return
	}

	for _, srv := range gc.MCP {
		client := mcp.NewClient(srv.Name, srv.Command, srv.Args, srv.Env)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		err := client.Connect(ctx)
		cancel()

		if err != nil {
			// Silent failure — user can check via /mcp status.
			continue
		}
		r.mcpClients[srv.Name] = client
	}
}

// mcpBannerStatus returns a string for the banner MCP line.
// e.g. "figma (5 tools), github (12 tools)" or "".
func (r *REPL) mcpBannerStatus() string {
	if len(r.mcpClients) == 0 {
		return ""
	}

	parts := make([]string, 0, len(r.mcpClients))
	for name, client := range r.mcpClients {
		tools := client.Tools()
		parts = append(parts, fmt.Sprintf("%s (%d tools)", name, len(tools)))
	}
	return strings.Join(parts, ", ")
}

// knownServerNames returns sorted list of known server names.
func knownServerNames() []string {
	names := make([]string, 0, len(knownServers))
	for k := range knownServers {
		names = append(names, k)
	}
	// Simple sort for consistent output.
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			if names[j] < names[i] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}
	return names
}
