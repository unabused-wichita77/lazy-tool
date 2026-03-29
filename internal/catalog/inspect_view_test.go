package catalog

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"lazy-tool/internal/app"
	"lazy-tool/internal/storage"
	"lazy-tool/pkg/models"
)

func TestBuildInspectView_sourceAndHealth(t *testing.T) {
	ctx := context.Background()
	p := filepath.Join(t.TempDir(), "c.db")
	st, err := storage.OpenSQLite(p)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()

	rec := models.CapabilityRecord{
		ID: "1", Kind: models.CapabilityKindTool, SourceID: "gw", SourceType: "gateway",
		CanonicalName: "gw__echo", OriginalName: "echo",
		GeneratedSummary: "x", SearchText: "echo", VersionHash: "v", LastSeenAt: time.Now(),
	}
	if err := st.UpsertCapability(ctx, rec); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertSourceHealth(ctx, "gw", false, "boom"); err != nil {
		t.Fatal(err)
	}
	reg, err := app.NewSourceRegistry([]models.Source{
		{ID: "gw", Type: models.SourceTypeGateway, Transport: models.TransportHTTP, URL: "http://127.0.0.1/mcp"},
	})
	if err != nil {
		t.Fatal(err)
	}

	v, err := BuildInspectView(ctx, st, reg, "gw__echo")
	if err != nil {
		t.Fatal(err)
	}
	if v.Record.ID != "1" {
		t.Fatalf("record: %+v", v.Record)
	}
	if v.Source == nil || v.Source.URL != "http://127.0.0.1/mcp" {
		t.Fatalf("source: %+v", v.Source)
	}
	if v.LastReindex == nil || v.LastReindex.OK || v.LastReindex.Message != "boom" {
		t.Fatalf("last_reindex: %+v", v.LastReindex)
	}
	if !v.SourceInConfig || !v.LastReindexRecorded {
		t.Fatalf("trust flags: SourceInConfig=%v LastReindexRecorded=%v", v.SourceInConfig, v.LastReindexRecorded)
	}
}

func TestBuildInspectView_missingHealthAndSource(t *testing.T) {
	ctx := context.Background()
	p := filepath.Join(t.TempDir(), "c2.db")
	st, err := storage.OpenSQLite(p)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	rec := models.CapabilityRecord{
		ID: "1", Kind: models.CapabilityKindTool, SourceID: "orphan", SourceType: "gateway",
		CanonicalName: "orphan__x", OriginalName: "x",
		GeneratedSummary: "x", SearchText: "x", VersionHash: "v", LastSeenAt: time.Now(),
	}
	if err := st.UpsertCapability(ctx, rec); err != nil {
		t.Fatal(err)
	}
	reg, err := app.NewSourceRegistry([]models.Source{
		{ID: "gw", Type: models.SourceTypeGateway, Transport: models.TransportHTTP, URL: "http://127.0.0.1/mcp"},
	})
	if err != nil {
		t.Fatal(err)
	}
	v, err := BuildInspectView(ctx, st, reg, "orphan__x")
	if err != nil {
		t.Fatal(err)
	}
	if v.SourceInConfig || v.Source != nil {
		t.Fatalf("expected no matching source, got %+v", v.Source)
	}
	if v.LastReindexRecorded || v.LastReindex != nil {
		t.Fatalf("expected no health row, got %+v", v.LastReindex)
	}
}
