package search

import "lazy-tool/pkg/models"

// GroupResultsBySource partitions a flat ranked list by source_id (first-seen order).
func GroupResultsBySource(results []models.SearchResult) []models.SourceResultGroup {
	if len(results) == 0 {
		return nil
	}
	var order []string
	by := make(map[string][]models.SearchResult)
	for _, r := range results {
		if _, ok := by[r.SourceID]; !ok {
			order = append(order, r.SourceID)
		}
		by[r.SourceID] = append(by[r.SourceID], r)
	}
	out := make([]models.SourceResultGroup, 0, len(order))
	for _, sid := range order {
		out = append(out, models.SourceResultGroup{SourceID: sid, Results: by[sid]})
	}
	return out
}
