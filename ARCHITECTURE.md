# Human Compiler Architecture

## Overview

The Human compiler is written in Go and follows a traditional multi-phase compiler architecture with one key innovation: the Intent IR (Intermediate Representation) that decouples the language from any specific output framework.

```
.human files + designs
        │
        ▼
   ┌─────────┐
   │  Lexer   │   Tokenizes .human source into tokens
   └────┬─────┘
        │
        ▼
   ┌─────────┐
   │  Parser  │   Builds Abstract Syntax Tree (AST) from tokens
   └────┬─────┘
        │
        ▼
   ┌──────────┐
   │ Analyzer │   Type checking, validation, semantic analysis
   └────┬─────┘
        │
        ▼
   ┌──────────┐
   │ IR Gen   │   Transforms AST into Intent IR
   └────┬─────┘
        │
        ▼
   ┌──────────┐
   │ Intent   │   Framework-agnostic representation
   │ IR       │   (serializable as YAML/JSON)
   └────┬─────┘
        │
        ├──────────┬──────────┬──────────┐
        ▼          ▼          ▼          ▼
   ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐
   │Frontend│ │Backend │ │Infra   │ │Quality │
   │Codegen │ │Codegen │ │Codegen │ │Engine  │
   └───┬────┘ └───┬────┘ └───┬────┘ └───┬────┘
       │          │          │          │
       ▼          ▼          ▼          ▼
    React/     Node/      Docker/    Tests/
    Angular    Python     Terraform  Audit
```

---

## Directory Structure

