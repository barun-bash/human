# Human Language Specification

> Definitive reference for the `.human` language, derived from the compiler source code (lexer, parser, IR builder, analyzer). Every syntax pattern documented here is accepted by the compiler.

**Version:** Matches compiler at commit `a81536b` (v0.4.1+)

---

## 1. File Structure

### Basics

- **File extension:** `.human`
- **Encoding:** UTF-8
- **Indentation:** Spaces (typically 2). Indentation defines block scope (like Python).
- **Keywords are case-insensitive:** `Page` = `page` = `PAGE`
- **Comments:** Lines starting with `#`
- **Strings:** Enclosed in double quotes (`"hello"`)
- **Numbers:** Integer (`42`, `500`) or decimal (`3.14`)
- **Colors:** Hex codes (`#6C5CE7`, `#ABC`)

### Section Headers (Decorative)

```
── theme ──
── frontend ──
── backend ──
```

Section headers use the `── name ──` format. They are **decorative only** — the parser records them but they have no semantic effect. You can use any name.

### Connector Words

The words `a`, `an`, `the`, `which`, `that`, `its`, `their` are **grammatical connectors**. The parser accepts them where expected but they are optional — they make your code read like English without changing meaning.

### Statement Order

Within a file, top-level blocks can appear in any order. Within a block, statement order doesn't affect semantics (but is preserved for readability).

---

## 2. All Block Types

The parser recognizes **16 top-level block types**. Each starts with a keyword at column 0.

### 2.1 `app` — Application Declaration

**Required.** Exactly one per file. Declares the application name and platform.

```
app TaskFlow is a web application
```

**Syntax:** `app <Name> is a <platform> application`

**Platforms:** `web`, `mobile`, `desktop`, `api`

The parser consumes: `app` → name → `is` → `a`/`an` → platform → rest of line.

---

### 2.2 `data` — Data Model

Declares a data entity with typed fields and relationships. Requires a colon and an indented body.

```
data User:
  has a name which is text
  has an email which is unique email
  has a password which is encrypted text
  has a role which is either "user" or "admin"
  has an optional bio which is text
  has a created datetime
  has many Task
```

**Field syntax (4 forms):**

| Form | Example | Result |
|------|---------|--------|
| Full form | `has a title which is text` | Field `title`, type `text` |
| With modifiers | `has an email which is unique email` | Field `email`, type `email`, modifier `unique` |
| Shorthand | `has a created datetime` | Field `created`, type `datetime` |
| Implicit type | `has a title` | Field `title`, type `text` (default) |

**Enum fields:**

```
has a status which is either "pending" or "in_progress" or "done"
```

**Modifiers** (placed before or after the field name):

| Modifier | Meaning |
|----------|---------|
| `optional` | Field is not required (placed before field name: `has an optional bio`) |
| `unique` | Value must be unique (placed after `which is`: `which is unique email`) |
| `encrypted` | Value is encrypted at rest (placed after `which is`: `which is encrypted text`) |

**Default values:**

```
has a status which defaults to "active"
```

**Relationships:**

| Pattern | Kind | Example |
|---------|------|---------|
| `belongs to a <Model>` | belongs_to | `belongs to a User` |
| `has many <Model>` | has_many | `has many Task` |
| `has many <Model> through <JoinModel>` | has_many_through | `has many Tag through TaskTag` |

**Auto-fields:** `id`, `created_at`, `updated_at` are auto-generated. Do NOT declare them.

**Multi-word field names** work naturally: `has a due date` creates field `due date` (normalized to `due_date` in generated code).

---

### 2.3 `page` — Frontend Page

Declares a page with display, interaction, and conditional statements.

```
page Dashboard:
  show a greeting with the user's name
  show a list of tasks sorted by due date
  each task shows its title, status, and due date
  clicking a task opens a detail panel
  there is a search bar that filters tasks by title
  if no tasks match, show "No tasks found"
  while loading, show a skeleton screen
  scrolling to bottom loads more tasks
```

