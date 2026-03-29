package catalog

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"lazy-tool/internal/storage"
	"lazy-tool/pkg/models"
)

func TestSetUserSummary_roundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "u.db")
	st, err := storage.OpenSQLite(p)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()
	rec := models.CapabilityRecord{
		ID:               "id1",
		Kind:             models.CapabilityKindTool,
		SourceID:         "s1",
		SourceType:       "gateway",
		CanonicalName:    "s1__t",
		OriginalName:     "t",
		GeneratedSummary: "gen",
		SearchText:       "s1 gateway tool t s1__t gen",
		InputSchemaJSON:  "{}",
		MetadataJSON:     "{}",
		VersionHash:      "h",
		LastSeenAt:       time.Now().UTC().Truncate(time.Second),
	}
	RefreshSearchText(&rec)
	if err := st.UpsertCapability(ctx, rec); err != nil {
		t.Fatal(err)
	}
	if err := SetUserSummary(ctx, st, "s1__t", "manual note"); err != nil {
		t.Fatal(err)
	}
	got, err := st.GetByCanonicalName(ctx, "s1__t")
	if err != nil {
		t.Fatal(err)
	}
	if got.UserSummary != "manual note" {
		t.Fatalf("user summary %q", got.UserSummary)
	}
	if got.SearchText == "" || !strings.Contains(got.SearchText, "manual") {
		t.Fatalf("search text should include manual summary: %q", got.SearchText)
	}
}
