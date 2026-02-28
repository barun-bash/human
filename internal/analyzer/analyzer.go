package analyzer

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/barun-bash/human/internal/codegen/themes"
	cerr "github.com/barun-bash/human/internal/errors"
	"github.com/barun-bash/human/internal/ir"
)

const suggestionThreshold = 0.6

// Analyze performs semantic analysis on a compiled IR Application.
// It validates cross-references, completeness, and consistency,
// returning any diagnostics found.
func Analyze(app *ir.Application, file string) *cerr.CompilerErrors {
	errs := cerr.New(file)

	// Build symbol tables
	models, modelList := collectNames(app.Data, func(m *ir.DataModel) string { return m.Name })
	pages, pageList := collectNames(app.Pages, func(p *ir.Page) string { return p.Name })
	_, componentList := collectNames(app.Components, func(c *ir.Component) string { return c.Name })
	apis, apiList := collectNames(app.APIs, func(a *ir.Endpoint) string { return a.Name })
	_, policyList := collectNames(app.Policies, func(p *ir.Policy) string { return p.Name })

	// componentList and policyList reserved for future cross-reference checks
	_ = componentList
	_ = policyList

	// 1. Duplicate names
	checkDuplicates(errs, app.Data, func(m *ir.DataModel) string { return m.Name }, "data model", "E301")
	checkDuplicates(errs, app.Pages, func(p *ir.Page) string { return p.Name }, "page", "E302")
	checkDuplicates(errs, app.Components, func(c *ir.Component) string { return c.Name }, "component", "E303")
	checkDuplicates(errs, app.APIs, func(a *ir.Endpoint) string { return a.Name }, "API", "E304")
	checkDuplicates(errs, app.Policies, func(p *ir.Policy) string { return p.Name }, "policy", "E305")

	// 2. Duplicate fields within a model
	checkDuplicateFields(errs, app.Data)

	// 3. Data model relation references
	checkRelationTargets(errs, app.Data, models, modelList)

	// 4. Through-table validation
	checkThroughTables(errs, app.Data, models)

	// 5. Database index validation
	checkIndexes(errs, app)

	// 6. Page navigation references
	checkPageNavigation(errs, app.Pages, pages, pageList)

	// 7. API model references
	checkAPIModelReferences(errs, app.APIs, models, modelList)

	// 8. Completeness
	checkCompleteness(errs, app, apis, apiList)

	// 9. Design system validation
	checkDesignSystem(errs, app)

	// 10. Architecture validation
	checkArchitecture(errs, app, models, modelList)

	// 11. Integration validation
	checkIntegrations(errs, app)

	// 12. Workflow-integration cross-references
	checkWorkflowIntegrationRefs(errs, app)

	// 13. Validation field references
	checkValidationFields(errs, app.APIs)

	// 14. Database engine validation
	checkDatabaseEngine(errs, app)

	// 15. Gateway route references
	checkGatewayRoutes(errs, app)

	// 16. Monitoring channel references
	checkMonitoringChannels(errs, app)

	// 17. Policy model references
	checkPolicyModelRefs(errs, app, models, modelList)

	// 18. Workflow/ErrorHandler/Pipeline model references
	checkActionModelRefs(errs, app, models, modelList)

	// 19. Trigger model references
	checkTriggerModelRefs(errs, app, models, modelList)

	return errs
}

// ── Symbol table helpers ──

// collectNames builds a case-insensitive lookup map and an original-case list
// from a slice of IR nodes.
func collectNames[T any](items []T, nameFunc func(T) string) (map[string]bool, []string) {
	m := make(map[string]bool)
	var list []string
	for _, item := range items {
		name := nameFunc(item)
		m[strings.ToLower(name)] = true
		list = append(list, name)
	}
	return m, list
}

// ── Duplicate detection ──

func checkDuplicates[T any](errs *cerr.CompilerErrors, items []T, nameFunc func(T) string, kind, code string) {
	seen := make(map[string]bool)
	for _, item := range items {
		name := nameFunc(item)
		lower := strings.ToLower(name)
		if seen[lower] {
			errs.AddError(code, fmt.Sprintf("Duplicate %s name %q", kind, name))
		}
		seen[lower] = true
	}
}

// ── Duplicate fields within a model ──

