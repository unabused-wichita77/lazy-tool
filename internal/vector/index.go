package vector

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/philippgille/chromem-go"
	"lazy-tool/pkg/models"
)

const collectionName = "lazy_tool_capabilities"

const fingerprintFile = ".embedding_fingerprint"

type Index struct {
	db   *chromem.DB
	coll *chromem.Collection
	dir  string
}

func Open(path string) (*Index, error) {
	_ = os.MkdirAll(path, 0o755)
	db, err := chromem.NewPersistentDB(path, false)
	if err != nil {
		return nil, err
	}
	coll, err := db.GetOrCreateCollection(collectionName, nil, nil)
	if err != nil {
		return nil, err
	}
	return &Index{db: db, coll: coll, dir: path}, nil
}

// Close drops references to the chromem DB. chromem-go v0.7.0 persists on each write and does not expose DB.Close;
// this is still called on Stack shutdown so we clear state and make Query a no-op after close. Idempotent.
func (v *Index) Close() error {
	if v == nil {
		return nil
	}
	v.db = nil
	v.coll = nil
	return nil
}

func NewInMemory() (*Index, error) {
	db := chromem.NewDB()
	coll, err := db.GetOrCreateCollection(collectionName, nil, nil)
	if err != nil {
		return nil, err
	}
	return &Index{db: db, coll: coll, dir: ""}, nil
}

// EmbeddingFingerprint hashes indexed embedding inputs so we can skip a full chromem reset when nothing changed (P1.2).
func EmbeddingFingerprint(recs []models.CapabilityRecord, embedderModel string) string {
	h := sha256.New()
	_, _ = h.Write([]byte(embedderModel))
	_, _ = h.Write([]byte{0})
	var lines []string
	for _, r := range recs {
		if len(r.EmbeddingVector) == 0 {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s\t%s\t%s", r.ID, r.VersionHash, r.EmbeddingModel))
	}
	sort.Strings(lines)
	_, _ = h.Write([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(h.Sum(nil))
}

func (v *Index) Reset(ctx context.Context) error {
	_ = ctx
	if v.db == nil {
		return ErrClosed
	}
	_ = v.db.DeleteCollection(collectionName)
	coll, err := v.db.GetOrCreateCollection(collectionName, nil, nil)
	if err != nil {
		return err
	}
	v.coll = coll
	return nil
}

func (v *Index) RebuildFromRecords(ctx context.Context, recs []models.CapabilityRecord) error {
	_, err := v.rebuildFromRecords(ctx, recs, "", false)
	return err
}

// RebuildFromRecordsIfUnchanged skips a full reset+rebuild when the embedding fingerprint matches the last run (P1.2).
func (v *Index) RebuildFromRecordsIfUnchanged(ctx context.Context, recs []models.CapabilityRecord, embedderModel string) (skipped bool, err error) {
	return v.rebuildFromRecords(ctx, recs, embedderModel, true)
}

func (v *Index) rebuildFromRecords(ctx context.Context, recs []models.CapabilityRecord, embedderModel string, useFingerprint bool) (skipped bool, err error) {
	fp := EmbeddingFingerprint(recs, embedderModel)
	if useFingerprint && v.dir != "" {
		b, err := os.ReadFile(filepath.Join(v.dir, fingerprintFile))
		if err == nil && string(b) == fp {
			return true, nil
		}
	}
	if err := v.Reset(ctx); err != nil {
		return false, err
	}
	var ids []string
	var embs [][]float32
	var metas []map[string]string
	var contents []string
	for _, rec := range recs {
		if len(rec.EmbeddingVector) == 0 {
			continue
		}
		ids = append(ids, rec.ID)
		embs = append(embs, rec.EmbeddingVector)
		metas = append(metas, map[string]string{"source_id": rec.SourceID, "canonical_name": rec.CanonicalName})
		contents = append(contents, rec.SearchText)
	}
	if len(ids) == 0 {
		if v.dir != "" && useFingerprint {
			_ = os.WriteFile(filepath.Join(v.dir, fingerprintFile), []byte(fp), 0o644)
		}
		return false, nil
	}
	if err := v.coll.Add(ctx, ids, embs, metas, contents); err != nil {
		return false, err
	}
	if v.dir != "" && useFingerprint {
		if err := os.WriteFile(filepath.Join(v.dir, fingerprintFile), []byte(fp), 0o644); err != nil {
			return false, err
		}
	}
	return false, nil
}

func (v *Index) Query(ctx context.Context, embedding []float32, limit int, sourceID string) ([]chromem.Result, error) {
	if len(embedding) == 0 {
		return nil, nil
	}
	if v.coll == nil {
		return nil, ErrClosed
	}
	where := map[string]string{}
	if sourceID != "" {
		where["source_id"] = sourceID
	}
	return v.coll.QueryEmbedding(ctx, embedding, limit, where, nil)
}

func ScoreMap(results []chromem.Result) map[string]float32 {
	m := make(map[string]float32)
	for _, r := range results {
		if r.ID == "" {
			continue
		}
		if s, ok := m[r.ID]; !ok || r.Similarity > s {
			m[r.ID] = r.Similarity
		}
	}
	return m
}
