package ir

import (
	"strings"

	"github.com/barun-bash/human/internal/parser"
)

// Build transforms a parsed AST into a framework-agnostic Intent IR.
// The returned Application contains all information needed by code generators.
func Build(prog *parser.Program) (*Application, error) {
	app := &Application{}

	// App declaration
	if prog.App != nil {
		app.Name = prog.App.Name
		app.Platform = prog.App.Platform
	}

	// Build configuration
	if prog.Build != nil {
		app.Config = buildConfig(prog.Build)
	}

	// Data models
	for _, d := range prog.Data {
		app.Data = append(app.Data, buildDataModel(d))
	}

	// Pages
	for _, p := range prog.Pages {
		app.Pages = append(app.Pages, buildPage(p))
	}

	// Components
	for _, c := range prog.Components {
		app.Components = append(app.Components, buildComponent(c))
	}

	// APIs
	for _, a := range prog.APIs {
		app.APIs = append(app.APIs, buildEndpoint(a))
	}

	// Policies
	for _, p := range prog.Policies {
		app.Policies = append(app.Policies, buildPolicy(p))
	}

	// Workflows and pipelines (separated by trigger type)
	for _, w := range prog.Workflows {
		if isPipelineTrigger(w.Event) {
			app.Pipelines = append(app.Pipelines, buildPipeline(w))
		} else {
			app.Workflows = append(app.Workflows, buildWorkflow(w))
		}
	}

	// Theme
	if prog.Theme != nil {
		app.Theme = buildTheme(prog.Theme)
	}

	// Authentication
	if prog.Authentication != nil {
		app.Auth = buildAuth(prog.Authentication)
	}

	// Database
	if prog.Database != nil {
		app.Database = buildDatabase(prog.Database)
	}

	// Integrations
	for _, i := range prog.Integrations {
		app.Integrations = append(app.Integrations, buildIntegration(i))
	}

	// Environments
	for _, e := range prog.Environments {
		app.Environments = append(app.Environments, buildEnvironment(e))
	}

	// Error handlers
	for _, e := range prog.ErrorHandlers {
		app.ErrorHandlers = append(app.ErrorHandlers, buildErrorHandler(e))
	}

	return app, nil
}

// ── Build Config ──

func buildConfig(b *parser.BuildDeclaration) *BuildConfig {
	cfg := &BuildConfig{}
	for _, s := range b.Statements {
		text := s.Text
		lower := strings.ToLower(text)
		switch {
		case strings.HasPrefix(lower, "frontend using "):
			cfg.Frontend = text[len("frontend using "):]
		case strings.HasPrefix(lower, "backend using "):
			cfg.Backend = text[len("backend using "):]
		case strings.HasPrefix(lower, "database using "):
			cfg.Database = text[len("database using "):]
		case strings.HasPrefix(lower, "deploy to "):
			cfg.Deploy = text[len("deploy to "):]
		}
	}
	return cfg
}

// ── Data Models ──

func buildDataModel(d *parser.DataDeclaration) *DataModel {
	model := &DataModel{Name: d.Name}

	for _, f := range d.Fields {
		df := &DataField{
			Name:     f.Name,
			Required: true,
		}

		// Determine type
		if len(f.EnumValues) > 0 {
			df.Type = "enum"
			df.EnumValues = f.EnumValues
		} else if f.Type != "" {
			df.Type = f.Type
		} else {
			df.Type = "text" // default
		}

		// Apply modifiers
		for _, mod := range f.Modifiers {
			switch mod {
			case "optional":
				df.Required = false
			case "unique":
				df.Unique = true
			case "encrypted":
				df.Encrypted = true
			}
		}

		if f.Default != "" {
			df.Default = f.Default
		}

		model.Fields = append(model.Fields, df)
	}

	for _, r := range d.Relationships {
		rel := &Relation{
			Kind:   r.Kind,
			Target: r.Target,
		}
		if r.Through != "" {
			rel.Kind = "has_many_through"
			rel.Through = r.Through
		}
		model.Relations = append(model.Relations, rel)
	}

	return model
}

// ── Pages ──

func buildPage(p *parser.PageDeclaration) *Page {
	page := &Page{Name: p.Name}
	for _, s := range p.Statements {
		page.Content = append(page.Content, classifyAction(s))
	}
	return page
}

// ── Components ──

