# Figma → .human Demo Workflow — Interactive Pipeline (Corrected)

## Goal

Implement the complete interactive workflow when a user imports a Figma design into the Human compiler. This is the "killer demo" flow: user selects a Figma design → chooses a design system → compiler generates .human → builds the app → serves Storybook at `/ui-storyboard`.

Read the project at `/Users/barunbashyal/Documents/Claude Projects/human/`.

---

## Pre-Implementation Checklist

Before starting, build the compiler and run tests to confirm a clean baseline:

```bash
cd "/Users/barunbashyal/Documents/Claude Projects/human/"
make build
go vet ./...
go test ./...
```

---

## Part 1: Validate & Fix Storybook Across All Frontends

### Current State (from codebase audit)

The Storybook generator (`internal/codegen/storybook/`) is **already multi-framework**:
- `.storybook/main.ts` uses correct framework addon per frontend (react-vite, vue3-vite, sveltekit, angular)
- Story files use correct imports (`@storybook/react`, `@storybook/vue3`, `@storybook/svelte`, `@storybook/angular`)
- File extensions: `.stories.tsx` for React, `.stories.ts` for Vue/Angular/Svelte
- Mock data generated in `src/mocks/data.ts` with factory functions
- Preview.ts generated with control matchers

**The gap:** Angular and Svelte generate their own package.json (in `internal/codegen/angular/workspace_gen.go` and `internal/codegen/svelte/workspace_gen.go`) and neither includes Storybook devDependencies or scripts. React and Vue get them via `internal/codegen/scaffold/packagejson.go`.

### Step 1: Build each example and verify Storybook output

```bash
cd "/Users/barunbashyal/Documents/Claude Projects/human/"
go build -o /tmp/human-cli ./cmd/human/main.go
```

**Test React (taskflow):**
```bash
cd /tmp && rm -rf storybook-react && mkdir storybook-react && cd storybook-react
cp "/Users/barunbashyal/Documents/Claude Projects/human/examples/taskflow/app.human" .
/tmp/human-cli build app.human
# Note: Storybook files are inside the framework directory, NOT at root
ls -la .human/output/react/.storybook/ 2>/dev/null || echo "NO .STORYBOOK CONFIG"
ls -la .human/output/react/src/stories/ 2>/dev/null || echo "NO STORYBOOK FILES"
cat .human/output/react/.storybook/main.ts 2>/dev/null
# Should contain: @storybook/react-vite
grep -A2 "storybook" .human/output/react/package.json 2>/dev/null
# Should contain: "storybook": "storybook dev -p 6006"
```

**Test Vue (blog):**
```bash
cd /tmp && rm -rf storybook-vue && mkdir storybook-vue && cd storybook-vue
cp "/Users/barunbashyal/Documents/Claude Projects/human/examples/blog/app.human" .
/tmp/human-cli build app.human
cat .human/output/vue/.storybook/main.ts 2>/dev/null
# Should contain: @storybook/vue3-vite
grep -A2 "storybook" .human/output/vue/package.json 2>/dev/null
```

**Test Angular (ecommerce):**
```bash
cd /tmp && rm -rf storybook-angular && mkdir storybook-angular && cd storybook-angular
cp "/Users/barunbashyal/Documents/Claude Projects/human/examples/ecommerce/app.human" .
/tmp/human-cli build app.human
cat .human/output/angular/.storybook/main.ts 2>/dev/null
# Should contain: @storybook/angular
# EXPECTED GAP: package.json will NOT have storybook scripts or devDeps
grep -A2 "storybook" .human/output/angular/package.json 2>/dev/null || echo "MISSING — needs fix"
```

**Test Svelte (saas):**
```bash
cd /tmp && rm -rf storybook-svelte && mkdir storybook-svelte && cd storybook-svelte
cp "/Users/barunbashyal/Documents/Claude Projects/human/examples/saas/app.human" .
/tmp/human-cli build app.human
cat .human/output/svelte/.storybook/main.ts 2>/dev/null
# Should contain: @storybook/sveltekit
# EXPECTED GAP: package.json will NOT have storybook scripts or devDeps
grep -A2 "storybook" .human/output/svelte/package.json 2>/dev/null || echo "MISSING — needs fix"
```

