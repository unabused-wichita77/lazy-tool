package embeddings

import "context"

type Noop struct{}

func (Noop) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	_ = ctx
	out := make([][]float32, len(texts))
	return out, nil
}

func (Noop) ModelName() string { return "noop" }
