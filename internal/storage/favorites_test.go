package storage

import (
	"context"
	"path/filepath"
	"testing"
)

func TestFavorites_roundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.db")
	s, err := OpenSQLite(p)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	ctx := context.Background()
	if err := s.AddFavorite(ctx, "cap-1"); err != nil {
		t.Fatal(err)
	}
	ids, err := s.ListFavoriteIDs(ctx)
	if err != nil || len(ids) != 1 || ids[0] != "cap-1" {
		t.Fatalf("got %v err %v", ids, err)
	}
	if err := s.RemoveFavorite(ctx, "cap-1"); err != nil {
		t.Fatal(err)
	}
	ids, err = s.ListFavoriteIDs(ctx)
	if err != nil || len(ids) != 0 {
		t.Fatalf("got %v", ids)
	}
}
