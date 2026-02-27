# Claude Code Organization for Human Compiler (Corrected)

## Architecture Overview

```
human/
├── CLAUDE.md                          # Always-on project context (~200 lines max)
├── .claude/
│   ├── settings.json                  # Permissions + hooks
│   ├── settings.local.json            # Local-only permissions (gitignored)
│   ├── STATUS.md                      # Agent coordination — workplan progress
│   ├── agent-activity.log             # Auto-appended by Stop hook
│   ├── skills/
│   │   ├── frontend-codegen/SKILL.md  # React, Vue, Angular, Svelte generators
│   │   ├── backend-codegen/SKILL.md   # Node, Python, Go generators
│   │   ├── quality-audit/SKILL.md     # Tests, security, lint, QA trail
│   │   ├── infra-deploy/SKILL.md      # Docker, Terraform, CI/CD
│   │   ├── figma-import/SKILL.md      # Figma → .human pipeline
│   │   └── human-lang/SKILL.md        # Language spec, IR, parser rules
│   └── agents/
│       ├── build-validator.md         # Validates builds across all stacks
│       ├── example-auditor.md         # Audits example apps for issues
│       └── codegen-reviewer.md        # Reviews generated code quality
```

---

## Part 1: CLAUDE.md — Lean Project Constitution

**Goal**: Every agent starts with the right mental model in <30 seconds of context loading. No stale info, no operational detail — just ground truth.

**Target**: ~200 lines, score 90+/100.

### What stays in CLAUDE.md

```markdown
# Human Compiler

Go compiler that turns structured English (.human files) into full-stack applications.

## Quick Reference

Build:        make build
Test:         go test ./...
Single pkg:   go test ./internal/codegen/react/...
Vet:          go vet ./...
Run REPL:     ./human
MCP server:   make mcp-embed && go build -o human-mcp ./cmd/human-mcp/

## Architecture

.human file → Lexer → Parser → IR (Intent IR) → Analyzer → Code Generators → Output

## Project Structure

cmd/human/          CLI entry point (main.go)
cmd/human-mcp/      MCP server entry point (needs `make mcp-embed` first)
internal/
  lexer/            Tokenizer (.human → tokens)
  parser/           Parser (tokens → AST → IR)
  ir/               Intent IR types and serialization
  analyzer/         Semantic analysis, warnings (W201-W503)
  codegen/          16 code generators:
    react/          React + TypeScript
    vue/            Vue 3 + TypeScript
    angular/        Angular + TypeScript
    svelte/         SvelteKit + TypeScript
    node/           Node.js (Express/Fastify)
    python/         Python (FastAPI/Django)
    gobackend/      Go (Gin/Fiber)
    postgres/       PostgreSQL migrations + seeds
    docker/         Dockerfile + docker-compose (includes nginx config)
    cicd/           GitHub Actions workflows
    architecture/   Monolith / microservices / serverless topology
    storybook/      Storybook stories per framework
    scaffold/       package.json, README, start scripts
    monitoring/     Prometheus + Grafana
    terraform/      AWS ECS/RDS, GCP Cloud Run/SQL
    themes/         7 design systems (Material, Shadcn, Ant, Chakra, Bootstrap, Tailwind, Untitled)
  quality/          Tests, security audit, lint, QA trail (NOT inside codegen/)
  figma/            Figma file classifier + mapper
  llm/              LLM connector (Anthropic, OpenAI, Ollama, Groq, Gemini, OpenRouter, custom)
  build/            Orchestrates full build pipeline
  repl/             Interactive REPL (32 commands, readline, tab completion)
  readline/         Terminal input with history + completion
  cli/              Terminal UI/UX (colors, themes, banners, spinners, styled output)
  cmdutil/          Shared CLI command utilities
  mcp/              MCP server protocol handlers
  config/           Project + global config, settings
  version/          Version + build metadata (ldflags)
  errors/           Error types with fix suggestions
examples/           13 example .human apps (taskflow, blog, ecommerce, saas, ...)

## Agent Coordination

After completing any WORKPLAN.md or CLAUDE_CODE_ORG.md part, append to .claude/STATUS.md:
  [date] Part N: [status] — [summary] — [test results]
Always read .claude/STATUS.md before starting work.

## Key Patterns

- IR is the single source of truth between parser and codegen
- Every generator reads IR, never raw AST
- `build with:` block in .human is REQUIRED (analyzer W201 if missing)
- Quality system (tests/security/lint) runs on every build, cannot be skipped
- Design system flows: .human theme: block → IR → codegen/themes → package.json deps
- Figma import is wired to REPL via /import command (figma_bridge.go + import.go)
- /ask includes validation retry loop (up to 3 attempts, feeds errors back to LLM)

## What NOT to Do

- Don't modify IR types without updating ALL generators
- Don't add npm deps to scaffold without matching ALL framework paths
- Don't hardcode framework-specific logic in shared codegen utilities
- Don't skip `go vet` — it catches real issues in template code
- Don't use `go run` for testing — always `go build` then execute the binary

## Go Version

Requires Go 1.25+. Module: github.com/barun-bash/human

## Current State

Phases 1-12 complete. 1,185+ tests across 34 packages.
Active work: WORKPLAN.md (Figma pipeline), CLAUDE_CODE_ORG.md (this plan).
```

