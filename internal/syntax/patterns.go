package syntax

// Category represents a logical grouping of syntax patterns.
type Category string

const (
	CatApp          Category = "app"
	CatData         Category = "data"
	CatPages        Category = "pages"
	CatComponents   Category = "components"
	CatEvents       Category = "events"
	CatStyling      Category = "styling"
	CatForms        Category = "forms"
	CatAPIs         Category = "apis"
	CatSecurity     Category = "security"
	CatPolicies     Category = "policies"
	CatDatabase     Category = "database"
	CatWorkflows    Category = "workflows"
	CatIntegrations Category = "integrations"
	CatArchitecture Category = "architecture"
	CatDevOps       Category = "devops"
	CatTheme        Category = "theme"
	CatBuild        Category = "build"
	CatConditional  Category = "conditional"
	CatErrors       Category = "errors"
)

// Pattern represents a single syntax pattern in the Human language.
type Pattern struct {
	Template    string   // "show a list of <data>"
	Description string   // "Renders a collection of data items"
	Category    Category
	Tags        []string // search tags
	Example     string   // full usage example
	Related     []string // related pattern templates
}

// CategoryLabel returns a human-readable label for a category.
func CategoryLabel(cat Category) string {
	labels := map[Category]string{
		CatApp:          "Application",
		CatData:         "Data Models",
		CatPages:        "Pages & Navigation",
		CatComponents:   "Components",
		CatEvents:       "Events & Interactions",
		CatStyling:      "Styling & Layout",
		CatForms:        "Forms & Inputs",
		CatAPIs:         "APIs & Endpoints",
		CatSecurity:     "Security & Authentication",
		CatPolicies:     "Policies & Authorization",
		CatDatabase:     "Database",
		CatWorkflows:    "Workflows & Events",
		CatIntegrations: "Integrations",
		CatArchitecture: "Architecture",
		CatDevOps:       "DevOps & Deployment",
		CatTheme:        "Theme & Design",
		CatBuild:        "Build Targets",
		CatConditional:  "Conditional Logic",
		CatErrors:       "Error Handling",
	}
	if label, ok := labels[cat]; ok {
		return label
	}
	return string(cat)
}

// AllCategories returns all categories in display order.
func AllCategories() []Category {
	return []Category{
		CatApp,
		CatData,
		CatPages,
		CatComponents,
		CatEvents,
		CatStyling,
		CatForms,
		CatAPIs,
		CatSecurity,
		CatPolicies,
		CatDatabase,
		CatWorkflows,
		CatIntegrations,
		CatArchitecture,
		CatDevOps,
		CatTheme,
		CatBuild,
		CatConditional,
		CatErrors,
	}
}

// ByCategory returns all patterns in a given category.
func ByCategory(cat Category) []Pattern {
	var result []Pattern
	for _, p := range allPatterns {
		if p.Category == cat {
			result = append(result, p)
		}
	}
	return result
}

// AllPatterns returns a copy of all registered patterns.
func AllPatterns() []Pattern {
	result := make([]Pattern, len(allPatterns))
	copy(result, allPatterns)
	return result
}

