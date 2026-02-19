# Contributing to Human

Thank you for your interest in contributing to the Human programming language compiler.

## Setup

### Prerequisites

- Go 1.21 or later
- Git
- Make (optional, but recommended)

### Getting Started

```bash
git clone https://github.com/barun-bash/human.git
cd human
make build
make test
```

## Coding Conventions

- Follow the [Go standard project layout](https://github.com/golang-standards/project-layout) (`cmd/`, `internal/`, `pkg/`)
- All compiler internals go in `internal/` — these are not importable by external packages
- Public IR types go in `pkg/humanir/` for plugin authors
- Use Go's `testing` package. No external test frameworks.
- Every package must have `*_test.go` files
- No external dependencies unless absolutely necessary. Standard library preferred.
- Error messages must be in plain English and suggest fixes in Human language

## Running Tests

```bash
# Run all tests
make test

# Run tests with verbose output
go test -v ./...

# Run tests for a specific package
go test -v ./internal/lexer/

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

## Linting

```bash
make lint
# or
go vet ./...
```

## Project Structure

```
cmd/human/       CLI entry point
internal/lexer/  Tokenizer
internal/parser/ Recursive descent parser + AST
internal/ir/     Intent IR definitions and builder
internal/analyzer/  Semantic analysis
internal/codegen/   Code generators (per target)
internal/quality/   Quality engine (tests, security, lint, QA)
internal/errors/    Error types and messages
internal/config/    Configuration loading
```

## Pull Request Guidelines

1. **One concern per PR.** Keep pull requests focused on a single change.
2. **Write tests.** Every new feature or bug fix should include tests.
3. **Run checks before submitting.** Make sure `make test` and `make lint` pass.
4. **Write clear commit messages.** Describe what changed and why.
5. **Test against the example.** Use `examples/taskflow/app.human` as a smoke test for parser and lexer changes.
6. **Keep error messages human-friendly.** If you add a new error, write it as advice from a helpful colleague, not a stack trace.

## What NOT to Do

- Do not add AI/LLM dependencies to the core compiler
- Do not use external Go dependencies unless absolutely necessary
- Do not generate code that requires a Human runtime
- Do not skip quality checks
- Do not make error messages technical — they should read like advice

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
