package search

import (
	"strings"
	"unicode"
)

func tokenize(s string) []string {
	s = strings.ToLower(s)
	var cur strings.Builder
	var out []string
	flush := func() {
		if cur.Len() == 0 {
			return
		}
		out = append(out, cur.String())
		cur.Reset()
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			cur.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return out
}

func keywordScore(searchText string, tokens []string) float64 {
	if len(tokens) == 0 {
		return 0
	}
	var score float64
	for _, t := range tokens {
		if len(t) < 2 {
			continue
		}
		c := strings.Count(searchText, t)
		if c > 0 {
			score += float64(c) * (1 + 0.1*float64(len(t)))
		}
	}
	return score
}
