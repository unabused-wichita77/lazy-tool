package search

import (
	"context"
	"math"
	"path/filepath"
	"testing"
	"time"

	"lazy-tool/internal/embeddings"
	"lazy-tool/internal/storage"
	"lazy-tool/pkg/models"
)

func TestSearch_explainScores_populatesBreakdown(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.db")
	st, err := storage.OpenSQLite(p)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()
	rec := models.CapabilityRecord{
		ID: "1", Kind: models.CapabilityKindTool, SourceID: "gw", SourceType: "gateway",
		CanonicalName: "gw__echo", OriginalName: "echo",
		GeneratedSummary: "echo tool", SearchText: "gw echo tool",
		VersionHash: "v", LastSeenAt: time.Now(),
	}
	if err := st.UpsertCapability(ctx, rec); err != nil {
		t.Fatal(err)
	}
	svc := NewService(st, nil, embeddings.Noop{}, ScoreWeights{
		ExactCanonical: 10,
		ExactName:      8,
		Substring:      2,
	}, false)
	ranked, err := svc.Search(ctx, models.SearchQuery{Text: "gw__echo", Limit: 5, ExplainScores: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(ranked.Results) != 1 {
		t.Fatalf("results: %d", len(ranked.Results))
	}
	bd := ranked.Results[0].ScoreBreakdown
	if bd == nil {
		t.Fatal("no breakdown")
	}
	// Ranker normalizes scores; breakdown is scaled to the same ratio as normalized Score.
	sum := 0.0
	for _, v := range bd {
		sum += v
	}
	if math.Abs(sum-ranked.Results[0].Score) > 0.02 {
		t.Fatalf("breakdown sum %v vs score %v: %#v", sum, ranked.Results[0].Score, bd)
	}
}
