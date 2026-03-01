# Human IR Schema Reference

Language-agnostic reference for the Intent IR — the typed intermediate representation between `.human` source and code generation.

All code generators read this IR exclusively; they never access the raw AST.

---

## Application (root)

The top-level node representing a complete application.

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Application name from `app:` declaration |
| `platform` | string | Target platform (`"web"`, `"mobile"`, etc.) |
| `config` | BuildConfig? | Framework and deployment choices from `build with:` block |
| `data` | DataModel[] | Data entities from `data:` blocks |
| `pages` | Page[] | Frontend pages from `page:` blocks |
| `components` | Component[] | Reusable UI components from `component:` blocks |
| `apis` | Endpoint[] | Backend API endpoints from `api:` blocks |
| `policies` | Policy[] | Authorization rules from `policy:` blocks |
| `workflows` | Workflow[] | Event-driven sequences from `when:` blocks |
| `theme` | Theme? | Visual configuration from `theme:` block |
| `auth` | Auth? | Authentication config from `security:` block |
| `database` | DatabaseConfig? | Database settings from `database:` block |
| `integrations` | Integration[] | Third-party services from `integrate:` blocks |
| `environments` | Environment[] | Deployment environments from `deploy:` blocks |
| `error_handlers` | ErrorHandler[] | Error recovery from `on error:` blocks |
| `pipelines` | Pipeline[] | CI/CD pipelines from `pipeline:` blocks |
| `architecture` | Architecture? | Architectural style from `architecture:` block |
| `monitoring` | MonitoringRule[] | Observability from `monitor:` block |

**Source syntax:**
```human
app: TaskFlow
  platform: web
```

---

## BuildConfig

Framework and deployment choices specified in the `build with:` block.

| Field | Type | Description |
|-------|------|-------------|
| `frontend` | string | Frontend framework (e.g. `"React with TypeScript"`) |
| `backend` | string | Backend framework (e.g. `"Node with Express"`) |
| `database` | string | Database engine (e.g. `"PostgreSQL"`) |
| `deploy` | string | Deployment target (e.g. `"Docker"`) |
| `ports` | PortConfig | Port numbers for services |

**Source syntax:**
```human
build with:
  frontend: React with TypeScript
  backend: Node with Express
  database: PostgreSQL
  deploy: Docker
```

---

## PortConfig

Port numbers for different services within the application.

| Field | Type | Description |
|-------|------|-------------|
| `frontend` | int | Frontend dev server port (default: 3000) |
| `backend` | int | Backend API server port (default: 3001) |
| `database` | int | Database connection port (default: 5432) |

---

## DataModel

A data entity with typed fields and relationships, from a `data:` block.

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Model name (PascalCase) |
| `fields` | DataField[] | Typed fields |
| `relations` | Relation[] | Relationships to other models |

**Source syntax:**
```human
data: Task
  title: text, required
  status: enum (open, in_progress, done), default: open
  due_date: datetime
  belongs_to: User
```

---

## DataField

A typed field within a data model.

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Field name (snake_case) |
| `type` | string | Type: `text`, `number`, `email`, `datetime`, `enum`, `boolean`, `url` |
| `required` | bool | Whether the field is required |
| `unique` | bool | Whether values must be unique |
| `encrypted` | bool | Whether the field is stored encrypted |
| `enum_values` | string[] | Allowed values for enum fields |
| `default` | string | Default value |

---

## Relation

A relationship between two data models.

| Field | Type | Description |
|-------|------|-------------|
| `kind` | string | `"belongs_to"`, `"has_many"`, `"has_many_through"` |
| `target` | string | Target model name |
| `through` | string | Join model for many-to-many relationships |

**Source syntax:**
```human
data: Task
  belongs_to: User
  has_many: Comments
```

---

## Page

A frontend page with content and interactions, from a `page:` block.

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Page name (PascalCase) |
| `content` | Action[] | Display, interaction, and navigation actions |

**Source syntax:**
```human
page: Dashboard
  show a list of tasks from API
  when user clicks a task, navigate to TaskDetail
```

---

## Component

A reusable UI component with typed props, from a `component:` block.

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Component name (PascalCase) |
| `props` | Prop[] | Input parameters |
| `content` | Action[] | Display and interaction actions |

**Source syntax:**
```human
component: TaskCard
  props: task
  show task title and status
  when user clicks, navigate to TaskDetail
```

---

## Prop

An input parameter for a component.

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Prop name |
| `type` | string | Optional type annotation |

---

## Action

A single step or statement in any block. This is the universal instruction unit used across pages, APIs, workflows, and pipelines.

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Action type (see table below) |
| `text` | string | Original statement text |
| `target` | string | Entity or element being acted upon |
| `value` | string | Value or destination |