func buildComponent(c *parser.ComponentDeclaration) *Component {
	comp := &Component{Name: c.Name}

	// Parse "accepts" into props: "task as Task" → Prop{Name:"task", Type:"Task"}
	for i := 0; i < len(c.Accepts); i++ {
		raw := c.Accepts[i]
		parts := strings.Fields(raw)
		prop := &Prop{Name: raw}
		if len(parts) >= 3 && strings.ToLower(parts[1]) == "as" {
			prop.Name = parts[0]
			prop.Type = parts[2]
		}
		comp.Props = append(comp.Props, prop)
	}

	for _, s := range c.Statements {
		comp.Content = append(comp.Content, classifyAction(s))
	}
	return comp
}

// ── API Endpoints ──

func buildEndpoint(a *parser.APIDeclaration) *Endpoint {
	ep := &Endpoint{
		Name: a.Name,
		Auth: a.Auth,
	}

	for _, name := range a.Accepts {
		ep.Params = append(ep.Params, &Param{Name: name})
	}

	for _, s := range a.Statements {
		// Extract structured validation from "check" statements
		if s.Kind == "check" {
			if rule := parseValidation(s.Text); rule != nil {
				ep.Validation = append(ep.Validation, rule)
				continue
			}
		}
		ep.Steps = append(ep.Steps, classifyAction(s))
	}

	return ep
}

// parseValidation extracts a structured validation rule from a "check" statement.
// Returns nil if the text cannot be parsed into a known pattern.
func parseValidation(text string) *ValidationRule {
	lower := strings.ToLower(text)

	// "check that <field> is not empty"
	if strings.Contains(lower, "is not empty") {
		field := extractFieldFromCheck(text, "is not empty")
		if field != "" {
			return &ValidationRule{Field: field, Rule: "not_empty"}
		}
	}

	// "check that <field> is a valid email"
	if strings.Contains(lower, "is a valid") {
		field := extractFieldFromCheck(text, "is a valid")
		valType := extractAfter(lower, "is a valid ")
		if field != "" && valType != "" {
			return &ValidationRule{Field: field, Rule: "valid_" + valType}
		}
	}

	// "check that <field> is at least <n> characters"
	if strings.Contains(lower, "is at least") && strings.Contains(lower, "characters") {
		field := extractFieldFromCheck(text, "is at least")
		value := extractBetween(lower, "is at least ", " characters")
		if field != "" {
			return &ValidationRule{Field: field, Rule: "min_length", Value: strings.TrimSpace(value)}
		}
	}

	// "check that <field> is less than <n> characters"
	if strings.Contains(lower, "is less than") && strings.Contains(lower, "characters") {
		field := extractFieldFromCheck(text, "is less than")
		value := extractBetween(lower, "is less than ", " characters")
		if field != "" {
			return &ValidationRule{Field: field, Rule: "max_length", Value: strings.TrimSpace(value)}
		}
	}

	// "check that <field> is not already taken"
	if strings.Contains(lower, "is not already taken") {
		field := extractFieldFromCheck(text, "is not already taken")
		if field != "" {
			return &ValidationRule{Field: field, Rule: "unique"}
		}
	}

	// "check that <field> is in the future"
	if strings.Contains(lower, "is in the future") {
		field := extractFieldFromCheck(text, "is in the future")
		if field != "" {
			return &ValidationRule{Field: field, Rule: "future_date"}
		}
	}

	// "check that <field> matches ..."
	if strings.Contains(lower, "matches") {
		field := extractFieldFromCheck(text, "matches")
		if field != "" {
			return &ValidationRule{Field: field, Rule: "matches"}
		}
	}

	// "check that current user is the owner or an admin"
	if strings.Contains(lower, "current user is") {
		return &ValidationRule{Field: "current_user", Rule: "authorization", Value: extractAfter(lower, "current user is ")}
	}

	return nil
}

// extractFieldFromCheck extracts the field name from "check that <field> <predicate>".
func extractFieldFromCheck(text, predicate string) string {
	lower := strings.ToLower(text)
	// Find "check that " prefix
	idx := strings.Index(lower, "check that ")
	if idx == -1 {
		// Try without "check that"
		idx = 0
	} else {
		idx += len("check that ")
	}
	// Find the predicate
	predIdx := strings.Index(lower[idx:], strings.ToLower(predicate))
	if predIdx == -1 {
		return ""
	}
	field := strings.TrimSpace(text[idx : idx+predIdx])
	return field
}

// ── Policies ──

func buildPolicy(p *parser.PolicyDeclaration) *Policy {
	pol := &Policy{Name: p.Name}
	for _, r := range p.Rules {
		rule := &PolicyRule{Text: r.Text}
		if r.Allowed {
			pol.Permissions = append(pol.Permissions, rule)
		} else {
			pol.Restrictions = append(pol.Restrictions, rule)
		}
	}
	return pol
}

// ── Workflows & Pipelines ──

