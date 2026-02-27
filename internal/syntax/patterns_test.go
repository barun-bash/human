package syntax

import (
	"testing"
)

func TestAllCategoriesHavePatterns(t *testing.T) {
	for _, cat := range AllCategories() {
		patterns := ByCategory(cat)
		if len(patterns) < 3 {
			t.Errorf("category %q has only %d patterns (want >= 3)", cat, len(patterns))
		}
	}
}

func TestAllPatternsHaveRequiredFields(t *testing.T) {
	for i, p := range AllPatterns() {
		if p.Template == "" {
			t.Errorf("pattern %d has empty Template", i)
		}
		if p.Description == "" {
			t.Errorf("pattern %d (%q) has empty Description", i, p.Template)
		}
		if p.Category == "" {
			t.Errorf("pattern %d (%q) has empty Category", i, p.Template)
		}
		if len(p.Tags) == 0 {
			t.Errorf("pattern %d (%q) has no Tags", i, p.Template)
		}
	}
}

func TestNoDuplicateTemplates(t *testing.T) {
	seen := make(map[string]bool)
	for _, p := range AllPatterns() {
		if seen[p.Template] {
			t.Errorf("duplicate template: %q", p.Template)
		}
		seen[p.Template] = true
	}
}

func TestSearchColor(t *testing.T) {
	results := Search("color")
	if len(results) == 0 {
		t.Fatal("expected results for 'color'")
	}

	// Should include styling and theme patterns.
	hasStyling := false
	hasTheme := false
	for _, p := range results {
		if p.Category == CatStyling {
			hasStyling = true
		}
		if p.Category == CatTheme {
			hasTheme = true
		}
	}
	if !hasStyling {
		t.Error("expected styling patterns in 'color' search")
	}
	if !hasTheme {
		t.Error("expected theme patterns in 'color' search")
	}
}

func TestSearchButton(t *testing.T) {
	results := Search("button")
	if len(results) == 0 {
		t.Fatal("expected results for 'button'")
	}

	hasEvents := false
	hasForms := false
	for _, p := range results {
		if p.Category == CatEvents {
			hasEvents = true
		}
		if p.Category == CatForms {
			hasForms = true
		}
	}
	if !hasEvents {
		t.Error("expected events patterns in 'button' search")
	}
	if !hasForms {
		t.Error("expected forms patterns in 'button' search")
	}
}

func TestSearchEmpty(t *testing.T) {
	results := Search("")
	if len(results) != len(AllPatterns()) {
		t.Errorf("empty search should return all patterns, got %d want %d", len(results), len(AllPatterns()))
	}
}

func TestByCategoryData(t *testing.T) {
	patterns := ByCategory(CatData)
	if len(patterns) == 0 {
		t.Fatal("expected data patterns")
	}
	for _, p := range patterns {
		if p.Category != CatData {
			t.Errorf("expected category data, got %q", p.Category)
		}
	}
}

func TestAutocomplete(t *testing.T) {
	results := Autocomplete("show")
	if len(results) == 0 {
		t.Fatal("expected results for 'show' autocomplete")
	}
	for _, p := range results {
		lower := p.Template
		if len(lower) < 4 || lower[:4] != "show" {
			t.Errorf("expected template starting with 'show', got %q", p.Template)
		}
	}
}

func TestAutocompleteEmpty(t *testing.T) {
	results := Autocomplete("")
	if results != nil {
		t.Error("empty prefix should return nil")
	}
}

func TestTotalPatternCount(t *testing.T) {
	all := AllPatterns()
	if len(all) < 100 {
		t.Errorf("expected at least 100 patterns, got %d", len(all))
	}
}

func TestCategoryLabel(t *testing.T) {
	label := CategoryLabel(CatData)
	if label != "Data Models" {
		t.Errorf("expected 'Data Models', got %q", label)
	}
	label = CategoryLabel(CatAPIs)
	if label != "APIs & Endpoints" {
		t.Errorf("expected 'APIs & Endpoints', got %q", label)
	}
}
