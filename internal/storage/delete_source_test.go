package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"lazy-tool/pkg/models"
)

func TestDeleteAllCapabilitiesForSource(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.db")
	s, err := OpenSQLite(p)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	ctx := context.Background()
	rec := models.CapabilityRecord{
		ID: "1", Kind: models.CapabilityKindTool, SourceID: "src-a", SourceType: "gateway",
		CanonicalName: "src_a__x", OriginalName: "x", SearchText: "x", VersionHash: "v",
		LastSeenAt: time.Now(), InputSchemaJSON: "{}", MetadataJSON: "{}",
	}
	if err := s.UpsertCapability(ctx, rec); err != nil {
		t.Fatal(err)
	}
	n, err := s.DeleteAllCapabilitiesForSource(ctx, "src-a")
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	all, err := s.ListAll(ctx)
	if err != nil || len(all) != 0 {
		t.Fatalf("list: %v err %v", all, err)
	}
}
