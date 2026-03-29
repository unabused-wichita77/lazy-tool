package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// SourceHealthRow is last-known reindex outcome per configured source (P1.4).
type SourceHealthRow struct {
	SourceID  string
	OK        bool
	Message   string
	UpdatedAt time.Time
}

func (s *SQLiteStore) ensureSourceHealth(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS source_health (
	source_id TEXT PRIMARY KEY,
	ok INTEGER NOT NULL,
	message TEXT NOT NULL,
	updated_at INTEGER NOT NULL
)`)
	return err
}

// UpsertSourceHealth records the outcome of indexing one source (empty message when ok).
func (s *SQLiteStore) UpsertSourceHealth(ctx context.Context, sourceID string, ok bool, message string) error {
	if err := s.ensureSourceHealth(ctx); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO source_health(source_id, ok, message, updated_at) VALUES(?,?,?,?)
ON CONFLICT(source_id) DO UPDATE SET
	ok=excluded.ok,
	message=excluded.message,
	updated_at=excluded.updated_at
`, sourceID, boolAsInt(ok), message, time.Now().Unix())
	return err
}

func boolAsInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// GetSourceHealth returns the last-known reindex row for sourceID, if any.
func (s *SQLiteStore) GetSourceHealth(ctx context.Context, sourceID string) (SourceHealthRow, bool, error) {
	if err := s.ensureSourceHealth(ctx); err != nil {
		return SourceHealthRow{}, false, err
	}
	var r SourceHealthRow
	var ts int64
	var okInt int
	err := s.db.QueryRowContext(ctx, `
SELECT source_id, ok, message, updated_at FROM source_health WHERE source_id = ?`, sourceID).Scan(&r.SourceID, &okInt, &r.Message, &ts)
	if errors.Is(err, sql.ErrNoRows) {
		return SourceHealthRow{}, false, nil
	}
	if err != nil {
		return SourceHealthRow{}, false, err
	}
	r.OK = okInt != 0
	r.UpdatedAt = time.Unix(ts, 0).UTC()
	return r, true, nil
}

// ListSourceHealth returns persisted rows (may be empty before first reindex).
func (s *SQLiteStore) ListSourceHealth(ctx context.Context) ([]SourceHealthRow, error) {
	if err := s.ensureSourceHealth(ctx); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT source_id, ok, message, updated_at FROM source_health ORDER BY source_id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []SourceHealthRow
	for rows.Next() {
		var r SourceHealthRow
		var ts int64
		var okInt int
		if err := rows.Scan(&r.SourceID, &okInt, &r.Message, &ts); err != nil {
			return nil, err
		}
		r.OK = okInt != 0
		r.UpdatedAt = time.Unix(ts, 0).UTC()
		out = append(out, r)
	}
	return out, rows.Err()
}
