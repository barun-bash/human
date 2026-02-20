package errors

import "strings"

// levenshtein computes the Damerau-Levenshtein distance between two strings.
// It counts insertions, deletions, substitutions, and transpositions of
// adjacent characters — each as a single edit. This is better for typo
// detection than plain Levenshtein (where a transposition costs 2).
func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	// Need two previous rows for transposition detection.
	prevprev := make([]int, lb+1)
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
			best := min3(ins, del, sub)

			// Transposition: swap of two adjacent characters
			if i > 1 && j > 1 && a[i-1] == b[j-2] && a[i-2] == b[j-1] {
				trans := prevprev[j-2] + 1
				if trans < best {
					best = trans
				}
			}
			curr[j] = best
		}
		prevprev = prev
		prev = curr
	}

	return prev[lb]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// Similarity returns a normalized similarity score between 0.0 and 1.0.
// 1.0 means identical strings, 0.0 means completely different.
func Similarity(a, b string) float64 {
	a = strings.ToLower(a)
	b = strings.ToLower(b)
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

// FindClosest returns the candidate most similar to target, or an empty
// string if no candidate exceeds the threshold.
func FindClosest(target string, candidates []string, threshold float64) string {
	best := ""
	bestScore := 0.0
	for _, c := range candidates {
		score := Similarity(target, c)
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
