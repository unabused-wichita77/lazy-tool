package search

import (
	"math"

	"lazy-tool/pkg/models"
)

// scaleScoreBreakdownsAfterRank scales each hit's ScoreBreakdown so components sum to the same ratio
// as the ranker's normalized Score vs pre-rank raw total (per capability id).
func scaleScoreBreakdownsAfterRank(results []models.SearchResult, rawTotalByCap map[string]float64) {
	for i := range results {
		id := results[i].CapabilityID
		raw, ok := rawTotalByCap[id]
		if !ok || raw < 1e-12 || results[i].ScoreBreakdown == nil {
			continue
		}
		norm := results[i].Score
		f := norm / raw
		for k, v := range results[i].ScoreBreakdown {
			results[i].ScoreBreakdown[k] = math.Round(v*f*100) / 100
		}
	}
}
