package syntax

import (
	"sort"
	"strings"

	cerr "github.com/barun-bash/human/internal/errors"
)

type scored struct {
	pattern Pattern
	score   float64
}

// Search returns patterns matching the query, sorted by relevance.
// Uses substring matching and fuzzy matching against templates, tags, and descriptions.
func Search(query string) []Pattern {
	if query == "" {
		return AllPatterns()
	}

	q := strings.ToLower(query)
	var results []scored

	for _, p := range allPatterns {
		score := scorePattern(p, q)
		if score > 0 {
			results = append(results, scored{pattern: p, score: score})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	patterns := make([]Pattern, len(results))
	for i, r := range results {
		patterns[i] = r.pattern
	}
	return patterns
}

// Autocomplete returns patterns whose templates start with the given prefix.
func Autocomplete(prefix string) []Pattern {
	if prefix == "" {
		return nil
	}

	p := strings.ToLower(prefix)
	var results []Pattern

	for _, pat := range allPatterns {
		if strings.HasPrefix(strings.ToLower(pat.Template), p) {
			results = append(results, pat)
		}
	}
	return results
}

// scorePattern scores how well a pattern matches a query string.
func scorePattern(p Pattern, query string) float64 {
	best := 0.0

	// Exact substring in template → 1.0
	if strings.Contains(strings.ToLower(p.Template), query) {
		best = 1.0
	}

	// Exact substring in tags → 0.9
	for _, tag := range p.Tags {
		if strings.Contains(strings.ToLower(tag), query) {
			if 0.9 > best {
				best = 0.9
			}
		}
	}

	// Exact substring in description → 0.8
	if strings.Contains(strings.ToLower(p.Description), query) {
		if 0.8 > best {
			best = 0.8
		}
	}

	// Fuzzy match against tags → 0.7
	if best == 0 {
		for _, tag := range p.Tags {
			sim := cerr.Similarity(query, tag)
			if sim > 0.6 {
				if 0.7 > best {
					best = 0.7
				}
			}
		}
	}

	return best
}
