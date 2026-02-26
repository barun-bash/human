package repl

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/barun-bash/human/internal/cli"
	"github.com/barun-bash/human/internal/cmdutil"
	"github.com/barun-bash/human/internal/figma"
)

// cmdImport handles the /import command for importing designs from Figma.
func cmdImport(r *REPL, args []string) {
	if len(args) < 2 {
		fmt.Fprintln(r.errOut, "Usage: /import figma <url>")
		return
	}

	source := strings.ToLower(args[0])
	if source != "figma" {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Unsupported import source: %s. Currently only 'figma' is supported.", source)))
		return
	}

	url := args[1]
	fileKey, nodeID, err := parseFigmaURL(url)
	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error(err.Error()))
		return
	}

	// Check Figma MCP is connected.
	figmaClient := r.mcpClients["figma"]
	if figmaClient == nil {
		fmt.Fprintln(r.errOut)
		fmt.Fprintln(r.errOut, cli.Error("Figma MCP server not connected."))
		fmt.Fprintln(r.errOut, cli.Info("Run: /mcp add figma"))
		fmt.Fprintln(r.errOut, cli.Muted("  You'll need a FIGMA_ACCESS_TOKEN from figma.com/developers"))
		return
	}

	// Step 1: Fetch metadata to understand the file structure.
	fmt.Fprintln(r.out, cli.Info("Fetching Figma design..."))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	metaArgs := map[string]any{"fileKey": fileKey}
	if nodeID != "" {
		metaArgs["nodeId"] = nodeID
	}

	result, err := figmaClient.CallTool(ctx, "get_metadata", metaArgs)
	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Figma MCP call failed: %v", err)))
		return
	}

	// Extract text content from MCP response.
	var responseText string
	for _, item := range result.Content {
		if item.Type == "text" {
			responseText += item.Text
		}
	}
	if result.IsError || responseText == "" {
		fmt.Fprintln(r.errOut, cli.Error("No data returned from Figma. Check the URL and your access token."))
		return
	}

	// Step 2: Parse MCP response into FigmaFile.
	file, err := figmaResponseToFile("Design", responseText)
	if err != nil {
		// If structured parsing fails, fall back to LLM-assisted generation
		// using the raw response as context.
		fmt.Fprintln(r.out, cli.Warn("Could not parse Figma response structurally."))
		fmt.Fprintln(r.out, cli.Info("Falling back to LLM-assisted generation..."))
		r.importViaLLM(responseText)
		return
	}

	pageCount := len(file.Pages)
	nodeCount := 0
	for _, page := range file.Pages {
		nodeCount += len(page.Nodes)
	}
	fmt.Fprintf(r.out, "%s  Found %d page(s), %d top-level component(s)\n",
		cli.Success(""), pageCount, nodeCount)

	// Step 3: Design system selection.
	ds := r.promptDesignSystem("")
	dsName := ds
	if ds == "" {
		dsName = "tailwind"
	}

	// Step 4: Optional plain English context.
	fmt.Fprintf(r.out, "\nAdd context in plain English? (y/n): ")
	answer, ok := r.scanLine()
	userDesc := ""
	if ok && isYes(answer) {
		fmt.Fprintln(r.out, cli.Info("Describe what this app does (press Enter when done):"))
		desc, descOK := r.scanLine()
		if descOK && desc != "" {
			userDesc = desc
		}
	}

	// Step 5: Generate .human file.
	// If LLM is available, use it with the Figma prompt for richer output.
	connector, llmCfg, connErr := loadREPLConnector()
	if connErr == nil && connector != nil {
		fmt.Fprintf(r.out, "\n%s  Generating with %s (%s) + Figma context...\n",
			cli.Info(""), llmCfg.Provider, llmCfg.Model)

		// Build combined prompt from Figma analysis + user description.
		figmaPrompt := figma.GenerateFigmaPrompt(file)
		fullPrompt := figmaPrompt
		if dsName != "" {
			fullPrompt += "\n\nUse design system: " + dsName
		}
		if userDesc != "" {
			fullPrompt += "\n\nAdditional context from the user:\n" + userDesc
		}

		connector.Instructions = r.instructions
		code, valid := r.generateWithRetry(connector, fullPrompt, 2)
		if code == "" {
			return
		}

		r.saveAndBuild(code, valid)
		return
	}

	// No LLM — use deterministic figma package.
	fmt.Fprintln(r.out, cli.Info("No LLM connected. Using deterministic Figma → Human pipeline."))
	config := &figma.GenerateConfig{
		AppName:  "App",
		Platform: "web",
		Frontend: "React with TypeScript",
		Backend:  "Node with Express",
		Database: "PostgreSQL",
	}

	code, err := figma.GenerateHumanFile(file, config)
	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Generation failed: %v", err)))
		return
	}

	r.saveAndBuild(code, true)
}

// importViaLLM uses the raw Figma response text as context for LLM generation.
func (r *REPL) importViaLLM(figmaContext string) {
	connector, llmCfg, err := loadREPLConnector()
	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error("LLM required for this import but not connected."))
		fmt.Fprintln(r.errOut, cli.Info("Run: /connect"))
		return
	}

	ds := r.promptDesignSystem("")

	prompt := "Generate a .human file based on this Figma design analysis:\n\n" + figmaContext
	if ds != "" {
		prompt += "\n\nUse design system: " + ds
	}

	fmt.Fprintf(r.out, "\n%s  Generating with %s (%s)...\n",
		cli.Info(""), llmCfg.Provider, llmCfg.Model)

	connector.Instructions = r.instructions
	code, valid := r.generateWithRetry(connector, prompt, 2)
	if code == "" {
		return
	}

	r.saveAndBuild(code, valid)
}

// saveAndBuild writes generated code to a file and optionally builds it.
func (r *REPL) saveAndBuild(code string, valid bool) {
	fmt.Fprintln(r.out)
	if valid {
		fmt.Fprintln(r.out, cli.Success("Generated code is valid .human syntax."))
	} else {
		fmt.Fprintln(r.out, cli.Warn("Generated code has syntax issues that may need manual fixes."))
	}

	filename := deriveFilename(code)

	// Check for overwrite.
	if _, err := os.Stat(filename); err == nil {
		fmt.Fprintf(r.out, "%s already exists. Overwrite? (y/n): ", filename)
		answer, ok := r.scanLine()
		if !ok || !isYes(answer) {
			fmt.Fprintln(r.out, cli.Info("Save cancelled."))
			return
		}
	}

	if err := os.WriteFile(filename, []byte(code+"\n"), 0644); err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Could not write file: %v", err)))
		return
	}
	fmt.Fprintln(r.out, cli.Success(fmt.Sprintf("Saved to %s", filename)))

	r.setProject(filename)
	r.clearSuggestions()

	fmt.Fprintf(r.out, "Build now? (y/n): ")
	answer, ok := r.scanLine()
	if !ok || !isYes(answer) {
		fmt.Fprintln(r.out, cli.Muted("  Run /build when you're ready."))
		return
	}

	if _, _, _, err := cmdutil.FullBuild(r.projectFile); err != nil {
		fmt.Fprintln(r.errOut, cli.Error(err.Error()))
	}
}
