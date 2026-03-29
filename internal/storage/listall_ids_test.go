package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"lazy-tool/pkg/models"
)

func TestListAllIDs_orderMatchesListAll(t *testing.T) {
	ctx := context.Background()
	p := filepath.Join(t.TempDir(), "ids.db")
	s, err := OpenSQLite(p)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	now := time.Now()
	// source_id order: s2 before s9; canonical breaks tie.
	recs := []models.CapabilityRecord{
		{ID: "b", Kind: models.CapabilityKindTool, SourceType: "g", SourceID: "s2", CanonicalName: "s2__b", OriginalName: "b", SearchText: "b", GeneratedSummary: "x", OriginalDescription: "d", VersionHash: "1", LastSeenAt: now, InputSchemaJSON: "{}", MetadataJSON: "{}"},
		{ID: "a", Kind: models.CapabilityKindTool, SourceType: "g", SourceID: "s2", CanonicalName: "s2__a", OriginalName: "a", SearchText: "a", GeneratedSummary: "x", OriginalDescription: "d", VersionHash: "1", LastSeenAt: now, InputSchemaJSON: "{}", MetadataJSON: "{}"},
		{ID: "c", Kind: models.CapabilityKindTool, SourceType: "g", SourceID: "s9", CanonicalName: "s9__c", OriginalName: "c", SearchText: "c", GeneratedSummary: "x", OriginalDescription: "d", VersionHash: "1", LastSeenAt: now, InputSchemaJSON: "{}", MetadataJSON: "{}"},
	}
	for _, rec := range recs {
		if err := s.UpsertCapability(ctx, rec); err != nil {
			t.Fatal(err)
		}
	}

	all, err := s.ListAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ids, err := s.ListAllIDs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != len(all) {
		t.Fatalf("len ids=%d all=%d", len(ids), len(all))
	}
	for i := range ids {
		if ids[i] != all[i].ID {
			t.Fatalf("idx %d: id=%s want %s", i, ids[i], all[i].ID)
		}
	}
}