### Step 2: Fix Angular package.json — add Storybook deps and scripts

**File:** `internal/codegen/angular/workspace_gen.go` (function `generatePackageJson`)

Add storybook devDependencies and scripts:

```go
import "github.com/barun-bash/human/internal/codegen/storybook"

// In generatePackageJson(), after design system deps injection:

// Storybook dependencies
for k, v := range storybook.DevDependencies("angular") {
    devDeps[k] = v
}

// In the scripts section, add storybook scripts after "test":
b.WriteString("    \"test\": \"ng test\",\n")
b.WriteString("    \"storybook\": \"storybook dev -p 6006\",\n")
b.WriteString("    \"build-storybook\": \"storybook build\"\n")
```

### Step 3: Fix Svelte package.json — add Storybook deps and scripts

**File:** `internal/codegen/svelte/workspace_gen.go` (function `generatePackageJson`)

Add storybook devDependencies and scripts:

```go
import "github.com/barun-bash/human/internal/codegen/storybook"

// In generatePackageJson(), after design system deps injection:

// Storybook dependencies
for k, v := range storybook.DevDependencies("svelte") {
    devDeps[k] = v
}

// In the scripts section, add storybook scripts:
b.WriteString("    \"check:watch\": \"svelte-kit sync && svelte-check --tsconfig ./tsconfig.json --watch\",\n")
b.WriteString("    \"storybook\": \"storybook dev -p 6006\",\n")
b.WriteString("    \"build-storybook\": \"storybook build\"\n")
```

### Step 4: Add `human storybook` CLI subcommand

**File:** `cmd/human/main.go`

Add a new `storybook` subcommand that:
1. Detects the frontend framework from the last build output (check for `react/`, `vue/`, `angular/`, `svelte/` dirs)
2. Changes to that frontend directory
3. Runs `npx storybook dev -p 6006`
4. Opens browser to `http://localhost:6006` (optional, platform-dependent)

```go
case "storybook":
    // Find the frontend directory in .human/output/
    outputDir := ".human/output"
    for _, fw := range []string{"react", "vue", "angular", "svelte"} {
        if dirExists(filepath.Join(outputDir, fw, ".storybook")) {
            cmd := exec.Command("npx", "storybook", "dev", "-p", "6006")
            cmd.Dir = filepath.Join(outputDir, fw)
            cmd.Stdout = os.Stdout
            cmd.Stderr = os.Stderr
            cmd.Stdin = os.Stdin
            os.Exit(runCmd(cmd))
        }
    }
    fmt.Fprintln(os.Stderr, "No Storybook found. Run 'human build' first.")
    os.Exit(1)
```

### Step 5: Update tests

**File:** `internal/codegen/angular/workspace_gen_test.go` or `generator_test.go`
- Add test: Angular package.json contains `@storybook/angular` in devDeps
- Add test: Angular package.json contains `"storybook"` script

**File:** `internal/codegen/svelte/workspace_gen_test.go` or `generator_test.go`
- Add test: Svelte package.json contains `@storybook/sveltekit` in devDeps
- Add test: Svelte package.json contains `"storybook"` script

### Step 6: Verify fixes

Re-run the tests from Step 1. All 4 frameworks should now show:
- `.storybook/main.ts` with correct framework addon
- `package.json` with `"storybook": "storybook dev -p 6006"` script
- `package.json` with framework-specific storybook devDependencies

```bash
go test ./internal/codegen/storybook/... ./internal/codegen/angular/... ./internal/codegen/svelte/... ./internal/codegen/scaffold/...
```

---

## Part 2: Design System Selection Subtask

### Current State (from codebase audit)

