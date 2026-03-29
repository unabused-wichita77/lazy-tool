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

func TestService_Search_hybridLexical(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.db")
	st, err := storage.OpenSQLite(p)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()
	rec := models.CapabilityRecord{
		ID:                  "1",
		Kind:                models.CapabilityKindTool,
		SourceID:            "github-gateway",
		SourceType:          "gateway",
		CanonicalName:       "github_gateway__create_issue",
		OriginalName:        "create_issue",
		OriginalDescription: "Create an issue in a repo",
		GeneratedSummary:    "Creates GitHub issues with title and body.",
		Tags:                []string{"title", "body", "repo"},
		InputSchemaJSON:     `{"properties":{"repo":{"type":"string"},"title":{"type":"string"}}}`,
		VersionHash:         "h1",
		LastSeenAt:          time.Now(),
	}
	rec.SearchText = "github-gateway create_issue repo title body issue"
	if err := st.UpsertCapability(ctx, rec); err != nil {
		t.Fatal(err)
	}
	svc := NewService(st, nil, embeddings.Noop{}, ScoreWeights{}, false)
	out, err := svc.Search(ctx, models.SearchQuery{Text: "create github issue", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Results) == 0 {
		t.Fatal("no results")
	}
	if out.Results[0].ProxyToolName != rec.CanonicalName {
		t.Fatalf("got %v", out.Results[0])
	}
	if out.Results[0].Kind != models.CapabilityKindTool {
		t.Fatalf("kind: got %q want tool", out.Results[0].Kind)
	}
}

func TestService_Search_exactCanonicalBeatsWeakerMatch(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.db")
	st, err := storage.OpenSQLite(p)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()
	weak := models.CapabilityRecord{
		ID: "w", Kind: models.CapabilityKindTool, SourceID: "s", SourceType: "gateway",
		CanonicalName: "s__other", OriginalName: "other",
		GeneratedSummary: "Mentions github_gateway__create_issue in passing.",
		SearchText:       "s other github_gateway__create_issue mention", VersionHash: "1", LastSeenAt: time.Now(),
		InputSchemaJSON: "{}", MetadataJSON: "{}",
	}
	exact := models.CapabilityRecord{
		ID: "e", Kind: models.CapabilityKindTool, SourceID: "s", SourceType: "gateway",
		CanonicalName: "github_gateway__create_issue", OriginalName: "create_issue",
		GeneratedSummary: "Creates issues.",
		SearchText:       "github_gateway create_issue", VersionHash: "2", LastSeenAt: time.Now(),
		InputSchemaJSON: "{}", MetadataJSON: "{}",
	}
	if err := st.UpsertCapability(ctx, weak); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertCapability(ctx, exact); err != nil {
		t.Fatal(err)
	}
	svc := NewService(st, nil, embeddings.Noop{}, ScoreWeights{}, false)
	out, err := svc.Search(ctx, models.SearchQuery{Text: "github_gateway__create_issue", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Results) < 1 || out.Results[0].ProxyToolName != exact.CanonicalName {
		t.Fatalf("want exact canonical first, got %+v", out.Results)
	}
}

func TestService_Search_exactOriginalName(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.db")
	st, err := storage.OpenSQLite(p)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()
	rec := models.CapabilityRecord{
		ID: "1", Kind: models.CapabilityKindTool, SourceID: "s", SourceType: "gateway",
		CanonicalName: "s__my_tool", OriginalName: "unique_orig_name",
		GeneratedSummary: "Summary.", SearchText: "s unique_orig_name summary", VersionHash: "1",
		LastSeenAt: time.Now(), InputSchemaJSON: "{}", MetadataJSON: "{}",
	}
	if err := st.UpsertCapability(ctx, rec); err != nil {
		t.Fatal(err)
	}
	svc := NewService(st, nil, embeddings.Noop{}, ScoreWeights{}, false)
	out, err := svc.Search(ctx, models.SearchQuery{Text: "unique_orig_name", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Results) != 1 || out.Results[0].CapabilityID != rec.ID {
		t.Fatalf("got %+v", out.Results)
	}
	if len(out.Results[0].WhyMatched) == 0 {
		t.Fatal("expected why_matched")
	}
}

func TestService_Search_ftsTagOrSource(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.db")
	st, err := storage.OpenSQLite(p)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()
	rec := models.CapabilityRecord{
		ID: "t1", Kind: models.CapabilityKindTool, SourceID: "special-source", SourceType: "gateway",
		CanonicalName: "special_source__noop", OriginalName: "noop",
		GeneratedSummary: "Unrelated summary text.",
		Tags:             []string{"quark"},
		SearchText:       "special-source noop unrelated quark",
		VersionHash:      "1", LastSeenAt: time.Now(),
		InputSchemaJSON: "{}", MetadataJSON: "{}",
	}
	if err := st.UpsertCapability(ctx, rec); err != nil {
		t.Fatal(err)
	}
	svc := NewService(st, nil, embeddings.Noop{}, ScoreWeights{}, false)
	out, err := svc.Search(ctx, models.SearchQuery{Text: "quark", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Results) == 0 || out.Results[0].ProxyToolName != rec.CanonicalName {
		t.Fatalf("got %+v", out.Results)
	}
	out2, err := svc.Search(ctx, models.SearchQuery{Text: "special-source", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(out2.Results) == 0 {
		t.Fatal("expected hit by source id in fts")
	}
}

func TestService_Search_userSummaryBoost(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.db")
	st, err := storage.OpenSQLite(p)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()
	base := models.CapabilityRecord{
		Kind: models.CapabilityKindTool, SourceID: "s", SourceType: "gateway",
		OriginalName: "t", GeneratedSummary: "same",
		SearchText: "s t shared-token", VersionHash: "1", LastSeenAt: time.Now(),
		InputSchemaJSON: "{}", MetadataJSON: "{}",
	}
	a := base
	a.ID = "a"
	a.CanonicalName = "s__a"
	b := base
	b.ID = "b"
	b.CanonicalName = "s__b"
	b.UserSummary = "operator pinned"
	if err := st.UpsertCapability(ctx, a); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertCapability(ctx, b); err != nil {
		t.Fatal(err)
	}
	svc := NewService(st, nil, embeddings.Noop{}, DefaultScoreWeights(), false)
	out, err := svc.Search(ctx, models.SearchQuery{Text: "shared-token", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Results) < 2 {
		t.Fatalf("need 2 hits, got %d", len(out.Results))
	}
	if out.Results[0].CapabilityID != b.ID {
		t.Fatalf("user-edited summary should rank first, got %+v", out.Results)
	}
	found := false
	for _, w := range out.Results[0].WhyMatched {
		if w == "user:edited-summary" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected user:edited-summary in why_matched: %#v", out.Results[0].WhyMatched)
	}
}

func TestService_Search_noopEmbeddingsNoPanic(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.db")
	st, err := storage.OpenSQLite(p)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()
	rec := models.CapabilityRecord{
		ID: "n1", Kind: models.CapabilityKindTool, SourceID: "s", SourceType: "gateway",
		CanonicalName: "s__x", OriginalName: "x", GeneratedSummary: "hello world",
		SearchText: "s x hello world", VersionHash: "1", LastSeenAt: time.Now(),
		InputSchemaJSON: "{}", MetadataJSON: "{}",
	}
	if err := st.UpsertCapability(ctx, rec); err != nil {
		t.Fatal(err)
	}
	svc := NewService(st, nil, embeddings.Noop{}, ScoreWeights{}, false)
	_, err = svc.Search(ctx, models.SearchQuery{Text: "hello world", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
}
