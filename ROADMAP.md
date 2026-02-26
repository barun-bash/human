# Human Language — Development Roadmap

## Vision

Build the first programming language designed for humans, not computers.
English in → production-ready applications out.

---

## Phase 1: Foundation (Weeks 1-4) ✅
**Goal: Parse a .human file and prove the grammar works**

### Week 1-2: Lexer + Token System
- [x] Define all token types
- [x] Build lexer that tokenizes .human files
- [x] Handle indentation-based scoping
- [x] Handle section headers (── name ──)
- [x] Handle strings, numbers, identifiers
- [x] Handle keywords (case-insensitive)
- [x] Handle comments (#)
- [x] Write comprehensive lexer tests
- [x] Test with sample .human files

### Week 3-4: Parser + AST
- [x] Define all AST node types
- [x] Build recursive descent parser
- [x] Parse: app declaration
- [x] Parse: data declarations (fields, types, relationships)
- [x] Parse: page declarations (display, interaction, conditions)
- [x] Parse: api declarations (input, validation, logic, response)
- [x] Parse: theme declaration
- [x] Parse: security declaration
- [x] Parse: database declaration
- [x] Error recovery (skip to next declaration on error)
- [x] Human-readable error messages
- [x] Write comprehensive parser tests

### Milestone: ✅ Can parse a complete .human file into an AST

---

## Phase 2: Intent IR (Weeks 5-6) ✅
**Goal: Transform AST into framework-agnostic representation**

- [x] Define IR node types (Application, Data, Page, API, etc.)
- [x] Build AST → IR transformer
- [x] Serialize IR to YAML
- [x] Serialize IR to JSON
- [x] Deserialize IR from YAML/JSON
- [ ] Validate IR completeness (every reference resolves)
- [x] Write IR tests
- [ ] Document IR schema

### Milestone: ✅ .human file → AST → Intent IR (YAML) works end-to-end

---

## Phase 3: First Code Generator — React + TypeScript (Weeks 7-10) ✅
**Goal: Generate a working React project from Intent IR**

### Week 7-8: Project Scaffolding
- [x] Generate package.json with correct dependencies
- [x] Generate tsconfig.json
- [x] Generate project structure (src/, pages/, components/, etc.)
- [x] Generate routing from page declarations
- [x] Generate data types from data declarations
- [ ] Generate theme/styling from theme declaration (Tailwind)

### Week 9-10: Component + Page Generation
- [x] Generate React components from component declarations
- [x] Generate pages from page declarations
- [x] Generate data fetching (hooks, API calls)
- [x] Generate forms from input statements
- [x] Generate lists, cards, tables from display statements
- [x] Generate event handlers from interaction statements
- [x] Generate conditional rendering from condition statements
- [x] Generate loading and error states

### Milestone: ✅ .human file → React + TypeScript project (7 files generated)

---

## Phase 4: First Backend Generator — Node + Express (Weeks 11-14) ✅
**Goal: Generate a working backend from Intent IR**

- [x] Generate Express server setup
- [x] Generate API routes from api declarations
- [x] Generate input validation from check statements
- [x] Generate data models (Prisma)
- [x] Generate database migrations (PostgreSQL)
- [x] Generate authentication middleware (JWT)
- [ ] Generate authorization middleware (from policies)
- [x] Generate error handling
- [x] Connect frontend to backend (API client generation)

### Milestone: ✅ Full-stack app from .human files (15 backend files generated)

---

## Phase 5: Quality Engine (Weeks 15-18) ⚠️ Partial
**Goal: Mandatory quality guarantees**

### Week 15-16: Auto Test Generation
- [x] Generate unit tests for every API (Jest/Vitest)
- [x] Generate unit tests for every component (React Testing Library)
- [x] Generate edge case tests from field types
- [ ] Generate integration tests for API flows
- [ ] Coverage tracking and threshold enforcement

### Week 17: Security Audit
- [ ] Dependency vulnerability scanning
- [ ] Input sanitization verification
- [x] Auth/authz coverage check
- [x] Secret detection in generated code
- [x] Security report generation

### Week 18: Code Quality + QA Trail
- [x] Linting of generated code (ESLint config)
- [ ] Duplication detection
- [ ] Performance pattern detection
- [ ] QA test plan generation
- [ ] Traceability matrix generation

### Milestone: ⚠️ Quality checks enforced on every build (partial — core reports work, advanced analysis remaining)

---

## Phase 6: CLI + Developer Experience (Weeks 19-20) ✅
**Goal: Polished command-line interface**

- [x] `human init` — interactive project creation
- [x] `human build` — compile to target
- [x] `human run` — development server
- [x] `human check` — validate .human files
- [x] `human test` — run generated tests
- [x] `human audit` — run security audit
- [x] `human deploy` — deploy to target (Docker, Terraform/AWS/GCP)
- [x] `human eject` — export as standalone project
- [x] Colored terminal output with 6 themes
- [x] Progress indicators (spinners, progress boxes)
- [x] Watch mode (`human build --watch`)
- [x] Interactive REPL with 20+ commands
- [x] Readline with tab completion and command history
- [x] AI-assisted commands (`/ask`, `/edit`, `/suggest`)
- [x] Self-update via GitHub releases (`/update`)
- [x] Build metadata embedding via ldflags

### Milestone: ✅ Complete developer experience from init to deploy, plus interactive REPL

---

## Phase 7: Design Import (Weeks 21-24) 🔄
**Goal: Figma and image files as input**

- [ ] Figma API integration (extract layers, styles, components)
- [ ] Image analysis for screenshots/JPEGs (requires LLM vision)
- [x] Visual → component mapping (heuristic classifier by name, structure, type)
- [x] Style extraction (colors, fonts, spacing) — theme extraction from Figma nodes
- [ ] Asset extraction (images, icons)
- [ ] Component map caching (deterministic after first import)
- [ ] Design enrichment via English statements
- [x] Data model inference from forms, cards, and tables
- [x] Complete `.human` source generation from design elements (pages, models, CRUD APIs)
- [x] LLM prompt generation for assisted design-to-code workflows
- [x] Figma demo example (`examples/figma-demo/app.human`) — SaaS analytics dashboard

### Milestone: 🔄 Figma → Human mapping intelligence complete; Figma API integration remaining

---

## Phase 8: Multi-Framework Output (Weeks 25-26) ✅
**Goal: Prove the IR is truly framework-agnostic**

- [x] Build Angular or Vue code generator
- [x] Same .human file produces both React and Angular/Vue
- [x] All quality checks work for new target
- [ ] Document how to build a code generator plugin

### Milestone: ✅ One .human source → multiple framework outputs (Vue, Angular, Svelte, Go generators wired)

---

## Phase 9: DevOps Generation (Weeks 27-30) ✅
**Goal: CI/CD, Docker, deployment from English**

- [x] Generate GitHub Actions workflows from pipeline declarations
- [x] Generate Dockerfiles and docker-compose
- [x] Generate Terraform for cloud deployment (AWS ECS/RDS, GCP Cloud Run/SQL)
- [x] Generate environment configurations (.env.example)
- [x] Project scaffolder (package.json workspaces, tsconfigs, vite config, README, start.sh)
- [ ] Git workflow commands (human feature, human release, etc.)
- [x] Monitoring configuration generation (Prometheus rules, Grafana dashboards)

### Milestone: ✅ Complete application lifecycle from .human files

---

## Phase 10: Architecture Support (Weeks 31-34) ✅
**Goal: Monolith, microservices, serverless from one keyword**

- [x] Monolith output (single project)
- [x] Microservices output (multiple projects, gateway, docker-compose)
- [x] Serverless output (Terraform Lambda/Cloud Functions)
- [x] Event-driven output (with message broker config)
- [x] Service-to-service communication generation

### Milestone: ✅ Architecture as a first-class language feature

---

## Phase 11: LLM Connector (Weeks 35-38) ✅
**Goal: Optional AI enhancement**

- [x] LLM connector interface (provider-agnostic)
- [x] Anthropic Claude integration
- [x] OpenAI integration
- [x] Ollama (local) integration
- [x] MCP server with 6 tools (human_build, human_validate, human_ir, human_examples, human_spec, human_read_file)
- [x] LLM system prompt for .human generation
- [ ] Smart interpretation (freeform → structured .human)
- [ ] Conversational editing mode (`human edit --with-llm`)
- [ ] Context building (project-wide understanding)
- [ ] Pattern suggestions (`human suggest`)

### Milestone: ✅ AI-enhanced editing while maintaining deterministic compilation

---

## Phase 12: Third-Party Integrations (Weeks 39-42) ✅
**Goal: First-class integration support**

- [x] Integration declaration parser
- [x] Built-in: Stripe (payments)
- [x] Built-in: SendGrid / email
- [x] Built-in: AWS S3 / file storage
- [x] Built-in: OAuth providers (Google, GitHub)
- [x] Built-in: Slack (messaging)
- [x] Typed service files, env vars, and dependencies per integration
- [ ] Custom API integration from declarations
- [ ] OpenAPI/Swagger spec import
- [ ] Integration test generation (mocked)

### Milestone: ✅ Third-party APIs usable in English

---

## Phase 13: Plugin Ecosystem (Weeks 43-46)
**Goal: Community can extend the language**

- [ ] Plugin interface specification
- [ ] Plugin loading system
- [ ] Plugin discovery and installation
- [ ] Plugin template / generator
- [ ] Documentation for plugin authors
- [ ] First community plugin examples

### Milestone: Open ecosystem for new targets and integrations

---

## Phase 14: Polish + Launch (Weeks 47-52)
**Goal: Ready for public release**

- [x] Project website (6 pages: landing, getting started, language spec, roadmap, manifesto, contributing)
- [ ] Tutorial: "Build your first app in Human"
- [x] 12 example applications (taskflow, blog, ecommerce, saas, api-only, test-app, recipes, projects, events, fitness, inventory, figma-demo)
- [ ] Performance optimization
- [x] Semantic error messages with codes and fix suggestions (17 errors, 10 warnings)
- [x] Installation scripts (install.sh, Homebrew formula, GoReleaser cross-platform binaries)
- [x] VS Code extension (syntax highlighting, snippets, bracket matching)
- [x] Landing page and branding
- [x] Open source release on GitHub
- [x] Language specification derived from compiler source
- [x] LLM system prompt for .human file generation

### Milestone: v1.0 public launch

---

## Current Output

Running `human build examples/taskflow/app.human` produces **85+ files** (React + Node stack). File count varies by stack:

| Generator | Files | Output |
|-----------|-------|--------|
| React + TypeScript | 13 | Types, API client, pages, components, router, Vite config, Tailwind |
| Vue + TypeScript | 16 | Components, pages, router, API, types, Pinia stores |
| Angular + TypeScript | 20 | Components, services, routing, signals, environments |
| Svelte + TypeScript | 19 | Pages, components, stores, SvelteKit routing |
| Node + Express | 19 | Prisma schema, auth + authorize middleware, policies, route handlers, integration services, server |
| Python + FastAPI | 11 | SQLAlchemy models, routes, auth, Alembic migrations, main app |
| Go + Gin | 10 | Handlers, routes, GORM models, auth middleware, go.mod |
| Docker + Compose | 5 | Dockerfiles, docker-compose.yml, .env.example, package.json |
| PostgreSQL | 2 | Migration (001_initial.sql), seed data |
| CI/CD | 6 | GitHub Actions workflows (test, build, deploy per environment) |
| Terraform | 10 | AWS ECS/RDS or GCP Cloud Run/SQL modules, variables, outputs |
| Monitoring | 8 | Prometheus alert rules, Grafana dashboard JSON |
| Architecture | 2-10 | Service topology, deployment diagrams (varies by architecture) |
| Quality Engine | 15+ | Test files, security-report.md, lint-report.md, build-report.md |
| Scaffold | 6-9 | Root package.json, stack-specific configs, README, start.sh |
| Storybook | varies | Component stories with relational mock data |

All generators are wired into the CLI. The scaffolder adapts to the selected stack — only files relevant to the chosen frontend/backend are generated.

### Example Gallery (12 applications)

| Example | Frontend | Backend | Design System | Unique Coverage |
|---------|----------|---------|---------------|-----------------|
| taskflow | React | Node | — | Reference example |
| blog | Vue | Python | — | CMS with nested comments |
| ecommerce | Angular | Go | — | Microservices architecture |
| saas | Svelte | Node | Shadcn | Serverless + tiered pricing |
| recipes | React | Node | Tailwind | Community content + favorites |
| projects | React | Node | Shadcn | Kanban board + teams |
| api-only | — | Node | — | Pure API (no frontend) |
| test-app | React | Node | — | Minimal test target |
| fitness | Vue | Python | Material | Vuetify integration |
| events | Angular | Node | Ant Design | ng-zorro-antd integration |
| inventory | React | Go | Chakra | @chakra-ui/react integration |
| figma-demo | React | Python | Untitled UI | Figma→Human translation demo |

### MCP Server

The MCP server (`cmd/human-mcp/`) exposes 6 tools over JSON-RPC 2.0 (stdin/stdout) for Claude Desktop integration:

| Tool | Description |
|------|-------------|
| `human_build` | Compile .human source through the full pipeline |
| `human_validate` | Validate without code generation, return diagnostics |
| `human_ir` | Parse and return Intent IR as YAML |
| `human_examples` | List or retrieve example .human applications |
| `human_spec` | Return the complete language specification |
| `human_read_file` | Read a file from the last build output |

---

## What's Next — v0.5.0

- 🔄 **Figma API integration** — connect the mapping intelligence (`internal/figma/`) to live Figma files via API
- 🔄 **Runtime correctness hardening** — end-to-end `docker compose up` validation, `tsc --noEmit` clean across all stacks
- 🔜 **Display statement intelligence** — smarter JSX/template generation from natural language descriptions
- 🔜 **Plugin system** — community-extensible code generators and integration adapters
- 🔜 **Human Cloud** — hosted builds (upload `.human`, get deployed app)

---

## Success Metrics

| Metric | Target | Status |
|---|---|---|
| Parse a .human file | Phase 1 | ✅ Done |
| Framework-agnostic IR | Phase 2 | ✅ Done |
| Generate a running React app | Phase 3 | ✅ Done |
| Full-stack app from English | Phase 4 | ✅ Done |
| Quality guarantees enforced | Phase 5 | ⚠️ Partial |
| CLI developer experience | Phase 6 | ⚠️ Partial |
| Design-to-code pipeline | Phase 7 | 🔄 In Progress |
| Multi-framework output (4 frontend + 3 backend) | Phase 8 | ✅ Done |
| DevOps generation (Docker, CI/CD, Terraform) | Phase 9 | ✅ Done |
| Architecture support (monolith, microservices, serverless) | Phase 10 | ✅ Done |
| LLM connector (Anthropic, OpenAI, Ollama) | Phase 11 | ✅ Done |
| Third-party integrations (Stripe, SendGrid, S3, OAuth, Slack) | Phase 12 | ✅ Done |
| Plugin ecosystem | Phase 13 | Upcoming |
| Public launch | Phase 14 | 🔄 In Progress |

---

## Principles Throughout

1. **Every phase produces something that works.** No phase is "just infrastructure."
2. **Test everything.** The compiler for a quality-enforced language must itself be well-tested.
3. **Error messages are a feature.** Every error helps the developer fix the problem.
4. **Determinism is sacred.** Same input, same output, always.
5. **Ship examples.** Every feature comes with a .human example that demonstrates it.