func checkDuplicateFields(errs *cerr.CompilerErrors, models []*ir.DataModel) {
	for _, model := range models {
		seen := make(map[string]bool)
		for _, field := range model.Fields {
			lower := strings.ToLower(field.Name)
			if seen[lower] {
				errs.AddError("E306", fmt.Sprintf("Data model %q has duplicate field %q", model.Name, field.Name))
			}
			seen[lower] = true
		}
	}
}

// ── Relation target validation ──

func checkRelationTargets(errs *cerr.CompilerErrors, models []*ir.DataModel, known map[string]bool, knownList []string) {
	for _, model := range models {
		for _, rel := range model.Relations {
			if !known[strings.ToLower(rel.Target)] {
				msg := fmt.Sprintf("Data model %q references %q which does not exist", model.Name, rel.Target)
				if suggestion := cerr.FindClosest(rel.Target, knownList, suggestionThreshold); suggestion != "" {
					errs.AddErrorWithSuggestion("E101", msg, fmt.Sprintf("Did you mean %q?", suggestion))
				} else {
					errs.AddError("E101", msg)
				}
			}

			if rel.Through != "" && !known[strings.ToLower(rel.Through)] {
				msg := fmt.Sprintf("Data model %q references through-model %q which does not exist", model.Name, rel.Through)
				if suggestion := cerr.FindClosest(rel.Through, knownList, suggestionThreshold); suggestion != "" {
					errs.AddErrorWithSuggestion("E101", msg, fmt.Sprintf("Did you mean %q?", suggestion))
				} else {
					errs.AddError("E101", msg)
				}
			}
		}
	}
}

// ── Through-table validation ──

func checkThroughTables(errs *cerr.CompilerErrors, models []*ir.DataModel, known map[string]bool) {
	// Build a map of model name → model for lookup
	modelMap := make(map[string]*ir.DataModel)
	for _, m := range models {
		modelMap[strings.ToLower(m.Name)] = m
	}

	for _, model := range models {
		for _, rel := range model.Relations {
			if rel.Kind != "has_many_through" || rel.Through == "" {
				continue
			}

			throughModel := modelMap[strings.ToLower(rel.Through)]
			if throughModel == nil {
				continue // already reported by checkRelationTargets
			}

			// The through-model must have belongs_to relations to both
			// the source model and the target model.
			hasSrc := false
			hasTgt := false
			for _, tRel := range throughModel.Relations {
				if tRel.Kind == "belongs_to" {
					if strings.EqualFold(tRel.Target, model.Name) {
						hasSrc = true
					}
					if strings.EqualFold(tRel.Target, rel.Target) {
						hasTgt = true
					}
				}
			}

			if !hasSrc {
				errs.AddError("E105", fmt.Sprintf(
					"Through-model %q must have a belongs_to relation to %q",
					rel.Through, model.Name))
			}
			if !hasTgt {
				errs.AddError("E105", fmt.Sprintf(
					"Through-model %q must have a belongs_to relation to %q",
					rel.Through, rel.Target))
			}
		}
	}
}

// ── Database index validation ──

func checkIndexes(errs *cerr.CompilerErrors, app *ir.Application) {
	if app.Database == nil {
		return
	}

	// Build model lookup
	modelMap := make(map[string]*ir.DataModel)
	var modelNames []string
	for _, m := range app.Data {
		modelMap[strings.ToLower(m.Name)] = m
		modelNames = append(modelNames, m.Name)
	}

	for _, idx := range app.Database.Indexes {
		model := modelMap[strings.ToLower(idx.Entity)]
		if model == nil {
			msg := fmt.Sprintf("Index references model %q which does not exist", idx.Entity)
			if suggestion := cerr.FindClosest(idx.Entity, modelNames, suggestionThreshold); suggestion != "" {
				errs.AddErrorWithSuggestion("E102", msg, fmt.Sprintf("Did you mean %q?", suggestion))
			} else {
				errs.AddError("E102", msg)
			}
			continue
		}

		// Validate each field in the index
		for _, field := range idx.Fields {
			if !resolveIndexField(field, model) {
				// Collect valid field names for suggestions
				var validFields []string
				for _, f := range model.Fields {
					validFields = append(validFields, f.Name)
				}
				for _, r := range model.Relations {
					if r.Kind == "belongs_to" {
						validFields = append(validFields, r.Target)
					}
				}

				msg := fmt.Sprintf("Index on %q references field %q which does not exist on that model", idx.Entity, field)
				if suggestion := cerr.FindClosest(field, validFields, suggestionThreshold); suggestion != "" {
					errs.AddErrorWithSuggestion("E102", msg, fmt.Sprintf("Did you mean %q?", suggestion))
				} else {
					errs.AddError("E102", msg)
				}
			}
		}
	}
}

