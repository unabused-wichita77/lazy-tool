package summarizer

import (
	"context"
	"strings"

	"lazy-tool/pkg/models"
)

type Noop struct{}

func (Noop) Summarize(ctx context.Context, rec models.CapabilityRecord) (string, error) {
	_ = ctx
	d := strings.TrimSpace(rec.OriginalDescription)
	if d == "" {
		return "Tool " + rec.OriginalName + " (no upstream description).", nil
	}
	if len(d) > 160 {
		d = d[:157] + "..."
	}
	return d, nil
}
