package summarizer

import (
	"context"

	"lazy-tool/pkg/models"
)

// Summarizer produces a short dense sentence for a capability record.
type Summarizer interface {
	Summarize(ctx context.Context, rec models.CapabilityRecord) (summary string, err error)
}
