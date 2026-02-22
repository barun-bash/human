# Changelog

All notable changes to the Human compiler are documented in this file.

---

## v0.4.2 — 2026-02-22

**Documentation and examples release.** Adds a compiler-derived language specification, an LLM system prompt for generating valid `.human` files, and two new example applications.

### New Documentation
- **`docs/HUMAN_LANG_SPEC.md`**: Complete language specification derived from compiler source — covers all 16 block types, 215 tokens, 12 field types, relationships, validation rules, 17 error codes, and 10 warning codes
- **`docs/HUMAN_LLM_PROMPT.md`**: LLM system prompt (~3600 tokens) with syntax quick reference, two complete examples, UI pattern mappings, design system and Figma guidance, generation checklist

### New Examples
- **`examples/recipes/app.human`**: Recipe sharing app — exercises relationships, search, favorites, file uploads, reviews (React + Node)
- **`examples/projects/app.human`**: Project management app — exercises teams, tasks, comments, policies, Slack integration, CI/CD pipelines (React + Node + shadcn)

### Website
- **`docs/roadmap.html`**: Updated project roadmap to reflect current status — Phases 1–12 complete, Phase 14 in progress, Current Output table expanded to 85+ files across 14 generators, Success Metrics table expanded to 14 rows with accurate statuses
- Updated website hero status fallback to v0.4.2 (`docs/version.js` and `docs/index.html`)

### Bug Fixes
- **ecommerce example**: Added missing Slack integration block (fixes W503 warning — Slack referenced without integration declared)
- **saas example**: Fixed border radius value from `"rounded on all elements"` to `"rounded"` (fixes W304 warning — validator expects bare keyword)

---

## v0.4.1 — 2026-02-21

**Runtime correctness and framework-aware Docker generation.**

### New Features
- **Framework-aware Docker generation**: backend Dockerfile, context path, and port are now dynamic based on configured stack (`./node` at 3000, `./python` at 8000, `./go` at 8080)
- **Python and Go backend Dockerfiles**: multi-stage builds for FastAPI (uvicorn) and Go (static binary)
- **Angular-specific frontend Dockerfile**: uses `NG_APP_API_URL` instead of `VITE_API_URL`, copies from `dist/app/browser/`
- **`.dockerignore` files** generated for all backend and frontend services (language-specific ignore patterns)
- **Terraform generator** updated to use consistent backend dir/port helpers

### Bug Fixes
- **CI/CD generator**: fixed workflow YAML syntax, proper job dependencies, correct artifact paths
- **Monitoring generator**: real PromQL expressions, proper metric types (`gauge` vs `counter`), valid Grafana dashboard JSON with correct panel structure
- **Terraform generator**: valid HCL output, correct resource references, proper variable interpolation

### Website
- Updated project website to match v0.4.0 README: multi-framework output in example/how-it-works sections, separated implemented vs planned in supported targets, added Infrastructure/Integrations/Design Systems columns, fixed quick start output path

---

## v0.4.0 — 2026-02-21

**Full-stack multi-framework release.** The compiler now generates production-ready code across 4 frontend frameworks, 3 backend languages, and 10+ infrastructure targets. 600+ tests across 28 packages.

### New Generators
- **Vue 3 + TypeScript** frontend code generator (pages, components, router, API client, Pinia stores)
- **Angular 17+ standalone** frontend code generator (components, services, routing, signals)
- **Svelte 5 + SvelteKit** frontend code generator (pages, components, stores, routing)
- **Python (FastAPI)** backend code generator (models, routes, auth, SQLAlchemy + Alembic)
- **Go (Gin)** backend code generator (handlers, routes, models, GORM, auth middleware)
- **GitHub Actions CI/CD** generator (test, build, deploy workflows from pipeline declarations)
- **Terraform** generator (AWS ECS/RDS, GCP Cloud Run/SQL from architecture + environment blocks)
- **Monitoring** generator (Prometheus rules + Grafana dashboards from monitoring declarations)
- **Storybook 8** UI storyboard generator with auto-generated stories and relational mock data
- **Architecture** support generator (monolith, microservices, serverless output)

