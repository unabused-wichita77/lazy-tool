package storage

import (
	"context"
	"time"
)

func (s *SQLiteStore) ensureFavorites(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS favorites (
	capability_id TEXT PRIMARY KEY,
	created_at INTEGER NOT NULL
)`)
	return err
}

// AddFavorite pins a capability by stable catalog id (P2.3).
func (s *SQLiteStore) AddFavorite(ctx context.Context, capabilityID string) error {
	if err := s.ensureFavorites(ctx); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO favorites(capability_id, created_at) VALUES(?,?)
ON CONFLICT(capability_id) DO UPDATE SET created_at=excluded.created_at
`, capabilityID, time.Now().Unix())
	return err
}

// RemoveFavorite unpins a capability.
func (s *SQLiteStore) RemoveFavorite(ctx context.Context, capabilityID string) error {
	if err := s.ensureFavorites(ctx); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM favorites WHERE capability_id=?`, capabilityID)
	return err
}

// ListFavoriteIDs returns pinned capability ids (unordered).
func (s *SQLiteStore) ListFavoriteIDs(ctx context.Context) ([]string, error) {
	if err := s.ensureFavorites(ctx); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT capability_id FROM favorites`)
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