// resolveIndexField checks whether a raw field name is valid for the given model.
// This mirrors the resolution logic in the postgres codegen: it checks belongs_to
// targets, exact field matches, and prefix field matches.
func resolveIndexField(rawField string, model *ir.DataModel) bool {
	lower := strings.ToLower(rawField)

	for _, rel := range model.Relations {
		if rel.Kind == "belongs_to" && strings.EqualFold(rel.Target, rawField) {
			return true
		}
	}

	for _, field := range model.Fields {
		fieldLower := strings.ToLower(field.Name)
		if fieldLower == lower {
			return true
		}
		if strings.HasPrefix(lower, fieldLower+" ") {
			return true
		}
	}

	return false
}

// ── Page navigation validation ──

var navigatePattern = regexp.MustCompile(`(?i)navigates?\s+to\s+(\w+)`)

func checkPageNavigation(errs *cerr.CompilerErrors, pages []*ir.Page, known map[string]bool, knownList []string) {
	for _, page := range pages {
		for _, action := range page.Content {
			matches := navigatePattern.FindAllStringSubmatch(action.Text, -1)
			for _, m := range matches {
				target := m[1]
				if !known[strings.ToLower(target)] {
					msg := fmt.Sprintf("Page %q navigates to %q which does not exist", page.Name, target)
					if suggestion := cerr.FindClosest(target, knownList, suggestionThreshold); suggestion != "" {
						errs.AddErrorWithSuggestion("E103", msg, fmt.Sprintf("Did you mean %q?", suggestion))
					} else {
						errs.AddError("E103", msg)
					}
				}
			}
		}
	}
}

// ── CRUD model reference helpers ──

var crudPattern = regexp.MustCompile(`(?i)\b(create|fetch|update|delete)\s+(?:a\s+|the\s+)?(\w+)\b`)

// isSkipWord returns true for common non-model words that follow CRUD verbs.
func isSkipWord(word string) bool {
	lower := strings.ToLower(word)
	return lower == "with" || lower == "the" || lower == "a" || lower == "an" ||
		lower == "success" || lower == "error" || lower == "token" ||
		lower == "it" || lower == "new" || lower == "existing" ||
		lower == "all" || lower == "current" || lower == "each" ||
		lower == "every" || lower == "this" || lower == "that" ||
		lower == "entry" || lower == "record" || lower == "item"
}

// checkCRUDRefs scans actions for CRUD-verb model references and emits
// diagnostics for unknown models. When asError is true it emits errors;
// otherwise it emits warnings.
func checkCRUDRefs(errs *cerr.CompilerErrors, label string, actions []*ir.Action, models map[string]bool, modelList []string, code string, asError bool) {
	for _, action := range actions {
		matches := crudPattern.FindAllStringSubmatch(action.Text, -1)
		for _, m := range matches {
			target := m[2]
			if isSkipWord(target) {
				continue
			}
			if !models[strings.ToLower(target)] {
				msg := fmt.Sprintf("%s references model %q which does not exist", label, target)
				suggestion := cerr.FindClosest(target, modelList, suggestionThreshold)
				hint := ""
				if suggestion != "" {
					hint = fmt.Sprintf("Did you mean %q?", suggestion)
				}
				if asError {
					if hint != "" {
						errs.AddErrorWithSuggestion(code, msg, hint)
					} else {
						errs.AddError(code, msg)
					}
				} else {
					if hint != "" {
						errs.AddWarningWithSuggestion(code, msg, hint)
					} else {
						errs.AddWarning(code, msg)
					}
				}
			}
		}
	}
}

// ── API model reference validation ──

func checkAPIModelReferences(errs *cerr.CompilerErrors, apis []*ir.Endpoint, models map[string]bool, modelList []string) {
	for _, api := range apis {
		checkCRUDRefs(errs, fmt.Sprintf("API %q", api.Name), api.Steps, models, modelList, "E104", true)
	}
}

