package storage

import (
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"lazy-tool/pkg/models"
	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func OpenSQLite(path string) (*SQLiteStore, error) {
	dsn := path
	if !strings.Contains(dsn, "?") {
		dsn += "?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	} else {
		dsn += "&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	s := &SQLiteStore{db: db}
	if _, err := s.db.Exec(migrateUp); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	if _, err := s.db.Exec(`ALTER TABLE capabilities ADD COLUMN user_summary TEXT NOT NULL DEFAULT ''`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			_ = db.Close()
			return nil, fmt.Errorf("migrate user_summary: %w", err)
		}
	}
	ctx := context.Background()
	if err := s.ensureFTS(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("fts: %w", err)
	}
	return s, nil
}

func (s *SQLiteStore) Close() error { return s.db.Close() }

func encodeFloat32(v []float32) []byte {
	if len(v) == 0 {
		return nil
	}
	b := make([]byte, 4*len(v))
	for i, f := range v {
		u := math.Float32bits(f)
		binary.LittleEndian.PutUint32(b[i*4:], u)
	}
	return b
}

func decodeFloat32(b []byte) []float32 {
	if len(b)%4 != 0 {
		return nil
	}
	n := len(b) / 4
	out := make([]float32, n)
	for i := 0; i < n; i++ {
		u := binary.LittleEndian.Uint32(b[i*4:])
		out[i] = math.Float32frombits(u)
	}
	return out
}

func (s *SQLiteStore) UpsertCapability(ctx context.Context, rec models.CapabilityRecord) error {
	tags, _ := json.Marshal(rec.Tags)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	_, err = tx.ExecContext(ctx, `
INSERT INTO capabilities (
	id, kind, source_id, source_type, canonical_name, original_name, original_description,
	generated_summary, user_summary, search_text, input_schema_json, metadata_json, tags_json,
	embedding_model, embedding_vector, version_hash, last_seen_at
) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
	kind=excluded.kind,
	source_id=excluded.source_id,
	source_type=excluded.source_type,
	canonical_name=excluded.canonical_name,
	original_name=excluded.original_name,
	original_description=excluded.original_description,
	generated_summary=excluded.generated_summary,
	user_summary=excluded.user_summary,
	search_text=excluded.search_text,
	input_schema_json=excluded.input_schema_json,
	metadata_json=excluded.metadata_json,
	tags_json=excluded.tags_json,
	embedding_model=excluded.embedding_model,
	embedding_vector=excluded.embedding_vector,
	version_hash=excluded.version_hash,
	last_seen_at=excluded.last_seen_at
`, rec.ID, rec.Kind, rec.SourceID, rec.SourceType, rec.CanonicalName, rec.OriginalName,
		rec.OriginalDescription, rec.GeneratedSummary, rec.UserSummary, rec.SearchText, rec.InputSchemaJSON,
		rec.MetadataJSON, string(tags), rec.EmbeddingModel, encodeFloat32(rec.EmbeddingVector),
		rec.VersionHash, rec.LastSeenAt.Unix())
	if err != nil {
		return err
	}
	if err := syncFTSRow(ctx, tx, rec); err != nil {
		return err
	}
	return tx.Commit()
}

func scanRec(row *sql.Row) (models.CapabilityRecord, error) {
	var rec models.CapabilityRecord
	var tagsJSON string
	var last int64
	var emb []byte
	err := row.Scan(
		&rec.ID, &rec.Kind, &rec.SourceID, &rec.SourceType, &rec.CanonicalName,
		&rec.OriginalName, &rec.OriginalDescription, &rec.GeneratedSummary, &rec.UserSummary, &rec.SearchText,
		&rec.InputSchemaJSON, &rec.MetadataJSON, &tagsJSON, &rec.EmbeddingModel, &emb,
		&rec.VersionHash, &last,
	)
	if err != nil {
		return rec, err
	}
	_ = json.Unmarshal([]byte(tagsJSON), &rec.Tags)
	rec.EmbeddingVector = decodeFloat32(emb)
	rec.LastSeenAt = time.Unix(last, 0)
	return rec, nil
}

func (s *SQLiteStore) GetCapability(ctx context.Context, id string) (models.CapabilityRecord, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, kind, source_id, source_type, canonical_name, original_name, original_description,
	generated_summary, user_summary, search_text, input_schema_json, metadata_json, tags_json,
	embedding_model, embedding_vector, version_hash, last_seen_at
FROM capabilities WHERE id=?`, id)
	return scanRec(row)
}

func (s *SQLiteStore) GetByCanonicalName(ctx context.Context, canonical string) (models.CapabilityRecord, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, kind, source_id, source_type, canonical_name, original_name, original_description,
	generated_summary, user_summary, search_text, input_schema_json, metadata_json, tags_json,
	embedding_model, embedding_vector, version_hash, last_seen_at
FROM capabilities WHERE canonical_name=?`, canonical)
	return scanRec(row)
}