// isPipelineTrigger returns true if the workflow event describes a CI/CD trigger.
func isPipelineTrigger(event string) bool {
	lower := strings.ToLower(event)
	return strings.HasPrefix(lower, "code is pushed") ||
		strings.HasPrefix(lower, "code is merged")
}

func buildWorkflow(w *parser.WorkflowDeclaration) *Workflow {
	wf := &Workflow{Trigger: w.Event}
	for _, s := range w.Statements {
		wf.Steps = append(wf.Steps, classifyAction(s))
	}
	return wf
}

func buildPipeline(w *parser.WorkflowDeclaration) *Pipeline {
	p := &Pipeline{Trigger: w.Event}
	for _, s := range w.Statements {
		p.Steps = append(p.Steps, classifyAction(s))
	}
	return p
}

// ── Theme ──

func buildTheme(t *parser.ThemeDeclaration) *Theme {
	theme := &Theme{
		Colors:  make(map[string]string),
		Fonts:   make(map[string]string),
		Options: make(map[string]string),
	}

	for _, s := range t.Properties {
		text := s.Text
		lower := strings.ToLower(text)

		switch {
		// "primary color is #6C5CE7"
		case strings.Contains(lower, "color is"):
			parts := strings.SplitN(lower, "color is", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				theme.Colors[name] = value
			}

		// "font is Inter for body and Poppins for headings"
		case strings.HasPrefix(lower, "font is"):
			fontText := text[len("font is"):]
			parseFontEntry(theme.Fonts, strings.TrimSpace(fontText))

		default:
			// Generic option: "dark mode is supported and toggles from the header"
			parts := strings.SplitN(text, " is ", 2)
			if len(parts) == 2 {
				theme.Options[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}

	return theme
}

// parseFontEntry parses "Inter for body and Poppins for headings" into the fonts map.
func parseFontEntry(fonts map[string]string, text string) {
	// Split by "and" to handle "Inter for body and Poppins for headings"
	segments := strings.Split(text, " and ")
	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		parts := strings.SplitN(seg, " for ", 2)
		if len(parts) == 2 {
			fonts[strings.TrimSpace(parts[1])] = strings.TrimSpace(parts[0])
		}
	}
}

// ── Authentication ──

func buildAuth(a *parser.AuthenticationDeclaration) *Auth {
	auth := &Auth{}

	for _, s := range a.Statements {
		lower := strings.ToLower(s.Text)

		if strings.HasPrefix(lower, "method ") {
			method := parseAuthMethod(s.Text[len("method "):])
			auth.Methods = append(auth.Methods, method)
		} else {
			auth.Rules = append(auth.Rules, classifyAction(s))
		}
	}

	return auth
}

// parseAuthMethod parses "JWT tokens that expire in 7 days" or
// "Google OAuth with redirect to /auth/google/callback".
func parseAuthMethod(text string) *AuthMethod {
	lower := strings.ToLower(text)
	method := &AuthMethod{Config: make(map[string]string)}

	switch {
	case strings.Contains(lower, "jwt"):
		method.Type = "jwt"
		if idx := strings.Index(lower, "expire in "); idx != -1 {
			method.Config["expiration"] = strings.TrimSpace(text[idx+len("expire in "):])
		}

	case strings.Contains(lower, "oauth"):
		method.Type = "oauth"
		// Extract provider: word before "OAuth"
		idx := strings.Index(lower, "oauth")
		if idx > 0 {
			method.Provider = strings.TrimSpace(text[:idx])
		}
		if rIdx := strings.Index(lower, "redirect to "); rIdx != -1 {
			method.Config["callback_url"] = strings.TrimSpace(text[rIdx+len("redirect to "):])
		}

	default:
		method.Type = "custom"
		method.Config["description"] = text
	}

	return method
}

// ── Database ──

func buildDatabase(d *parser.DatabaseDeclaration) *DatabaseConfig {
	db := &DatabaseConfig{}

	for _, s := range d.Statements {
		lower := strings.ToLower(s.Text)

		switch {
		case strings.HasPrefix(lower, "use "):
			db.Engine = s.Text[len("use "):]

		case strings.HasPrefix(lower, "index "):
			if idx := parseIndex(s.Text[len("index "):]); idx != nil {
				db.Indexes = append(db.Indexes, idx)
			}

		default:
			db.Rules = append(db.Rules, classifyAction(s))
		}
	}

	return db
}

// parseIndex parses "User by email" or "Task by user and status".
func parseIndex(text string) *Index {
	parts := strings.SplitN(text, " by ", 2)
	if len(parts) != 2 {
		return nil
	}
	entity := strings.TrimSpace(parts[0])
	fieldStr := strings.TrimSpace(parts[1])

	// Split fields by " and "
	rawFields := strings.Split(fieldStr, " and ")
	var fields []string
	for _, f := range rawFields {
		f = strings.TrimSpace(f)
		if f != "" {
			fields = append(fields, f)
		}
	}

	return &Index{Entity: entity, Fields: fields}
}

// ── Integrations ──

func buildIntegration(i *parser.IntegrationDeclaration) *Integration {
	integ := &Integration{
		Service:     i.Service,
		Credentials: make(map[string]string),
	}

	for _, s := range i.Statements {
		lower := strings.ToLower(s.Text)

		switch {
		// "api key from environment variable SENDGRID_API_KEY"
		case strings.Contains(lower, "from environment variable"):
			parts := strings.SplitN(s.Text, "from environment variable ", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0]) // "api key"
				envVar := strings.TrimSpace(parts[1])
				integ.Credentials[key] = envVar
			}

		// "use for sending transactional emails"
		case strings.HasPrefix(lower, "use for "):
			integ.Purpose = strings.TrimSpace(s.Text[len("use for "):])
		}
	}

	return integ
}

