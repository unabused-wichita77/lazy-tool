package catalog

import (
	"context"

	"lazy-tool/internal/storage"
)

// SetUserSummary stores a manual summary override for one capability (by canonical proxy name).
// Empty text clears the override. Search text is rebuilt for lexical matching.
func SetUserSummary(ctx context.Context, st *storage.SQLiteStore, canonicalName, text string) error {
	rec, err := st.GetByCanonicalName(ctx, canonicalName)
	if err != nil {
		return err
	}
	rec.UserSummary = text
	RefreshSearchText(&rec)
	return st.UpsertCapability(ctx, rec)
}
