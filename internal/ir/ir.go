package ir

import "strings"

// Application is the root IR node representing a complete application.
// It is framework-agnostic and serializable — given only this IR,
// any code generator can produce a working application.
type Application struct {
	Name          string          `json:"name"`
	Platform      string          `json:"platform"`
	Config        *BuildConfig    `json:"config,omitempty"`
	Data          []*DataModel    `json:"data,omitempty"`
	Pages         []*Page         `json:"pages,omitempty"`
	Components    []*Component    `json:"components,omitempty"`
	APIs          []*Endpoint     `json:"apis,omitempty"`
	Policies      []*Policy       `json:"policies,omitempty"`
	Workflows     []*Workflow     `json:"workflows,omitempty"`
	Theme         *Theme          `json:"theme,omitempty"`
	Auth          *Auth           `json:"auth,omitempty"`
	Database      *DatabaseConfig `json:"database,omitempty"`
	Integrations  []*Integration  `json:"integrations,omitempty"`
	Environments  []*Environment  `json:"environments,omitempty"`
	ErrorHandlers []*ErrorHandler  `json:"error_handlers,omitempty"`
	Pipelines     []*Pipeline      `json:"pipelines,omitempty"`
	Architecture  *Architecture    `json:"architecture,omitempty"`
	Monitoring    []*MonitoringRule `json:"monitoring,omitempty"`
}

// ── Build Configuration ──

// BuildConfig holds the target framework and deployment choices.
type BuildConfig struct {
	Frontend string `json:"frontend,omitempty"` // e.g. "React with TypeScript"
	Backend  string `json:"backend,omitempty"`  // e.g. "Node with Express"
	Database string `json:"database,omitempty"` // e.g. "PostgreSQL"
	Deploy   string `json:"deploy,omitempty"`   // e.g. "Docker"
}

// ── Data Layer ──

// DataModel represents a data entity with typed fields and relationships.
type DataModel struct {
	Name      string       `json:"name"`
	Fields    []*DataField `json:"fields,omitempty"`
	Relations []*Relation  `json:"relations,omitempty"`
}