func (s *SQLiteStore) listRows(ctx context.Context, query string, arg any) ([]models.CapabilityRecord, error) {
	var rows *sql.Rows
	var err error
	if arg == nil {
		rows, err = s.db.QueryContext(ctx, query)
	} else {
		rows, err = s.db.QueryContext(ctx, query, arg)
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []models.CapabilityRecord
	for rows.Next() {
		var rec models.CapabilityRecord
		var tagsJSON string
		var last int64
		var emb []byte
		if err := rows.Scan(
			&rec.ID, &rec.Kind, &rec.SourceID, &rec.SourceType, &rec.CanonicalName,
			&rec.OriginalName, &rec.OriginalDescription, &rec.GeneratedSummary, &rec.UserSummary, &rec.SearchText,
			&rec.InputSchemaJSON, &rec.MetadataJSON, &tagsJSON, &rec.EmbeddingModel, &emb,
			&rec.VersionHash, &last,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(tagsJSON), &rec.Tags)
		rec.EmbeddingVector = decodeFloat32(emb)
		rec.LastSeenAt = time.Unix(last, 0)
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) ListBySource(ctx context.Context, sourceID string) ([]models.CapabilityRecord, error) {
	return s.listRows(ctx, `
SELECT id, kind, source_id, source_type, canonical_name, original_name, original_description,
	generated_summary, user_summary, search_text, input_schema_json, metadata_json, tags_json,
	embedding_model, embedding_vector, version_hash, last_seen_at
FROM capabilities WHERE source_id=? ORDER BY canonical_name`, sourceID)
}

func (s *SQLiteStore) ListAll(ctx context.Context) ([]models.CapabilityRecord, error) {
	return s.listRows(ctx, `
SELECT id, kind, source_id, source_type, canonical_name, original_name, original_description,
	generated_summary, user_summary, search_text, input_schema_json, metadata_json, tags_json,
	embedding_model, embedding_vector, version_hash, last_seen_at
FROM capabilities ORDER BY source_id, canonical_name`, nil)
}

// ListAllIDs returns capability ids in the same order as ListAll (source_id, canonical_name).
func (s *SQLiteStore) ListAllIDs(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM capabilities ORDER BY source_id, canonical_name`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// likeSubstringPattern builds a SQL LIKE pattern that matches needle as a literal substring
// (escapes %, _, and \ for use with ESCAPE '\').
func likeSubstringPattern(needle string) string {
	var b strings.Builder
	b.WriteByte('%')
	for _, r := range needle {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '%':
			b.WriteString(`\%`)
		case '_':
			b.WriteString(`\_`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('%')
	return b.String()
}

// ListIDsBySearchTextSubstring returns capability IDs whose search_text contains needle as a
// literal substring. Callers should pass the same lowercased needle the search service uses;
// stored search_text is built lowercased by the catalog.
func (s *SQLiteStore) ListIDsBySearchTextSubstring(ctx context.Context, needle string, sourceIDs []string) ([]string, error) {
	if needle == "" {
		return nil, nil
	}
	pat := likeSubstringPattern(needle)
	var q string
	var args []any
	if len(sourceIDs) == 0 {
		q = `SELECT id FROM capabilities WHERE search_text LIKE ? ESCAPE '\' ORDER BY source_id, canonical_name`
		args = []any{pat}
	} else {
		ph := placeholders(len(sourceIDs))
		q = `SELECT id FROM capabilities WHERE search_text LIKE ? ESCAPE '\' AND source_id IN (` + ph + `) ORDER BY source_id, canonical_name`
		args = append([]any{pat}, anySlice(sourceIDs)...)
	}
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) DeleteStale(ctx context.Context, sourceID string, keep map[string]struct{}) (int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM capabilities WHERE source_id=?`, sourceID)
	if err != nil {
		return 0, err
	}
	var toDelete []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return 0, err
		}
		if _, ok := keep[id]; !ok {
			toDelete = append(toDelete, id)
		}
	}
	_ = rows.Close()
	if len(toDelete) == 0 {
		return 0, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	for _, id := range toDelete {
		if err := deleteFTSRow(ctx, tx, id); err != nil {
			return 0, err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM capabilities WHERE id=?`, id); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return len(toDelete), nil
}

// DeleteAllCapabilitiesForSource removes every catalog row for source_id (e.g. source disabled in config).
func (s *SQLiteStore) DeleteAllCapabilitiesForSource(ctx context.Context, sourceID string) (int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM capabilities WHERE source_id=?`, sourceID)
	if err != nil {
		return 0, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return 0, err
		}
		ids = append(ids, id)
	}
	_ = rows.Close()
	if len(ids) == 0 {
		return 0, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	for _, id := range ids {
		if err := deleteFTSRow(ctx, tx, id); err != nil {
			return 0, err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM capabilities WHERE id=?`, id); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return len(ids), nil
}
