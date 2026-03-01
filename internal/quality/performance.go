package quality

import (
	"fmt"
	"strings"

	"github.com/barun-bash/human/internal/ir"
)

// PerformanceFinding represents a detected performance anti-pattern in the IR.
type PerformanceFinding struct {
	Kind     string // "n-plus-one", "missing-pagination", "missing-index", "large-payload"
	Severity string // "warning", "info"
	Target   string
	Message  string
	Fix      string // suggested fix in Human syntax
}

// checkPerformance scans the IR for performance anti-patterns.
func checkPerformance(app *ir.Application) []PerformanceFinding {
	var findings []PerformanceFinding

	findings = append(findings, checkNPlusOne(app)...)
	findings = append(findings, checkMissingPagination(app)...)
	findings = append(findings, checkMissingIndexes(app)...)
	findings = append(findings, checkLargePayloads(app)...)

	return findings
}

// checkNPlusOne detects pages that loop over a model and reference a related model inside the loop.
func checkNPlusOne(app *ir.Application) []PerformanceFinding {
	var findings []PerformanceFinding

	// Build relation map: model -> list of relation targets
	relations := map[string][]string{}
	for _, model := range app.Data {
		for _, rel := range model.Relations {
			relations[strings.ToLower(model.Name)] = append(relations[strings.ToLower(model.Name)], strings.ToLower(rel.Target))
		}
	}

	for _, page := range app.Pages {
		for i, action := range page.Content {
			if action.Type != "loop" {
				continue
			}
			loopModel := strings.ToLower(action.Target)
			if loopModel == "" {
				continue
			}

			// Look at subsequent actions for relation target references
			targets := relations[loopModel]
			if len(targets) == 0 {
				continue
			}

			for j := i + 1; j < len(page.Content); j++ {
				inner := page.Content[j]
				innerTarget := strings.ToLower(inner.Target)
				innerText := strings.ToLower(inner.Text)

				for _, relTarget := range targets {
					if innerTarget == relTarget || strings.Contains(innerText, relTarget) {
						findings = append(findings, PerformanceFinding{
							Kind:     "n-plus-one",
							Severity: "warning",
							Target:   page.Name,
							Message:  fmt.Sprintf("Page %s loops over %s and accesses related %s inside the loop — potential N+1 query", page.Name, action.Target, relTarget),
							Fix:      fmt.Sprintf("Add 'include %s' to the data fetch for %s", relTarget, action.Target),
						})
						break
					}
				}
			}
		}
	}

	return findings
}

// checkMissingPagination detects APIs that fetch all records without pagination.
func checkMissingPagination(app *ir.Application) []PerformanceFinding {
	var findings []PerformanceFinding

	for _, ep := range app.APIs {
		hasFetchAll := false
		hasPaginate := false

		for _, step := range ep.Steps {
			lower := strings.ToLower(step.Text)
			if step.Type == "query" && (strings.Contains(lower, "fetch all") || strings.Contains(lower, "get all") || strings.Contains(lower, "list all") || strings.Contains(lower, "find all")) {
				hasFetchAll = true
			}
			if strings.Contains(lower, "paginate") || strings.Contains(lower, "pagination") || strings.Contains(lower, "limit") || strings.Contains(lower, "per page") {
				hasPaginate = true
			}
		}

		// Also check params for pagination indicators
		for _, p := range ep.Params {
			lower := strings.ToLower(p.Name)
			if lower == "page" || lower == "limit" || lower == "offset" || lower == "per_page" {
				hasPaginate = true
			}
		}

		if hasFetchAll && !hasPaginate {
			findings = append(findings, PerformanceFinding{
				Kind:     "missing-pagination",
				Severity: "warning",
				Target:   ep.Name,
				Message:  fmt.Sprintf("Endpoint %s fetches all records without pagination — may cause performance issues with large datasets", ep.Name),
				Fix:      fmt.Sprintf("Add 'paginate results, 20 per page' to the %s endpoint", ep.Name),
			})
		}
	}

	return findings
}