```
human-lang/
├── cmd/
│   └── human/
│       └── main.go              # CLI entry point
│
├── internal/
│   ├── lexer/
│   │   ├── lexer.go             # Tokenizer
│   │   ├── lexer_test.go
│   │   ├── token.go             # Token types
│   │   └── keywords.go          # Reserved keywords
│   │
│   ├── parser/
│   │   ├── parser.go            # Parser (tokens → AST)
│   │   ├── parser_test.go
│   │   ├── ast.go               # AST node definitions
│   │   ├── data.go              # Parse data declarations
│   │   ├── page.go              # Parse page declarations
│   │   ├── api.go               # Parse API declarations
│   │   ├── security.go          # Parse security declarations
│   │   ├── architecture.go      # Parse architecture declarations
│   │   └── devops.go            # Parse devops declarations
│   │
│   ├── analyzer/
│   │   ├── analyzer.go          # Semantic analysis
│   │   ├── analyzer_test.go
│   │   ├── typechecker.go       # Type checking
│   │   └── validator.go         # Business rule validation
│   │
│   ├── ir/
│   │   ├── ir.go                # Intent IR definitions
│   │   ├── ir_test.go
│   │   ├── builder.go           # AST → IR transformation
│   │   └── serialize.go         # IR ↔ YAML/JSON
│   │
│   ├── codegen/
│   │   ├── codegen.go           # Code generator interface
│   │   ├── registry.go          # Target plugin registry
│   │   │
│   │   ├── frontend/
│   │   │   ├── react/
│   │   │   │   ├── generator.go      # React + TS code generation
│   │   │   │   ├── generator_test.go
│   │   │   │   ├── components.go     # Component generation
│   │   │   │   ├── pages.go          # Page generation
│   │   │   │   ├── theme.go          # Theme/styling generation
│   │   │   │   └── templates/        # Go templates for React code
│   │   │   │       ├── component.tmpl
│   │   │   │       ├── page.tmpl
│   │   │   │       └── app.tmpl
│   │   │   │
│   │   │   ├── angular/
│   │   │   │   └── generator.go
│   │   │   ├── vue/
│   │   │   │   └── generator.go
│   │   │   └── svelte/
│   │   │       └── generator.go
│   │   │
│   │   ├── backend/
│   │   │   ├── node/
│   │   │   │   ├── generator.go      # Node + Express generation
│   │   │   │   ├── routes.go         # API route generation
│   │   │   │   ├── models.go         # Data model generation
│   │   │   │   └── templates/
│   │   │   ├── python/
│   │   │   │   └── generator.go
│   │   │   └── go/
│   │   │       └── generator.go
│   │   │
│   │   ├── database/
│   │   │   ├── postgresql/
│   │   │   │   ├── generator.go
│   │   │   │   └── migrations.go
│   │   │   ├── mongodb/
│   │   │   │   └── generator.go
│   │   │   └── sqlite/
│   │   │       └── generator.go
│   │   │
│   │   └── infra/
│   │       ├── docker/
│   │       │   └── generator.go
│   │       ├── terraform/
│   │       │   └── generator.go
│   │       └── github/
│   │           └── generator.go      # GitHub Actions generation
│   │
│   ├── quality/
│   │   ├── quality.go           # Quality engine orchestrator
│   │   ├── testgen/
│   │   │   ├── testgen.go       # Auto test generation
│   │   │   ├── unit.go          # Unit test generation
│   │   │   ├── integration.go   # Integration test generation
│   │   │   └── edge_cases.go    # Edge case generation
│   │   ├── security/
│   │   │   ├── audit.go         # Security audit engine
│   │   │   ├── dependency.go    # Dependency vulnerability scan
│   │   │   ├── injection.go     # Injection detection
│   │   │   └── secrets.go       # Secret detection
│   │   ├── lint/
│   │   │   ├── lint.go          # Code quality checks
│   │   │   ├── duplication.go   # Duplication detection
│   │   │   └── performance.go   # Performance pattern detection
│   │   └── qa/
│   │       ├── qa.go            # QA trail generation
│   │       ├── testplan.go      # Test plan generation
│   │       └── traceability.go  # Traceability matrix
│   │
│   ├── design/
│   │   ├── design.go            # Design import orchestrator
│   │   ├── figma.go             # Figma API integration
│   │   ├── image.go             # Image/screenshot analysis
│   │   └── component_map.go    # Visual → component mapping
│   │
│   ├── llm/
│   │   ├── connector.go         # LLM connector interface
│   │   ├── anthropic.go         # Claude integration
│   │   ├── openai.go            # OpenAI integration
│   │   ├── ollama.go            # Local LLM integration
│   │   └── interpreter.go       # Freeform → structured conversion
│   │
│   ├── errors/
│   │   ├── errors.go            # Error types
│   │   └── messages.go          # Human-readable error messages
│   │
│   └── config/
│       ├── config.go            # Configuration loading
│       └── defaults.go          # Default values
│
├── pkg/
│   └── humanir/
│       └── types.go             # Public IR types (for plugins)
│
├── plugins/                     # Plugin system
│   └── README.md
│
├── examples/                    # Example .human programs
│   ├── todo-app/
│   │   ├── app.human
│   │   └── human.config
│   ├── blog/
│   │   ├── app.human
│   │   ├── frontend.human
│   │   ├── backend.human
│   │   └── human.config
│   └── saas-starter/
│       ├── app.human
│       ├── frontend.human
│       ├── backend.human
│       ├── devops.human
│       ├── integrations.human
│       └── human.config
│
├── LANGUAGE_SPEC.md             # Language specification
├── ARCHITECTURE.md              # This file
├── ROADMAP.md                   # Development roadmap
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## Phase Details

### Phase 1: Lexer

The lexer reads `.human` source text and produces a stream of tokens.

**Key design decisions:**
- Line-oriented (indentation matters, like Python)
- Keywords are case-insensitive (`Page` = `page` = `PAGE`)
- Strings are enclosed in double quotes
- Comments start with `#`
- Section headers are `── name ──`

