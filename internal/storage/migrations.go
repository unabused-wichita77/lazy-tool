package storage

const migrateUp = `
CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY);
CREATE TABLE IF NOT EXISTS capabilities (
	id TEXT PRIMARY KEY,
	kind TEXT NOT NULL,
	source_id TEXT NOT NULL,
	source_type TEXT NOT NULL,
	canonical_name TEXT NOT NULL UNIQUE,
	original_name TEXT NOT NULL,
	original_description TEXT,
	generated_summary TEXT,
	search_text TEXT,
	input_schema_json TEXT,
	metadata_json TEXT,
	tags_json TEXT,
	embedding_model TEXT,
	embedding_vector BLOB,
	version_hash TEXT NOT NULL,
	last_seen_at INTEGER NOT NULL,
	user_summary TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_capabilities_source ON capabilities(source_id);
CREATE INDEX IF NOT EXISTS idx_capabilities_canonical ON capabilities(canonical_name);
`