**Display statements** (start with `show`, `each`, `display`):

```
show a hero section with the app name
show a list of published posts sorted by date
each task shows its title, status, and priority
show the user's name, email, and avatar
```

**Interaction statements** (start with `clicking`, `dragging`, `scrolling`, `hovering`, `typing`, `pressing`):

```
clicking a task navigates to TaskDetail
clicking "Save" updates the user profile
dragging a task reorders the list
scrolling to bottom loads more tasks
```

**Input elements** (start with `there is a`):

```
there is a search bar that filters tasks by title
there is a dropdown to filter by status
there is a form to create a Task
there is a file upload for avatar
there is a floating button to add a new task
there is a date range picker to filter by due date
```

**Conditional display** (start with `if`, `when`, `while`, `unless`):

```
if no tasks match, show "No tasks found"
if user is not logged in, show login button
while loading, show a skeleton screen
```

**Navigation:**

```
clicking the "Get Started" button navigates to Dashboard
```

The analyzer validates navigation targets — the page must exist (error E103).

---

### 2.4 `component` — Reusable UI Component

Declares a reusable component with typed props.

```
component TaskCard:
  accepts task as Task
  show the task title in bold
  show the status as a colored badge
  if task is overdue, show the due date in red
  clicking the card triggers on_click
```

**Props:** `accepts <name> as <Type>` — parsed into prop name and type. Multiple props separated by commas.

**Body:** Same statement types as `page` (display, interaction, conditional).

---

### 2.5 `api` — Backend API Endpoint

Declares a backend endpoint with auth, inputs, validation, and logic.

```
api CreateTask:
  requires authentication
  accepts title, description, status, priority, and due date
  check that title is not empty
  check that title is less than 200 characters
  check that due date is in the future
  create a Task with the given fields and current user as owner
  set status to "pending" if not provided
  respond with the created task
```

**Special statements:**

| Statement | Meaning |
|-----------|---------|
| `requires authentication` | Endpoint requires auth token (sets `auth: true` in IR) |
| `accepts <params>` | Comma/and-separated parameter list |
| `check that <field> is not empty` | Validation: required field |
| `check that <field> is a valid email` | Validation: email format |
| `check that <field> is at least N characters` | Validation: min length |
| `check that <field> is less than N characters` | Validation: max length |
| `check that <field> is not already taken` | Validation: unique check |
| `check that <field> is in the future` | Validation: future date |
| `check that <field> matches ...` | Validation: pattern match |
| `check that current user is the owner or an admin` | Authorization check |
| `respond with ...` | API response |
| `create a <Model>` | Create entity |
| `update the <Model>` | Update entity |
| `delete the <Model>` | Delete entity |
| `fetch the <Model> by <field>` | Query entity |

The analyzer validates that referenced models exist (error E104) and that if any API requires auth, an `authentication` block is defined (error E201).

---

### 2.6 `policy` — Authorization Rules

Declares permission and restriction rules for a role.

```
policy FreeUser:
  can create up to 50 tasks per month
  can view only their own tasks
  cannot delete completed tasks
  cannot export data
```

**Rules:** Each line starts with `can` (permission) or `cannot` (restriction). The rest of the line is the rule text.

---

### 2.7 `when` — Workflow / Pipeline

Declares an event-driven action sequence. Also used for CI/CD pipelines.

```
when a user signs up:
  create their account
  assign FreeUser policy
  send welcome email with template "welcome"
  after 3 days, send email with template "getting-started"
```

**Syntax:** `when <event description>:` followed by indented steps.

**CI/CD pipelines:** If the trigger starts with `code is pushed` or `code is merged`, the IR classifies it as a pipeline (not a workflow):

```
when code is pushed to a feature branch:
  run all tests
  check code formatting
  report results back to the pull request

when code is merged to main:
  run all tests
  build the application
  deploy to production environment
  if health checks fail, rollback automatically
```

