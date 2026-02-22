package mcp

// AllTools returns the tool definitions for all 6 MCP tools.
func AllTools() []Tool {
	return []Tool{
		{
			Name:        "human_build",
			Description: "Compile a .human source file into production-ready code. Runs the full pipeline: parse, IR, analyze, and code generation. Returns a file manifest and key file contents.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"source": map[string]any{
						"type":        "string",
						"description": "The .human source code to compile.",
					},
					"output_dir": map[string]any{
						"type":        "string",
						"description": "Optional output directory. If omitted, uses a temporary directory.",
					},
				},
				"required": []string{"source"},
			},
		},
		{
			Name:        "human_validate",
			Description: "Validate a .human source file without generating code. Runs parse and semantic analysis, returning structured diagnostics (errors, warnings, suggestions).",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"source": map[string]any{
						"type":        "string",
						"description": "The .human source code to validate.",
					},
				},
				"required": []string{"source"},
			},
		},
		{
			Name:        "human_ir",
			Description: "Parse a .human source file and return the Intent IR as YAML. Useful for inspecting the intermediate representation before code generation.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"source": map[string]any{
						"type":        "string",
						"description": "The .human source code to convert to IR.",
					},
				},
				"required": []string{"source"},
			},
		},
		{
			Name:        "human_examples",
			Description: "List available example .human applications, or return the source of a specific example. Examples demonstrate language features and best practices.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{
						"type":        "string",
						"description": "Name of a specific example to retrieve (e.g. 'taskflow', 'blog'). If omitted, lists all available examples.",
					},
				},
			},
		},
		{
			Name:        "human_spec",
			Description: "Return the complete Human language specification (LANGUAGE_SPEC.md). Use this to understand the grammar, keywords, and syntax rules.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "human_read_file",
			Description: "Read a file from the last build output. Use after human_build to inspect individual generated files.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Relative path within the build output directory (e.g. 'react/src/App.tsx', 'node/src/index.ts').",
					},
				},
				"required": []string{"path"},
			},
		},
	}
}
