# Human Language Specification v0.1

## Overview

**Human** is a natural language programming language that compiles structured English and design files into production-ready full-stack applications. It is deterministic, target-agnostic, and enforces mandatory quality guarantees.

**File extension:** `.human`
**CLI command:** `human`
**Compiler language:** Go

---

## 1. Core Principles

1. **English is the syntax.** If you can describe what you want in structured English, you can build it.
2. **Deterministic compilation.** Same `.human` file always produces the same output. No randomness.
3. **Target-agnostic.** One source, any output framework. The developer chooses the target.
4. **Quality is mandatory.** Tests, security audit, code quality, and QA trail are compiler-enforced. Cannot be skipped.
5. **Design-aware.** Figma files, images, and screenshots are first-class inputs alongside English.
6. **Ejectable.** Generated code is clean, readable, and fully owned by the developer.
7. **LLM-optional.** Core compiler works offline with no AI dependency. LLM connector available as enhancement.

---

## 2. File Structure

A Human project has the following structure:

```
my-app/
├── app.human              # Main application definition
├── frontend.human         # UI pages, components, themes
├── backend.human          # APIs, data, logic, security
├── devops.human           # Architecture, CI/CD, deployment
├── integrations.human     # Third-party API connections
├── designs/               # Figma files, images, screenshots
│   ├── homepage.figma
│   └── dashboard.png
├── assets/                # Static assets (images, fonts, icons)
├── human.config            # Project configuration
└── .human/                # Compiler cache and generated IR
    └── intent/            # Intermediate representation files
```

### human.config

```
name: my-app
version: 0.1.0

target:
  frontend: react with typescript
  backend: node with express
  database: postgresql
  deploy: vercel

options:
  test_coverage_minimum: 90
  strict_security: true
  accessibility: WCAG-AA
```

---

## 3. Language Grammar

### 3.1 Top-Level Declarations

Every `.human` file begins with declarations that define what things exist.

#### Application Declaration
```
app <Name> is a <platform> application

platform := "web" | "mobile" | "desktop" | "api"
```

Examples:
```
app FinanceTracker is a web application
app FitnessPal is a mobile application
app PhotoEditor is a desktop application
app PaymentGateway is an api application
```

#### Section Declaration
```
── <section_name> ──
```

Sections organize code within a file. Recognized sections:
- `frontend`, `backend`, `devops`, `integrations`
- `logic`, `security`, `database`, `policies`
- `theme`, `workflows`, `monitoring`
- `pipeline`, `environments`

---

### 3.2 Data Declarations

Data declarations define the entities/models in your application.

```
data <Name>:
  has a <field_name> which is <type>
  has a <field_name> which is <type>
  belongs to a <OtherData>
  has many <OtherData>
```

#### Field Types

| Human Type | Meaning | Example |
|---|---|---|
| `text` | String | `has a name which is text` |
| `number` | Integer or float | `has an age which is number` |
| `decimal` | Precise decimal | `has a price which is decimal` |
| `boolean` | True/false | `has an active flag which is boolean` |
| `date` | Date only | `has a birthday which is date` |
| `datetime` | Date and time | `has a created datetime` |
| `email` | Email (auto-validated) | `has an email which is email` |
| `url` | URL (auto-validated) | `has a website which is url` |
| `file` | File upload | `has an avatar which is file` |
| `image` | Image file | `has a photo which is image` |
| `json` | Arbitrary JSON | `has metadata which is json` |

#### Field Modifiers

```
has a <field> which is <type>              # required by default
has an optional <field> which is <type>    # nullable
has a <field> which is unique <type>       # unique constraint
has a <field> which is encrypted <type>    # encrypted at rest
has a <field> which is either "a" or "b"   # enum
has a <field> which defaults to <value>    # default value
```

#### Relationships

```
belongs to a <Data>                 # foreign key, many-to-one
has many <Data>                     # one-to-many
has many <Data> through <JoinData>  # many-to-many
```

#### Full Example

