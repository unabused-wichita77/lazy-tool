package search

import (
	"context"

	"github.com/philippgille/chromem-go"
	"lazy-tool/internal/vector"
)

// VectorQuery runs embedding similarity against the chromem index (spec §14 vector leg).
func VectorQuery(ctx context.Context, idx *vector.Index, embedding []float32, limit int, sourceID string) ([]chromem.Result, error) {
	if idx == nil {
		return nil, nil
	}
	return idx.Query(ctx, embedding, limit, sourceID)
}