### Action Types

| Type | Description | Example `.human` statement |
|------|-------------|---------------------------|
| `display` | Show/render something | `show a list of tasks` |
| `interact` | Click, drag, scroll, hover | `when user clicks delete button` |
| `input` | Form element, search, dropdown | `show a form with title and description` |
| `navigate` | Page navigation | `navigate to Dashboard` |
| `condition` | If/when/while/unless | `if task is overdue, highlight in red` |
| `loop` | Each/every iteration | `for each task, show a TaskCard` |
| `query` | Fetch/get data | `fetch all tasks for current user` |
| `create` | Create entity | `create a new Task with the given data` |
| `update` | Update/set entity | `update the task status to done` |
| `delete` | Delete entity | `delete the task` |
| `validate` | Check/validate data | `check that email is not empty` |
| `respond` | API response | `respond with the created task` |
| `send` | Send email/notification | `send a welcome email to the user` |
| `assign` | Set/assign value | `set status to "active"` |
| `alert` | Alert team/user | `alert the admin team` |
| `log` | Logging/tracking | `log the action for audit` |
| `delay` | After X time | `after 24 hours` |
| `retry` | Retry logic | `retry up to 3 times` |
| `configure` | Configuration setting | `enable cors` |

---

## Endpoint

A backend API endpoint from an `api:` block.

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Endpoint name (e.g. `"CreateTask"`, `"GetUsers"`) |
| `auth` | bool | Whether authentication is required |
| `params` | Param[] | Input parameters |
| `validation` | ValidationRule[] | Validation checks from `check that` statements |
| `steps` | Action[] | Implementation steps |

**Source syntax:**
```human
api: CreateTask
  accepts: title, description, due_date
  authenticate
  check that title is not empty
  check that due_date is in the future
  create a new Task with the given data
  respond with the created task
```

---

## Param

An API input parameter.

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Parameter name |

---

## ValidationRule

A structured validation check extracted from `check that ...` statements.

| Field | Type | Description |
|-------|------|-------------|
| `field` | string | Field being validated |
| `rule` | string | Rule type: `not_empty`, `valid_email`, `min_length`, `max_length`, `unique`, `future_date`, `matches` |
| `value` | string | Rule parameter (e.g. `"8"` for `min_length`) |
| `message` | string | Custom error message |

**Source syntax:**
```human
check that email is not empty
check that email is a valid email
check that password is at least 8 characters
check that username is unique
```

---

## Policy

Authorization rules for a role, from a `policy:` block.

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Role name (e.g. `"Admin"`, `"Member"`) |
| `permissions` | PolicyRule[] | Allowed actions |
| `restrictions` | PolicyRule[] | Denied actions |

**Source syntax:**
```human
policy: Admin
  can manage all Tasks
  can manage all Users

policy: Member
  can create Tasks
  can edit own Tasks
  cannot delete other users' Tasks
```

---

## PolicyRule

A single permission or restriction statement.

| Field | Type | Description |
|-------|------|-------------|
| `text` | string | Original rule text |

---

## Workflow

An event-driven action sequence from a `when:` block.

| Field | Type | Description |
|-------|------|-------------|
| `trigger` | string | Event that triggers the workflow |
| `steps` | Action[] | Actions to execute |

**Source syntax:**
```human
when: user signs up
  send a welcome email
  create a default workspace
```

---

## Pipeline

A CI/CD pipeline from a `pipeline:` block.

| Field | Type | Description |
|-------|------|-------------|
| `trigger` | string | Code event trigger (e.g. `"push to main"`) |
| `steps` | Action[] | Pipeline steps |

**Source syntax:**
```human
pipeline: deploy
  on push to main
  run tests
  build Docker image
  deploy to production
```

---

## Theme

Visual configuration from the `theme:` block.

| Field | Type | Description |
|-------|------|-------------|
| `design_system` | string | Design system: `material`, `shadcn`, `ant`, `chakra`, `bootstrap`, `tailwind`, `untitled` |
| `colors` | map[string]string | Color tokens (e.g. `primary: "#3B82F6"`) |
| `fonts` | map[string]string | Font configuration (e.g. `heading: "Inter"`) |
| `spacing` | string | Spacing preset: `compact`, `comfortable`, `spacious` |
| `border_radius` | string | Border radius preset: `sharp`, `smooth`, `rounded`, `pill` |
| `dark_mode` | bool | Whether dark mode is enabled |
| `options` | map[string]string | Additional theme properties |

**Source syntax:**
```human
theme:
  design: material
  primary color: #3B82F6
  font: Inter
  spacing: comfortable
  corners: rounded
  dark mode: on
```

---

## Auth

Authentication and security configuration from the `security:` block.

