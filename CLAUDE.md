# Human Compiler

Go compiler that turns structured English (.human files) into full-stack applications.

## Quick Reference

```bash
make build          # Build binary (embeds version via ldflags)
make test           # Run all tests
make mcp            # Build MCP server (embeds spec + examples)
make install        # Install to /usr/local/bin
make clean          # Remove build artifacts
go test ./...       # Run tests without Make
go vet ./...        # Lint
go test ./internal/codegen/react/...  # Single package
```

## Architecture

```
.human file → Lexer → Parser → IR (Intent IR) → Analyzer → Code Generators → Output
```

The **Intent IR** is the key abstraction — a typed, serializable, framework-agnostic intermediate representation. Same IR compiles to any target framework.

## Project Structure

```
cmd/human/            CLI entry point (main.go)
cmd/human-mcp/        MCP server (needs `make mcp-embed` first)
internal/
  lexer/              Tokenizer (.human → tokens)
  parser/             Parser (tokens → AST → IR)
  ir/                 Intent IR types and serialization
  analyzer/           Semantic analysis, warnings (W201-W503)
  codegen/            16 code generators:
    react/              React + TypeScript
    vue/                Vue 3 + TypeScript
    angular/            Angular + TypeScript
    svelte/             SvelteKit + TypeScript
    node/               Node.js (Express/Fastify)
    python/             Python (FastAPI/Django)
    gobackend/          Go (Gin/Fiber)
    postgres/           PostgreSQL migrations + seeds
    docker/             Dockerfile + docker-compose (nginx inline)
    terraform/          AWS ECS/RDS, GCP Cloud Run/SQL
    cicd/               GitHub Actions workflows
    architecture/       Monolith / microservices / serverless
    storybook/          Storybook stories per framework
    scaffold/           package.json, README, start scripts
    monitoring/         Prometheus + Grafana
    themes/             7 design systems (Material, Shadcn, Ant, Chakra, Bootstrap, Tailwind, Untitled)
  quality/            Tests, security audit, lint, QA trail
  build/              Orchestrates full build pipeline
  repl/               Interactive REPL (32 commands, readline, tab completion)
  readline/           Terminal input with history + completion
  cli/                Terminal UI/UX (colors, themes, banners, spinners)
  cmdutil/            Shared CLI command utilities
  config/             Project + global config, settings
  version/            Version + build metadata (ldflags)
  llm/                LLM connector (Anthropic, OpenAI, Ollama, Groq, Gemini, OpenRouter, custom)
  mcp/                MCP server protocol handlers
  figma/              Figma design → .human mapping intelligence
  errors/             Error types with fix suggestions
examples/             13 example .human apps
```

## Key Patterns

- IR is the single source of truth between parser and codegen
- Every generator reads IR, never raw AST
- `build with:` block in .human is REQUIRED (analyzer W201 if missing)
- Quality system (tests/security/lint) runs on every build, cannot be skipped
- Design system flow: .human `theme:` block → IR → `codegen/themes` → package.json deps
- Figma import wired to REPL via `/import` command (`figma_bridge.go` + `import.go`)
- `/ask` includes validation retry loop (up to 3 attempts, feeds errors back to LLM)
- Angular/Svelte Storybook deps injected in `workspace_gen.go`, not `packagejson.go`

## What NOT to Do

- Don't modify IR types without updating ALL generators
- Don't add npm deps to scaffold without matching ALL framework paths
- Don't hardcode framework-specific logic in shared codegen utilities
- Don't skip `go vet` — it catches real issues in template code
- Don't use `go run` for testing — always `go build` then execute the binary
- Don't add AI/LLM deps to core compiler — LLM connector is separate, optional

## Agent Coordination

After completing any workplan part, append to `.claude/STATUS.md`:
```
[date] Part N: [status] — [summary] — [test results]
```
Always read `.claude/STATUS.md` before starting work.

## Current State

Phases 1-12 complete. 1,185+ tests across 34 packages.
Active work: `WORKPLAN.md` (Figma pipeline), `CLAUDE_CODE_ORG.md` (Claude Code setup).

## Repository

- **Repo:** https://github.com/barun-bash/human
- **Module:** `github.com/barun-bash/human`
- **Go:** 1.25+
- **License:** MIT
