# CLAUDE.md — Project Context for Claude Code

## What is this project?

Human is a programming language where you write in structured English and the compiler produces production-ready, full-stack applications. The compiler is written in Go.

**One sentence:** English in → production-ready code out.

## Repository

- **Repo:** https://github.com/barun-bash/human
- **Language:** Go (1.25+)
- **Module path:** `github.com/barun-bash/human`
- **License:** MIT

## Key Documents (read these first)

- `README.md` — Project overview, quick example, how it works
- `LANGUAGE_SPEC.md` — Complete grammar reference for the Human language
- `MANIFESTO.md` — Why this project exists
- `ARCHITECTURE.md` — Compiler design, directory structure, phase details
- `ROADMAP.md` — 52-week development plan across 14 phases

## Architecture Overview

```
.human files + designs
    ↓
Lexer (tokenize English into tokens)
    ↓
Parser (build AST from tokens)
    ↓
Analyzer (validate semantics, resolve references)
    ↓
IR Generator (AST → Intent IR, framework-agnostic)
    ↓
Code Generators (IR → React/Node/Python/Go/Docker/etc.)
    ↓
Quality Engine (tests + security + lint + QA — mandatory, cannot skip)
    ↓
Output: production-ready, deployable code
```

The **Intent IR** is the key innovation — a typed, serializable, framework-agnostic intermediate representation. Same IR compiles to any target framework.

## Directory Structure

```
human/
├── cmd/
│   ├── human/main.go                # CLI entry point
│   └── human-mcp/                   # MCP server (JSON-RPC 2.0, 6 tools)
├── internal/
│   ├── lexer/                       # Tokenizer (token.go, lexer.go)
│   ├── parser/                      # Recursive descent parser (ast.go, parser.go)
│   ├── analyzer/                    # Semantic analysis + validation
│   ├── ir/                          # Intent IR types, AST→IR builder, YAML/JSON serialization
│   ├── codegen/                     # 16 code generators:
│   │   ├── react/                   #   React, Angular, Vue, Svelte (frontend)
│   │   ├── angular/, vue/, svelte/
│   │   ├── node/, python/, gobackend/ # Node+Express, FastAPI, Go+Gin (backend)
│   │   ├── postgres/                #   PostgreSQL migrations + seeds
│   │   ├── docker/                  #   Dockerfiles + docker-compose
│   │   ├── terraform/               #   AWS ECS/RDS, GCP Cloud Run/SQL
│   │   ├── cicd/                    #   GitHub Actions workflows
│   │   ├── monitoring/              #   Prometheus + Grafana
│   │   ├── architecture/            #   Monolith/microservices/serverless topology
│   │   ├── scaffold/                #   Project scaffolding (package.json, configs)
│   │   ├── storybook/               #   Storybook stories
│   │   └── themes/                  #   7 design systems (Material, Shadcn, Chakra, etc.)
│   ├── quality/                     # Quality engine (tests, security, lint reports)
│   ├── build/                       # Build pipeline orchestrator
│   ├── cli/                         # Terminal output (colors, themes, banners, spinners)
│   ├── cmdutil/                     # Shared CLI command utilities
│   ├── repl/                        # Interactive REPL (20+ commands, readline, tab completion)
│   ├── readline/                    # Line editor with history + completion
│   ├── config/                      # Project + global config, settings
│   ├── version/                     # SemVer parsing, build metadata (ldflags)
│   ├── llm/                         # LLM connector (Anthropic, OpenAI, Ollama, Groq, Gemini)
│   ├── mcp/                         # MCP client for external tool servers
│   ├── figma/                       # Figma design → .human mapping intelligence
│   └── errors/                      # Error types with fix suggestions
├── examples/                        # 13 example apps (taskflow, blog, ecommerce, saas, ...)
├── docs/                            # Website (GitHub Pages)
├── brand/                           # Logos and assets
├── go.mod
├── Makefile                         # build, test, install (with ldflags for version embedding)
├── LANGUAGE_SPEC.md
├── ARCHITECTURE.md
├── ROADMAP.md
├── MANIFESTO.md
└── README.md
```

## Current Status

**Phases 1–12 complete.** The compiler is fully functional: lexer, parser, analyzer, IR, 16 code generators, quality engine, interactive REPL, LLM connector, MCP server, and 13 example apps. 600+ tests across 30+ packages.

What's shipping now:
- **Phase 13** — Plugin ecosystem (community-extensible generators)
- **Phase 14** — Polish + launch (performance, tutorials, v1.0)

## Language Design Decisions

