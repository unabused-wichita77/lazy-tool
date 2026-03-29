package ranking

import (
	"context"

	"lazy-tool/pkg/models"
)

// Ranker merges candidate results into a final ordered list with explanations.
type Ranker interface {
	Rank(ctx context.Context, q models.SearchQuery, candidates []models.SearchResult) (models.RankedResults, error)
}