// checkMissingIndexes detects pages that filter/sort by fields not covered by database indexes.
func checkMissingIndexes(app *ir.Application) []PerformanceFinding {
	var findings []PerformanceFinding

	// Build index coverage map: "entity:field" -> true
	indexed := map[string]bool{}
	if app.Database != nil {
		for _, idx := range app.Database.Indexes {
			for _, field := range idx.Fields {
				indexed[strings.ToLower(idx.Entity)+":"+strings.ToLower(field)] = true
			}
		}
	}

	// Check page actions for filter/sort references
	for _, page := range app.Pages {
		for _, action := range page.Content {
			lower := strings.ToLower(action.Text)
			if !strings.Contains(lower, "filter") && !strings.Contains(lower, "sort") && !strings.Contains(lower, "search") {
				continue
			}
			target := strings.ToLower(action.Target)
			if target == "" {
				continue
			}

			// Try to find which model and field is being filtered
			for _, model := range app.Data {
				if !strings.Contains(lower, strings.ToLower(model.Name)) {
					continue
				}
				for _, field := range model.Fields {
					if strings.Contains(lower, strings.ToLower(field.Name)) {
						key := strings.ToLower(model.Name) + ":" + strings.ToLower(field.Name)
						if !indexed[key] {
							findings = append(findings, PerformanceFinding{
								Kind:     "missing-index",
								Severity: "info",
								Target:   page.Name,
								Message:  fmt.Sprintf("Page %s filters/sorts %s by %s but no database index exists", page.Name, model.Name, field.Name),
								Fix:      fmt.Sprintf("Add 'index %s by %s' to the database block", model.Name, field.Name),
							})
						}
					}
				}
			}
		}
	}

	return findings
}

// checkLargePayloads detects APIs that respond with models having many fields without field selection.
func checkLargePayloads(app *ir.Application) []PerformanceFinding {
	var findings []PerformanceFinding

	for _, ep := range app.APIs {
		// Check for respond steps
		for _, step := range ep.Steps {
			if step.Type != "respond" {
				continue
			}

			// Find the model being returned
			target := step.Target
			if target == "" {
				// Try to infer from text
				for _, model := range app.Data {
					if strings.Contains(strings.ToLower(step.Text), strings.ToLower(model.Name)) {
						target = model.Name
						break
					}
				}
			}
			if target == "" {
				continue
			}

			// Count fields in the model
			for _, model := range app.Data {
				if !strings.EqualFold(model.Name, target) {
					continue
				}
				if len(model.Fields) > 10 {
					// Check if endpoint mentions field selection
					hasSelection := false
					for _, s := range ep.Steps {
						lower := strings.ToLower(s.Text)
						if strings.Contains(lower, "select") || strings.Contains(lower, "only") || strings.Contains(lower, "fields") || strings.Contains(lower, "exclude") {
							hasSelection = true
							break
						}
					}
					if !hasSelection {
						findings = append(findings, PerformanceFinding{
							Kind:     "large-payload",
							Severity: "info",
							Target:   ep.Name,
							Message:  fmt.Sprintf("Endpoint %s returns model %s with %d fields — consider selecting only needed fields", ep.Name, model.Name, len(model.Fields)),
							Fix:      fmt.Sprintf("Add 'respond with only name, status, created_at from %s' to select specific fields", model.Name),
						})
					}
				}
			}
		}
	}

	return findings
}

// renderPerformanceReport produces a standalone performance-report.md.
func renderPerformanceReport(findings []PerformanceFinding) string {
	var b strings.Builder

	b.WriteString("# Performance Report\n\n")
	b.WriteString("Generated by Human compiler quality engine.\n\n")

	warns := 0
	infos := 0
	for _, f := range findings {
		switch f.Severity {
		case "warning":
			warns++
		case "info":
			infos++
		}
	}

	fmt.Fprintf(&b, "**Summary:** %d warnings, %d info\n\n", warns, infos)

	if len(findings) == 0 {
		b.WriteString("No performance issues found.\n")
		return b.String()
	}

	b.WriteString("## Findings\n\n")
	b.WriteString("| Severity | Kind | Target | Message | Suggested Fix |\n")
	b.WriteString("|----------|------|--------|---------|---------------|\n")
	for _, f := range findings {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n", f.Severity, f.Kind, f.Target, f.Message, f.Fix)
	}
	b.WriteString("\n")

	return b.String()
}

// renderPerformanceSection produces a markdown section for the build report.
func renderPerformanceSection(findings []PerformanceFinding) string {
	var b strings.Builder

	b.WriteString("## Performance\n\n")

	warns := 0
	for _, f := range findings {
		if f.Severity == "warning" {
			warns++
		}
	}

	fmt.Fprintf(&b, "**Summary:** %d findings (%d warnings)\n\n", len(findings), warns)

	if len(findings) == 0 {
		b.WriteString("No performance issues found.\n\n")
		return b.String()
	}

	b.WriteString("| Kind | Target | Message |\n")
	b.WriteString("|------|--------|---------|\n")
	for _, f := range findings {
		fmt.Fprintf(&b, "| %s | %s | %s |\n", f.Kind, f.Target, f.Message)
	}
	b.WriteString("\n")

	return b.String()
}
