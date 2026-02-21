package prompts

import "strings"

// ExtractHumanCode extracts .human code from an LLM response by stripping
// markdown code fences. If the response contains a ```human fence, it extracts
// the content within. If no fence is found, returns the raw response.
func ExtractHumanCode(response string) string {
	// Try to find a ```human code fence first.
	code := extractFence(response, "human")
	if code != "" {
		return code
	}

	// Try a generic ``` fence.
	code = extractFence(response, "")
	if code != "" {
		return code
	}

	// No fence found — return the raw response trimmed.
	return strings.TrimSpace(response)
}

// extractFence finds and extracts content from the first code fence with
// the given language tag. Returns "" if not found.
func extractFence(text, lang string) string {
	opener := "```"
	if lang != "" {
		opener = "```" + lang
	}

	lines := strings.Split(text, "\n")
	var result []string
	inFence := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if !inFence {
			if strings.HasPrefix(trimmed, opener) {
				inFence = true
				continue
			}
		} else {
			if trimmed == "```" {
				// End of fence — return what we have.
				return strings.TrimSpace(strings.Join(result, "\n"))
			}
			result = append(result, line)
		}
	}

	// Unclosed fence — return what we found anyway.
	if inFence && len(result) > 0 {
		return strings.TrimSpace(strings.Join(result, "\n"))
	}

	return ""
}

// Suggestion is a parsed suggestion from the LLM's analysis.
type Suggestion struct {
	Category string // e.g. "security", "performance"
	Text     string // the suggestion text
}

// ExtractSuggestions parses categorized suggestions from an LLM response.
// Expected format: lines starting with [category] followed by suggestion text.
func ExtractSuggestions(response string) []Suggestion {
	var suggestions []Suggestion

	for _, line := range strings.Split(response, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Look for [category] prefix.
		if !strings.HasPrefix(line, "[") {
			continue
		}

		closeBracket := strings.Index(line, "]")
		if closeBracket < 2 {
			continue
		}

		category := strings.ToLower(line[1:closeBracket])
		text := strings.TrimSpace(line[closeBracket+1:])

		if text == "" {
			continue
		}

		suggestions = append(suggestions, Suggestion{
			Category: category,
			Text:     text,
		})
	}

	return suggestions
}

// EstimateTokens provides a rough token count estimate for a text string.
// Uses the ~4 characters per token heuristic.
func EstimateTokens(text string) int {
	if len(text) == 0 {
		return 0
	}
	return (len(text) + 3) / 4
}

// ContextWindowSize returns the approximate context window size for known models.
func ContextWindowSize(model string) int {
	model = strings.ToLower(model)

	switch {
	case strings.Contains(model, "claude"):
		return 200000
	case strings.Contains(model, "gpt-4o"):
		return 128000
	case strings.Contains(model, "gpt-4"):
		return 128000
	case strings.Contains(model, "gpt-3.5"):
		return 16000
	case strings.Contains(model, "llama"):
		return 8000
	default:
		return 8000
	}
}