**What moved OUT of CLAUDE.md** (vs the current version):
- Figma workflow details → `.claude/skills/figma-import/SKILL.md`
- Language spec details → `.claude/skills/human-lang/SKILL.md`
- Token categories → `human-lang` skill
- Example application details → `quality-audit` skill
- Coding conventions (obvious ones) → removed (Go standard is implied)
- Key Documents list → removed (agents can find files themselves)

---

## Part 2: Skills — Domain Knowledge On-Demand

Each skill's description is always visible to Claude for matching. Full content loads when invoked via `/skill-name` or when Claude decides the skill is relevant.

### 2a. `/frontend-codegen` — Frontend Code Generation

```
.claude/skills/frontend-codegen/
├── SKILL.md
├── react-patterns.md       # React-specific codegen patterns
├── vue-patterns.md         # Vue 3 composition API patterns
├── angular-patterns.md     # Angular module/component patterns
└── svelte-patterns.md      # SvelteKit patterns
```

**SKILL.md**:
```yaml
---
name: frontend-codegen
description: >
  Generate frontend code from Human IR. Use when working on React, Vue,
  Angular, or Svelte code generators in internal/codegen/. Covers component
  generation, routing, state management, Storybook stories, and design
  system integration.
allowed-tools: Read, Grep, Glob, Bash, Edit, Write
---
```

Content covers:
- How IR pages/components map to framework constructs
- Template rendering pattern (Go templates in each generator)
- Storybook story generation (framework-aware: .stories.tsx for React, .stories.ts for others)
- Design system token injection via `internal/codegen/themes/`
- 7 design systems: Material UI, Shadcn/ui, Ant Design, Chakra UI, Bootstrap, Tailwind CSS, Untitled UI
- Package.json dependency management via `internal/codegen/scaffold/packagejson.go`
- Angular/Svelte workspace generators write their own package.json (separate from scaffold)
- **Gotcha**: Angular/Svelte Storybook deps are injected in `workspace_gen.go`, not `packagejson.go`
- **Gotcha**: Vue stories use `@storybook/vue3`, not `@storybook/vue`
- Test file generation alongside components

### 2b. `/backend-codegen` — Backend Code Generation

```
.claude/skills/backend-codegen/
├── SKILL.md
├── node-patterns.md        # Express/Fastify patterns
├── python-patterns.md      # FastAPI/Django patterns
└── go-patterns.md          # Gin/Fiber patterns
```

**SKILL.md**:
```yaml
---
name: backend-codegen
description: >
  Generate backend code from Human IR. Use when working on Node.js,
  Python, or Go code generators in internal/codegen/. Covers API route
  generation, middleware, database queries, authentication, and
  service-to-service communication.
allowed-tools: Read, Grep, Glob, Bash, Edit, Write
---
```