---

### 2.8 `theme` — Visual Theme

Configures colors, fonts, spacing, border radius, dark mode, and design system.

```
theme:
  primary color is #6C5CE7
  secondary color is #00B894
  font is Inter for body and Poppins for headings
  border radius is smooth on all elements
  dark mode is supported and toggles from the header
  spacing is comfortable
  design system is shadcn
```

**Recognized properties:**

| Property | Syntax | Values |
|----------|--------|--------|
| Colors | `<name> color is <hex>` | Any name: primary, secondary, accent, danger |
| Fonts | `font is <Font> for <context>` | Context: body, headings. Multiple via `and` |
| Spacing | `spacing is <value>` | `compact`, `comfortable`, `spacious` |
| Border radius | `border radius is <value>` | `sharp`, `smooth`, `rounded`, `pill` (bare keyword only) |
| Dark mode | `dark mode is supported ...` | Any truthy description |
| Design system | `design system is <name>` | See section 6 |

---

### 2.9 `authentication` — Security Configuration

Configures auth methods, rate limiting, CORS, and security rules.

```
authentication:
  method JWT tokens that expire in 7 days
  method Google OAuth with redirect to /auth/google/callback
  passwords are hashed with bcrypt using 12 rounds
  all api requests require a valid token except SignUp and Login
  rate limit all endpoints to 100 requests per minute per user
  sanitize all text inputs against XSS
  enable CORS only for our frontend domain
```

**Auth methods:**

| Pattern | Type | Config |
|---------|------|--------|
| `method JWT tokens that expire in <duration>` | jwt | expiration |
| `method <Provider> OAuth with redirect to <url>` | oauth | provider, callback_url |

Other lines in the body become security rules (rate limiting, CORS, etc.).

---

### 2.10 `database` — Database Configuration

```
database:
  use PostgreSQL
  when the app starts, create tables if they don't exist
  index User by email
  index Task by user and status
  backup daily at 3am
  keep backups for 30 days
```

**Recognized statements:**

| Statement | Effect |
|-----------|--------|
| `use <engine>` | Sets database engine (PostgreSQL, MySQL, etc.) |
| `index <Model> by <field> [and <field>]` | Creates database index |
| `backup ...` | Backup configuration |
| `keep backups for ...` | Retention policy |

The analyzer validates that indexed models and fields exist (error E102).

---

### 2.11 `integrate with` — Third-Party Integrations

```
integrate with SendGrid:
  api key from environment variable SENDGRID_API_KEY
  use for sending transactional emails

integrate with AWS S3:
  api key from environment variable AWS_ACCESS_KEY
  secret from environment variable AWS_SECRET_KEY
  use for storing user avatars and file attachments

integrate with Slack:
  api key from environment variable SLACK_WEBHOOK_URL
  use for team notifications and alerts
```

**Syntax:** `integrate with <Service Name>:` — service name can be multiple words (e.g., `AWS S3`).

**Recognized statements:**

| Pattern | Extracted to |
|---------|-------------|
| `api key from environment variable <VAR>` | credentials |
| `secret from environment variable <VAR>` | credentials |
| `use for <purpose>` | purpose |
| `sender email is <value>` | config.sender_email |
| `region is <value>` | config.region |
| `bucket is <value>` | config.bucket |
| `webhook endpoint is <value>` | config.webhook_endpoint |
| `channel is <value>` | config.channel |
| `template "<name>"` | templates list |

**Auto-detected types:** The IR infers integration type from the service name:

| Service contains | Type |
|-----------------|------|
| SendGrid, Mailgun, SES, Postmark | email |
| S3, GCS, Cloudinary, Minio | storage |
| Stripe, PayPal, Braintree, Square | payment |
| Slack, Discord, Twilio, Telegram | messaging |
| Google, GitHub, Facebook, Auth0, Okta | oauth |

---

### 2.12 `environment` — Deployment Environments