// ── Completeness checks ──

func checkCompleteness(errs *cerr.CompilerErrors, app *ir.Application, apis map[string]bool, apiList []string) {
	_ = apis
	_ = apiList

	// E201: If any API requires auth, app.Auth must be configured
	for _, api := range app.APIs {
		if api.Auth && app.Auth == nil {
			errs.AddError("E201", fmt.Sprintf(
				"API %q requires authentication but no authentication block is defined", api.Name))
			break // one error is enough
		}
	}

	// W201: If app has pages/data/APIs but no build with: block, warn
	if app.Config == nil || (app.Config.Frontend == "" && app.Config.Backend == "" && app.Config.Database == "") {
		hasContent := len(app.Pages) > 0 || len(app.Data) > 0 || len(app.APIs) > 0
		if hasContent {
			errs.AddWarning("W201", "No build targets specified — add a 'build with:' block to generate frontend, backend, and database code. Without it, only CI/CD and scaffold files are produced.")
		}
	}

	// E202: If database is configured, at least one data model must exist
	if app.Config != nil && app.Config.Database != "" && len(app.Data) == 0 {
		errs.AddError("E202", "Build config specifies a database but no data models are defined")
	}

	// E203: If frontend is configured, at least one page must exist
	if app.Config != nil && app.Config.Frontend != "" && len(app.Pages) == 0 {
		errs.AddError("E203", "Build config specifies a frontend but no pages are defined")
	}
}

// ── Design system validation ──

func checkDesignSystem(errs *cerr.CompilerErrors, app *ir.Application) {
	if app.Theme == nil || app.Theme.DesignSystem == "" {
		return
	}

	ds := themes.Registry(app.Theme.DesignSystem)
	if ds == nil {
		allIDs := themes.AllIDs()
		msg := fmt.Sprintf("Unknown design system %q", app.Theme.DesignSystem)
		if suggestion := cerr.FindClosest(app.Theme.DesignSystem, allIDs, 0.4); suggestion != "" {
			errs.AddWarningWithSuggestion("W301", msg,
				fmt.Sprintf("Did you mean %q? Supported: %s", suggestion, strings.Join(allIDs, ", ")))
		} else {
			errs.AddWarning("W301",
				fmt.Sprintf("%s. Supported: %s", msg, strings.Join(allIDs, ", ")))
		}
		return
	}

	// Validate spacing value
	if app.Theme.Spacing != "" {
		validSpacing := []string{"compact", "comfortable", "spacious"}
		found := false
		for _, v := range validSpacing {
			if app.Theme.Spacing == v {
				found = true
				break
			}
		}
		if !found {
			errs.AddWarning("W303", fmt.Sprintf(
				"Unknown spacing %q — expected one of: compact, comfortable, spacious. Defaulting to comfortable",
				app.Theme.Spacing))
		}
	}

	// Validate border radius value
	if app.Theme.BorderRadius != "" {
		validRadius := []string{"sharp", "smooth", "rounded", "pill"}
		found := false
		for _, v := range validRadius {
			if app.Theme.BorderRadius == v {
				found = true
				break
			}
		}
		if !found {
			errs.AddWarning("W304", fmt.Sprintf(
				"Unknown border radius %q — expected one of: sharp, smooth, rounded, pill. Defaulting to smooth",
				app.Theme.BorderRadius))
		}
	}

	// Check framework compatibility
	if app.Config != nil && app.Config.Frontend != "" {
		framework := inferFramework(app.Config.Frontend)
		if framework != "" && !themes.HasFrameworkSupport(app.Theme.DesignSystem, framework) {
			errs.AddWarning("W302", fmt.Sprintf(
				"Design system %q has no %s library — will use Tailwind CSS with %s color palette as fallback",
				ds.Name, framework, ds.Name))
		}
	}
}

// inferFramework extracts the framework name from a build config string.
func inferFramework(frontend string) string {
	lower := strings.ToLower(frontend)
	for _, fw := range []string{"react", "vue", "angular", "svelte"} {
		if strings.Contains(lower, fw) {
			return fw
		}
	}
	return ""
}

// ── Architecture validation ──

