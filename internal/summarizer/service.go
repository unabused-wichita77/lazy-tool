package summarizer

import "lazy-tool/internal/config"

// New matches the spec’s suggested entrypoint name (Phase 26 / §17 `service.go`).
// It delegates to [FromConfig].
func New(cfg *config.Config) Summarizer {
	return FromConfig(cfg)
}