Design system infrastructure **already works end-to-end**:
- `.human` file's `theme:` block → IR builder normalizes name → `app.Theme.DesignSystem` → codegen
- 7 design systems supported: material, shadcn, ant, chakra, bootstrap, tailwind, untitled
- `internal/codegen/themes/deps.go` → `Dependencies(systemID, framework)` returns correct npm packages
- `internal/codegen/themes/generate.go` → generates CSS variables, theme.ts, tailwind.config.js per system
- Alias normalization: "mui" → "material", "antd" → "ant", etc.
- Silent fallback to Tailwind when design system doesn't support the framework

**The gap:** No interactive design system picker in the REPL. Users must manually edit the `.human` file.

### Important: Correct file paths

| Workplan reference | Actual path |
|---|---|
| `internal/cli/repl.go` | `internal/repl/repl.go` |
| `internal/cli/llm.go` | `internal/repl/ask.go` + `internal/llm/` |
| `docs/HUMAN_LLM_PROMPT.md` | `internal/llm/prompts/prompts.go` (embedded in Go code) |

### Step 1: Add interactive design system picker

**New file:** `internal/repl/designpicker.go`

Create a design system picker function that can be called from multiple places (import flow, `/ask` enhancement, standalone command):

```go
package repl

// designSystems lists available design systems with descriptions.
var designSystems = []struct {
    ID   string
    Name string
    Desc string
}{
    {"material", "Material UI", "Google's design system"},
    {"shadcn", "Shadcn/ui", "Modern, minimal components"},
    {"ant", "Ant Design", "Enterprise-grade UI"},
    {"chakra", "Chakra UI", "Accessible, simple"},
    {"bootstrap", "Bootstrap", "Classic, widely supported"},
    {"tailwind", "Tailwind CSS", "Utility-first, no components"},
    {"untitled", "Untitled UI", "Clean, modern, React Aria"},
}

// promptDesignSystem displays the interactive picker and returns the selected ID.
// Returns empty string if user cancels.
func (r *REPL) promptDesignSystem() string {
    // Display numbered list
    // Read user selection (1-7, or 8 for Custom)
    // If Custom: prompt for primary color, font, border radius, spacing
    // Return the design system ID
}
```

### Step 2: Add framework-design system compatibility warning

When the user selects a design system, check compatibility with the target frontend and warn if it will fall back to Tailwind:

| Design System | React | Vue | Angular | Svelte |
|---|---|---|---|---|
| Material UI | native | native | native | **fallback** |
| Shadcn/ui | native | native | **fallback** | native |
| Ant Design | native | native | native | **fallback** |
| Chakra UI | native | **fallback** | **fallback** | **fallback** |
| Bootstrap | native | native | native | native |
| Tailwind CSS | native | native | native | native |
| Untitled UI | native | native | native | native |

Source: `internal/codegen/themes/theme.go` — the `DesignSystem.Frameworks` map.

If the user selects Chakra + Vue, show: "Chakra UI doesn't support Vue. Tailwind CSS will be used with Chakra's color palette."

### Step 3: Wire picker into `/ask` command

**File:** `internal/repl/ask.go`

After the user provides their description, before sending to LLM, optionally prompt for design system if the description doesn't already specify one:

```go
// In cmdAsk(), after reading user description:
if !containsDesignSystem(description) {
    r.print("Would you like to choose a design system? [y/N]: ")
    if r.readYesNo() {
        ds := r.promptDesignSystem()
        if ds != "" {
            description += "\n\nUse design system: " + ds
        }
    }
}
```

### Step 4: Verify design system affects output

Build the figma-demo example with different design systems and confirm:

```bash
for ds in "material" "shadcn" "tailwind"; do
    echo "=== Building with $ds ==="
    cd /tmp && rm -rf "ds-test-$ds" && mkdir "ds-test-$ds" && cd "ds-test-$ds"
    cp "/Users/barunbashyal/Documents/Claude Projects/human/examples/figma-demo/app.human" .
    # macOS sed requires -i '' (empty string argument)
    sed -i '' "s/design system is .*/design system is $ds/" app.human
    /tmp/human-cli build app.human
    echo "Files: $(find .human/output -type f | wc -l)"
    echo "Design system deps:"
    grep -E "@mui|@radix|antd|@chakra|bootstrap|tailwindcss" .human/output/react/package.json 2>/dev/null || echo "(none specific)"
    echo "---"
done
```