func checkArchitecture(errs *cerr.CompilerErrors, app *ir.Application, models map[string]bool, modelList []string) {
	if app.Architecture == nil {
		return
	}

	// W401: Validate architecture style
	validStyles := []string{"monolith", "microservices", "serverless"}
	style := strings.ToLower(app.Architecture.Style)
	styleValid := false
	for _, v := range validStyles {
		if style == v {
			styleValid = true
			break
		}
	}
	if !styleValid && style != "" {
		msg := fmt.Sprintf("Unknown architecture style %q", app.Architecture.Style)
		if suggestion := cerr.FindClosest(app.Architecture.Style, validStyles, 0.4); suggestion != "" {
			errs.AddWarningWithSuggestion("W401", msg,
				fmt.Sprintf("Did you mean %q? Supported: %s", suggestion, strings.Join(validStyles, ", ")))
		} else {
			errs.AddWarning("W401",
				fmt.Sprintf("%s. Supported: %s", msg, strings.Join(validStyles, ", ")))
		}
	}

	// E401: Microservices must define at least one service
	if strings.Contains(style, "microservice") && len(app.Architecture.Services) == 0 {
		errs.AddError("E401", "Microservices architecture declared but no services are defined")
	}

	// W402: Service model references
	for _, svc := range app.Architecture.Services {
		for _, modelName := range svc.Models {
			if !models[strings.ToLower(modelName)] {
				msg := fmt.Sprintf("Service %q references model %q which does not exist", svc.Name, modelName)
				if suggestion := cerr.FindClosest(modelName, modelList, suggestionThreshold); suggestion != "" {
					errs.AddWarningWithSuggestion("W402", msg, fmt.Sprintf("Did you mean %q?", suggestion))
				} else {
					errs.AddWarning("W402", msg)
				}
			}
		}
	}

	// W403: Service talks_to references
	serviceNames := make(map[string]bool)
	var serviceNameList []string
	for _, svc := range app.Architecture.Services {
		serviceNames[strings.ToLower(svc.Name)] = true
		serviceNameList = append(serviceNameList, svc.Name)
	}
	for _, svc := range app.Architecture.Services {
		for _, target := range svc.TalksTo {
			if !serviceNames[strings.ToLower(target)] {
				msg := fmt.Sprintf("Service %q talks to %q which is not defined", svc.Name, target)
				if suggestion := cerr.FindClosest(target, serviceNameList, suggestionThreshold); suggestion != "" {
					errs.AddWarningWithSuggestion("W403", msg, fmt.Sprintf("Did you mean %q?", suggestion))
				} else {
					errs.AddWarning("W403", msg)
				}
			}
		}
	}

	// E402: Serverless without APIs
	if strings.Contains(style, "serverless") && len(app.APIs) == 0 {
		errs.AddError("E402", "Serverless architecture declared but no APIs are defined — each API becomes a Lambda function")
	}
}

// ── Integration validation ──

var (
	sendEmailPattern = regexp.MustCompile(`(?i)\bsend\s+(email|notification|welcome email|reminder email)\b`)
	slackAlertPattern = regexp.MustCompile(`(?i)\b(alert|notify|message)\b.*\bslack\b|\bslack\b.*\b(alert|notify|message)\b`)
)

func checkIntegrations(errs *cerr.CompilerErrors, app *ir.Application) {
	if len(app.Integrations) == 0 {
		return
	}

	// E501: Duplicate service
	seen := make(map[string]bool)
	for _, integ := range app.Integrations {
		lower := strings.ToLower(integ.Service)
		if seen[lower] {
			errs.AddError("E501", fmt.Sprintf("Duplicate integration: %q is declared more than once", integ.Service))
		}
		seen[lower] = true
	}

	// W501: Integration without credentials (except local services like Ollama)
	localServices := map[string]bool{"ollama": true}
	for _, integ := range app.Integrations {
		if len(integ.Credentials) == 0 && !localServices[strings.ToLower(integ.Service)] {
			errs.AddWarning("W501", fmt.Sprintf(
				"Integration %q has no credentials configured — it will need API keys at runtime",
				integ.Service))
		}
	}
}

