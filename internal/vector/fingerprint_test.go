package vector

import (
	"testing"
	"time"

	"lazy-tool/pkg/models"
)

func TestEmbeddingFingerprint_changesWithVersion(t *testing.T) {
	base := models.CapabilityRecord{
		ID: "a", Kind: models.CapabilityKindTool, SourceID: "s", SourceType: "gateway",
		CanonicalName: "s__x", OriginalName: "x",
		SearchText: "hello", VersionHash: "v1", LastSeenAt: time.Now(),
		EmbeddingModel: "m", EmbeddingVector: []float32{1, 0, 0},
	}
	fp1 := EmbeddingFingerprint([]models.CapabilityRecord{base}, "m")
	base.VersionHash = "v2"
	fp2 := EmbeddingFingerprint([]models.CapabilityRecord{base}, "m")
	if fp1 == fp2 {
		t.Fatal("fingerprint should change when version_hash changes")
	}
}

func TestEmbeddingFingerprint_emptyEmbeddings(t *testing.T) {
	fp := EmbeddingFingerprint([]models.CapabilityRecord{
		{ID: "a", VersionHash: "v"},
	}, "noop")
	if fp == "" {
		t.Fatal("expected non-empty fingerprint for embedder model only")
	}
}
