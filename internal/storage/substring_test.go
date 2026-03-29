package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"lazy-tool/pkg/models"
)

func TestListIDsBySearchTextSubstring_likeMetacharactersAndFilter(t *testing.T) {
	ctx := context.Background()
	p := filepath.Join(t.TempDir(), "sub.db")
	s, err := OpenSQLite(p)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	base := models.CapabilityRecord{
		Kind: models.CapabilityKindTool, SourceType: "g",
		OriginalDescription: "d", VersionHash: "1", LastSeenAt: time.Now(),
		InputSchemaJSON: "{}", MetadataJSON: "{}",
	}
	a := base
	a.ID, a.SourceID = "a", "s1"
	a.CanonicalName, a.OriginalName = "s1__a", "a"
	a.SearchText = "alpha 100% done tool_name x"
	b := base
	b.ID, b.SourceID = "b", "s2"
	b.CanonicalName, b.OriginalName = "s2__b", "b"
	b.SearchText = "beta other"
	for _, rec := range []models.CapabilityRecord{a, b} {
		if err := s.UpsertCapability(ctx, rec); err != nil {
			t.Fatal(err)
		}
	}

	ids, err := s.ListIDsBySearchTextSubstring(ctx, "100% done", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != "a" {
		t.Fatalf("100%% done: got %+v", ids)
	}

	ids, err = s.ListIDsBySearchTextSubstring(ctx, "tool_name", []string{"s1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != "a" {
		t.Fatalf("source filter: got %+v", ids)
	}

	ids, err = s.ListIDsBySearchTextSubstring(ctx, "tool_name", []string{"s2"})
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 0 {
		t.Fatalf("want no match for s2, got %+v", ids)
	}

	ids, err = s.ListIDsBySearchTextSubstring(ctx, `%`, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != "a" {
		t.Fatalf("literal percent: got %+v", ids)
	}
}

func TestListIDsBySearchTextSubstring_emptyNeedle(t *testing.T) {
	ctx := context.Background()
	p := filepath.Join(t.TempDir(), "e.db")
	s, err := OpenSQLite(p)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	ids, err := s.ListIDsBySearchTextSubstring(ctx, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 0 {
		t.Fatalf("want nil, got %+v", ids)
	}
}
