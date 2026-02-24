package repl

import (
	"fmt"
	"os"
	"strings"

	"github.com/barun-bash/human/internal/cli"
	"github.com/barun-bash/human/internal/cmdutil"
)

// humanMDTemplate is the default HUMAN.md template created by /instructions init.
const humanMDTemplate = `# Project Instructions

These instructions are automatically loaded by the Human compiler REPL and passed
as context to all AI operations (/ask, /edit, /suggest). Edit this file to set
persistent preferences for your project.

## Project Description

<!-- Describe what this application does -->

## Tech Stack

<!-- Specify your preferred frameworks and tools -->
<!-- Examples: -->
<!-- - Frontend: React with TypeScript -->
<!-- - Backend: Go -->
<!-- - Database: PostgreSQL -->
<!-- - Styling: Tailwind CSS with Shadcn components -->

## Design System

<!-- Describe your visual design preferences -->
<!-- Examples: -->
<!-- - Use a clean, minimal design -->
<!-- - Primary color: #3B82F6 -->
<!-- - Font: Inter -->

## Coding Conventions

<!-- Specify code style and patterns -->
<!-- Examples: -->
<!-- - Use functional components, not class components -->
<!-- - All API responses should include pagination -->
<!-- - Use snake_case for database fields -->

## Deployment Target

<!-- Specify where this app will be deployed -->
<!-- Examples: -->
<!-- - Docker Compose for local development -->
<!-- - AWS ECS for production -->
<!-- - Vercel for frontend -->
`

// cmdInstructions handles the /instructions command.
//
// Subcommands:
//   - /instructions       — show current HUMAN.md content
//   - /instructions edit  — open HUMAN.md in $EDITOR
//   - /instructions init  — create a template HUMAN.md
func cmdInstructions(r *REPL, args []string) {
	sub := ""
	if len(args) > 0 {
		sub = strings.ToLower(args[0])
	}

	switch sub {
	case "":
		instructionsShow(r)
	case "edit":
		instructionsEdit(r)
	case "init":
		instructionsInit(r)
	default:
		fmt.Fprintf(r.errOut, "Unknown /instructions subcommand: %s\n", sub)
		fmt.Fprintln(r.errOut, "Usage: /instructions [edit|init]")
	}
}

// instructionsShow displays the current HUMAN.md content.
func instructionsShow(r *REPL) {
	if r.instructions == "" {
		path := r.instructionsPath()
		if path == "" {
			fmt.Fprintln(r.out, "No project loaded. Use /open <file.human> first.")
		} else {
			fmt.Fprintln(r.out, "No HUMAN.md found. Run /instructions init to create one.")
		}
		return
	}

	fmt.Fprintln(r.out)
	fmt.Fprintln(r.out, cli.Heading("Project Instructions (HUMAN.md)"))
	fmt.Fprintln(r.out, strings.Repeat("\u2500", 50))
	fmt.Fprintln(r.out, r.instructions)
	fmt.Fprintln(r.out)
}

// instructionsEdit opens HUMAN.md in the user's preferred editor.
func instructionsEdit(r *REPL) {
	path := r.instructionsPath()
	if path == "" {
		fmt.Fprintln(r.errOut, "No project loaded. Use /open <file.human> first.")
		return
	}

	// Create the file if it doesn't exist so the editor has something to open.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.WriteFile(path, []byte(humanMDTemplate), 0644); err != nil {
			fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Could not create %s: %v", path, err)))
			return
		}
		fmt.Fprintln(r.out, cli.Muted(fmt.Sprintf("  Created %s", path)))
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}

	if err := cmdutil.RunCommand(".", editor, path); err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Could not open editor: %v", err)))
		return
	}

	// Reload instructions after editing.
	r.loadInstructions()
}

// instructionsInit creates a template HUMAN.md in the project directory.
func instructionsInit(r *REPL) {
	path := r.instructionsPath()
	if path == "" {
		fmt.Fprintln(r.errOut, "No project loaded. Use /open <file.human> first.")
		return
	}

	// Check for existing file.
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintf(r.out, "HUMAN.md already exists. Overwrite? (y/n): ")
		answer, ok := r.scanLine()
		if !ok || !isYes(answer) {
			fmt.Fprintln(r.out, cli.Info("Cancelled."))
			return
		}
	}

	if err := os.WriteFile(path, []byte(humanMDTemplate), 0644); err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Could not write %s: %v", path, err)))
		return
	}

	fmt.Fprintln(r.out, cli.Success(fmt.Sprintf("Created %s", path)))
	fmt.Fprintln(r.out, cli.Muted("  Edit this file to set project-wide AI preferences."))
	fmt.Fprintln(r.out, cli.Muted("  Use /instructions edit to open it in your editor."))

	// Load it.
	r.loadInstructions()
}