- **File extension:** `.human`
- **Indentation-based scoping** (like Python, no curly braces)
- **Keywords are case-insensitive** (`Page` = `page` = `PAGE`)
- **Strings** enclosed in double quotes
- **Comments** start with `#`
- **Section headers** use `── name ──` format
- **English connectors** (`is`, `has`, `which`, `the`, `a`) are part of the grammar

## Token Categories

Declarations: app, data, page, component, api, service, policy, workflow, theme, architecture, environment, integrate, database, authentication, build

Types: text, number, decimal, boolean, date, datetime, email, url, file, image, json

Actions: show, fetch, create, update, delete, send, respond, navigate, check, validate, filter, sort, paginate, search

Conditions: if, when, while, unless, until, after, before, every

Connectors: is, are, has, with, from, to, in, on, for, by, as, and, or, not, the, a, an, which, that, either

Modifiers: requires, accepts, only, every, each, all, optional, unique, encrypted

## Coding Conventions

- Go standard project layout (`cmd/`, `internal/`, `pkg/`)
- All compiler internals in `internal/` (not importable by external packages)
- Public IR types in `pkg/humanir/` (for plugin authors)
- Every package has `*_test.go` files
- Error messages must be in plain English and suggest fixes in Human language
- Use Go's `testing` package, no external test frameworks
- Run `go vet` and `go test ./...` before committing

## Build Commands

```bash
make build       # Build the compiler binary
make test        # Run all tests
make install     # Install to /usr/local/bin
make clean       # Remove build artifacts
make lint        # Run go vet
```

Or without Make (note: `make build` embeds version via ldflags):
```bash
go build -o human ./cmd/human/
go test ./...
go vet ./...
```

## Testing Strategy

- Lexer tests: Feed .human source strings, verify token sequences
- Parser tests: Feed token streams, verify AST structure
- IR tests: Feed ASTs, verify IR output matches expected YAML/JSON
- Integration tests: Feed .human files end-to-end, verify final output
- Use `examples/taskflow/app.human` as the primary integration test target

## The Example Application

`examples/taskflow/app.human` is a complete task management application that exercises every language feature:
- App declaration and theme
- Pages (Home, Dashboard, Profile) with display/interaction/conditional statements
- Data models (User, Task, Tag, TaskTag) with all field types and relationships
- APIs (SignUp, Login, CRUD operations) with auth, validation, and logic
- Security (JWT, OAuth, rate limiting, CORS)
- Policies (FreeUser, ProUser, Admin)
- Workflows (signup sequence, overdue notifications, completion tracking)
- Error handling (database retry, validation feedback)
- Database configuration with indexes
- Integrations (SendGrid, AWS S3, Slack)
- DevOps (Git branches, CI/CD pipelines, environments, monitoring)
- Build target specification

**When building any compiler phase, test it against this file.** If the lexer can tokenize it, the parser can parse it, and the IR can represent it — the phase is complete.

## What NOT to do

- Do not add any AI/LLM dependency to the core compiler. LLM connector is a separate, optional package.
- Do not use external Go dependencies unless absolutely necessary. Standard library preferred.
- Do not generate code that requires a Human runtime. Generated code must be standalone.
- Do not skip quality checks in the compiler. If we enforce quality on users, we enforce it on ourselves.
- Do not make error messages technical. They should read like advice from a helpful colleague.

## Interactive REPL

The CLI includes a full interactive REPL (`internal/repl/`) with 20+ commands:

**Core:** `/open`, `/new`, `/check`, `/build`, `/deploy`, `/stop`, `/status`, `/run`, `/test`, `/audit`, `/eject`
**AI-assisted:** `/ask`, `/edit`, `/suggest`, `/connect`, `/disconnect`, `/model`
**System:** `/theme`, `/config`, `/history`, `/update`, `/version`, `/help`, `/quit`
**Navigation:** `/cd`, `/pwd`, `/examples`, `/instructions`, `/review`, `/mcp`

Features: readline with tab completion, command history, plan mode, auto-detect project, MCP server connections, self-update via GitHub releases.

## Figma Design Conversion

When converting Figma designs to `.human` files, scope to 1-3 screens at a time (larger imports exceed context limits). Use Figma MCP tools: `get_metadata` → `get_screenshot` + `get_design_context` → write `.human` → `human_validate` → `human_build`.

Docker port convention: frontend `73xx`, backend `74xx`, database `74xx` range.

## Version & Build

Version is embedded via ldflags at build time (`make build`). The `internal/version/` package provides SemVer parsing and comparison. The REPL checks GitHub releases on startup (24h cache) and shows update notifications.
