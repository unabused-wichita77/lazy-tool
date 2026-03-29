package search

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"lazy-tool/internal/embeddings"
	"lazy-tool/internal/metrics"
	"lazy-tool/internal/storage"
	"lazy-tool/pkg/models"
)

func TestService_searchEmptyQuery_maxCatalogIDs_truncatesAndMetrics(t *testing.T) {
	ctx := context.Background()
	p := filepath.Join(t.TempDir(), "eq.db")
	st, err := storage.OpenSQLite(p)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()

	base := models.CapabilityRecord{
		Kind:            models.CapabilityKindTool,
		SourceID:        "s",
		SourceType:      "gateway",
		InputSchemaJSON: "{}", MetadataJSON: "{}",
		VersionHash: "v", LastSeenAt: time.Now(),
	}
	for _, id := range []string{"a", "b", "c"} {
		r := base
		r.ID = id
		r.CanonicalName = "s__" + id
		r.OriginalName = id
		r.GeneratedSummary = id
		r.SearchText = id
		if err := st.UpsertCapability(ctx, r); err != nil {
			t.Fatal(err)
		}
	}

	var gotTotal, gotProc int
	var gotTrunc bool
	prev := metrics.SearchEmptyQueryScan
	metrics.SearchEmptyQueryScan = func(total, proc int, trunc bool) {
		gotTotal, gotProc, gotTrunc = total, proc, trunc
	}
	defer func() { metrics.SearchEmptyQueryScan = prev }()

	svc := NewService(st, nil, embeddings.Noop{}, ScoreWeights{Favorite: 5}, false)
	svc.EmptyQueryMaxCatalogIDs = 2
	out, err := svc.Search(ctx, models.SearchQuery{
		Text:        "",
		Limit:       10,
		FavoriteIDs: map[string]struct{}{"a": {}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotTotal != 3 || gotProc != 2 || !gotTrunc {
		t.Fatalf("metric: total=%d proc=%d trunc=%v", gotTotal, gotProc, gotTrunc)
	}
	if out.CandidatePath != models.SearchCandidatePathEmptyQueryTruncated {
		t.Fatalf("CandidatePath=%q", out.CandidatePath)
	}
	if len(out.Results) != 1 || out.Results[0].CapabilityID != "a" {
		t.Fatalf("results=%+v", out.Results)
	}
}

func TestService_searchEmptyQuery_customBatch(t *testing.T) {
	ctx := context.Background()
	p := filepath.Join(t.TempDir(), "eq2.db")
	st, err := storage.OpenSQLite(p)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	base := models.CapabilityRecord{
		Kind:            models.CapabilityKindTool,
		SourceID:        "s",
		SourceType:      "gateway",
		InputSchemaJSON: "{}", MetadataJSON: "{}",
		VersionHash: "v", LastSeenAt: time.Now(),
	}
	for _, id := range []string{"x", "y"} {
		r := base
		r.ID = id
		r.CanonicalName = "s__" + id
		r.OriginalName = id
		r.GeneratedSummary = id
		r.SearchText = id
		if err := st.UpsertCapability(ctx, r); err != nil {
			t.Fatal(err)
		}
	}
	svc := NewService(st, nil, embeddings.Noop{}, ScoreWeights{Favorite: 1}, false)
	svc.EmptyQueryIDBatch = 1
	_, err = svc.Search(ctx, models.SearchQuery{Text: "", Limit: 10, FavoriteIDs: map[string]struct{}{"x": {}, "y": {}}})
	if err != nil {
		t.Fatal(err)
	}
}