func checkWorkflowIntegrationRefs(errs *cerr.CompilerErrors, app *ir.Application) {
	if len(app.Workflows) == 0 && len(app.ErrorHandlers) == 0 {
		return
	}

	// Build set of integration types present.
	hasEmail := false
	hasMessaging := false
	for _, integ := range app.Integrations {
		switch integ.Type {
		case "email":
			hasEmail = true
		case "messaging":
			hasMessaging = true
		}
	}

	// Collect all action texts from workflows and error handlers.
	type actionSource struct {
		label string
		text  string
	}
	var actions []actionSource

	for _, wf := range app.Workflows {
		for _, step := range wf.Steps {
			actions = append(actions, actionSource{label: fmt.Sprintf("Workflow %q", wf.Trigger), text: step.Text})
		}
	}
	for _, eh := range app.ErrorHandlers {
		for _, step := range eh.Steps {
			actions = append(actions, actionSource{label: fmt.Sprintf("Error handler %q", eh.Condition), text: step.Text})
		}
	}

	for _, a := range actions {
		// W502: sends email but no email integration
		if !hasEmail && sendEmailPattern.MatchString(a.text) {
			errs.AddWarning("W502", fmt.Sprintf(
				"%s sends email but no email integration is declared — add an 'integrate with SendGrid' (or similar) block",
				a.label))
		}

		// W503: references Slack but no messaging integration
		if !hasMessaging && slackAlertPattern.MatchString(a.text) {
			errs.AddWarning("W503", fmt.Sprintf(
				"%s references Slack but no messaging integration is declared — add an 'integrate with Slack' (or similar) block",
				a.label))
		}
	}
}

// ── Validation field references (W107) ──

func checkValidationFields(errs *cerr.CompilerErrors, apis []*ir.Endpoint) {
	for _, api := range apis {
		if len(api.Validation) == 0 || len(api.Params) == 0 {
			continue
		}
		paramNames := make(map[string]bool)
		var paramList []string
		for _, p := range api.Params {
			paramNames[strings.ToLower(p.Name)] = true
			paramList = append(paramList, p.Name)
		}
		for _, v := range api.Validation {
			if !paramNames[strings.ToLower(v.Field)] {
				msg := fmt.Sprintf("API %q validation references field %q which is not a declared parameter", api.Name, v.Field)
				if suggestion := cerr.FindClosest(v.Field, paramList, suggestionThreshold); suggestion != "" {
					errs.AddWarningWithSuggestion("W107", msg, fmt.Sprintf("Did you mean %q?", suggestion))
				} else {
					errs.AddWarning("W107", msg)
				}
			}
		}
	}
}

// ── Database engine validation (W305) ──

var knownEngines = []string{"PostgreSQL", "MySQL", "MariaDB", "SQLite", "MongoDB", "Redis", "DynamoDB", "CockroachDB"}

func checkDatabaseEngine(errs *cerr.CompilerErrors, app *ir.Application) {
	if app.Database == nil || app.Database.Engine == "" {
		return
	}
	for _, engine := range knownEngines {
		if strings.EqualFold(app.Database.Engine, engine) {
			return
		}
	}
	msg := fmt.Sprintf("Unknown database engine %q", app.Database.Engine)
	if suggestion := cerr.FindClosest(app.Database.Engine, knownEngines, 0.4); suggestion != "" {
		errs.AddWarningWithSuggestion("W305", msg, fmt.Sprintf("Did you mean %q?", suggestion))
	} else {
		errs.AddWarning("W305", fmt.Sprintf("%s. Supported: %s", msg, strings.Join(knownEngines, ", ")))
	}
}

// ── Gateway route references (W404) ──

func checkGatewayRoutes(errs *cerr.CompilerErrors, app *ir.Application) {
	if app.Architecture == nil || app.Architecture.Gateway == nil {
		return
	}
	serviceNames := make(map[string]bool)
	var serviceNameList []string
	for _, svc := range app.Architecture.Services {
		serviceNames[strings.ToLower(svc.Name)] = true
		serviceNameList = append(serviceNameList, svc.Name)
	}
	for path, svcName := range app.Architecture.Gateway.Routes {
		if !serviceNames[strings.ToLower(svcName)] {
			msg := fmt.Sprintf("Gateway route %q targets service %q which is not defined", path, svcName)
			if suggestion := cerr.FindClosest(svcName, serviceNameList, suggestionThreshold); suggestion != "" {
				errs.AddWarningWithSuggestion("W404", msg, fmt.Sprintf("Did you mean %q?", suggestion))
			} else {
				errs.AddWarning("W404", msg)
			}
		}
	}
}

