package search

import (
	"testing"

	"lazy-tool/pkg/models"
)

func TestGroupResultsBySource_order(t *testing.T) {
	results := []models.SearchResult{
		{SourceID: "b", ProxyToolName: "b__1", CapabilityID: "1"},
		{SourceID: "a", ProxyToolName: "a__1", CapabilityID: "2"},
		{SourceID: "a", ProxyToolName: "a__2", CapabilityID: "3"},
	}
	g := GroupResultsBySource(results)
	if len(g) != 2 || g[0].SourceID != "b" || g[1].SourceID != "a" {
		t.Fatalf("unexpected grouping: %+v", g)
	}
	if len(g[1].Results) != 2 {
		t.Fatal(g[1].Results)
	}
}
