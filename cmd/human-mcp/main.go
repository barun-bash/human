package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/barun-bash/human/internal/cli"
	"github.com/barun-bash/human/internal/mcp"
)

//go:embed embedded/LANGUAGE_SPEC.md
var languageSpec string

//go:embed embedded/examples
var examplesFS embed.FS

func main() {
	// Disable ANSI colors — MCP uses stdio for JSON-RPC, not terminal output
	cli.ColorEnabled = false

	// Load examples from embedded filesystem
	examples := make(map[string]string)
	entries, err := examplesFS.ReadDir("embedded/examples")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not read embedded examples: %v\n", err)
	} else {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
			content, err := examplesFS.ReadFile("embedded/examples/" + entry.Name())
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not read example %s: %v\n", entry.Name(), err)
				continue
			}
			examples[name] = string(content)
		}
	}

	// Create transport over stdin/stdout
	transport := mcp.NewTransport(os.Stdin, os.Stdout)

	// Create and run the MCP server
	server := mcp.NewServer(transport, languageSpec, examples)
	if err := server.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