```
environment staging:
  url is staging.taskflow.example.com
  uses staging database
  seeds with test data

environment production:
  url is taskflow.example.com
  uses production database
  requires manual approval for deployment
```

**Syntax:** `environment <name>:` followed by property statements. Properties using `<key> is <value>` are extracted as config key-value pairs.

---

### 2.13 `build with` — Build Configuration

```
build with:
  frontend using React with TypeScript
  backend using Node with Express
  database using PostgreSQL
  deploy to Docker
```

**Recognized statements:**

| Pattern | Field |
|---------|-------|
| `frontend using <framework>` | frontend |
| `backend using <framework>` | backend |
| `database using <engine>` | database |
| `deploy to <target>` | deploy |

**Frontend frameworks:** React, Vue, Angular, Svelte (+ TypeScript)
**Backend frameworks:** Node (Express), Python (FastAPI, Django), Go (Gin)
**Databases:** PostgreSQL, MySQL
**Deploy targets:** Docker, AWS Lambda, Vercel

---

### 2.14 `architecture` — Architecture Style

```
architecture: monolith
```

Or with service definitions for microservices:

```
architecture: microservices
  service UserService:
    handles user management
    owns User
    runs on port 3001
    has its own database
    talks to OrderService to check active orders

  service OrderService:
    handles order processing
    owns Order, OrderItem
    runs on port 3002

  gateway:
    routes /api/users to UserService
    routes /api/orders to OrderService
    handles rate limiting and CORS

  message broker using RabbitMQ
```

**Styles:** `monolith`, `microservices`, `serverless`, `event-driven` (normalized to `microservices`), `modular` (normalized to `monolith`)

**Service properties:** `handles`, `owns`/`manages`, `runs on port`, `has its own database`, `talks to`

**Gateway:** `routes <path> to <Service>`, plus rules for rate limiting/CORS.

---

### 2.15 `if` (top-level) — Error Handlers

Top-level `if` blocks define error recovery logic.

```
if database is unreachable:
  retry 3 times with 1 second delay
  if still failing, respond with "service temporarily unavailable"
  alert the engineering team via Slack

if an api request fails validation:
  respond with a clear message explaining what is wrong
  log the attempt for analytics
```

**Syntax:** `if <condition>:` at the top level (column 0).

Note: `if` inside a page/api/workflow body is a conditional statement, not an error handler.

---

### 2.16 Top-Level Statements

Several constructs live at the top level without their own block:

**Source control:**
```
source control using Git on GitHub
repository: https://github.com/example/taskflow
```

**Branches:**
```
branches:
  main is the production branch
  staging is the pre-release branch
  feature branches from main with prefix "feat/"
```

**Monitoring:**
```
track response times for all api endpoints
track error rates per endpoint
track active users daily and monthly
alert on Slack if error rate exceeds 5 percent
alert on Slack if response time exceeds 2 seconds
log all api requests to CloudWatch
keep logs for 90 days
```

---

## 3. Data Model Reference

### Field Types

| Type | Keyword | Description |
|------|---------|-------------|
| Text | `text` | String/varchar |
| Number | `number` | Integer |
| Decimal | `decimal` | Float/double |
| Boolean | `boolean` | True/false |
| Date | `date` | Date only |
| DateTime | `datetime` | Date + time |
| Email | `email` | Email address (validated) |
| URL | `url` | Web address |
| File | `file` | File upload/reference |
| Image | `image` | Image upload/reference |
| JSON | `json` | Arbitrary JSON data |
| Timestamp | `timestamp` | Unix timestamp |

### Type Inference

If no type is specified, the default is `text`:
```
has a title              # → type: text
has a count number       # → type: number (shorthand)
has a title which is text  # → type: text (explicit)
```

### Enum Fields

```
has a role which is either "user" or "admin" or "moderator"
```

Enums are stored as `type: "enum"` in the IR with an `enum_values` array.

### Relationships

