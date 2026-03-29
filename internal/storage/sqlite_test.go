package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"lazy-tool/pkg/models"
)

func TestSQLite_UpsertRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "t.db")
	s, err := OpenSQLite(p)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	ctx := context.Background()
	rec := models.CapabilityRecord{
		ID:                  "id1",
		Kind:                models.CapabilityKindTool,
		SourceID:            "s1",
		SourceType:          "gateway",
		CanonicalName:       "s1__toola",
		OriginalName:        "toolA",
		OriginalDescription: "does a",
		GeneratedSummary:    "Does a thing.",
		SearchText:          "toola does",
		InputSchemaJSON:     "{}",
		MetadataJSON:        "{}",
		Tags:                []string{"x"},
		VersionHash:         "abc",
		LastSeenAt:          time.Now().UTC().Truncate(time.Second),
	}
	if err := s.UpsertCapability(ctx, rec); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetCapability(ctx, "id1")
	if err != nil {
		t.Fatal(err)
	}
	if got.CanonicalName != rec.CanonicalName {
		t.Fatal(got.CanonicalName)
	}
	got2, err := s.GetByCanonicalName(ctx, "s1__toola")
	if err != nil {
		t.Fatal(err)
	}
	if got2.ID != rec.ID {
		t.Fatal(got2.ID)
	}
}
