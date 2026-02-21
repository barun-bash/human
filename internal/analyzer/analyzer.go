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

	// Silence unused-variable warnings — lists are needed for future suggestions
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

// ── API model reference validation ──

var crudPattern = regexp.MustCompile(`(?i)\b(create|fetch|update|delete)\s+(?:a\s+|the\s+)?(\w+)\b`)

func checkAPIModelReferences(errs *cerr.CompilerErrors, apis []*ir.Endpoint, models map[string]bool, modelList []string) {
	for _, api := range apis {
		for _, step := range api.Steps {
			matches := crudPattern.FindAllStringSubmatch(step.Text, -1)
			for _, m := range matches {
				verb := strings.ToLower(m[1])
				target := m[2]

				// Skip common non-model words that follow CRUD verbs
				lower := strings.ToLower(target)
				if lower == "with" || lower == "the" || lower == "a" || lower == "an" ||
					lower == "success" || lower == "error" || lower == "token" ||
					lower == "it" || lower == "new" || lower == "existing" ||
					lower == "all" || lower == "current" || lower == "each" ||
					lower == "every" || lower == "this" || lower == "that" ||
					lower == "entry" || lower == "record" || lower == "item" {
					continue
				}

				// Only flag if it looks like it should be a model reference
				// (CRUD verbs typically act on data models)
				_ = verb
				if !models[strings.ToLower(target)] {
					msg := fmt.Sprintf("API %q references model %q which does not exist", api.Name, target)
					if suggestion := cerr.FindClosest(target, modelList, suggestionThreshold); suggestion != "" {
						errs.AddErrorWithSuggestion("E104", msg, fmt.Sprintf("Did you mean %q?", suggestion))
					} else {
						errs.AddError("E104", msg)
					}
				}
			}
		}
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
	validStyles := []string{"monolith", "microservices", "serverless", "event-driven"}
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