**belongs_to** — Foreign key to another model:
```
data Task:
  belongs to a User
```

**has_many** — One-to-many:
```
data User:
  has many Task
```

**has_many_through** — Many-to-many via join table:
```
data Task:
  has many Tag through TaskTag

data Tag:
  has many Task through TaskTag

data TaskTag:
  belongs to a Task
  belongs to a Tag
```

The analyzer validates:
- Relation targets exist (E101)
- Through-models have belongs_to relations to both sides (E105)

### Database Indexes

```
database:
  index User by email
  index Task by user and status
  index Task by user and due date
```

The analyzer validates index targets and fields (E102).

---

## 4. Page & Component Reference

### Display Patterns

```
show a hero section with the app name
show a list of tasks sorted by due date
show a summary card with total tasks and completed tasks
show the user's name, email, and avatar
show the status as a colored badge
show account creation date in relative format like "joined 3 months ago"
```

### Iteration

```
each task shows its title, status, and due date
each pricing card shows the plan name, price, and features
```

### Interaction Subjects

| Keyword | Use |
|---------|-----|
| `clicking` | Click/tap events |
| `dragging` | Drag-and-drop |
| `scrolling` | Scroll events |
| `hovering` | Mouse hover |
| `typing` | Text input |
| `pressing` | Key press |

### Interaction Verbs

After the subject: `navigates to`, `opens`, `triggers`, `shows`, `loads`, `reorders`, `does`.

```
clicking a task navigates to TaskDetail
clicking "Save" updates the user profile
dragging a task reorders the list
scrolling to bottom loads more tasks
```

### Input Elements

All start with `there is a`:

```
there is a search bar that filters tasks by title
there is a dropdown to filter by status
there is a form to create a Task
there is a file upload for avatar
there is a floating button to add a new task
there is a date range picker to filter by due date
there is a text input for title
there is a rich text editor for content
there is a tag selector to add or remove Tags
there is a toggle for published status
```

### Component Props

```
component TaskCard:
  accepts task as Task
  accepts user as User, showActions as boolean
```

`accepts <name> as <Type>` — comma-separated for multiple.

---

## 5. API Statement Reference

### Authentication

```
requires authentication
```

Sets the endpoint's `auth` flag. Requires an `authentication` block to be defined in the file (validated by E201).

### Input Parameters

```
accepts title, description, status, priority, and due date
accepts name, email, and password
accepts post_id
```

Comma-separated, with optional `and` before the last parameter. Multi-word param names work.

### Validation Rules

All validation starts with `check that`:

| Pattern | Rule | Example |
|---------|------|---------|
| `is not empty` | not_empty | `check that title is not empty` |
| `is a valid email` | valid_email | `check that email is a valid email` |
| `is at least N characters` | min_length | `check that password is at least 8 characters` |
| `is less than N characters` | max_length | `check that title is less than 200 characters` |
| `is not already taken` | unique | `check that email is not already taken` |
| `is in the future` | future_date | `check that due date is in the future` |
| `matches ...` | matches | `check that password matches confirmation` |
| `current user is ...` | authorization | `check that current user is the owner or an admin` |

### CRUD Operations

```
create a User with the given fields
fetch the task by task_id
update the task with the given fields
delete the task
```

### Response

```
respond with the created task
respond with the user and auth token
respond with "task deleted"
respond with tasks and pagination info
```

### Other API Statements

```
set status to "pending" if not provided
assign "user" role
sort by due date
support filtering by status
paginate with 20 per page
send welcome email to the user
```

---

## 6. Build Configuration

### Frontend Frameworks

| Value | Framework |
|-------|-----------|
| `React with TypeScript` | React + TS |
| `Vue with TypeScript` | Vue + TS |
| `Angular` | Angular |
| `Svelte` | Svelte |

### Backend Frameworks