```
data User:
  has a name which is text
  has an email which is unique email
  has a password which is encrypted text
  has a role which is either "user" or "admin" or "moderator"
  has an optional bio which is text
  has a created datetime
  has many Post
  has many Comment

data Post:
  belongs to a User
  has a title which is text
  has a body which is text
  has a status which is either "draft" or "published" or "archived"
  has a published date
  has many Comment
  has many Tag through PostTag

data Comment:
  belongs to a User
  belongs to a Post
  has a body which is text
  has a created datetime
```

---

### 3.3 Frontend Declarations

#### Page Declaration

```
page <Name>:
  <content_statements>
```

Content statements describe what appears on the page and how it behaves.

##### Display Statements

```
show <what>                                    # render something
show a list of <data>                          # render a collection
show each <item>'s <field> and <field>         # specify fields
show <data> in a <layout>                      # specify layout
show "static text"                             # static content
show a <element> with <properties>             # specific element
```

Layouts: `card`, `table`, `grid`, `list`, `row`, `column`, `form`

##### Interaction Statements

```
clicking <element> does <action>               # click handler
clicking <element> navigates to <page>         # navigation
clicking <element> opens <thing>               # modal/panel/link
typing in <element> does <action>              # input handler
hovering over <element> shows <thing>          # hover effect
pressing <key> does <action>                   # keyboard shortcut
scrolling to bottom loads more <data>          # infinite scroll
dragging <element> reorders the list           # drag and drop
```

##### Input Statements

```
there is a text input for <purpose>            # text field
there is a search bar that filters <data>      # search + filter
there is a dropdown to select <options>        # select
there is a checkbox for <purpose>              # checkbox
there is a date picker for <purpose>           # date input
there is a file upload for <purpose>           # file input
there is a form to create <data>               # auto-form
```

##### Conditional Display

```
if <condition>, show <thing>                   # conditional render
if no <data> match, show "<message>"           # empty state
while loading, show a spinner                  # loading state
if there is an error, show the error message   # error state
```

#### Full Page Example

```
page Dashboard:
  show a greeting with the user's name
  show a summary card with total income and total expenses this month
  
  show a list of recent transactions sorted by date newest first
  each transaction shows its title, amount, category, and date
  each transaction shows the amount in green if income, red if expense
  clicking a transaction opens a detail panel on the right
  
  there is a search bar that filters transactions by title
  there is a dropdown to filter by category
  there is a date range picker to filter by date
  
  if no transactions match, show "No transactions found"
  while loading, show a skeleton screen
  
  there is a floating button to add a new transaction
  clicking the add button opens a form to create a Transaction
```

#### Component Declaration

Reusable UI pieces.

```
component <Name>:
  accepts <prop> as <type>
  <content_statements>
```

Example:

```
component TransactionCard:
  accepts transaction as Transaction
  show the transaction title in bold
  show the amount aligned right
  show the category as a colored badge
  show the date in relative format like "2 hours ago"
  clicking the card triggers on_click
```

#### Design Import

```
design <name> from "<file_path>"
  <enrichment_statements>
```

Example:

```
design dashboard from "designs/dashboard.figma"
  the sidebar is a shared component across all pages
  the chart section uses recharts
  the transaction list is scrollable and loads more on scroll
  make the layout responsive for mobile and tablet
```

#### Theme Declaration

```
theme:
  primary color is <color>
  secondary color is <color>
  font is <font> for body and <font> for headings
  border radius is <style>
  dark mode is supported
  spacing is <density>
```

Example:

```
theme:
  primary color is #6C5CE7
  secondary color is #00B894
  danger color is #D63031
  font is Inter for body and Poppins for headings
  border radius is smooth on all elements
  dark mode is supported and toggles from the header
  spacing is comfortable
```

---

### 3.4 Backend Declarations

#### API Declaration

```
api <Name>:
  accepts <fields>
  requires <precondition>
  <logic_statements>
  respond with <response>
```

Logic statements:

```
check that <validation>                        # validation
if <condition>, respond with "<message>"       # conditional response
create <data>                                  # insert
update <data>                                  # update
delete <data>                                  # delete
fetch <data> from <source>                     # query
send <notification>                            # side effect
respond with <data>                            # return
```

Example:

```
api CreatePost:
  requires authentication
  accepts title, body, and category
  check that title is not empty
  check that title is less than 200 characters
  check that body is not empty
  check that category is a valid category
  create a Post with the given fields and current user as author
  set status to "draft"
  respond with the created post

api ListPosts:
  requires authentication
  return all posts where status is "published"
  sort by published date newest first
  support filtering by category
  support searching by title
  paginate with 20 per page
  respond with posts and pagination info

api DeletePost:
  requires authentication
  accepts post_id
  fetch the post by post_id
  if post does not exist, respond with "post not found"
  check that current user is the author or an admin
  delete the post
  respond with "post deleted"
```

#### Security Declaration

```
authentication:
  method <auth_method>
  <auth_rules>
```

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

#### Policy Declaration

```
policy <Name>:
  can <permission>
  cannot <restriction>
```

Example:

```
policy FreeUser:
  can create up to 50 posts per month
  can view only their own posts
  can edit only their own posts
  cannot delete published posts
  cannot export data

policy Admin:
  can view all users and their data
  can edit any post
  can delete any post
  can export data
  can view system analytics
```

#### Database Declaration

```
database:
  use <database_type>
  <database_rules>
```

Example:

```
database:
  use PostgreSQL
  when the app starts, create tables if they don't exist
  index User by email
  index Post by user and published date
  index Comment by post and created date
  backup daily at 3am
  keep backups for 30 days
```

#### Workflow Declaration

```
when <event>:
  <action_sequence>
```

Example:

```
when a user signs up:
  create their account
  assign FreeUser policy
  send welcome email with template "welcome"
  after 3 days, send email with template "getting-started"
  after 14 days, if they have fewer than 5 posts,
    send email with template "need-help"

when a post is published:
  notify all followers of the author
  add to the public feed
  update the author's post count
  index for search
```

#### Error Handling

```
if <error_condition>:
  <recovery_actions>
```

Example:

```
if database is unreachable:
  retry 3 times with 1 second delay
  if still failing, respond with "service temporarily unavailable"
  alert the engineering team via Slack

if an api request fails validation:
  respond with a clear message explaining what is wrong
  log the attempt for analytics
  do not reveal internal details
```

---

### 3.5 Integration Declarations

```
integrate with <ServiceName>:
  api key from environment variable <VAR_NAME>
  use for <purpose>

integrate with custom api "<Name>":
  base url from environment variable <VAR_NAME>
  authentication using <method>
  endpoint <Name>:
    method <HTTP_METHOD> to <path>
    sends <fields>
    returns <fields>
```

Example:

```
integrate with Stripe:
  api key from environment variable STRIPE_KEY
  use for payment processing

integrate with custom api "InventoryService":
  base url from environment variable INVENTORY_API_URL
  authentication using api key in header "X-API-Key"
  
  endpoint CheckStock:
    method GET to /stock/{product_id}
    returns quantity as number and warehouse as text
```

---

### 3.6 Architecture Declaration

```
architecture: <type>
```

Types: `monolith`, `modular monolith`, `microservices`, `event-driven microservices`, `serverless`, `hybrid`

#### Monolith

```
architecture: monolith
```

#### Microservices

```
architecture: microservices

  service <Name>:
    handles <responsibilities>
    runs on port <number>
    has its own database
    talks to <OtherService> to <purpose>

  gateway:
    routes <path> to <Service>
    handles rate limiting and CORS
```

#### Event-Driven

```
architecture: event-driven microservices
  message broker using <Broker>

  service <Name>:
    publishes "<event>" when <condition>
    listens for "<event>" and <action>
```

#### Serverless

```
architecture: serverless
  each api endpoint runs as an independent function
  scale automatically based on traffic
```

---

### 3.7 DevOps Declarations

#### Source Control