Content covers:
- How IR api/data/policy/workflow nodes map to backend constructs
- Database generator: PostgreSQL only (`internal/codegen/postgres/`) — no MySQL, MongoDB, or SQLite
- Authentication flow generation (JWT, OAuth)
- Middleware chain generation
- Go backend package is `gobackend/` (not `golang/`)
- **Gotcha**: Go backend needs `git` in Docker for dependency fetching
- **Gotcha**: Node Docker uses `npm install` not `npm ci` (no lockfile in generated code)
- **Gotcha**: Port conflicts — each backend framework has different defaults

### 2c. `/quality-audit` — Tests, Security, Lint, QA

```
.claude/skills/quality-audit/
├── SKILL.md
└── quality-checklist.md    # Per-generator quality requirements
```

**SKILL.md**:
```yaml
---
name: quality-audit
description: >
  Generate and validate quality artifacts from Human IR. Use when working
  on test generation, security auditing, code linting, or QA trail in
  internal/quality/. Note: quality/ is a top-level internal package,
  NOT inside codegen/.
allowed-tools: Read, Grep, Glob, Bash, Edit, Write
---
```

Content covers:
- Quality system is mandatory — runs on every build, no skip flag
- `internal/quality/` is a peer of `internal/codegen/`, not nested inside it
- Test generation: unit, edge-case, integration, frontend render tests
- Security audit: dependency scanning, input sanitization, auth checks
- Coverage threshold: hardcoded at 90% in `internal/quality/coverage.go` (not configurable)
- QA trail: traceability matrix from .human spec → tests → security → QA
- `examples/taskflow/app.human` is the primary integration test target

### 2d. `/infra-deploy` — Docker, Terraform, CI/CD

```
.claude/skills/infra-deploy/
├── SKILL.md
├── docker-patterns.md      # Dockerfile + compose patterns
└── cicd-patterns.md        # GitHub Actions patterns
```

**SKILL.md**:
```yaml
---
name: infra-deploy
description: >
  Generate infrastructure and deployment code from Human IR. Use when
  working on Docker, Terraform, CI/CD, or architecture generators in
  internal/codegen/. Covers containerization, orchestration, cloud
  deployment, and monitoring setup.
allowed-tools: Read, Grep, Glob, Bash, Edit, Write
---
```

Content covers:
- Docker multi-stage builds per framework
- docker-compose orchestration (frontend + backend + db + monitoring)
- nginx reverse proxy config is **embedded inline in Dockerfiles** (not separate nginx.conf files)
  - React and Angular Dockerfiles include nginx with `/api` proxy pass and SPA `try_files`
- CI/CD pipeline generation (GitHub Actions only — no GitLab CI)
- Architecture modes: monolith, microservices, serverless
- Monitoring: Prometheus + Grafana config generation
- Terraform: AWS ECS/RDS and GCP Cloud Run/SQL
- **Gotcha**: nginx `/api` proxy pass is required — missing causes 404s on API calls
- **Gotcha**: Docker frontend builds need correct `EXPOSE` port per framework
- Port convention: frontend `73xx`, backend `74xx`, database `74xx` range

### 2e. `/figma-import` — Figma → .human Pipeline

```
.claude/skills/figma-import/
├── SKILL.md
└── figma-api-reference.md  # Figma node types and mapping rules
```

**SKILL.md**:
```yaml
---
name: figma-import
description: >
  Import Figma designs into .human files. Use when working on the Figma
  integration in internal/figma/, the MCP bridge in internal/repl/
  (figma_bridge.go, import.go), or the /import REPL command. Covers
  Figma API, node classification, design-to-IR mapping, and LLM generation.
---
```

