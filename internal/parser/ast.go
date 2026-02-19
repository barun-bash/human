package parser

// Program is the root AST node representing a complete .human file.
type Program struct {
	App            *AppDeclaration
	Data           []*DataDeclaration
	Pages          []*PageDeclaration
	Components     []*ComponentDeclaration
	APIs           []*APIDeclaration
	Policies       []*PolicyDeclaration
	Workflows      []*WorkflowDeclaration
	Theme          *ThemeDeclaration
	Authentication *AuthenticationDeclaration
	Database       *DatabaseDeclaration
	Integrations   []*IntegrationDeclaration
	Environments   []*EnvironmentDeclaration
	ErrorHandlers  []*ErrorHandlerDeclaration
	Build          *BuildDeclaration
	Sections       []string     // section header names in order
	Statements     []*Statement // top-level statements not in any block
}

// AppDeclaration represents: app <Name> is a <platform> application
type AppDeclaration struct {
	Name     string // e.g. "TaskFlow"
	Platform string // e.g. "web", "mobile", "desktop", "api"
	Line     int
}

// DataDeclaration represents a data model with fields and relationships.
//
//	data User:
//	  has a name which is text
//	  belongs to a Team
//	  has many Post
type DataDeclaration struct {
	Name          string
	Fields        []*Field
	Relationships []*Relationship
	Line          int
}

// Field represents a single field within a data declaration.
//
//	has a name which is text
//	has an optional bio which is text
//	has a role which is either "user" or "admin"
//	has a created datetime               (shorthand)
type Field struct {
	Name       string   // field name, e.g. "name", "email"
	Type       string   // type keyword, e.g. "text", "email", "datetime"
	Modifiers  []string // "optional", "unique", "encrypted"
	EnumValues []string // for "either" fields: ["user", "admin"]
	Default    string   // default value (from "defaults to")
	Line       int
}

// Relationship represents a relationship between data models.
//
//	belongs to a User         → Kind="belongs_to"
//	has many Post             → Kind="has_many"
//	has many Tag through PostTag → Kind="has_many", Through="PostTag"
type Relationship struct {
	Kind    string // "belongs_to" or "has_many"
	Target  string // related model name
	Through string // join table for many-to-many
	Line    int
}

// PageDeclaration represents a frontend page with display and interaction statements.
//
//	page Dashboard:
//	  show a greeting with the user's name
//	  clicking a task navigates to the task detail
type PageDeclaration struct {
	Name       string
	Statements []*Statement
	Line       int
}

// ComponentDeclaration represents a reusable UI component.
//
//	component TransactionCard:
//	  accepts transaction as Transaction
//	  show the transaction title in bold
type ComponentDeclaration struct {
	Name       string
	Accepts    []string
	Statements []*Statement
	Line       int
}

// APIDeclaration represents a backend API endpoint.
//
//	api CreateTask:
//	  requires authentication
//	  accepts title, description, and status
//	  check that title is not empty
//	  respond with the created task
type APIDeclaration struct {
	Name       string
	Auth       bool     // true if "requires authentication"
	Accepts    []string // parameter names
	Statements []*Statement
	Line       int
}

// PolicyDeclaration represents authorization rules for a role.
//
//	policy FreeUser:
//	  can create up to 50 tasks per month
//	  cannot delete completed tasks
type PolicyDeclaration struct {
	Name  string
	Rules []*PolicyRule
	Line  int
}

// PolicyRule represents a single permission rule within a policy.
type PolicyRule struct {
	Allowed bool   // true for "can", false for "cannot"
	Text    string // the full rule text
	Line    int
}

// WorkflowDeclaration represents an event-driven action sequence.
// Also used for CI/CD pipelines (when code is pushed/merged).
//
//	when a user signs up:
//	  create their account
//	  send welcome email
type WorkflowDeclaration struct {
	Event      string // trigger description: "a user signs up"
	Statements []*Statement
	Line       int
}

// ThemeDeclaration represents visual theme configuration.
//
//	theme:
//	  primary color is #6C5CE7
//	  font is Inter for body
type ThemeDeclaration struct {
	Properties []*Statement
	Line       int
}

// AuthenticationDeclaration represents security/auth configuration.
//
//	authentication:
//	  method JWT tokens that expire in 7 days
//	  rate limit all endpoints to 100 requests per minute
type AuthenticationDeclaration struct {
	Statements []*Statement
	Line       int
}

// DatabaseDeclaration represents database configuration.
//
//	database:
//	  use PostgreSQL
//	  index User by email
type DatabaseDeclaration struct {
	Statements []*Statement
	Line       int
}

// IntegrationDeclaration represents a third-party service integration.
//
//	integrate with SendGrid:
//	  api key from environment variable SENDGRID_API_KEY
//	  use for sending transactional emails
type IntegrationDeclaration struct {
	Service    string
	Statements []*Statement
	Line       int
}

// EnvironmentDeclaration represents a deployment environment.
//
//	environment staging:
//	  url is "staging.example.com"
//	  uses staging database
type EnvironmentDeclaration struct {
	Name       string
	Statements []*Statement
	Line       int
}

// ErrorHandlerDeclaration represents an error handling block.
//
//	if database is unreachable:
//	  retry 3 times with 1 second delay
//	  alert the engineering team
type ErrorHandlerDeclaration struct {
	Condition  string // e.g. "database is unreachable"
	Statements []*Statement
	Line       int
}

// BuildDeclaration represents build target configuration.
//
//	build with:
//	  frontend using React with TypeScript
//	  backend using Node with Express
type BuildDeclaration struct {
	Statements []*Statement
	Line       int
}

// Statement represents a single line of structured English within a block.
// The Kind field identifies the leading keyword for quick categorization.
type Statement struct {
	Kind string // lowercase first keyword: "show", "clicking", "if", "check", etc.
	Text string // the full reconstructed text of the statement
	Line int
}