// DataField is a typed field within a data model.
type DataField struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`                  // text, number, email, datetime, enum, etc.
	Required   bool     `json:"required"`
	Unique     bool     `json:"unique,omitempty"`
	Encrypted  bool     `json:"encrypted,omitempty"`
	EnumValues []string `json:"enum_values,omitempty"` // for enum fields
	Default    string   `json:"default,omitempty"`
}

// Relation is a relationship between data models.
type Relation struct {
	Kind    string `json:"kind"`              // belongs_to, has_many, has_many_through
	Target  string `json:"target"`
	Through string `json:"through,omitempty"` // join model for many-to-many
}

// ── Frontend ──

// Page represents a frontend page with content and interactions.
type Page struct {
	Name    string    `json:"name"`
	Content []*Action `json:"content,omitempty"`
}

// Component represents a reusable UI component.
type Component struct {
	Name    string    `json:"name"`
	Props   []*Prop   `json:"props,omitempty"`
	Content []*Action `json:"content,omitempty"`
}

// Prop is an input parameter for a component.
type Prop struct {
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
}

// ── Backend ──

// Endpoint represents a backend API endpoint.
type Endpoint struct {
	Name       string            `json:"name"`
	Auth       bool              `json:"auth"`
	Params     []*Param          `json:"params,omitempty"`
	Validation []*ValidationRule `json:"validation,omitempty"`
	Steps      []*Action         `json:"steps,omitempty"`
}

// Param is an API input parameter.
type Param struct {
	Name string `json:"name"`
}

// ValidationRule is a structured validation check extracted from
// "check that ..." statements.
type ValidationRule struct {
	Field   string `json:"field"`
	Rule    string `json:"rule"`            // not_empty, valid_email, min_length, max_length, unique, future_date, matches
	Value   string `json:"value,omitempty"` // for rules that take a value
	Message string `json:"message,omitempty"`
}

// ── Authorization ──

// Policy represents authorization rules for a role.
type Policy struct {
	Name         string        `json:"name"`
	Permissions  []*PolicyRule `json:"permissions,omitempty"`
	Restrictions []*PolicyRule `json:"restrictions,omitempty"`
}

// PolicyRule is a single permission or restriction.
type PolicyRule struct {
	Text string `json:"text"` // original rule text
}

// ── Workflows & Pipelines ──

// Workflow represents an event-driven action sequence.
type Workflow struct {
	Trigger string    `json:"trigger"`
	Steps   []*Action `json:"steps,omitempty"`
}

// Pipeline represents a CI/CD pipeline triggered by code events.
type Pipeline struct {
	Trigger string    `json:"trigger"`
	Steps   []*Action `json:"steps,omitempty"`
}

// ── Actions ──

// Action represents a single step or statement in any block.
// The Type field categorizes the action for code generators.
//
// Action types:
//
//	display    - show/render something
//	interact   - clicking, dragging, scrolling, hovering
//	input      - form element, search, dropdown, file upload
//	navigate   - page navigation
//	condition  - if/when/while/unless
//	loop       - each/every iteration
//	query      - fetch/get data
//	create     - create entity
//	update     - update/set entity
//	delete     - delete entity
//	validate   - check/validate data
//	respond    - API response
//	send       - send email/notification
//	assign     - set/assign value
//	alert      - alert team/user
//	log        - logging/tracking
//	delay      - after X time
//	retry      - retry logic
//	configure  - configuration setting
type Action struct {
	Type   string `json:"type"`
	Text   string `json:"text"`
	Target string `json:"target,omitempty"` // entity or element being acted upon
	Value  string `json:"value,omitempty"`  // value or destination
}

// ── Theme ──

// Theme holds visual configuration extracted from theme properties.
type Theme struct {
	DesignSystem string            `json:"design_system,omitempty"` // material, shadcn, ant, chakra, bootstrap, tailwind, untitled
	Colors       map[string]string `json:"colors,omitempty"`
	Fonts        map[string]string `json:"fonts,omitempty"`
	Spacing      string            `json:"spacing,omitempty"`       // compact, comfortable, spacious
	BorderRadius string            `json:"border_radius,omitempty"` // sharp, smooth, rounded, pill
	DarkMode     bool              `json:"dark_mode,omitempty"`
	Options      map[string]string `json:"options,omitempty"` // other properties
}

// ── Security ──

// Auth holds authentication and security configuration.
type Auth struct {
	Methods []*AuthMethod `json:"methods,omitempty"`
	Rules   []*Action     `json:"rules,omitempty"` // rate limiting, CORS, sanitization, etc.
}

// AuthMethod is a specific authentication approach.
type AuthMethod struct {
	Type     string            `json:"type"`               // jwt, oauth
	Provider string            `json:"provider,omitempty"`  // for OAuth: google, github, etc.
	Config   map[string]string `json:"config,omitempty"`    // expiration, callback_url, etc.
}

// ── Database ──

// DatabaseConfig holds database engine and configuration.
type DatabaseConfig struct {
	Engine  string    `json:"engine,omitempty"` // PostgreSQL, MySQL, etc.
	Indexes []*Index  `json:"indexes,omitempty"`
	Rules   []*Action `json:"rules,omitempty"` // backup, retention, startup tasks
}

// Index is a database index definition.
type Index struct {
	Entity string   `json:"entity"`
	Fields []string `json:"fields"`
}

// ── Integrations ──

// Integration represents a third-party service connection.
type Integration struct {
	Service     string            `json:"service"`
	Type        string            `json:"type,omitempty"`        // email, storage, payment, messaging, oauth
	Credentials map[string]string `json:"credentials,omitempty"` // env var mappings
	Config      map[string]string `json:"config,omitempty"`      // region, sender_email, bucket, webhook_endpoint, channel
	Templates   []string          `json:"templates,omitempty"`   // email template names
	Purpose     string            `json:"purpose,omitempty"`
}

// InferIntegrationType returns the integration type based on service name.
func InferIntegrationType(service string) string {
	s := strings.ToLower(service)
	switch {
	case strings.Contains(s, "sendgrid") || strings.Contains(s, "mailgun") ||
		strings.Contains(s, "ses") || strings.Contains(s, "postmark") ||
		strings.Contains(s, "mailchimp"):
		return "email"
	case strings.Contains(s, "s3") || strings.Contains(s, "gcs") ||
		strings.Contains(s, "cloudinary") || strings.Contains(s, "minio"):
		return "storage"
	case strings.Contains(s, "stripe") || strings.Contains(s, "paypal") ||
		strings.Contains(s, "braintree") || strings.Contains(s, "square"):
		return "payment"
	case strings.Contains(s, "slack") || strings.Contains(s, "discord") ||
		strings.Contains(s, "twilio") || strings.Contains(s, "telegram"):
		return "messaging"
	case strings.Contains(s, "google") || strings.Contains(s, "github") ||
		strings.Contains(s, "facebook") || strings.Contains(s, "auth0") ||
		strings.Contains(s, "okta"):
		return "oauth"
	default:
		return ""
	}
}

// ── Deployment ──

// Environment represents a deployment environment.
type Environment struct {
	Name   string            `json:"name"`
	Config map[string]string `json:"config,omitempty"` // url, database, flags
	Rules  []*Action         `json:"rules,omitempty"`
}

// ── Error Handling ──

// ErrorHandler represents error recovery logic.
type ErrorHandler struct {
	Condition string    `json:"condition"`
	Steps     []*Action `json:"steps,omitempty"`
}

// ── Architecture ──

// Architecture describes the application's architectural style.
type Architecture struct {
	Style    string        `json:"style"`              // monolith, microservices, serverless
	Services []*ServiceDef `json:"services,omitempty"` // for microservices
	Gateway  *GatewayDef   `json:"gateway,omitempty"`  // for microservices
	Broker   string        `json:"broker,omitempty"`   // message broker (e.g., RabbitMQ, Kafka)
}

// ServiceDef defines a microservice.
type ServiceDef struct {
	Name           string   `json:"name"`
	Handles        string   `json:"handles,omitempty"`         // responsibility description
	Port           int      `json:"port,omitempty"`
	Models         []string `json:"models,omitempty"`          // data model names this service owns
	HasOwnDatabase bool     `json:"has_own_database,omitempty"`
	TalksTo        []string `json:"talks_to,omitempty"`        // other services it communicates with
}

// GatewayDef defines an API gateway for microservices.
type GatewayDef struct {
	Routes map[string]string `json:"routes,omitempty"` // path → service name
	Rules  []string          `json:"rules,omitempty"`  // rate limiting, CORS, etc.
}

// ── Monitoring ──

// MonitoringRule represents an observability directive.
type MonitoringRule struct {
	Kind      string `json:"kind"`                // track, alert, log
	Metric    string `json:"metric,omitempty"`    // what to track/log
	Channel   string `json:"channel,omitempty"`   // alert channel (e.g., "Slack")
	Condition string `json:"condition,omitempty"` // alert trigger condition
	Service   string `json:"service,omitempty"`   // log destination (e.g., "CloudWatch")
	Duration  string `json:"duration,omitempty"`  // retention duration
}