Content covers:
- Figma URL parsing (design, file, board, branch formats) — `parseFigmaURL()` in `figma_bridge.go`
- Node classification: frame → page, component → component, text → content
- `figmaResponseToFile()` conversion handles 3 JSON formats: document tree, flat node list, single node
- MCP server tools: `get_design_context`, `get_screenshot`, `get_metadata`
- **Current state**: Figma IS wired to REPL via `/import figma <url>` command
  - `internal/repl/import.go` — command handler, design system prompt, LLM/deterministic generation
  - `internal/repl/figma_bridge.go` — URL parsing, JSON→FigmaFile conversion (19 tests)
- LLM fallback: if structured parsing fails, raw Figma response is fed to LLM as context
- Deterministic fallback: `figma.GenerateHumanFile()` when no LLM is connected
- **Gap**: No LLM vision integration — uses text/metadata only, not screenshots
- Scope to 1-3 screens per import (larger imports exceed context limits)

### 2f. `/human-lang` — Language Spec & Compiler Internals

```
.claude/skills/human-lang/
├── SKILL.md
├── grammar-reference.md    # Condensed grammar rules
└── ir-node-types.md        # IR type reference
```

**SKILL.md**:
```yaml
---
name: human-lang
description: >
  Human language specification and compiler internals. Use when working
  on the lexer, parser, IR types, or analyzer. Covers grammar rules,
  token categories, IR node types, and semantic analysis warnings.
allowed-tools: Read, Grep, Glob
---
```

Content covers:
- Token categories (keywords, types, modifiers, actions, conditions, connectors)
- Grammar rules for each declaration type (app, data, page, api, etc.)
- IR node type hierarchy (Application → Data/Page/API/Policy/Workflow/etc.)
- Analyzer warning codes:
  - W201: No build targets specified
  - W301-W304: Page reference issues (undefined, unused, missing API, bad navigation)
  - W401-W403: Architecture issues (invalid style, undefined model/service refs)
  - W501-W503: Integration issues (missing credentials, email without provider, Slack without integration)
- **Rule**: IR is the contract — parsers write it, generators read it, nothing else
- **Rule**: `build with:` is required — analyzer emits W201 without it

---

## Part 3: Subagents — Isolated Task Runners

Subagents handle repeated tasks that would pollute the main context. Each one has a focused scope, restricted tools, and returns only results.

### 3a. `build-validator` — Validates Builds Across All Stacks

```yaml
---
name: build-validator
description: >
  Validates that a .human file builds successfully for a given tech stack.
  Runs the full pipeline: parse → analyze → codegen → Docker build.
  Use when testing build pipeline changes across multiple frameworks.
tools: Bash, Read, Glob, Grep
model: sonnet
maxTurns: 20
---
```

**Prompt**: Given a .human example file and target stack, run `./build/human build`,
verify output file count, check for compilation errors, optionally run
`docker compose build`. Report PASS/FAIL with specific errors.

**When to spawn**: After modifying any generator, scaffold, or Docker template.
Run against all 13 examples or a targeted subset.

### 3b. `example-auditor` — Audits Example Apps

```yaml
---
name: example-auditor
description: >
  Audits an example .human app for correctness. Checks that the .human
  file parses, IR is valid, all generators produce output, and Docker
  builds succeed. Reports issues by severity.
tools: Bash, Read, Glob, Grep
model: sonnet
maxTurns: 30
---
```

**Prompt**: For a given example in `examples/`, run the full audit:
parse the .human file, check analyzer warnings, build all targets,
verify file counts match expectations, run `go test` for the relevant
packages. Return a structured report.

**When to spawn**: Before releases, after major refactors, during example audits.

### 3c. `codegen-reviewer` — Reviews Generated Code

```yaml
---
name: codegen-reviewer
description: >
  Reviews code generated by a specific generator for quality issues.
  Checks for framework-specific anti-patterns, missing imports,
  incorrect routing, and test coverage gaps.
tools: Read, Glob, Grep
model: sonnet
maxTurns: 15
---
```

**Prompt**: Given a generator output directory, review the generated code
for: missing imports, incorrect framework idioms, broken routing,
missing error handling, test file presence. Compare against the
IR to verify all nodes were generated.