// ── Environments ──

func buildEnvironment(e *parser.EnvironmentDeclaration) *Environment {
	env := &Environment{
		Name:   e.Name,
		Config: make(map[string]string),
	}

	for _, s := range e.Statements {
		lower := strings.ToLower(s.Text)

		// "url is staging.taskflow.example.com"
		if strings.Contains(lower, " is ") {
			parts := strings.SplitN(s.Text, " is ", 2)
			if len(parts) == 2 {
				env.Config[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
			continue
		}

		env.Rules = append(env.Rules, classifyAction(s))
	}

	return env
}

// ── Error Handlers ──

func buildErrorHandler(e *parser.ErrorHandlerDeclaration) *ErrorHandler {
	eh := &ErrorHandler{Condition: e.Condition}
	for _, s := range e.Statements {
		eh.Steps = append(eh.Steps, classifyAction(s))
	}
	return eh
}

// ── Action Classification ──

// classifyAction converts a parser Statement into a typed Action.
// The statement's Kind is mapped to an Action Type for code generators.
func classifyAction(s *parser.Statement) *Action {
	action := &Action{Text: s.Text}

	switch s.Kind {
	// Display
	case "show", "display", "render":
		action.Type = "display"

	// Interaction
	case "clicking", "dragging", "scrolling", "hovering", "typing":
		action.Type = "interact"

	// Input elements
	case "there":
		action.Type = "input"

	// Navigation
	case "navigate", "navigates", "redirect":
		action.Type = "navigate"

	// Conditions
	case "if", "when", "while", "unless", "until":
		action.Type = "condition"

	// Iteration
	case "each", "every", "for":
		action.Type = "loop"

	// Data queries
	case "fetch", "get", "find", "load", "support", "paginate", "sort":
		action.Type = "query"

	// Data mutations
	case "create":
		action.Type = "create"
	case "update", "set":
		action.Type = "update"
	case "delete", "remove":
		action.Type = "delete"

	// Validation
	case "check", "validate":
		action.Type = "validate"

	// Response
	case "respond":
		action.Type = "respond"

	// Communication
	case "send", "notify":
		action.Type = "send"

	// Assignment
	case "assign":
		action.Type = "assign"

	// Alerting
	case "alert":
		action.Type = "alert"

	// Logging/tracking
	case "log", "track":
		action.Type = "log"

	// Timing
	case "after":
		action.Type = "delay"

	// Retry
	case "retry":
		action.Type = "retry"

	// Build/deploy
	case "run", "build", "deploy", "report":
		action.Type = "configure"

	// Configuration/rules
	case "method", "rate", "sanitize", "enable", "passwords", "all",
		"use", "index", "backup", "keep",
		"frontend", "backend", "database":
		action.Type = "configure"

	default:
		action.Type = "configure"
	}

	return action
}

// ── String helpers ──

// extractAfter returns the substring after the first occurrence of prefix.
func extractAfter(s, prefix string) string {
	idx := strings.Index(s, prefix)
	if idx == -1 {
		return ""
	}
	return strings.TrimSpace(s[idx+len(prefix):])
}

// extractBetween returns the substring between start and end markers.
func extractBetween(s, start, end string) string {
	sIdx := strings.Index(s, start)
	if sIdx == -1 {
		return ""
	}
	after := s[sIdx+len(start):]
	eIdx := strings.Index(after, end)
	if eIdx == -1 {
		return after
	}
	return after[:eIdx]
}
