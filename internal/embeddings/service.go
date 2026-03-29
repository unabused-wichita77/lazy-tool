package embeddings

import "lazy-tool/internal/config"

// New matches the spec’s suggested entrypoint name (Phase 7 / §17 `service.go`).
// It delegates to [FromConfig].
func New(cfg *config.Config) Embedder {
	return FromConfig(cfg)
}