**When to spawn**: After modifying a code generator template.

---

## Part 4: Hooks — Deterministic Automation

Hooks fire every time, no exceptions. Unlike skills (invoked by Claude's judgment), hooks guarantee execution.

### `.claude/settings.json` (merged with existing permissions)

```json
{
  "permissions": {
    "allow": [
      "Bash",
      "Write",
      "Edit"
    ]
  },
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Edit|Write",
        "hooks": [
          {
            "type": "command",
            "command": "fp=$(echo '$TOOL_INPUT' | jq -r '.file_path // .path // empty'); if [ -n \"$fp\" ] && echo \"$fp\" | grep -qE '\\.go$'; then gofmt -w \"$fp\" 2>/dev/null; fi",
            "timeout": 10
          }
        ]
      },
      {
        "matcher": "Edit|Write",
        "hooks": [
          {
            "type": "command",
            "command": "fp=$(echo '$TOOL_INPUT' | jq -r '.file_path // .path // empty'); if [ -n \"$fp\" ] && echo \"$fp\" | grep -qE 'internal/codegen/.+\\.go$'; then echo 'Codegen template modified — consider running build-validator against affected examples'; fi",
            "timeout": 5
          }
        ]
      }
    ],
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "cmd=$(echo '$TOOL_INPUT' | jq -r '.command'); if echo \"$cmd\" | grep -qE 'rm -rf|rm -r.*internal|rm -r.*cmd'; then echo '{\"decision\":\"block\",\"reason\":\"Blocked: destructive delete of source directories\"}'; fi",
            "timeout": 5
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "cd \"$CLAUDE_PROJECT_DIR\" && go vet ./... 2>&1 | tail -5 || true",
            "timeout": 30
          }
        ]
      },
      {
        "hooks": [
          {
            "type": "command",
            "command": "cd \"$CLAUDE_PROJECT_DIR\" && echo \"$(date '+%Y-%m-%d %H:%M') $(git diff --stat HEAD 2>/dev/null | tail -1)\" >> .claude/agent-activity.log",
            "timeout": 10
          }
        ]
      }
    ]
  }
}
```

### Hook Inventory

| Hook | Event | Matcher | What it does |
|------|-------|---------|-------------|
| Go formatter | PostToolUse | Edit\|Write | Runs `gofmt` on any edited .go file |
| Codegen warning | PostToolUse | Edit\|Write | Warns when codegen templates are modified |
| Destructive delete blocker | PreToolUse | Bash | Blocks `rm -rf` on source directories |
| Go vet on stop | Stop | * | Runs `go vet ./...` when Claude finishes a response |
| Activity logger | Stop | * | Appends timestamp + git diff summary to agent-activity.log |

### Hooks to add later

| Hook | Event | What it does |
|------|-------|-------------|
| Test runner | PostToolUse (Edit\|Write) | Runs `go test` on modified package after edit |
| .human validator | PostToolUse (Edit\|Write) | Validates .human files on save via `./build/human check` |

---

## Part 5: Agent Coordination Protocol

### Goal

Enable any agent to check progress on workplan parts without asking the user.

### Implementation

1. **Add to CLAUDE.md** (under "Key Patterns"):
```
## Agent Coordination
After completing any WORKPLAN.md or CLAUDE_CODE_ORG.md part, append to .claude/STATUS.md:
  [date] Part N: [status] — [summary] — [test results]
Always read .claude/STATUS.md before starting work.
```

2. **Create `.claude/STATUS.md`**:
```markdown
# Workplan Progress
<!-- Agents: append status lines here after completing work. Read before starting. -->
```

3. **Stop hook** (already included in Part 4 settings.json):
   - Appends timestamp + git diff summary to `.claude/agent-activity.log` on every response

### Files Changed
- CLAUDE.md — 2 lines added (Agent Coordination section)
- `.claude/STATUS.md` — new, ~3 lines
- `.claude/agent-activity.log` — auto-created by Stop hook
- `.claude/settings.json` — Stop hook added

### Validation
- Spin up a second agent, ask "what's the workplan status?" — it should read STATUS.md and answer accurately without asking you.

---

## Part 6: Implementation Plan

### Phase 1: CLAUDE.md rewrite (30 min)
Trim current CLAUDE.md to the ~200-line version from Part 1. Move displaced content into skill drafts.

### Phase 2: Core skills (2 hours)
Create the 6 skill directories and SKILL.md files. Start with `human-lang` and `frontend-codegen` since those are most frequently needed. Reference files can come later.

### Phase 3: Hooks + coordination (30 min)
Merge hooks into existing `.claude/settings.json` (preserving current permissions). Create `.claude/STATUS.md`. Test each hook by triggering the relevant tool.

### Phase 4: Subagents (1 hour)
Create the 3 agent markdown files. Test build-validator against one example.

### Phase 5: Reference files (ongoing)
Add supporting .md files to skills as you encounter repeated explanations. Extract from conversation history and WORKPLAN.md.

---

## Token Budget Analysis

| Component | Est. Tokens | When loaded |
|-----------|------------|-------------|
| CLAUDE.md | ~800 | Always |
| Skill descriptions (6x) | ~600 | Always (for matching) |
| Single skill content | ~1,500 | On-demand per task |
| Subagent prompt | ~500 | On spawn only |
| Hooks | 0 | Shell scripts, no context |
| **Worst case** | **~3,400** | CLAUDE.md + 1 skill |
| **Typical** | **~1,400** | CLAUDE.md + descriptions |

Compare to current CLAUDE.md alone: ~2,000+ tokens, much of it stale/misleading.

---

## Decision Matrix: Where Does This Go?

| Information type | Where it goes | Why |
|-----------------|--------------|-----|
| Build commands, project structure | CLAUDE.md | Needed every session |
| "Don't do X" rules | CLAUDE.md | Must be always-visible |
| Current project state | CLAUDE.md (1 line) | Prevents wrong assumptions |
| Agent coordination protocol | CLAUDE.md (2 lines) | All agents must see it |
| React codegen patterns | frontend-codegen skill | Only needed when editing React generator |
| Figma URL parsing rules | figma-import skill | Only needed during Figma work |
| IR node type reference | human-lang skill | Only needed during parser/IR work |
| Analyzer warning codes | human-lang skill | Only needed during analyzer work |
| "Validate all 13 examples" | build-validator agent | Repeated task, needs isolation |
| "Run gofmt after edit" | PostToolUse hook | Must happen every time, deterministic |
| "What's the next task" | .claude/STATUS.md | Updated by agents, read by agents |
| Conversation-specific context | Main agent context | Ephemeral, not persisted |

---

## Corrections from Original Workplan

| # | What was wrong | What's correct |
|---|---------------|---------------|
| 1 | Module path `github.com/anthropics/human` | `github.com/barun-bash/human` |
| 2 | Listed `mysql/`, `mongodb/`, `sqlite/` generators | Only `postgres/` exists |
| 3 | "Figma only wired to MCP, not REPL" | `/import` command + `figma_bridge.go` exist and work |
| 4 | Go backend dir `golang/` | Actual name is `gobackend/` |
| 5 | Analyzer warnings "W1xx-W4xx" | Actual range is W201-W503 |
| 6 | `internal/codegen/quality/` | Quality is `internal/quality/` (peer of codegen) |
| 7 | Coverage "default 90%, configurable" | 90% is hardcoded, not configurable |
| 8 | "12 example apps" | 13 example apps |
| 9 | "20+ REPL commands" | 32 commands |
| 10 | `cli/` = "CLI argument parsing" | `cli/` = terminal UI/UX (colors, themes, banners, spinners) |
| 11 | Skills "auto-loaded on context" | Descriptions always visible; full content loads on invocation |
| 12 | Hooks had broken shell quoting in JSON | Fixed `jq` + `grep` escaping |
| 13 | settings.json replaced permissions | Merged with existing `permissions.allow` array |