// ── Monitoring channel references (W504) ──

func checkMonitoringChannels(errs *cerr.CompilerErrors, app *ir.Application) {
	if len(app.Monitoring) == 0 {
		return
	}
	integrationLookup := make(map[string]bool)
	for _, integ := range app.Integrations {
		integrationLookup[strings.ToLower(integ.Service)] = true
		if integ.Type != "" {
			integrationLookup[strings.ToLower(integ.Type)] = true
		}
	}
	for _, rule := range app.Monitoring {
		if rule.Kind == "alert" && rule.Channel != "" {
			if !integrationLookup[strings.ToLower(rule.Channel)] {
				errs.AddWarning("W504", fmt.Sprintf(
					"Monitoring alert channel %q has no matching integration declared", rule.Channel))
			}
		}
	}
}

// ── Policy model references (W109) ──

func checkPolicyModelRefs(errs *cerr.CompilerErrors, app *ir.Application, models map[string]bool, modelList []string) {
	for _, policy := range app.Policies {
		for _, rules := range [][]*ir.PolicyRule{policy.Permissions, policy.Restrictions} {
			for _, rule := range rules {
				matches := crudPattern.FindAllStringSubmatch(rule.Text, -1)
				for _, m := range matches {
					target := m[2]
					if isSkipWord(target) {
						continue
					}
					if !models[strings.ToLower(target)] {
						msg := fmt.Sprintf("Policy %q references model %q which does not exist", policy.Name, target)
						if suggestion := cerr.FindClosest(target, modelList, suggestionThreshold); suggestion != "" {
							errs.AddWarningWithSuggestion("W109", msg, fmt.Sprintf("Did you mean %q?", suggestion))
						} else {
							errs.AddWarning("W109", msg)
						}
					}
				}
			}
		}
	}
}

// ── Workflow/ErrorHandler/Pipeline CRUD model references ──

func checkActionModelRefs(errs *cerr.CompilerErrors, app *ir.Application, models map[string]bool, modelList []string) {
	for _, wf := range app.Workflows {
		checkCRUDRefs(errs, fmt.Sprintf("Workflow %q", wf.Trigger), wf.Steps, models, modelList, "W109", false)
	}
	for _, eh := range app.ErrorHandlers {
		checkCRUDRefs(errs, fmt.Sprintf("Error handler %q", eh.Condition), eh.Steps, models, modelList, "W109", false)
	}
	for _, pl := range app.Pipelines {
		checkCRUDRefs(errs, fmt.Sprintf("Pipeline %q", pl.Trigger), pl.Steps, models, modelList, "W109", false)
	}
}

// ── Trigger model references (W106) ──

var triggerModelPattern = regexp.MustCompile(`(?i)\b(\w+)\s+(?:is\s+)?(?:created|updated|deleted|completed|overdue|signs?\s+up)\b`)

func checkTriggerModelRefs(errs *cerr.CompilerErrors, app *ir.Application, models map[string]bool, modelList []string) {
	type triggerSource struct {
		label   string
		trigger string
	}
	var triggers []triggerSource
	for _, wf := range app.Workflows {
		triggers = append(triggers, triggerSource{label: "Workflow", trigger: wf.Trigger})
	}
	for _, pl := range app.Pipelines {
		triggers = append(triggers, triggerSource{label: "Pipeline", trigger: pl.Trigger})
	}

	// Words that appear before trigger verbs but are not model names
	triggerSkip := map[string]bool{
		"becomes": true, "gets": true, "was": true, "been": true,
		"when": true, "if": true, "once": true, "after": true,
	}

	for _, ts := range triggers {
		matches := triggerModelPattern.FindAllStringSubmatch(ts.trigger, -1)
		for _, m := range matches {
			target := m[1]
			lower := strings.ToLower(target)
			if isSkipWord(target) || triggerSkip[lower] {
				continue
			}
			if !models[lower] {
				msg := fmt.Sprintf("%s trigger %q references model %q which does not exist", ts.label, ts.trigger, target)
				if suggestion := cerr.FindClosest(target, modelList, suggestionThreshold); suggestion != "" {
					errs.AddWarningWithSuggestion("W106", msg, fmt.Sprintf("Did you mean %q?", suggestion))
				} else {
					errs.AddWarning("W106", msg)
				}
			}
		}
	}
}