---

## Part 3: Figma Import + Plain English → .human Generation

### Current State (from codebase audit)

**What already exists:**

1. **`/ask` command** (`internal/repl/ask.go`) — already does freeform English → .human via LLM:
   - Loads LLM connector (project config → global config → env vars)
   - Streams LLM response with real-time output
   - Uses system prompt from `internal/llm/prompts/prompts.go`
   - Includes project instructions from HUMAN.md
   - Validates via `llm.ExtractAndValidate()`
   - Derives filename from `app <Name>` declaration
   - Prompts to save and auto-build

2. **Figma classification/mapping** (`internal/figma/`) — complete but disconnected:
   - `classifier.go` — 5-tier heuristic, 18 component types
   - `mapper.go` — classified nodes → Human language statements
   - `inference.go` — data model extraction from forms/cards/tables
   - `generator.go` — `GenerateHumanFile()` orchestrates full pipeline
   - `prompt.go` — `GenerateFigmaPrompt()` and `GeneratePagePrompt()` for LLM prompts

3. **Figma MCP registered** (`internal/repl/mcp.go`) — `@anthropic/figma-mcp-server` as known server with `FIGMA_ACCESS_TOKEN` env var

4. **LLM connector** (`internal/llm/`) — 7 providers (anthropic, openai, ollama, groq, openrouter, gemini, custom)

**What does NOT exist:**
- `/import` command (no way to import Figma URLs from REPL)
- `/describe` command (use `/ask` instead — it already does this)
- Bridge from Figma MCP tool outputs → `internal/figma/` package
- Validation retry loop (single-pass only)
- Combo flow: Figma context + user description → LLM → .human

### Step 1: Add `/import` command for Figma URLs

**New file:** `internal/repl/import.go`

The `/import` command orchestrates the Figma → .human pipeline:

```go
// cmdImport handles: /import figma <url>
func (r *REPL) cmdImport(args string) {
    // 1. Parse args — expect "figma <url>"
    // 2. Check Figma MCP is connected (r.mcpClients["figma"])
    //    If not: suggest "/mcp add figma"
    // 3. Extract fileKey and nodeId from Figma URL
    // 4. Call Figma MCP tools:
    //    a. get_metadata(fileKey) → find page/screen node IDs
    //    b. get_design_context(fileKey, nodeId) → get component tree
    // 5. Parse MCP response into figma.FigmaFile struct
    // 6. Prompt for design system selection (r.promptDesignSystem())
    // 7. Ask "Add context in plain English? [y/N]"
    //    If yes: read multiline description
    // 8. Generate .human file:
    //    Option A (no LLM): figma.GenerateHumanFile(file, config)
    //    Option B (with LLM): figma.GenerateFigmaPrompt(file) + user description → LLM
    // 9. Validate output
    // 10. Prompt to save and build (reuse /ask flow)
}
```

### Step 2: Register `/import` in commands

**File:** `internal/repl/commands.go`

Add to `registerCommands()`:

```go
r.register(&Command{
    Name:        "/import",
    Description: "Import from Figma or other sources",
    Usage:       "/import figma <url>",
    Handler:     r.cmdImport,
    Category:    "design",
})
```

Add to the help `order` slice after `/ask`.

### Step 3: Add validation retry loop

**File:** `internal/repl/ask.go` (or shared utility)

When LLM-generated code fails validation, retry up to 3 times:

```go
func (r *REPL) generateWithRetry(connector *llm.Connector, prompt string, maxRetries int) (string, error) {
    for attempt := 0; attempt <= maxRetries; attempt++ {
        result, err := connector.Ask(ctx, prompt)
        if err != nil {
            return "", err
        }
        if result.Valid {
            return result.Code, nil
        }
        if attempt == maxRetries {
            // Return best effort with warning
            r.printWarn("Generated code has validation issues: %s", result.ParseError)
            return result.Code, nil
        }
        // Feed errors back to LLM
        prompt = fmt.Sprintf("Fix these errors in the Human code:\n%s\n\nOriginal code:\n```human\n%s\n```", result.ParseError, result.Code)
        r.printInfo("Validation failed, retrying (%d/%d)...", attempt+1, maxRetries)
    }
    return "", fmt.Errorf("generation failed after %d retries", maxRetries)
}
```

### Step 4: Wire Figma MCP response → figma package

Create a bridge that converts Figma MCP tool responses into `figma.FigmaFile` structs:

**New file:** `internal/repl/figma_bridge.go`

```go
package repl

import (
    "encoding/json"
    "github.com/barun-bash/human/internal/figma"
)

// parseFigmaResponse converts raw JSON from Figma MCP get_design_context
// into a figma.FigmaFile suitable for the classifier/mapper/generator pipeline.
func parseFigmaResponse(rawJSON []byte) (*figma.FigmaFile, error) {
    // Parse Figma node tree JSON into FigmaFile struct
    // Map node types, properties, children
    // Extract layout, fills, strokes, effects
    // Return populated FigmaFile
}
```

### Step 5: Do NOT create `/describe` — enhance `/ask` instead

The `/ask` command already does exactly what `/describe` was supposed to do. Instead of creating a new command, enhance `/ask` to optionally include design system context:

**File:** `internal/repl/ask.go`

After reading the user's description, if no design system is mentioned and no `.human` file is loaded:
1. Offer design system selection
2. Append to the LLM prompt

This avoids command proliferation and keeps the UX simple.

---

## Part 4: End-to-End Validation

### Test 1: Full build with Storybook — all 4 frameworks

```bash
cd "/Users/barunbashyal/Documents/Claude Projects/human/"
go build -o /tmp/human-cli ./cmd/human/main.go

# React (taskflow)
cd /tmp && rm -rf e2e-react && mkdir e2e-react && cd e2e-react
cp "/Users/barunbashyal/Documents/Claude Projects/human/examples/taskflow/app.human" .
/tmp/human-cli build app.human
echo "=== React Storybook ==="
test -f .human/output/react/.storybook/main.ts && echo "PASS: main.ts exists" || echo "FAIL"
grep "react-vite" .human/output/react/.storybook/main.ts >/dev/null && echo "PASS: correct framework" || echo "FAIL"
grep '"storybook"' .human/output/react/package.json >/dev/null && echo "PASS: script exists" || echo "FAIL"
grep "@storybook/react" .human/output/react/package.json >/dev/null && echo "PASS: deps exist" || echo "FAIL"

# Vue (blog)
cd /tmp && rm -rf e2e-vue && mkdir e2e-vue && cd e2e-vue
cp "/Users/barunbashyal/Documents/Claude Projects/human/examples/blog/app.human" .
/tmp/human-cli build app.human
echo "=== Vue Storybook ==="
test -f .human/output/vue/.storybook/main.ts && echo "PASS: main.ts exists" || echo "FAIL"
grep "vue3-vite" .human/output/vue/.storybook/main.ts >/dev/null && echo "PASS: correct framework" || echo "FAIL"
grep '"storybook"' .human/output/vue/package.json >/dev/null && echo "PASS: script exists" || echo "FAIL"
grep "@storybook/vue3" .human/output/vue/package.json >/dev/null && echo "PASS: deps exist" || echo "FAIL"

# Angular (ecommerce)
cd /tmp && rm -rf e2e-angular && mkdir e2e-angular && cd e2e-angular
cp "/Users/barunbashyal/Documents/Claude Projects/human/examples/ecommerce/app.human" .
/tmp/human-cli build app.human
echo "=== Angular Storybook ==="
test -f .human/output/angular/.storybook/main.ts && echo "PASS: main.ts exists" || echo "FAIL"
grep "storybook/angular" .human/output/angular/.storybook/main.ts >/dev/null && echo "PASS: correct framework" || echo "FAIL"
grep '"storybook"' .human/output/angular/package.json >/dev/null && echo "PASS: script exists" || echo "FAIL: NEEDS FIX"
grep "@storybook/angular" .human/output/angular/package.json >/dev/null && echo "PASS: deps exist" || echo "FAIL: NEEDS FIX"

# Svelte (saas)
cd /tmp && rm -rf e2e-svelte && mkdir e2e-svelte && cd e2e-svelte
cp "/Users/barunbashyal/Documents/Claude Projects/human/examples/saas/app.human" .
/tmp/human-cli build app.human
echo "=== Svelte Storybook ==="
test -f .human/output/svelte/.storybook/main.ts && echo "PASS: main.ts exists" || echo "FAIL"
grep "sveltekit" .human/output/svelte/.storybook/main.ts >/dev/null && echo "PASS: correct framework" || echo "FAIL"
grep '"storybook"' .human/output/svelte/package.json >/dev/null && echo "PASS: script exists" || echo "FAIL: NEEDS FIX"
grep "@storybook/sveltekit" .human/output/svelte/package.json >/dev/null && echo "PASS: deps exist" || echo "FAIL: NEEDS FIX"
```