| Value | Framework |
|-------|-----------|
| `Node with Express` | Express.js |
| `Python with FastAPI` | FastAPI |
| `Python with Django` | Django |
| `Go with Gin` | Go Gin |

### Design Systems

7 supported design systems (specified in `theme` block):

| ID | Name | React | Vue | Angular | Svelte |
|----|------|-------|-----|---------|--------|
| `material` | Material UI | Yes | Yes (Vuetify) | Yes | Tailwind fallback |
| `shadcn` | Shadcn/ui | Yes | Yes | No | Yes |
| `ant` | Ant Design | Yes | Yes | Yes (ng-zorro) | No |
| `chakra` | Chakra UI | Yes | No | No | No |
| `bootstrap` | Bootstrap | Yes | Yes | Yes | Yes |
| `tailwind` | Tailwind CSS | Yes | Yes | Yes | Yes |
| `untitled` | Untitled UI | Yes | Yes | Yes | Yes |

**Aliases:** `MUI` → material, `Shadcn/ui` → shadcn, `Ant Design`/`antd` → ant, `Chakra UI` → chakra, `TailwindCSS` → tailwind.

If a design system doesn't support the chosen frontend framework, the compiler falls back to Tailwind CSS with the design system's color palette (warning W302).

---

## 7. Integrations

### Pattern

```
integrate with <Service Name>:
  <credential statements>
  <config statements>
  <purpose statement>
```

### Credential Pattern

```
api key from environment variable SENDGRID_API_KEY
secret from environment variable AWS_SECRET_KEY
```

### Known Services

| Service | Auto-detected Type |
|---------|-------------------|
| SendGrid, Mailgun, SES, Postmark, Mailchimp | email |
| AWS S3, GCS, Cloudinary, Minio | storage |
| Stripe, PayPal, Braintree, Square | payment |
| Slack, Discord, Twilio, Telegram | messaging |
| Google, GitHub, Facebook, Auth0, Okta | oauth |

The analyzer warns if an integration has no credentials (W501), except for local services like Ollama.

---

## 8. Infrastructure

### Environments

```
environment staging:
  url is staging.example.com
  uses staging database
  seeds with test data
  resets weekly

environment production:
  url is example.com
  uses production database
  requires manual approval for deployment
  has auto-scaling enabled
```

### Pipelines

Pipelines are `when` blocks whose trigger starts with `code is pushed` or `code is merged`:

```
when code is pushed to a feature branch:
  run all tests
  check code formatting

when code is merged to main:
  run all tests
  build the application
  deploy to production environment
```

### Monitoring

Top-level statements:

```
track response times for all api endpoints
track error rates per endpoint
alert on Slack if error rate exceeds 5 percent
log all api requests to CloudWatch
keep logs for 90 days
```

### Architecture

```
architecture: monolith
architecture: microservices
architecture: serverless
```

For microservices, define services, gateway, and message broker in the indented body (see section 2.14).

For serverless, at least one API must be defined (validated by E402).

---

## 9. Semantic Validation

The analyzer produces errors (compilation fails) and warnings (compilation continues).

### Errors

| Code | Description |
|------|-------------|
| **E101** | Data model references a model that does not exist (in relationships) |
| **E102** | Database index references a model or field that does not exist |
| **E103** | Page navigates to a page that does not exist |
| **E104** | API references a model that does not exist (in CRUD operations) |
| **E105** | Through-model missing required belongs_to relation to source or target |
| **E201** | API requires authentication but no `authentication` block is defined |
| **E202** | Build config specifies a database but no data models are defined |
| **E203** | Build config specifies a frontend but no pages are defined |
| **E301** | Duplicate data model name |
| **E302** | Duplicate page name |
| **E303** | Duplicate component name |
| **E304** | Duplicate API name |
| **E305** | Duplicate policy name |
| **E306** | Duplicate field name within a data model |
| **E401** | Microservices architecture declared but no services defined |
| **E402** | Serverless architecture declared but no APIs defined |
| **E501** | Duplicate integration (same service declared twice) |

