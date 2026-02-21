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

### Milestone: ✅ Full-stack app from .human files (13 backend files generated)

---

## Phase 5: Quality Engine (Weeks 15-18) ✅
**Goal: Mandatory quality guarantees**

### Week 15-16: Auto Test Generation
- [x] Generate unit tests for every API (Jest/Vitest)
- [ ] Generate unit tests for every component (React Testing Library)
- [ ] Generate edge case tests from field types
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

### Milestone: Nothing builds without passing all quality checks

---

## Phase 6: CLI + Developer Experience (Weeks 19-20)
**Goal: Polished command-line interface**

- [x] `human init` — interactive project creation
- [x] `human build` — compile to target
- [x] `human run` — development server
- [x] `human check` — validate .human files
- [x] `human test` — run generated tests
- [x] `human audit` — run security audit
- [ ] `human deploy` — deploy to target
- [x] `human eject` — export as standalone project
- [x] Colored terminal output
- [ ] Progress indicators
- [x] Watch mode (rebuild on file change)

### Milestone: Complete developer experience from init to deploy

---

## Phase 7: Design Import (Weeks 21-24)
**Goal: Figma and image files as input**

- [ ] Figma API integration (extract layers, styles, components)
- [ ] Image analysis for screenshots/JPEGs (requires LLM vision)
- [ ] Visual → component mapping
- [ ] Style extraction (colors, fonts, spacing)
- [ ] Asset extraction (images, icons)
- [ ] Component map caching (deterministic after first import)
- [ ] Design enrichment via English statements

### Milestone: Figma file + English → working React components

---

## Phase 8: Second Frontend Target (Weeks 25-26)
**Goal: Prove the IR is truly framework-agnostic**

- [ ] Build Angular or Vue code generator
- [ ] Same .human file produces both React and Angular/Vue
- [ ] All quality checks work for new target
- [ ] Document how to build a code generator plugin

### Milestone: One .human source → multiple framework outputs

---

## Phase 9: DevOps Generation (Weeks 27-30) — Started
**Goal: CI/CD, Docker, deployment from English**

- [ ] Generate GitHub Actions workflows from pipeline declarations
- [x] Generate Dockerfiles and docker-compose
- [ ] Generate Terraform for cloud deployment
- [x] Generate environment configurations (.env.example)
- [x] Project scaffolder (package.json workspaces, tsconfigs, vite config, README, start.sh)
- [ ] Git workflow commands (human feature, human release, etc.)
- [ ] Monitoring configuration generation

### Milestone: Complete application lifecycle from .human files

---

## Phase 10: Architecture Support (Weeks 31-34)
**Goal: Monolith, microservices, serverless from one keyword**

- [ ] Monolith output (single project)
- [ ] Microservices output (multiple projects, gateway, docker-compose)
- [ ] Serverless output (Lambda/Cloud Functions)
- [ ] Event-driven output (with message broker config)
- [ ] Service-to-service communication generation

### Milestone: Architecture as a first-class language feature

---

## Phase 11: LLM Connector (Weeks 35-38)
**Goal: Optional AI enhancement**

- [ ] LLM connector interface (provider-agnostic)
- [ ] Anthropic Claude integration
- [ ] OpenAI integration
- [ ] Ollama (local) integration
- [ ] Smart interpretation (freeform → structured .human)
- [ ] Conversational editing mode
- [ ] Context building (project-wide understanding)
- [ ] Pattern suggestions

### Milestone: AI-enhanced editing while maintaining deterministic compilation

---

## Phase 12: Third-Party Integrations (Weeks 39-42)
**Goal: First-class integration support**

- [ ] Integration declaration parser
- [ ] Built-in: Stripe
- [ ] Built-in: SendGrid / email
- [ ] Built-in: AWS S3 / file storage
- [ ] Built-in: OAuth providers (Google, GitHub, etc.)
- [ ] Custom API integration from declarations
- [ ] OpenAPI/Swagger spec import
- [ ] Integration test generation (mocked)

### Milestone: Third-party APIs usable in English

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

- [ ] Comprehensive documentation website
- [ ] Tutorial: "Build your first app in Human"
- [ ] 5+ complete example applications
- [ ] Performance optimization
- [ ] Error message review (every error is helpful)
- [ ] Installation scripts (brew, apt, scoop, etc.)
- [ ] VS Code extension (syntax highlighting, autocomplete)
- [x] Landing page and branding
- [ ] Open source release

### Milestone: v1.0 public launch

---

## Current Output

Running `human build examples/taskflow/app.human` produces **45 files**:

| Generator | Files | Output |
|-----------|-------|--------|
| React + TypeScript | 7 | Types, API client, pages, components, router |
| Node + Express | 13 | Prisma schema, auth middleware, error handler, route files, server |
| Docker + Compose | 5 | Dockerfiles, docker-compose.yml, .env.example, package.json |
| PostgreSQL | 2 | Migration (001_initial.sql), seed data |
| Quality Engine | 11 | 8 test files, security-report.md, lint-report.md, build-report.md |
| Scaffold | 7 | Root + node + react package.json, tsconfigs, vite config, README, start.sh |

The scaffolder overwrites Docker's root `package.json` (adding npm workspaces) and `.env.example` (identical content), so 9 files are written but 2 replace existing ones.

---

## Success Metrics

| Metric | Target | Status |
|---|---|---|
| Parse a .human file | Phase 1 | ✅ Done |
| Generate a running React app | Phase 3 | ✅ Done |
| Full-stack app from English | Phase 4 | ✅ Done |
| Docker deployment config | Phase 9 | ✅ Done |
| Project scaffolder (runnable output) | Phase 9 | ✅ Done |
| Quality guarantees enforced | Phase 5 | ✅ Done |
| Design-to-code pipeline | Phase 7 | Upcoming |
| Multi-framework output | Phase 8 | Upcoming |
| Public launch | Phase 14 | Upcoming |

---

## Principles Throughout

1. **Every phase produces something that works.** No phase is "just infrastructure."
2. **Test everything.** The compiler for a quality-enforced language must itself be well-tested.
3. **Error messages are a feature.** Every error helps the developer fix the problem.
4. **Determinism is sacred.** Same input, same output, always.
5. **Ship examples.** Every feature comes with a .human example that demonstrates it.