| Field | Type | Description |
|-------|------|-------------|
| `methods` | AuthMethod[] | Authentication approaches |
| `rules` | Action[] | Security rules (rate limiting, CORS, etc.) |

**Source syntax:**
```human
security:
  use JWT authentication
  token expires after 7 days
  rate limit 100 requests per minute per user
  enable cors
  sanitize all inputs
```

---

## AuthMethod

A specific authentication approach.

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Auth type: `jwt`, `oauth` |
| `provider` | string | OAuth provider: `google`, `github`, etc. |
| `config` | map[string]string | Configuration (e.g. `expiration`, `callback_url`) |

---

## DatabaseConfig

Database engine and configuration from the `database:` block.

| Field | Type | Description |
|-------|------|-------------|
| `engine` | string | Database engine (e.g. `"PostgreSQL"`, `"MySQL"`) |
| `indexes` | Index[] | Database indexes |
| `rules` | Action[] | Database rules (backup, retention, startup) |

**Source syntax:**
```human
database:
  engine: PostgreSQL
  index Task by status
  index Task by due_date, status
  backup daily
  retain logs for 30 days
```

---

## Index

A database index definition.

| Field | Type | Description |
|-------|------|-------------|
| `entity` | string | Model name the index belongs to |
| `fields` | string[] | Fields included in the index |

---

## Integration

A third-party service connection from an `integrate:` block.

| Field | Type | Description |
|-------|------|-------------|
| `service` | string | Service name (e.g. `"SendGrid"`, `"Stripe"`) |
| `type` | string | Inferred type: `email`, `storage`, `payment`, `messaging`, `oauth` |
| `credentials` | map[string]string | Env var mappings (e.g. `"API key": "SENDGRID_API_KEY"`) |
| `config` | map[string]string | Configuration (e.g. `region`, `sender_email`) |
| `templates` | string[] | Email template names |
| `purpose` | string | Integration purpose description |

**Source syntax:**
```human
integrate: SendGrid
  API key: SENDGRID_API_KEY
  sender: noreply@example.com
  templates: welcome, password_reset, task_assigned
```

---

## Environment

A deployment environment from a `deploy:` block.

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Environment name (e.g. `"staging"`, `"production"`) |
| `config` | map[string]string | Environment-specific config (url, database, flags) |
| `rules` | Action[] | Deployment rules |

**Source syntax:**
```human
deploy:
  staging:
    url: https://staging.example.com
    database: postgres://staging-db/app
  production:
    url: https://example.com
    database: postgres://prod-db/app
```

---

## ErrorHandler

Error recovery logic from an `on error:` block.

| Field | Type | Description |
|-------|------|-------------|
| `condition` | string | Error condition |
| `steps` | Action[] | Recovery steps |

**Source syntax:**
```human
on error:
  if payment fails, retry up to 3 times
  if service unavailable, alert the admin team
  log all errors for debugging
```

---

## Architecture

Architectural style from the `architecture:` block.

| Field | Type | Description |
|-------|------|-------------|
| `style` | string | `"monolith"`, `"microservices"`, `"serverless"` |
| `services` | ServiceDef[] | Microservice definitions |
| `gateway` | GatewayDef? | API gateway config |
| `broker` | string | Message broker (e.g. `"RabbitMQ"`, `"Kafka"`) |

**Source syntax:**
```human
architecture:
  style: microservices
  services:
    UserService handles users on port 3001
    TaskService handles tasks on port 3002
  gateway:
    /users -> UserService
    /tasks -> TaskService
  broker: RabbitMQ
```

---

## ServiceDef

A microservice definition within the architecture.

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Service name |
| `handles` | string | Responsibility description |
| `port` | int | Service port |
| `models` | string[] | Data model names owned by this service |
| `has_own_database` | bool | Whether the service has a dedicated database |
| `talks_to` | string[] | Other services it communicates with |

---

## GatewayDef

API gateway configuration for microservices.

| Field | Type | Description |
|-------|------|-------------|
| `routes` | map[string]string | Path-to-service mapping (e.g. `"/users" → "UserService"`) |
| `rules` | string[] | Gateway rules (rate limiting, CORS) |

---

## MonitoringRule

An observability directive from the `monitor:` block.

| Field | Type | Description |
|-------|------|-------------|
| `kind` | string | `"track"`, `"alert"`, `"log"` |
| `metric` | string | What to track or log |
| `channel` | string | Alert channel (e.g. `"Slack"`) |
| `condition` | string | Alert trigger condition |
| `service` | string | Log destination (e.g. `"CloudWatch"`) |
| `duration` | string | Retention duration |

**Source syntax:**
```human
monitor:
  track response time per endpoint
  alert team on Slack if error rate > 5%
  log all requests to CloudWatch
  retain logs for 90 days
```
