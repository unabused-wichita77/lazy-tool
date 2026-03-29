package search

import "testing"

func TestKeywordScore(t *testing.T) {
	s := keywordScore("hello world create issue", []string{"create", "issue"})
	if s <= 0 {
		t.Fatal(s)
	}
}