### Test 2: Design system switching

```bash
# macOS-compatible sed (uses -i '' not -i)
for ds in "material" "shadcn" "tailwind"; do
    echo "=== Building with $ds ==="
    cd /tmp && rm -rf "ds-test-$ds" && mkdir "ds-test-$ds" && cd "ds-test-$ds"
    cp "/Users/barunbashyal/Documents/Claude Projects/human/examples/figma-demo/app.human" .
    sed -i '' "s/design system is .*/design system is $ds/" app.human
    /tmp/human-cli build app.human
    echo "Files: $(find .human/output -type f | wc -l)"
    echo "Design system deps:"
    grep -oE '"@mui[^"]*"|"@radix[^"]*"|"@chakra[^"]*"|"tailwindcss[^"]*"' .human/output/react/package.json 2>/dev/null || echo "(base only)"
    echo "---"
done
```

### Test 3: Go test suite

```bash
cd "/Users/barunbashyal/Documents/Claude Projects/human/"
go vet ./...
go test ./...
```

### Test 4: `/import` command (manual REPL test)

Since the REPL uses readline and doesn't accept piped input, test manually:

```bash
/tmp/human-cli
# In REPL:
# /mcp add figma
# (enter FIGMA_ACCESS_TOKEN when prompted)
# /import figma https://figma.com/design/<fileKey>/<fileName>?node-id=<nodeId>
# (follow interactive prompts)
```

If no Figma token is available, test the design system picker standalone:

```bash
/tmp/human-cli
# /ask
# > A dashboard with user stats, project list, and activity feed.
# > Use React, Node, PostgreSQL.
# (should offer design system selection if /ask was enhanced)
```

---

## Validation Checklist

### Part 1 — Storybook
- [ ] Storybook `.storybook/main.ts` exists for React with `@storybook/react-vite`
- [ ] Storybook `.storybook/main.ts` exists for Vue with `@storybook/vue3-vite`
- [ ] Storybook `.storybook/main.ts` exists for Angular with `@storybook/angular`
- [ ] Storybook `.storybook/main.ts` exists for Svelte with `@storybook/sveltekit`
- [ ] React `package.json` includes storybook scripts and devDeps ✅ (already works)
- [ ] Vue `package.json` includes storybook scripts and devDeps ✅ (already works)
- [ ] Angular `package.json` includes storybook scripts and devDeps (FIX NEEDED)
- [ ] Svelte `package.json` includes storybook scripts and devDeps (FIX NEEDED)
- [ ] `human storybook` CLI subcommand launches Storybook dev server (NEW)
- [ ] Story file extensions correct: `.stories.tsx` (React), `.stories.ts` (Vue/Angular/Svelte)

