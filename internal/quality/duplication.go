package quality

import (
	"fmt"
	"sort"
	"strings"

	"github.com/barun-bash/human/internal/ir"
)

// DuplicationFinding represents a detected duplication pattern in the IR.
type DuplicationFinding struct {
	Kind    string   // "api-duplicate", "similar-pages", "repeated-validation"
	Items   []string
	Message string
}

// checkDuplication scans the IR for duplicated patterns.
func checkDuplication(app *ir.Application) []DuplicationFinding {
	var findings []DuplicationFinding

	findings = append(findings, checkDuplicateAPILogic(app)...)
	findings = append(findings, checkSimilarPages(app)...)
	findings = append(findings, checkRepeatedValidation(app)...)

	return findings
}

// checkDuplicateAPILogic compares endpoint pairs for duplicated logic:
// same params (by name set), same model in steps, same validation rules.
func checkDuplicateAPILogic(app *ir.Application) []DuplicationFinding {
	var findings []DuplicationFinding

	for i := 0; i < len(app.APIs); i++ {
		for j := i + 1; j < len(app.APIs); j++ {
			a, b := app.APIs[i], app.APIs[j]

			// Need at least 1 param each to compare
			if len(a.Params) == 0 || len(b.Params) == 0 {
				continue
			}

			// Check same params (by name set)
			if !sameParamSet(a.Params, b.Params) {
				continue
			}

			// Check same model reference in steps
			aModel := stepModel(a.Steps)
			bModel := stepModel(b.Steps)
			if aModel == "" || bModel == "" || aModel != bModel {
				continue
			}

			// Check same validation rules
			if !sameValidationRules(a.Validation, b.Validation) {
				continue
			}

			findings = append(findings, DuplicationFinding{
				Kind:    "api-duplicate",
				Items:   []string{a.Name, b.Name},
				Message: fmt.Sprintf("Endpoints %s and %s have identical params, target model '%s', and validation rules — consider extracting shared logic", a.Name, b.Name, aModel),
			})
		}
	}

	return findings
}

// checkSimilarPages compares page pairs for duplicated structure:
// same action type sequence and same model reference.
func checkSimilarPages(app *ir.Application) []DuplicationFinding {
	var findings []DuplicationFinding

	for i := 0; i < len(app.Pages); i++ {
		for j := i + 1; j < len(app.Pages); j++ {
			a, b := app.Pages[i], app.Pages[j]

			if len(a.Content) == 0 || len(b.Content) == 0 {
				continue
			}

			// Compare action type sequences
			if !sameActionTypes(a.Content, b.Content) {
				continue
			}

			// Check that both reference the same model
			aModel := actionModel(a.Content)
			bModel := actionModel(b.Content)
			if aModel == "" || bModel == "" || aModel != bModel {
				continue
			}

			findings = append(findings, DuplicationFinding{
				Kind:    "similar-pages",
				Items:   []string{a.Name, b.Name},
				Message: fmt.Sprintf("Pages %s and %s have identical structure targeting model '%s' — consider extracting a shared component", a.Name, b.Name, aModel),
			})
		}
	}

	return findings
}

// checkRepeatedValidation collects all field+rule combos across endpoints
// and flags any that appear 3+ times.
func checkRepeatedValidation(app *ir.Application) []DuplicationFinding {
	var findings []DuplicationFinding

	type combo struct {
		Field string
		Rule  string
	}
	counts := map[combo][]string{}

	for _, ep := range app.APIs {
		for _, v := range ep.Validation {
			key := combo{Field: strings.ToLower(v.Field), Rule: v.Rule}
			counts[key] = append(counts[key], ep.Name)
		}
	}

	for key, endpoints := range counts {
		if len(endpoints) >= 3 {
			findings = append(findings, DuplicationFinding{
				Kind:    "repeated-validation",
				Items:   endpoints,
				Message: fmt.Sprintf("Validation '%s %s' is repeated across %d endpoints (%s) — consider a shared validation rule", key.Field, key.Rule, len(endpoints), strings.Join(endpoints, ", ")),
			})
		}
	}

	return findings
}

// sameParamSet checks if two param lists have the same set of names.
func sameParamSet(a, b []*ir.Param) bool {
	if len(a) != len(b) {
		return false
	}
	setA := make(map[string]bool, len(a))
	for _, p := range a {
		setA[strings.ToLower(p.Name)] = true
	}
	for _, p := range b {
		if !setA[strings.ToLower(p.Name)] {
			return false
		}
	}
	return true
}

// stepModel extracts the most common Target from action steps.
func stepModel(steps []*ir.Action) string {
	counts := map[string]int{}
	for _, s := range steps {
		if s.Target != "" {
			counts[strings.ToLower(s.Target)]++
		}
	}
	best := ""
	bestN := 0
	for k, n := range counts {
		if n > bestN {
			best = k
			bestN = n
		}
	}
	return best
}

// sameValidationRules checks if two sets of validation rules are equivalent.
func sameValidationRules(a, b []*ir.ValidationRule) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 {
		return true
	}

	type vr struct{ field, rule string }
	setA := make([]vr, len(a))
	setB := make([]vr, len(b))

	for i, v := range a {
		setA[i] = vr{strings.ToLower(v.Field), v.Rule}
	}
	for i, v := range b {
		setB[i] = vr{strings.ToLower(v.Field), v.Rule}
	}

	sort.Slice(setA, func(i, j int) bool {
		if setA[i].field == setA[j].field {
			return setA[i].rule < setA[j].rule
		}
		return setA[i].field < setA[j].field
	})
	sort.Slice(setB, func(i, j int) bool {
		if setB[i].field == setB[j].field {
			return setB[i].rule < setB[j].rule
		}
		return setB[i].field < setB[j].field
	})

	for i := range setA {
		if setA[i] != setB[i] {
			return false
		}
	}
	return true
}

// sameActionTypes checks if two action slices have the same type sequence.
func sameActionTypes(a, b []*ir.Action) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Type != b[i].Type {
			return false
		}
	}
	return true
}

// actionModel extracts the most common Target from actions.
func actionModel(actions []*ir.Action) string {
	counts := map[string]int{}
	for _, a := range actions {
		if a.Target != "" {
			counts[strings.ToLower(a.Target)]++
		}
	}
	best := ""
	bestN := 0
	for k, n := range counts {
		if n > bestN {
			best = k
			bestN = n
		}
	}
	return best
}

// renderDuplicationSection produces a markdown section for the build report.
func renderDuplicationSection(findings []DuplicationFinding) string {
	var b strings.Builder

	b.WriteString("## Duplication\n\n")
	fmt.Fprintf(&b, "**Summary:** %d duplication findings\n\n", len(findings))

	if len(findings) == 0 {
		b.WriteString("No duplication issues found.\n\n")
		return b.String()
	}

	b.WriteString("| Kind | Items | Message |\n")
	b.WriteString("|------|-------|---------|\n")
	for _, f := range findings {
		fmt.Fprintf(&b, "| %s | %s | %s |\n", f.Kind, strings.Join(f.Items, ", "), f.Message)
	}
	b.WriteString("\n")

	return b.String()
}
