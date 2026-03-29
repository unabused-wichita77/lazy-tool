package search

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"lazy-tool/internal/embeddings"
	"lazy-tool/internal/storage"
	"lazy-tool/pkg/models"
)

func TestSearch_fullCatalogSubstringDisabled_skipsSubstringCandidates(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.db")
	st, err := storage.OpenSQLite(p)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()
	rec := models.CapabilityRecord{
		ID: "1", Kind: models.CapabilityKindTool, SourceID: "s", SourceType: "gateway",
		CanonicalName: "s__x", OriginalName: "x",
		GeneratedSummary: "hi", SearchText: "s x tool z marker",
		VersionHash: "1", LastSeenAt: time.Now(), InputSchemaJSON: "{}", MetadataJSON: "{}",
	}
	if err := st.UpsertCapability(ctx, rec); err != nil {
		t.Fatal(err)
	}
	svc := NewService(st, nil, embeddings.Noop{}, ScoreWeights{}, false)
	svc.FullCatalogSubstring = false
	ranked, err := svc.Search(ctx, models.SearchQuery{Text: "z", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if ranked.CandidatePath != models.SearchCandidatePathFullCatalogSubstringDisabled {
		t.Fatalf("candidate path: %q", ranked.CandidatePath)
	}
	if len(ranked.Results) != 0 {
		t.Fatalf("expected no substring-only hits when disabled, got %d", len(ranked.Results))
	}
}