### New Features
- **7 built-in design systems**: Material UI, Shadcn/ui, Ant Design, Chakra UI, Bootstrap, Tailwind CSS, Untitled UI — injected into scaffold package.json per framework
- **Third-party integration support**: Stripe (payments), SendGrid (email), AWS S3 (storage), OAuth (Google/GitHub), Slack (messaging) — generates typed service files, env vars, and npm/pip/go dependencies
- **LLM connector**: optional Anthropic, OpenAI, and Ollama providers for `human edit --with-llm`, `human ask`, and `human suggest` commands
- **VS Code extension**: syntax highlighting, keyword/type/action snippets, and bracket matching for `.human` files
- **4 new example apps**: Blog (Vue + Python), E-commerce (Angular + Go), SaaS (Svelte + Node), API-only (Node)
- **Install script** (`install.sh`), **Homebrew formula** (`barun-bash/tap/human`), and **GoReleaser** config for cross-platform binary releases
- **Authorization middleware** generation from policy declarations
- **`human deploy`** command for deploying to configured environments
- **`human audit`** and **`human eject`** commands
- **`human build --watch`** mode for automatic rebuilds on file change
- **Stack-aware scaffolder**: only generates files relevant to the chosen frontend/backend (React+Node, Vue+Python, Angular+Go, etc.)
- **Frontend routing** wired with 404 handling across all 4 frameworks (React Router, Vue Router, Angular Router, SvelteKit)

### Bug Fixes
- React display statements now generate real JSX (lists, cards, conditionals) instead of literal Human language text
- Vue, Angular, and Svelte display statements generate real framework-specific UI elements
- Node route handlers generate real CRUD logic with auth, bcrypt, userId scoping, and Prisma queries
- Docker `.env.example` and `docker-compose.yml` now include integration config env vars (AWS_REGION, S3_BUCKET), not just credentials
- AWS-specific env vars no longer leak to non-AWS storage integrations (e.g., Cloudinary)
- React Dockerfile includes `ARG VITE_API_URL` + `ENV` before build step so Vite can inline the API URL
- `start.sh` uses subshells for Prisma commands so `set -e` failures don't corrupt working directory
- Prisma enum generation fixed (proper enum block output, no duplicate fields)
- Prisma index field resolution fixed (references actual model field names)
- Go generator: fixed `has_many_through` relationships and GORM struct tags
- Angular generator: fixed missing `package.json` and model import paths
- Svelte generator: fixed missing `package.json` start script
- React generator: added missing Vite entry files (`index.html`, `main.tsx`, `vite-env.d.ts`)
- `install.sh`: creates install directory if it does not exist
- PostgreSQL generator: fixed runtime correctness issues
- Python and Go backend generators: fixed runtime correctness issues

### Quality & Infrastructure
- Semantic analyzer with structured error codes (E101-E501 errors, W301-W503 warnings)
- Quality engine deepened: component tests, edge case generation, integration tests, coverage tracking
- Runtime validation across all generators (TypeScript `tsc --noEmit`, Vite build, `bash -n`)
- 600+ test functions across 28 Go packages
- `docs/version.js` as single source of truth for website version info

---

## v0.2.0 — 2026-02-20

**Semantic analysis release.** Added a full semantic analyzer that validates `.human` files beyond syntax — checking references, types, and relationships.

### New Features
- Semantic analyzer with error codes E101-E501 and warnings W401-W503
- `human init` command with interactive project creation
- `human run` and `human test` commands
- Colored terminal output

### Bug Fixes
- Fixed `human init` template to use correct block header syntax
- Fixed PostgreSQL index column resolution

---

## v0.1.1 — 2026-02-20

### Bug Fixes
- Version bump and documentation fixes

---

## v0.1.0 — 2026-02-19

**Initial release.** Lexer, parser, Intent IR, React + Node code generators, PostgreSQL migrations, Docker infrastructure, quality engine, and project scaffolder.

### Features
- Lexer with indentation tracking, case-insensitive keywords, section headers, comments
- Recursive descent parser for all Human language constructs
- Intent IR: typed, serializable, framework-agnostic intermediate representation
- React + TypeScript frontend generator (pages, components, router, API client, types)
- Node + Express backend generator (Prisma schema, routes, auth middleware, policies, error handling)
- PostgreSQL generator (initial migration, seed data)
- Docker generator (Dockerfiles, docker-compose.yml, .env.example)
- Quality engine (test generation, security report, lint report, build report)
- Project scaffolder (package.json workspaces, tsconfigs, vite config, README, start.sh)
- CLI: `human build`, `human check`
- 228+ tests across the compiler