```
source control using Git on <Provider>
repository: <url>

branches:
  <branch> is the <purpose> branch
  <pattern> branch from <base> with prefix "<prefix>"
```

#### Pipeline / CI/CD

```
when <git_event>:
  <pipeline_actions>
```

Pipeline actions:

```
run all tests
check code formatting
check for security vulnerabilities
build the application
deploy to <environment>
run smoke tests against <environment>
run health checks
if <check> fails, rollback automatically
notify the team on <channel>
report results back to the pull request
```

#### Environments

```
environment <name>:
  url is <domain>
  uses <env> database
  <environment_rules>
```

#### Monitoring

```
track <metric>
alert on <channel> if <condition>
log <what> to <service>
keep logs for <duration>
```

---

### 3.8 Build Target Declaration

```
build with:
  frontend using <framework> with <language>
  backend using <language> with <framework>
  database using <database>
  deploy to <platform>
```

#### Supported Targets (v1)

**Frontend:**
- React with TypeScript
- Angular with TypeScript
- Vue with TypeScript
- Svelte with TypeScript
- HTMX with vanilla JavaScript

**Backend:**
- Node with Express
- Node with Fastify
- Python with FastAPI
- Python with Django
- Go with Gin
- Go with Fiber
- Rust with Axum

**Database:**
- PostgreSQL
- MySQL
- MongoDB
- SQLite
- Supabase

**Deploy:**
- Vercel
- AWS (Lambda + API Gateway)
- GCP (Cloud Run)
- Docker (self-hosted)
- Kubernetes

---

## 4. Mandatory Quality System

These are not optional. The compiler enforces all four pillars on every build.

### 4.1 Automatic Tests

Every data, api, page, and component declaration generates:

- **Unit tests**: Each stated behavior, each validation rule, each conditional branch
- **Edge case tests**: Empty inputs, boundary values, special characters, unicode
- **Integration tests**: API-to-database flows, service-to-service contracts
- **Frontend tests**: Render, interaction, accessibility, responsive breakpoints

Minimum coverage: configurable (default 90%). Build fails below threshold.

### 4.2 Security Audit

Every build runs:

- Dependency vulnerability scan (block on critical CVE)
- Input sanitization verification (SQL injection, XSS)
- Authentication/authorization check (every protected route)
- Secret detection (no hardcoded keys, tokens, passwords)
- Infrastructure security (HTTPS, CORS, headers, encryption)

### 4.3 Code Quality

Every build ensures:

- Consistent formatting (no configuration needed)
- No unused code, no type errors, no unsafe patterns
- Duplication detection with refactoring suggestions
- Performance pattern detection (N+1 queries, missing indexes, unbounded fetches)
- Accessibility compliance (WCAG level configurable)

### 4.4 QA Trail

Every feature generates:

- Test plan derived from `.human` specifications
- Test execution records per build
- Regression tracking across versions
- Traceability matrix: requirement → tests → security → QA

---

## 5. Intent IR (Intermediate Representation)

The Intent IR is the framework-agnostic representation between Human source and generated code. It is a typed, structured, serializable format (YAML/JSON).

### IR Node Types

```
Application       # root node
├── Config        # build targets, options
├── Data[]        # entity definitions
│   ├── Field[]   # field definitions with types
│   └── Relation[] # relationships
├── Page[]        # frontend pages
│   ├── DataBinding[]  # data sources
│   ├── Layout[]       # visual structure
│   ├── Interaction[]  # event handlers
│   └── Condition[]    # conditional rendering
├── Component[]   # reusable UI pieces
├── API[]         # backend endpoints
│   ├── Auth        # authentication requirement
│   ├── Input[]     # accepted parameters
│   ├── Validation[] # input checks
│   ├── Logic[]     # business logic steps
│   └── Response    # return value
├── Policy[]      # authorization rules
├── Workflow[]    # event-driven sequences
├── Integration[] # third-party connections
├── Architecture  # deployment architecture
├── Pipeline[]    # CI/CD definitions
└── Environment[] # deployment environments
```

### Example IR Output

