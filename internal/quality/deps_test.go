package quality

import (
	"strings"
	"testing"
)

func TestParseNpmAudit_WithVulnerabilities(t *testing.T) {
	mockJSON := `{
		"metadata": {
			"vulnerabilities": {
				"critical": 1,
				"high": 2,
				"moderate": 3,
				"low": 1,
				"total": 7
			}
		},
		"vulnerabilities": {
			"lodash": {
				"name": "lodash",
				"severity": "critical",
				"via": [{"title": "Prototype Pollution", "url": "https://example.com/1", "name": "lodash"}]
			},
			"axios": {
				"name": "axios",
				"severity": "high",
				"via": [{"title": "Server-Side Request Forgery", "url": "https://example.com/2", "name": "axios"}]
			}
		}
	}`

	report, err := parseNpmAudit([]byte(mockJSON))
	if err != nil {
		t.Fatalf("parseNpmAudit: %v", err)
	}

	if report.Total != 7 {
		t.Errorf("expected total 7, got %d", report.Total)
	}
	if report.Critical != 1 {
		t.Errorf("expected 1 critical, got %d", report.Critical)
	}
	if report.High != 2 {
		t.Errorf("expected 2 high, got %d", report.High)
	}
	if report.Moderate != 3 {
		t.Errorf("expected 3 moderate, got %d", report.Moderate)
	}
	if report.Low != 1 {
		t.Errorf("expected 1 low, got %d", report.Low)
	}

	if len(report.Advisories) != 2 {
		t.Fatalf("expected 2 advisories, got %d", len(report.Advisories))
	}

	// Check advisory details
	found := map[string]bool{}
	for _, a := range report.Advisories {
		found[a.Module] = true
		if a.Module == "lodash" {
			if a.Severity != "critical" {
				t.Errorf("expected critical severity for lodash, got %s", a.Severity)
			}
			if a.Title != "Prototype Pollution" {
				t.Errorf("expected Prototype Pollution title, got %s", a.Title)
			}
		}
	}
	if !found["lodash"] || !found["axios"] {
		t.Error("missing expected advisory modules")
	}
}

func TestParseNpmAudit_ZeroVulnerabilities(t *testing.T) {
	mockJSON := `{
		"metadata": {
			"vulnerabilities": {
				"critical": 0,
				"high": 0,
				"moderate": 0,
				"low": 0,
				"total": 0
			}
		},
		"vulnerabilities": {}
	}`

	report, err := parseNpmAudit([]byte(mockJSON))
	if err != nil {
		t.Fatalf("parseNpmAudit: %v", err)
	}

	if report.Total != 0 {
		t.Errorf("expected total 0, got %d", report.Total)
	}
	if len(report.Advisories) != 0 {
		t.Errorf("expected 0 advisories, got %d", len(report.Advisories))
	}
}

func TestParseNpmAudit_InvalidJSON(t *testing.T) {
	_, err := parseNpmAudit([]byte("not json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseNpmAudit_ViaWithStringOnly(t *testing.T) {
	// Some npm audit entries have plain strings in "via" instead of objects
	mockJSON := `{
		"metadata": {
			"vulnerabilities": {
				"critical": 0,
				"high": 1,
				"moderate": 0,
				"low": 0,
				"total": 1
			}
		},
		"vulnerabilities": {
			"express": {
				"name": "express",
				"severity": "high",
				"via": ["qs"]
			}
		}
	}`

	report, err := parseNpmAudit([]byte(mockJSON))
	if err != nil {
		t.Fatalf("parseNpmAudit: %v", err)
	}

	if len(report.Advisories) != 1 {
		t.Fatalf("expected 1 advisory, got %d", len(report.Advisories))
	}
	// Should fall back to generated title
	if !strings.Contains(report.Advisories[0].Title, "express") {
		t.Errorf("expected fallback title with module name, got %s", report.Advisories[0].Title)
	}
}

func TestRenderDependencyAudit_WithReport(t *testing.T) {
	report := &VulnerabilityReport{
		Total: 3, Critical: 1, High: 1, Moderate: 1, Low: 0,
		Advisories: []Advisory{
			{Module: "lodash", Severity: "critical", Title: "Prototype Pollution", URL: "https://example.com"},
		},
	}

	output := renderDependencyAudit(report)
	if !strings.Contains(output, "# Dependency Audit") {
		t.Error("missing report header")
	}
	if !strings.Contains(output, "3 total") {
		t.Error("missing total count")
	}
	if !strings.Contains(output, "1 critical") {
		t.Error("missing critical count")
	}
	if !strings.Contains(output, "Prototype Pollution") {
		t.Error("missing advisory title")
	}
}

func TestRenderDependencyAudit_Nil(t *testing.T) {
	output := renderDependencyAudit(nil)
	if !strings.Contains(output, "skipped") {
		t.Error("expected skip message for nil report")
	}
}

func TestRenderDependencyAudit_Clean(t *testing.T) {
	report := &VulnerabilityReport{}
	output := renderDependencyAudit(report)
	if !strings.Contains(output, "No vulnerabilities found") {
		t.Error("expected clean message for zero vulnerabilities")
	}
}

func TestRenderDependencySection(t *testing.T) {
	report := &VulnerabilityReport{
		Total: 5, Critical: 2, High: 1, Moderate: 1, Low: 1,
	}

	section := renderDependencySection(report)
	if !strings.Contains(section, "## Dependencies") {
		t.Error("missing section header")
	}
	if !strings.Contains(section, "| Critical | 2 |") {
		t.Error("missing critical count")
	}
	if !strings.Contains(section, "| **Total** | **5** |") {
		t.Error("missing total count")
	}
}

func TestRenderDependencySection_Nil(t *testing.T) {
	section := renderDependencySection(nil)
	if !strings.Contains(section, "skipped") {
		t.Error("expected skip message for nil report")
	}
}