**Token types:**
```go
type TokenType int

const (
    // Structural
    TOKEN_NEWLINE
    TOKEN_INDENT
    TOKEN_DEDENT
    TOKEN_COLON
    TOKEN_SECTION_HEADER    // ── name ──
    
    // Literals
    TOKEN_STRING            // "hello"
    TOKEN_NUMBER            // 42, 3.14
    TOKEN_IDENTIFIER        // user_name, PostTitle
    
    // Keywords (declarations)
    TOKEN_APP
    TOKEN_DATA
    TOKEN_PAGE
    TOKEN_COMPONENT
    TOKEN_API
    TOKEN_SERVICE
    TOKEN_POLICY
    TOKEN_THEME
    TOKEN_ARCHITECTURE
    TOKEN_ENVIRONMENT
    TOKEN_INTEGRATE
    
    // Keywords (actions)
    TOKEN_SHOW
    TOKEN_FETCH
    TOKEN_CREATE
    TOKEN_UPDATE
    TOKEN_DELETE
    TOKEN_SEND
    TOKEN_RESPOND
    TOKEN_NAVIGATE
    
    // Keywords (conditions)
    TOKEN_IF
    TOKEN_WHEN
    TOKEN_WHILE
    TOKEN_UNLESS
    
    // Keywords (connectors)
    TOKEN_IS
    TOKEN_ARE
    TOKEN_HAS
    TOKEN_WITH
    TOKEN_FROM
    TOKEN_TO
    TOKEN_IN
    TOKEN_ON
    TOKEN_FOR
    TOKEN_BY
    TOKEN_AS
    TOKEN_AND
    TOKEN_OR
    TOKEN_NOT
    TOKEN_THE
    TOKEN_A
    TOKEN_AN
    TOKEN_WHICH
    TOKEN_THAT
    TOKEN_EITHER
    
    // Keywords (modifiers)
    TOKEN_REQUIRES
    TOKEN_ACCEPTS
    TOKEN_ONLY
    TOKEN_EVERY
    TOKEN_EACH
    TOKEN_ALL
    TOKEN_OPTIONAL
    TOKEN_UNIQUE
    TOKEN_ENCRYPTED
    
    // Special
    TOKEN_EOF
    TOKEN_COMMENT
)
```

### Phase 2: Parser

Recursive descent parser that builds an AST from the token stream.

**Key design decisions:**
- Top-down parsing, one declaration at a time
- Each declaration type has its own sub-parser
- Error recovery: skip to next declaration on error
- Produces clear, human-readable error messages

### Phase 3: Analyzer

Semantic analysis phase that validates the AST.

**Checks:**
- All referenced data types exist
- All page references resolve to real pages
- API inputs match data field types
- Policies reference valid roles and actions
- No circular dependencies
- All required fields are provided

### Phase 4: IR Generation

Transforms the validated AST into the Intent IR.

**Key property:** The IR is a complete, self-contained representation of the application. Given only the IR (no source files), any code generator can produce a working application.

### Phase 5: Code Generation

Pluggable code generators that transform IR into framework-specific code.

**Interface:**
```go
type CodeGenerator interface {
    Name() string
    Generate(ir *IntentIR, outputDir string) error
    SupportedTargets() []string
}
```

Each generator:
1. Reads the IR
2. Applies framework-specific templates
3. Writes a complete, buildable project
4. Includes package.json / go.mod / requirements.txt etc.

### Phase 6: Quality Engine

Runs mandatory quality checks on the generated code.

**Pipeline:**
```
Generated Code
    │
    ├── Test Generator → writes test files
    ├── Security Auditor → scans for vulnerabilities
    ├── Code Linter → checks code quality
    └── QA Trail Generator → creates test plan + traceability
    │
    ▼
Quality Report (pass/fail with human-readable messages)
```

---

## Concurrency Model

The compiler uses Go's goroutines for parallelism:

```
Parse all .human files          → concurrent per file
Analyze                         → sequential (needs full AST)
Generate IR                     → sequential
Code generation                 → concurrent per target (frontend/backend/infra)
Quality checks                  → concurrent per pillar (tests/security/lint/qa)
```

---

## Plugin System

Third-party code generators implement the `CodeGenerator` interface and are loaded at runtime.

```
$HOME/.human/plugins/
├── human-target-flutter/
│   └── plugin.so
├── human-target-react-native/
│   └── plugin.so
└── human-target-htmx/
    └── plugin.so
```

Plugins are distributed as Go plugins (`.so`) or as separate binaries that communicate via stdin/stdout JSON protocol.

---

## Error Philosophy

Every error message must:
1. Be written in plain English
2. Explain what went wrong
3. Suggest a fix in Human language (not in generated code)
4. Reference the exact line in the `.human` file

```
Error in frontend.human, line 14:
  page Dashboard fetches "transactions" but no data named 
  "Transaction" is defined anywhere.
  
  Define it in your backend:
    data Transaction:
      has an amount which is number
      has a category which is text
      has a date which is date
```
