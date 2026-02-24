package repl

import "strings"

// findClosest returns the candidate most similar to target, or an empty
// string if no candidate exceeds the threshold. Uses a simple Levenshtein
// distance comparison.
func findClosest(target string, candidates []string, threshold float64) string {
	target = strings.ToLower(target)
	best := ""
	bestScore := 0.0

	for _, c := range candidates {
		score := similarity(target, strings.ToLower(c))
		if score > bestScore {
			bestScore = score
			best = c
		}
	}

	if bestScore >= threshold {
		return best
	}
	return ""
}

// similarity returns a normalized score between 0.0 and 1.0.
func similarity(a, b string) float64 {
	if a == b {
		return 1.0
	}
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	if maxLen == 0 {
		return 1.0
	}
	dist := levenshtein(a, b)
	return 1.0 - float64(dist)/float64(maxLen)
}

// levenshtein computes the edit distance between two strings.
func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	prev := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr := make([]int, lb+1)
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			ins := curr[j-1] + 1
			del := prev[j] + 1
			sub := prev[j-1] + cost
			best := ins
			if del < best {
				best = del
			}
			if sub < best {
				best = sub
			}
			curr[j] = best
		}
		prev = curr
	}

	return prev[lb]
}