### Part 2 — Design System
- [ ] Interactive design system picker displays 7 options + Custom
- [ ] Custom option prompts for color, font, radius, spacing
- [ ] Framework-design system compatibility warning on mismatch (e.g., Chakra + Vue)
- [ ] Selected design system reflected in generated theme files
- [ ] Selected design system reflected in package.json dependencies
- [ ] Design system picker wired into `/ask` flow (optional prompt)

### Part 3 — Figma Import + LLM
- [ ] `/import` command registered in REPL
- [ ] `/import figma <url>` parses fileKey and nodeId from URL
- [ ] Requires Figma MCP connection (`/mcp add figma`), shows clear error if not connected
- [ ] Calls Figma MCP tools to fetch design context
- [ ] Bridges MCP response → `internal/figma/` package structs
- [ ] Offers design system selection during import flow
- [ ] Offers optional plain English description to enrich generation
- [ ] Generates .human file (via figma package or LLM, depending on context)
- [ ] Validation retry loop: up to 3 attempts with error feedback to LLM
- [ ] Prompts to save file and auto-build

### Part 4 — Tests
- [ ] `go vet ./...` clean
- [ ] `go test ./...` all pass
- [ ] E2E: all 4 framework builds produce Storybook output
- [ ] E2E: design system switching changes package.json deps
- [ ] E2E: `human storybook` launches dev server from build output

---

## Implementation Order (Recommended)

1. **Part 1, Steps 2-3** — Fix Angular/Svelte package.json (small, isolated changes)
2. **Part 1, Step 4** — Add `human storybook` CLI subcommand
3. **Part 1, Steps 5-6** — Tests and verification
4. **Part 2, Steps 1-2** — Design system picker + compatibility warning
5. **Part 2, Step 3** — Wire picker into `/ask`
6. **Part 3, Step 3** — Validation retry loop (reusable utility)
7. **Part 3, Steps 1-2** — `/import` command + registration
8. **Part 3, Step 4** — Figma MCP → figma package bridge
9. **Part 4** — Full E2E validation

## Files Modified/Created

### Modified
- `internal/codegen/angular/workspace_gen.go` — Add storybook deps + scripts
- `internal/codegen/svelte/workspace_gen.go` — Add storybook deps + scripts
- `internal/repl/commands.go` — Register `/import`, add to help order
- `internal/repl/ask.go` — Add optional design system prompt + retry loop
- `cmd/human/main.go` — Add `storybook` subcommand

### New
- `internal/repl/designpicker.go` — Interactive design system selection UI
- `internal/repl/import.go` — `/import figma <url>` command handler
- `internal/repl/figma_bridge.go` — Figma MCP response → figma.FigmaFile parser

### Tests (modified or new)
- `internal/codegen/angular/generator_test.go` — Storybook assertions
- `internal/codegen/svelte/generator_test.go` — Storybook assertions
- `internal/repl/import_test.go` — URL parsing, flow tests

## Commit

```bash
git add internal/codegen/angular/workspace_gen.go \
        internal/codegen/svelte/workspace_gen.go \
        internal/repl/designpicker.go \
        internal/repl/import.go \
        internal/repl/figma_bridge.go \
        internal/repl/commands.go \
        internal/repl/ask.go \
        cmd/human/main.go \
        internal/codegen/angular/generator_test.go \
        internal/codegen/svelte/generator_test.go \
        internal/repl/import_test.go
git commit -m "Figma demo workflow: storybook fixes, design system picker, /import command

- Fix Angular/Svelte package.json missing storybook deps and scripts
- Add human storybook CLI subcommand
- Add interactive design system picker with compatibility warnings
- Add /import figma <url> command with MCP bridge
- Add validation retry loop for LLM-generated code
- Wire design system selection into /ask flow"
git push
```