Human source:
```
api GetUser:
  requires authentication
  accepts user_id
  fetch the user by user_id
  if user does not exist, respond with "not found"
  respond with the user profile
```

Intent IR (YAML):
```yaml
type: api
name: GetUser
auth:
  required: true
input:
  - name: user_id
    type: identifier
    required: true
logic:
  - type: query
    entity: User
    filter: { id: "$user_id" }
    assign: "user"
  - type: condition
    if: { operator: "not_exists", value: "$user" }
    then:
      type: respond
      status: 404
      body: { message: "not found" }
  - type: respond
    status: 200
    body: { data: "$user.profile" }
```

---

## 6. LLM Connector (Optional)

The LLM connector is an optional plugin that enhances the developer experience.

### Configuration

```
connect to LLM:
  provider is <Provider> using <Model>
  use for: interpretation, suggestions, context
```

### Capabilities

1. **Smart Interpretation**: Converts freeform English into structured `.human` grammar
2. **Conversational Editing**: Edit your app through natural dialogue
3. **Context Building**: Understands entire project, answers questions, finds issues
4. **Pattern Suggestions**: Recommends improvements based on project analysis
5. **Design Import Assist**: Uses vision to interpret Figma files and screenshots

### Boundary

The LLM connector NEVER:
- Replaces the deterministic compiler
- Is required for any core operation
- Produces non-reproducible output (all suggestions are written to `.human` files)
- Runs without explicit user opt-in

---

## 7. CLI Reference

```
human init <name>              Create new project
human build                    Compile .human files to target code
human build --inspect          Show generated files without deploying
human run                      Start development server
human check                    Validate .human files
human test                     Run all generated tests
human audit                    Run security audit
human deploy                   Deploy to configured environment
human eject                    Export generated code as standalone project
human edit --with-llm          Start conversational editing session
human ask "<question>"         Ask LLM about your project
human suggest                  Get improvement suggestions from LLM
human feature "<name>"         Create feature branch
human commit "<message>"       Test, lint, commit
human push                     Push and trigger pipeline
human release "<version>"      Tag and deploy release
human rollback                 Revert to last stable deployment
human integrate <service>      Add third-party integration
human plugin add <plugin>      Add compiler target plugin
human plugin list              List available plugins
```

---

## 8. Error Messages

All error messages are in plain English and suggest fixes in Human language.

### Examples

```
Error: Your Post data has a "category" field but no predefined 
categories. Consider adding:
  has a category which is either "tech" or "lifestyle" or "news"
Or if categories are dynamic:
  has a category which is text

Error: api DeletePost lets any authenticated user delete any post. 
You probably want to restrict this. Add:
  check that current user is the author or an admin

Error: page Dashboard fetches transactions but has no loading state. 
Users will see a blank screen while data loads. Add:
  while loading, show a skeleton screen

Warning: User data has 50,000+ rows but no indexes on frequently 
queried fields. Add to your database section:
  index User by email
  index User by created date
```

---

## 9. Reserved Keywords

```
# Declarations
app, data, page, component, api, service, agent, policy, workflow
theme, architecture, environment, integrate

# Types
text, number, decimal, boolean, date, datetime, email, url, file, 
image, json

# Relationships
belongs to, has many, has a, has an, through

# Actions
show, fetch, create, update, delete, send, respond, navigate
check, validate, filter, sort, paginate, search
publish, listen, notify, alert, log, track

# Conditions
if, when, while, unless, until, after, before, every

# Modifiers
requires, accepts, only, either, or, and, not, with, from, to
using, in, on, at, for, by, as, is, are, the, a, an

# Quality
test, audit, check, verify, ensure, must, cannot, limit

# Architecture
monolith, microservices, serverless, gateway, broker

# DevOps
deploy, build, run, push, release, rollback, branch, merge
pipeline, monitor, alert, log
```

---

## 10. Version History

| Version | Date | Changes |
|---|---|---|
| 0.1 | 2025-02-19 | Initial specification |

---

*Human: The first programming language designed for humans, not computers.*
