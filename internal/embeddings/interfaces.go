package embeddings

import "context"

// Embedder turns text into a dense vector for semantic search.
type Embedder interface {
	Embed(ctx context.Context, texts []string) (vectors [][]float32, err error)
	ModelName() string
}