var allPatterns = []Pattern{
	// ── App ──
	{
		Template:    "app <Name> is a <platform> application",
		Description: "Declare the application and its platform type",
		Category:    CatApp,
		Tags:        []string{"app", "application", "declare", "web", "mobile", "desktop", "api"},
		Example:     "app TaskFlow is a web application",
		Related:     []string{"build with:"},
	},
	{
		Template:    "── <section> ──",
		Description: "Section divider to organize code within a file",
		Category:    CatApp,
		Tags:        []string{"section", "organize", "divider", "frontend", "backend"},
		Example:     "── frontend ──",
	},
	{
		Template:    "name: <project_name>",
		Description: "Set the project name in configuration",
		Category:    CatApp,
		Tags:        []string{"name", "project", "config", "identity"},
		Example:     "name: my-app",
	},

	// ── Data Models ──
	{
		Template:    "data <Name>:",
		Description: "Define a data entity/model",
		Category:    CatData,
		Tags:        []string{"data", "model", "entity", "define", "schema"},
		Example:     "data User:",
		Related:     []string{"has a <field> which is <type>", "belongs to a <Data>"},
	},
	{
		Template:    "has a <field> which is <type>",
		Description: "Add a required field to a data model",
		Category:    CatData,
		Tags:        []string{"field", "property", "attribute", "required"},
		Example:     "has a name which is text",
	},
	{
		Template:    "has an optional <field> which is <type>",
		Description: "Add a nullable/optional field",
		Category:    CatData,
		Tags:        []string{"field", "optional", "nullable"},
		Example:     "has an optional bio which is text",
	},
	{
		Template:    "has a <field> which is unique <type>",
		Description: "Add a field with a unique constraint",
		Category:    CatData,
		Tags:        []string{"field", "unique", "constraint"},
		Example:     "has an email which is unique email",
	},
	{
		Template:    "has a <field> which is encrypted <type>",
		Description: "Add an encrypted-at-rest field",
		Category:    CatData,
		Tags:        []string{"field", "encrypted", "security"},
		Example:     "has a password which is encrypted text",
	},
	{
		Template:    "has a <field> which is either <value> or <value>",
		Description: "Add an enum/choice field",
		Category:    CatData,
		Tags:        []string{"field", "enum", "choice", "either", "or"},
		Example:     `has a role which is either "user" or "admin" or "moderator"`,
	},
	{
		Template:    "has a <field> which defaults to <value>",
		Description: "Add a field with a default value",
		Category:    CatData,
		Tags:        []string{"field", "default", "value"},
		Example:     `has a status which defaults to "draft"`,
	},
	{
		Template:    "belongs to a <Data>",
		Description: "Many-to-one relationship (foreign key)",
		Category:    CatData,
		Tags:        []string{"relationship", "belongs", "foreign key", "many-to-one"},
		Example:     "belongs to a User",
		Related:     []string{"has many <Data>"},
	},
	{
		Template:    "has many <Data>",
		Description: "One-to-many relationship",
		Category:    CatData,
		Tags:        []string{"relationship", "has many", "one-to-many"},
		Example:     "has many Post",
		Related:     []string{"belongs to a <Data>", "has many <Data> through <JoinData>"},
	},
	{
		Template:    "has many <Data> through <JoinData>",
		Description: "Many-to-many relationship via join table",
		Category:    CatData,
		Tags:        []string{"relationship", "many-to-many", "join", "through"},
		Example:     "has many Tag through PostTag",
	},

	// Field types
	{
		Template:    "text",
		Description: "String field type",
		Category:    CatData,
		Tags:        []string{"type", "string", "text"},
	},
	{
		Template:    "number",
		Description: "Integer or float field type",
		Category:    CatData,
		Tags:        []string{"type", "number", "integer", "float"},
	},
	{
		Template:    "decimal",
		Description: "Precise decimal field type (for money, etc.)",
		Category:    CatData,
		Tags:        []string{"type", "decimal", "money", "precise"},
	},
	{
		Template:    "boolean",
		Description: "True/false field type",
		Category:    CatData,
		Tags:        []string{"type", "boolean", "flag", "true", "false"},
	},
	{
		Template:    "date",
		Description: "Date-only field type",
		Category:    CatData,
		Tags:        []string{"type", "date"},
	},
	{
		Template:    "datetime",
		Description: "Date and time field type",
		Category:    CatData,
		Tags:        []string{"type", "datetime", "timestamp"},
	},
	{
		Template:    "email",
		Description: "Email field type (auto-validated)",
		Category:    CatData,
		Tags:        []string{"type", "email", "validation"},
	},
	{
		Template:    "url",
		Description: "URL field type (auto-validated)",
		Category:    CatData,
		Tags:        []string{"type", "url", "link"},
	},
	{
		Template:    "file",
		Description: "File upload field type",
		Category:    CatData,
		Tags:        []string{"type", "file", "upload"},
	},
	{
		Template:    "image",
		Description: "Image file field type",
		Category:    CatData,
		Tags:        []string{"type", "image", "photo", "picture"},
	},
	{
		Template:    "json",
		Description: "Arbitrary JSON field type",
		Category:    CatData,
		Tags:        []string{"type", "json", "metadata", "arbitrary"},
	},

	// ── Pages ──
	{
		Template:    "page <Name>:",
		Description: "Define a page in the application",
		Category:    CatPages,
		Tags:        []string{"page", "view", "screen", "route"},
		Example:     "page Dashboard:",
		Related:     []string{"show <what>", "clicking <element> navigates to <page>"},
	},
	{
		Template:    "show <what>",
		Description: "Render content on a page",
		Category:    CatPages,
		Tags:        []string{"show", "display", "render"},
		Example:     "show a greeting with the user's name",
	},
	{
		Template:    "show a list of <data>",
		Description: "Render a collection of data items",
		Category:    CatPages,
		Tags:        []string{"show", "list", "collection", "display"},
		Example:     "show a list of recent transactions sorted by date newest first",
	},
	{
		Template:    "show each <item>'s <field> and <field>",
		Description: "Specify which fields to display for each item",
		Category:    CatPages,
		Tags:        []string{"show", "each", "fields", "display"},
		Example:     "each transaction shows its title, amount, category, and date",
	},
	{
		Template:    "show <data> in a <layout>",
		Description: "Display data in a specific layout (card, table, grid, etc.)",
		Category:    CatPages,
		Tags:        []string{"show", "layout", "card", "table", "grid"},
		Example:     "show users in a table",
	},
	{
		Template:    `show "<text>"`,
		Description: "Display static text content",
		Category:    CatPages,
		Tags:        []string{"show", "text", "static", "content"},
		Example:     `show "Welcome to TaskFlow"`,
	},

	// ── Components ──
	{
		Template:    "component <Name>:",
		Description: "Define a reusable UI component",
		Category:    CatComponents,
		Tags:        []string{"component", "reusable", "ui", "widget"},
		Example:     "component TransactionCard:",
		Related:     []string{"accepts <prop> as <type>"},
	},
	{
		Template:    "accepts <prop> as <type>",
		Description: "Define a component prop/parameter",
		Category:    CatComponents,
		Tags:        []string{"prop", "parameter", "accepts", "input"},
		Example:     "accepts transaction as Transaction",
	},
	{
		Template:    "design <name> from <file>",
		Description: "Import a design file (Figma, image) as a component",
		Category:    CatComponents,
		Tags:        []string{"design", "import", "figma", "image"},
		Example:     `design dashboard from "designs/dashboard.figma"`,
	},

	// ── Events & Interactions ──
	{
		Template:    "clicking <element> does <action>",
		Description: "Handle click events on an element",
		Category:    CatEvents,
		Tags:        []string{"click", "button", "event", "handler", "action"},
		Example:     "clicking the delete button deletes the post",
	},
	{
		Template:    "clicking <element> navigates to <page>",
		Description: "Navigate to another page on click",
		Category:    CatEvents,
		Tags:        []string{"click", "navigate", "route", "link"},
		Example:     "clicking a transaction opens a detail panel on the right",
	},
	{
		Template:    "clicking <element> opens <thing>",
		Description: "Open a modal, panel, or link on click",
		Category:    CatEvents,
		Tags:        []string{"click", "open", "modal", "panel"},
		Example:     "clicking the add button opens a form to create a Transaction",
	},
	{
		Template:    "typing in <element> does <action>",
		Description: "Handle input/typing events",
		Category:    CatEvents,
		Tags:        []string{"typing", "input", "handler", "change"},
		Example:     "typing in the search bar filters the list",
	},
	{
		Template:    "hovering over <element> shows <thing>",
		Description: "Show content on hover",
		Category:    CatEvents,
		Tags:        []string{"hover", "tooltip", "show", "mouse"},
		Example:     "hovering over a user avatar shows their name",
	},
	{
		Template:    "pressing <key> does <action>",
		Description: "Handle keyboard shortcuts",
		Category:    CatEvents,
		Tags:        []string{"keyboard", "shortcut", "press", "key"},
		Example:     "pressing Escape closes the modal",
	},
	{
		Template:    "scrolling to bottom loads more <data>",
		Description: "Infinite scroll / load more on scroll",
		Category:    CatEvents,
		Tags:        []string{"scroll", "infinite", "load more", "pagination"},
		Example:     "scrolling to bottom loads more posts",
	},
	{
		Template:    "dragging <element> reorders the list",
		Description: "Drag and drop reordering",
		Category:    CatEvents,
		Tags:        []string{"drag", "drop", "reorder", "sort"},
		Example:     "dragging a task reorders the list",
	},

	// ── Styling ──
	{
		Template:    "show the <field> in bold",
		Description: "Bold text styling",
		Category:    CatStyling,
		Tags:        []string{"bold", "style", "text", "font"},
		Example:     "show the transaction title in bold",
	},
	{
		Template:    "show the <field> aligned right",
		Description: "Right-align content",
		Category:    CatStyling,
		Tags:        []string{"align", "right", "layout"},
		Example:     "show the amount aligned right",
	},
	{
		Template:    "show the <field> as a colored badge",
		Description: "Display as a colored badge/chip",
		Category:    CatStyling,
		Tags:        []string{"badge", "chip", "color", "tag"},
		Example:     "show the category as a colored badge",
	},
	{
		Template:    "show the <field> in relative format",
		Description: "Display dates in relative format (e.g., '2 hours ago')",
		Category:    CatStyling,
		Tags:        []string{"date", "relative", "format", "time"},
		Example:     `show the date in relative format like "2 hours ago"`,
	},
	{
		Template:    "show the <field> in green if <condition>, red if <condition>",
		Description: "Conditional color styling based on data",
		Category:    CatStyling,
		Tags:        []string{"color", "conditional", "green", "red"},
		Example:     "each transaction shows the amount in green if income, red if expense",
	},
	{
		Template:    "make the layout responsive for mobile and tablet",
		Description: "Add responsive breakpoints",
		Category:    CatStyling,
		Tags:        []string{"responsive", "mobile", "tablet", "breakpoint"},
		Example:     "make the layout responsive for mobile and tablet",
	},

	// ── Forms & Inputs ──
	{
		Template:    "there is a text input for <purpose>",
		Description: "Add a text input field",
		Category:    CatForms,
		Tags:        []string{"input", "text", "field", "form"},
		Example:     "there is a text input for the post title",
	},
	{
		Template:    "there is a search bar that filters <data>",
		Description: "Add a search bar with filtering",
		Category:    CatForms,
		Tags:        []string{"search", "filter", "bar", "find"},
		Example:     "there is a search bar that filters transactions by title",
	},
	{
		Template:    "there is a dropdown to select <options>",
		Description: "Add a dropdown/select input",
		Category:    CatForms,
		Tags:        []string{"dropdown", "select", "options", "choose"},
		Example:     "there is a dropdown to filter by category",
	},
	{
		Template:    "there is a checkbox for <purpose>",
		Description: "Add a checkbox input",
		Category:    CatForms,
		Tags:        []string{"checkbox", "toggle", "boolean"},
		Example:     "there is a checkbox to mark the task as complete",
	},
	{
		Template:    "there is a date picker for <purpose>",
		Description: "Add a date picker input",
		Category:    CatForms,
		Tags:        []string{"date", "picker", "calendar"},
		Example:     "there is a date range picker to filter by date",
	},
	{
		Template:    "there is a file upload for <purpose>",
		Description: "Add a file upload input",
		Category:    CatForms,
		Tags:        []string{"file", "upload", "attachment"},
		Example:     "there is a file upload for the user avatar",
	},
	{
		Template:    "there is a form to create <data>",
		Description: "Auto-generate a form for creating a data entity",
		Category:    CatForms,
		Tags:        []string{"form", "create", "auto", "generate"},
		Example:     "there is a form to create a Transaction",
	},
	{
		Template:    "there is a floating button to <action>",
		Description: "Add a floating action button (FAB)",
		Category:    CatForms,
		Tags:        []string{"button", "floating", "fab", "action"},
		Example:     "there is a floating button to add a new transaction",
	},

	// ── APIs ──
	{
		Template:    "api <Name>:",
		Description: "Define an API endpoint",
		Category:    CatAPIs,
		Tags:        []string{"api", "endpoint", "route", "rest"},
		Example:     "api CreatePost:",
		Related:     []string{"requires authentication", "accepts <fields>", "respond with <data>"},
	},
	{
		Template:    "accepts <fields>",
		Description: "Declare accepted input parameters",
		Category:    CatAPIs,
		Tags:        []string{"accepts", "input", "parameters", "fields"},
		Example:     "accepts title, body, and category",
	},
	{
		Template:    "check that <validation>",
		Description: "Add input validation",
		Category:    CatAPIs,
		Tags:        []string{"check", "validate", "validation", "rule"},
		Example:     "check that title is not empty",
	},
	{
		Template:    "create a <Data> with <fields>",
		Description: "Insert a new record",
		Category:    CatAPIs,
		Tags:        []string{"create", "insert", "new", "record"},
		Example:     "create a Post with the given fields and current user as author",
	},
	{
		Template:    "update the <Data>",
		Description: "Update an existing record",
		Category:    CatAPIs,
		Tags:        []string{"update", "modify", "edit", "save"},
		Example:     "update the post status to published",
	},
	{
		Template:    "delete the <Data>",
		Description: "Delete a record",
		Category:    CatAPIs,
		Tags:        []string{"delete", "remove", "destroy"},
		Example:     "delete the post",
	},
	{
		Template:    "fetch <data> from <source>",
		Description: "Query/retrieve data",
		Category:    CatAPIs,
		Tags:        []string{"fetch", "query", "get", "retrieve", "read"},
		Example:     "fetch the user by user_id",
	},
	{
		Template:    "respond with <data>",
		Description: "Return a response from the API",
		Category:    CatAPIs,
		Tags:        []string{"respond", "return", "response", "output"},
		Example:     "respond with the created post",
	},
	{
		Template:    "if <condition>, respond with <message>",
		Description: "Conditional response (error handling, not found, etc.)",
		Category:    CatAPIs,
		Tags:        []string{"if", "respond", "error", "not found"},
		Example:     `if post does not exist, respond with "post not found"`,
	},
	{
		Template:    "sort by <field> newest first",
		Description: "Sort results in descending order",
		Category:    CatAPIs,
		Tags:        []string{"sort", "order", "newest", "descending"},
		Example:     "sort by published date newest first",
	},
	{
		Template:    "support filtering by <field>",
		Description: "Enable filtering on a field",
		Category:    CatAPIs,
		Tags:        []string{"filter", "where", "query", "search"},
		Example:     "support filtering by category",
	},
	{
		Template:    "paginate with <count> per page",
		Description: "Add pagination to results",
		Category:    CatAPIs,
		Tags:        []string{"paginate", "pagination", "page", "limit"},
		Example:     "paginate with 20 per page",
	},
	{
		Template:    "send <notification>",
		Description: "Trigger a side effect (email, notification, etc.)",
		Category:    CatAPIs,
		Tags:        []string{"send", "notify", "email", "notification"},
		Example:     `send welcome email with template "welcome"`,
	},

	// ── Security ──
	{
		Template:    "authentication:",
		Description: "Define the authentication configuration block",
		Category:    CatSecurity,
		Tags:        []string{"auth", "authentication", "login", "security"},
		Example:     "authentication:",
		Related:     []string{"method <auth_method>", "requires authentication"},
	},
	{
		Template:    "method JWT tokens that expire in <duration>",
		Description: "Configure JWT-based authentication",
		Category:    CatSecurity,
		Tags:        []string{"jwt", "token", "auth", "expire"},
		Example:     "method JWT tokens that expire in 7 days",
	},
	{
		Template:    "method <Provider> OAuth",
		Description: "Configure OAuth authentication with a provider",
		Category:    CatSecurity,
		Tags:        []string{"oauth", "google", "github", "social", "login"},
		Example:     "method Google OAuth with redirect to /auth/google/callback",
	},
	{
		Template:    "requires authentication",
		Description: "Mark an API endpoint as requiring auth",
		Category:    CatSecurity,
		Tags:        []string{"auth", "required", "protected", "secure"},
		Example:     "requires authentication",
	},
	{
		Template:    "passwords are hashed with <algorithm>",
		Description: "Configure password hashing",
		Category:    CatSecurity,
		Tags:        []string{"password", "hash", "bcrypt", "security"},
		Example:     "passwords are hashed with bcrypt using 12 rounds",
	},
	{
		Template:    "rate limit <scope> to <limit> per <period>",
		Description: "Add rate limiting to endpoints",
		Category:    CatSecurity,
		Tags:        []string{"rate limit", "throttle", "limit", "security"},
		Example:     "rate limit all endpoints to 100 requests per minute per user",
	},
	{
		Template:    "sanitize all text inputs against XSS",
		Description: "Enable XSS input sanitization",
		Category:    CatSecurity,
		Tags:        []string{"sanitize", "xss", "security", "input"},
		Example:     "sanitize all text inputs against XSS",
	},
	{
		Template:    "enable CORS only for <domain>",
		Description: "Configure CORS policy",
		Category:    CatSecurity,
		Tags:        []string{"cors", "domain", "origin", "security"},
		Example:     "enable CORS only for our frontend domain",
	},

	// ── Policies ──
	{
		Template:    "policy <Name>:",
		Description: "Define an authorization policy/role",
		Category:    CatPolicies,
		Tags:        []string{"policy", "role", "permission", "authorization"},
		Example:     "policy Admin:",
		Related:     []string{"can <permission>", "cannot <restriction>"},
	},
	{
		Template:    "can <permission>",
		Description: "Grant a permission to a policy",
		Category:    CatPolicies,
		Tags:        []string{"can", "allow", "permission", "grant"},
		Example:     "can view all users and their data",
	},
	{
		Template:    "cannot <restriction>",
		Description: "Restrict a capability for a policy",
		Category:    CatPolicies,
		Tags:        []string{"cannot", "deny", "restrict", "forbid"},
		Example:     "cannot delete published posts",
	},
	{
		Template:    "can create up to <limit> per <period>",
		Description: "Set a rate limit on a permission",
		Category:    CatPolicies,
		Tags:        []string{"limit", "rate", "quota", "permission"},
		Example:     "can create up to 50 posts per month",
	},

	// ── Database ──
	{
		Template:    "database:",
		Description: "Define database configuration block",
		Category:    CatDatabase,
		Tags:        []string{"database", "db", "storage", "config"},
		Example:     "database:",
		Related:     []string{"use <database_type>", "index <Data> by <field>"},
	},
	{
		Template:    "use <database_type>",
		Description: "Specify the database engine",
		Category:    CatDatabase,
		Tags:        []string{"use", "engine", "postgresql", "mysql", "mongodb"},
		Example:     "use PostgreSQL",
	},
	{
		Template:    "index <Data> by <field>",
		Description: "Create a database index on a field",
		Category:    CatDatabase,
		Tags:        []string{"index", "performance", "query", "optimize"},
		Example:     "index User by email",
	},
	{
		Template:    "when the app starts, create tables if they don't exist",
		Description: "Auto-create tables on startup",
		Category:    CatDatabase,
		Tags:        []string{"migrate", "create", "tables", "startup"},
		Example:     "when the app starts, create tables if they don't exist",
	},
	{
		Template:    "backup daily at <time>",
		Description: "Schedule automatic database backups",
		Category:    CatDatabase,
		Tags:        []string{"backup", "schedule", "daily", "recovery"},
		Example:     "backup daily at 3am",
	},
	{
		Template:    "keep backups for <duration>",
		Description: "Set backup retention policy",
		Category:    CatDatabase,
		Tags:        []string{"backup", "retention", "keep", "policy"},
		Example:     "keep backups for 30 days",
	},

	// ── Workflows ──
	{
		Template:    "when <event>:",
		Description: "Define a workflow triggered by an event",
		Category:    CatWorkflows,
		Tags:        []string{"when", "event", "trigger", "workflow"},
		Example:     "when a user signs up:",
		Related:     []string{"send <notification>", "after <delay>, <action>"},
	},
	{
		Template:    "after <delay>, <action>",
		Description: "Schedule a delayed action in a workflow",
		Category:    CatWorkflows,
		Tags:        []string{"after", "delay", "schedule", "timer"},
		Example:     `after 3 days, send email with template "getting-started"`,
	},
	{
		Template:    "notify all <audience> of <event>",
		Description: "Send notifications to a group",
		Category:    CatWorkflows,
		Tags:        []string{"notify", "notification", "alert", "audience"},
		Example:     "notify all followers of the author",
	},
	{
		Template:    "assign <policy> policy",
		Description: "Assign a policy/role in a workflow",
		Category:    CatWorkflows,
		Tags:        []string{"assign", "policy", "role", "workflow"},
		Example:     "assign FreeUser policy",
	},
	{
		Template:    "when a <Data> is <action>:",
		Description: "Trigger workflow on data lifecycle event",
		Category:    CatWorkflows,
		Tags:        []string{"when", "created", "updated", "deleted", "published"},
		Example:     "when a post is published:",
	},

	// ── Integrations ──
	{
		Template:    "integrate with <Service>:",
		Description: "Connect to a third-party service",
		Category:    CatIntegrations,
		Tags:        []string{"integrate", "service", "third-party", "connect"},
		Example:     "integrate with Stripe:",
		Related:     []string{"api key from environment variable <VAR>", "use for <purpose>"},
	},
	{
		Template:    "api key from environment variable <VAR>",
		Description: "Configure API key from environment",
		Category:    CatIntegrations,
		Tags:        []string{"api key", "environment", "secret", "config"},
		Example:     "api key from environment variable STRIPE_KEY",
	},
	{
		Template:    "use for <purpose>",
		Description: "Describe the integration purpose",
		Category:    CatIntegrations,
		Tags:        []string{"purpose", "use", "payment", "email"},
		Example:     "use for payment processing",
	},
	{
		Template:    `integrate with custom api "<Name>":`,
		Description: "Connect to a custom/internal API",
		Category:    CatIntegrations,
		Tags:        []string{"custom", "api", "internal", "microservice"},
		Example:     `integrate with custom api "InventoryService":`,
	},
	{
		Template:    "endpoint <Name>:",
		Description: "Define an endpoint within a custom API integration",
		Category:    CatIntegrations,
		Tags:        []string{"endpoint", "method", "path", "route"},
		Example:     "endpoint CheckStock:",
	},

	// ── Architecture ──
	{
		Template:    "architecture: monolith",
		Description: "Use monolithic architecture",
		Category:    CatArchitecture,
		Tags:        []string{"monolith", "architecture", "single", "simple"},
	},
	{
		Template:    "architecture: microservices",
		Description: "Use microservices architecture",
		Category:    CatArchitecture,
		Tags:        []string{"microservices", "architecture", "distributed"},
		Related:     []string{"service <Name>:", "gateway:"},
	},
	{
		Template:    "architecture: serverless",
		Description: "Use serverless/FaaS architecture",
		Category:    CatArchitecture,
		Tags:        []string{"serverless", "lambda", "function", "faas"},
	},
	{
		Template:    "architecture: event-driven microservices",
		Description: "Use event-driven microservices with message broker",
		Category:    CatArchitecture,
		Tags:        []string{"event-driven", "message", "broker", "async"},
	},
	{
		Template:    "service <Name>:",
		Description: "Define a microservice",
		Category:    CatArchitecture,
		Tags:        []string{"service", "microservice", "component"},
		Example:     "service UserService:",
	},
	{
		Template:    "handles <responsibilities>",
		Description: "Describe what a microservice handles",
		Category:    CatArchitecture,
		Tags:        []string{"handles", "responsibility", "domain"},
		Example:     "handles user management and authentication",
	},
	{
		Template:    "talks to <Service> to <purpose>",
		Description: "Declare service-to-service communication",
		Category:    CatArchitecture,
		Tags:        []string{"talks to", "communication", "dependency"},
		Example:     "talks to PaymentService to process payments",
	},
	{
		Template:    "gateway:",
		Description: "Define the API gateway configuration",
		Category:    CatArchitecture,
		Tags:        []string{"gateway", "api", "routing", "proxy"},
	},
	{
		Template:    "publishes <event> when <condition>",
		Description: "Publish an event to the message broker",
		Category:    CatArchitecture,
		Tags:        []string{"publish", "event", "message", "async"},
		Example:     `publishes "order.created" when an order is placed`,
	},
	{
		Template:    "listens for <event> and <action>",
		Description: "Subscribe to an event from the message broker",
		Category:    CatArchitecture,
		Tags:        []string{"listen", "subscribe", "event", "handler"},
		Example:     `listens for "order.created" and processes payment`,
	},

	// ── DevOps ──
	{
		Template:    "when code is pushed to <branch>:",
		Description: "Define a CI/CD pipeline trigger",
		Category:    CatDevOps,
		Tags:        []string{"ci", "cd", "pipeline", "push", "trigger"},
		Example:     "when code is pushed to main:",
	},
	{
		Template:    "run all tests",
		Description: "Run the test suite in a pipeline",
		Category:    CatDevOps,
		Tags:        []string{"test", "ci", "pipeline", "verify"},
	},
	{
		Template:    "check code formatting",
		Description: "Run code formatter check in a pipeline",
		Category:    CatDevOps,
		Tags:        []string{"format", "lint", "ci", "style"},
	},
	{
		Template:    "check for security vulnerabilities",
		Description: "Run security vulnerability scan",
		Category:    CatDevOps,
		Tags:        []string{"security", "vulnerability", "scan", "audit"},
	},
	{
		Template:    "deploy to <environment>",
		Description: "Deploy to a target environment",
		Category:    CatDevOps,
		Tags:        []string{"deploy", "release", "environment", "ship"},
		Example:     "deploy to staging",
	},
	{
		Template:    "if <check> fails, rollback automatically",
		Description: "Auto-rollback on failure",
		Category:    CatDevOps,
		Tags:        []string{"rollback", "revert", "failure", "safety"},
		Example:     "if health check fails, rollback automatically",
	},
	{
		Template:    "environment <name>:",
		Description: "Define a deployment environment",
		Category:    CatDevOps,
		Tags:        []string{"environment", "staging", "production", "config"},
		Example:     "environment staging:",
	},
	{
		Template:    "url is <domain>",
		Description: "Set the environment URL",
		Category:    CatDevOps,
		Tags:        []string{"url", "domain", "host"},
		Example:     "url is staging.example.com",
	},
	{
		Template:    "track <metric>",
		Description: "Monitor a metric",
		Category:    CatDevOps,
		Tags:        []string{"track", "monitor", "metric", "observability"},
		Example:     "track response time for all API endpoints",
	},
	{
		Template:    "alert on <channel> if <condition>",
		Description: "Set up alerting for a condition",
		Category:    CatDevOps,
		Tags:        []string{"alert", "notification", "monitor", "threshold"},
		Example:     "alert on Slack if error rate exceeds 1%",
	},
	{
		Template:    "log <what> to <service>",
		Description: "Configure logging destination",
		Category:    CatDevOps,
		Tags:        []string{"log", "logging", "service", "output"},
		Example:     "log all errors to DataDog",
	},

	// ── Theme ──
	{
		Template:    "theme:",
		Description: "Define the visual theme/design system",
		Category:    CatTheme,
		Tags:        []string{"theme", "design", "style", "visual"},
		Example:     "theme:",
		Related:     []string{"primary color is <color>", "font is <font>"},
	},
	{
		Template:    "primary color is <color>",
		Description: "Set the primary brand color",
		Category:    CatTheme,
		Tags:        []string{"primary", "color", "brand", "palette"},
		Example:     "primary color is #6C5CE7",
	},
	{
		Template:    "secondary color is <color>",
		Description: "Set the secondary brand color",
		Category:    CatTheme,
		Tags:        []string{"secondary", "color", "palette"},
		Example:     "secondary color is #00B894",
	},
	{
		Template:    "danger color is <color>",
		Description: "Set the danger/error color",
		Category:    CatTheme,
		Tags:        []string{"danger", "error", "color", "red"},
		Example:     "danger color is #D63031",
	},
	{
		Template:    "font is <font> for body and <font> for headings",
		Description: "Set body and heading fonts",
		Category:    CatTheme,
		Tags:        []string{"font", "typography", "body", "heading"},
		Example:     "font is Inter for body and Poppins for headings",
	},
	{
		Template:    "border radius is <style>",
		Description: "Set the border radius style (sharp, smooth, rounded, pill)",
		Category:    CatTheme,
		Tags:        []string{"border", "radius", "rounded", "sharp"},
		Example:     "border radius is smooth on all elements",
	},
	{
		Template:    "dark mode is supported",
		Description: "Enable dark mode support",
		Category:    CatTheme,
		Tags:        []string{"dark mode", "theme", "toggle", "night"},
		Example:     "dark mode is supported and toggles from the header",
	},
	{
		Template:    "spacing is <density>",
		Description: "Set spacing density (compact, comfortable, spacious)",
		Category:    CatTheme,
		Tags:        []string{"spacing", "density", "compact", "comfortable"},
		Example:     "spacing is comfortable",
	},
	{
		Template:    "use <design_system> design system",
		Description: "Use a pre-built design system (Material, Shadcn, etc.)",
		Category:    CatTheme,
		Tags:        []string{"design system", "material", "shadcn", "ant", "chakra"},
		Example:     "use Shadcn design system",
	},

	// ── Build ──
	{
		Template:    "build with:",
		Description: "Define the build target configuration",
		Category:    CatBuild,
		Tags:        []string{"build", "target", "config", "stack"},
		Example:     "build with:",
		Related:     []string{"frontend using <framework>", "backend using <language>"},
	},
	{
		Template:    "frontend using <framework> with <language>",
		Description: "Set the frontend framework and language",
		Category:    CatBuild,
		Tags:        []string{"frontend", "react", "vue", "angular", "svelte"},
		Example:     "frontend using React with TypeScript",
	},
	{
		Template:    "backend using <language> with <framework>",
		Description: "Set the backend language and framework",
		Category:    CatBuild,
		Tags:        []string{"backend", "node", "python", "go", "express", "fastapi"},
		Example:     "backend using Node with Express",
	},
	{
		Template:    "database using <database>",
		Description: "Set the database engine",
		Category:    CatBuild,
		Tags:        []string{"database", "postgresql", "mysql", "mongodb"},
		Example:     "database using PostgreSQL",
	},
	{
		Template:    "deploy to <platform>",
		Description: "Set the deployment platform",
		Category:    CatBuild,
		Tags:        []string{"deploy", "docker", "aws", "gcp", "vercel"},
		Example:     "deploy to Docker",
	},

	// ── Conditional ──
	{
		Template:    "if <condition>, show <thing>",
		Description: "Conditionally render content",
		Category:    CatConditional,
		Tags:        []string{"if", "conditional", "show", "render"},
		Example:     "if user is admin, show the admin panel",
	},
	{
		Template:    `if no <data> match, show "<message>"`,
		Description: "Show an empty state message",
		Category:    CatConditional,
		Tags:        []string{"empty", "state", "no data", "message"},
		Example:     `if no transactions match, show "No transactions found"`,
	},
	{
		Template:    "while loading, show a spinner",
		Description: "Show a loading indicator while data loads",
		Category:    CatConditional,
		Tags:        []string{"loading", "spinner", "skeleton", "wait"},
		Example:     "while loading, show a skeleton screen",
	},
	{
		Template:    "if there is an error, show the error message",
		Description: "Show error state when something fails",
		Category:    CatConditional,
		Tags:        []string{"error", "message", "display", "handle"},
		Example:     "if there is an error, show the error message",
	},

	// ── Errors ──
	{
		Template:    "if <service> is unreachable:",
		Description: "Handle service unavailability",
		Category:    CatErrors,
		Tags:        []string{"error", "unreachable", "down", "unavailable"},
		Example:     "if database is unreachable:",
	},
	{
		Template:    "retry <count> times with <delay>",
		Description: "Retry a failed operation",
		Category:    CatErrors,
		Tags:        []string{"retry", "resilience", "delay", "attempt"},
		Example:     "retry 3 times with 1 second delay",
	},
	{
		Template:    "if still failing, respond with <message>",
		Description: "Final fallback after retries exhausted",
		Category:    CatErrors,
		Tags:        []string{"fallback", "fail", "respond", "degraded"},
		Example:     `if still failing, respond with "service temporarily unavailable"`,
	},
	{
		Template:    "alert the <team> via <channel>",
		Description: "Alert a team on critical failure",
		Category:    CatErrors,
		Tags:        []string{"alert", "team", "slack", "notify"},
		Example:     "alert the engineering team via Slack",
	},
	{
		Template:    "if an api request fails validation:",
		Description: "Handle API validation failures",
		Category:    CatErrors,
		Tags:        []string{"validation", "fail", "api", "error"},
		Example:     "if an api request fails validation:",
	},
	{
		Template:    "do not reveal internal details",
		Description: "Hide internal error details from responses",
		Category:    CatErrors,
		Tags:        []string{"security", "hide", "internal", "detail"},
		Example:     "do not reveal internal details",
	},
}
