package service

import (
	"strings"
	"unicode"
)

// normalizeName lowercases and keeps only letters and digits, and drops a
// trailing plural "s": "Pull-Ups" -> "pullup".
func normalizeName(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	n := b.String()
	if len(n) > 3 && strings.HasSuffix(n, "s") && !strings.HasSuffix(n, "ss") {
		n = n[:len(n)-1]
	}
	return n
}

// similarNames reports whether two exercise names are close enough to warn
// about: equal after normalization, a small typo apart, or one containing
// the other (only for names of 4+ characters, so "row" or "dip" don't match
// everything).
func similarNames(a, b string) bool {
	x, y := normalizeName(a), normalizeName(b)
	if x == "" || y == "" {
		return false
	}
	if x == y {
		return true
	}
	short := min(len([]rune(x)), len([]rune(y)))
	if short >= 4 && (strings.Contains(x, y) || strings.Contains(y, x)) {
		return true
	}
	maxDist := 1
	if short >= 8 {
		maxDist = 2
	}
	return short >= 4 && levenshtein(x, y) <= maxDist
}

func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	cur := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(rb)]
}
