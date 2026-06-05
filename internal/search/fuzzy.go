package search

import "strings"

// fuzzyScore ranks how well query matches text (higher is better). Empty query returns 0.
func fuzzyScore(query, text string) int {
	query = strings.ToLower(strings.TrimSpace(query))
	text = strings.ToLower(text)
	if query == "" {
		return 0
	}
	if text == query {
		return 10000
	}
	if strings.HasPrefix(text, query) {
		return 5000 + len(query)
	}
	// Subsequence match with bonuses for consecutive runs and early position.
	qi, score, run, first := 0, 0, 0, -1
	for ti := 0; ti < len(text) && qi < len(query); ti++ {
		if text[ti] == query[qi] {
			if first < 0 {
				first = ti
			}
			run++
			score += run * 4
			if ti < 10 {
				score += 2
			}
			qi++
		} else {
			run = 0
		}
	}
	if qi < len(query) {
		return -1
	}
	if first == 0 {
		score += 50
	}
	return score
}
