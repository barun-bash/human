package quality

import (
	"fmt"
	"strings"

	"github.com/barun-bash/human/internal/ir"
)

// CoverageReport holds test coverage statistics.
type CoverageReport struct {
	EndpointsTested int
	EndpointsTotal  int
	PagesTested     int
	PagesTotal      int
	FieldsTested    int
	FieldsTotal     int
	Overall         float64 // percentage
}

// calculateCoverage counts testable items vs tested items and computes coverage.
func calculateCoverage(app *ir.Application, result *Result) *CoverageReport {
	cov := &CoverageReport{}

	// Endpoints: every endpoint gets at least one API test
	cov.EndpointsTotal = len(app.APIs)
	cov.EndpointsTested = result.TestFiles // each API test file covers one endpoint

	// Pages: every page gets a component test
	cov.PagesTotal = len(app.Pages)
	cov.PagesTested = result.ComponentTestFiles

	// Fields: only fields in models that have Create endpoints are edge-tested
	cov.FieldsTotal = countAllFields(app)
	cov.FieldsTested = countTestedFields(app)

	total := cov.EndpointsTotal + cov.PagesTotal + cov.FieldsTotal
	tested := cov.EndpointsTested + cov.PagesTested + cov.FieldsTested

	if total > 0 {
		cov.Overall = float64(tested) / float64(total) * 100
	}

	return cov
}

// renderCoverageSection produces a markdown section for the build report.
func renderCoverageSection(cov *CoverageReport) string {
	var b strings.Builder

	b.WriteString("## Test Coverage\n\n")
	b.WriteString("| Category | Tested | Total | Coverage |\n")
	b.WriteString("|----------|--------|-------|----------|\n")

	epPct := pct(cov.EndpointsTested, cov.EndpointsTotal)
	pgPct := pct(cov.PagesTested, cov.PagesTotal)
	fdPct := pct(cov.FieldsTested, cov.FieldsTotal)

	fmt.Fprintf(&b, "| Endpoints | %d | %d | %.0f%% |\n", cov.EndpointsTested, cov.EndpointsTotal, epPct)
	fmt.Fprintf(&b, "| Pages | %d | %d | %.0f%% |\n", cov.PagesTested, cov.PagesTotal, pgPct)
	fmt.Fprintf(&b, "| Fields | %d | %d | %.0f%% |\n", cov.FieldsTested, cov.FieldsTotal, fdPct)
	fmt.Fprintf(&b, "| **Overall** | **%d** | **%d** | **%.0f%%** |\n",
		cov.EndpointsTested+cov.PagesTested+cov.FieldsTested,
		cov.EndpointsTotal+cov.PagesTotal+cov.FieldsTotal,
		cov.Overall)
	b.WriteString("\n")

	if cov.Overall < 90 {
		b.WriteString("**Warning:** Test coverage is below 90%. Consider adding tests for untested items.\n\n")
	}

	return b.String()
}

// countAllFields counts all fields across all data models.
func countAllFields(app *ir.Application) int {
	total := 0
	for _, model := range app.Data {
		total += len(model.Fields)
	}
	return total
}

// countTestedFields counts fields in models that have a Create endpoint.
func countTestedFields(app *ir.Application) int {
	total := 0
	for _, model := range app.Data {
		if hasCreateEndpoint(app, model.Name) {
			total += len(model.Fields)
		}
	}
	return total
}

// hasCreateEndpoint checks if a Create{ModelName} endpoint exists.
func hasCreateEndpoint(app *ir.Application, modelName string) bool {
	target := "Create" + modelName
	for _, ep := range app.APIs {
		if strings.EqualFold(ep.Name, target) {
			return true
		}
	}
	return false
}

// pct computes percentage, returning 0 if total is 0.
func pct(tested, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(tested) / float64(total) * 100
}
