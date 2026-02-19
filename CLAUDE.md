# CLAUDE.md — Project Context for Claude Code

## What is this project?

Human is a programming language where you write in structured English and the compiler produces production-ready, full-stack applications. The compiler is written in Go.

**One sentence:** English in → production-ready code out.

## Repository

- **Repo:** https://github.com/barun-bash/human
- **Language:** Go (1.21+)
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
│   └── human/
│       └── main.go                  # CLI entry point
├── internal/
│   ├── lexer/
│   │   ├── token.go                 # Token type definitions
│   │   ├── lexer.go                 # Tokenizer
│   │   └── lexer_test.go            # Lexer tests
│   ├── parser/
│   │   ├── ast.go                   # AST node definitions
│   │   ├── parser.go                # Recursive descent parser
│   │   └── parser_test.go           # Parser tests
│   ├── analyzer/                    # Semantic analysis (future)
│   ├── ir/
│   │   ├── ir.go                    # Intent IR type definitions
│   │   ├── builder.go               # AST → IR transformation
│   │   ├── serialize.go             # IR ↔ YAML/JSON
│   │   └── ir_test.go               # IR tests
│   ├── codegen/                     # Code generators (future)
│   │   ├── frontend/react/
│   │   ├── backend/node/
│   │   ├── database/postgresql/
│   │   └── infra/docker/
│   ├── quality/                     # Quality engine (future)
│   ├── design/                      # Figma/image import (future)
│   ├── llm/                         # LLM connector (future)
│   ├── errors/                      # Error types and messages
│   └── config/                      # Configuration loading
├── examples/
│   └── taskflow/
│       └── app.human                # Reference example application
├── go.mod
├── Makefile
├── LANGUAGE_SPEC.md
├── ARCHITECTURE.md
├── ROADMAP.md
├── MANIFESTO.md
└── README.md
```

## Current Status

**Phase 1: Foundation** — Building the lexer and parser.

What exists so far:
- Documentation: README, Language Spec, Architecture, Roadmap, Manifesto
- Example: examples/taskflow/app.human (complete task management app)
- Go module initialized

What needs to be built next (in order):
1. **Lexer + Tokens** (`internal/lexer/`) — Tokenize .human files
2. **Parser + AST** (`internal/parser/`) — Build syntax tree from tokens
3. **Intent IR** (`internal/ir/`) — Framework-agnostic intermediate representation
4. **CLI** (`cmd/human/main.go`) — Command-line interface

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

Or without Make:
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

## Immediate Next Task

Build the lexer and token system in `internal/lexer/`:

1. `token.go` — Define all token types (100+ tokens covering declarations, types, actions, conditions, connectors, modifiers)
2. `lexer.go` — Tokenizer that handles indentation tracking, keyword recognition (case-insensitive), string/number parsing, section headers, comments
3. `lexer_test.go` — Comprehensive tests covering app declarations, data models, APIs, enums, indentation, comments, section headers

Then build the parser in `internal/parser/`:

1. `ast.go` — AST node definitions for all language constructs
2. `parser.go` — Recursive descent parser consuming tokens from lexer
3. `parser_test.go` — Tests parsing complete .human files into AST

**Test target:** Successfully parse `examples/taskflow/app.human` end-to-end.