### Warnings

| Code | Description |
|------|-------------|
| **W301** | Unknown design system (with suggestions) |
| **W302** | Design system has no library for chosen frontend framework (Tailwind fallback) |
| **W303** | Unknown spacing value (expected: compact, comfortable, spacious) |
| **W304** | Unknown border radius value (expected: sharp, smooth, rounded, pill) |
| **W401** | Unknown architecture style |
| **W402** | Service references a model that does not exist |
| **W403** | Service talks_to a service that does not exist |
| **W501** | Integration has no credentials configured |
| **W502** | Workflow sends email but no email integration is declared |
| **W503** | Workflow references Slack but no messaging integration is declared |

All errors and warnings include "did you mean?" suggestions when a close match is found (using Levenshtein distance).

---

## 10. What NOT to Write

### Never do these

| Wrong | Why | Right |
|-------|-----|-------|
| `import React` | No imports — the compiler handles all dependencies | (just use React in `build with`) |
| `let count = 0` | No variable declarations | `has a count which is number` |
| `for task in tasks:` | No for-loops | `each task shows its title` |
| `function validate():` | No function definitions | `api Validate:` |
| `{ ... }` or `;` | No braces or semicolons | Indentation-based scoping |
| `has an id which is number` | Don't declare `id` | Auto-generated |
| `has a created_at which is datetime` | Don't declare `created_at` | Auto-generated |
| `has an updated_at which is datetime` | Don't declare `updated_at` | Auto-generated |

### Common mistakes

1. **Forgetting the colon** after block headers:
   ```
   # Wrong:
   data User
     has a name

   # Right:
   data User:
     has a name
   ```

2. **Using `is` instead of `using` in build config:**
   ```
   # Works but non-standard:
   frontend is React

   # Preferred:
   frontend using React with TypeScript
   ```

3. **Declaring auto-fields:** `id`, `created_at`, `updated_at` are automatic. Declaring them creates duplicate fields.

4. **Missing through-model:** For many-to-many relationships, you must define the join model with `belongs to` on both sides:
   ```
   data TaskTag:
     belongs to a Task
     belongs to a Tag
   ```

5. **Referencing non-existent models:** The analyzer catches typos — always use the exact model name as declared.

---

## Appendix: Complete Token List

The lexer recognizes **215 token types** organized into categories:

**Structural:** EOF, NEWLINE, INDENT, DEDENT, COLON, COMMA, SECTION_HEADER, COMMENT

**Literals:** STRING_LIT, NUMBER_LIT, COLOR_LIT, IDENTIFIER, POSSESSIVE

**Declarations (17):** app, data, page, component, api, service, agent, policy, workflow, theme, architecture, environment, integrate, database, authentication, build, design

**Types (11):** text, number, decimal, boolean, date, datetime, email, url, file, image, json

**Actions (33):** show, fetch, create, update, delete, send, respond, navigate, check, validate, filter, sort, paginate, search, set, return, publish, listen, notify, alert, log, track, run, deploy, keep, backup, retry, rollback, index, enable, support, assign, use

**Conditions (8):** if, when, while, unless, until, after, before, every

**Connectors (23):** is, are, has, with, from, to, in, on, for, by, as, and, or, not, the, a, an, which, that, either, of, its, their, using, per, at

**Modifiers (8):** requires, accepts, only, each, all, optional, unique, encrypted

**Relationships (4):** belongs, many, through, defaults

**Interaction Subjects (6):** clicking, typing, hovering, pressing, scrolling, dragging

**Interaction Verbs (7):** does, navigates, opens, triggers, shows, loads, reorders

**Policy (3):** can, cannot, must

**Architecture (5):** monolith, microservices, serverless, gateway, broker

**DevOps (8):** pipeline, monitor, release, merge, push, source, repository, branches

**Other (14):** there, no, do, method, endpoint, except, rate, limit, uses, sanitize, variable, key
