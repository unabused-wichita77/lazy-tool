package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"lazy-tool/internal/app"
	"lazy-tool/internal/config"
	"lazy-tool/internal/connectors"
	"lazy-tool/internal/embeddings"
	"lazy-tool/internal/search"
	"lazy-tool/internal/storage"
	"lazy-tool/internal/summarizer"
	"lazy-tool/internal/vector"
)

type Stack struct {
	Cfg        *config.Config
	Store      *storage.SQLiteStore
	Vec        *vector.Index
	Registry   *app.SourceRegistry
	Factory    connectors.Factory
	Search     *search.Service
	Summarizer summarizer.Summarizer
	Embedder   embeddings.Embedder
}

func OpenStack(cfgPath string) (*Stack, error) {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, err
	}
	if cfg.Storage.SQLitePath == "" {
		return nil, fmt.Errorf("storage.sqlite_path is required")
	}
	if err := os.MkdirAll(filepath.Dir(cfg.Storage.SQLitePath), 0o755); err != nil {
		return nil, err
	}
	vecPath := cfg.Storage.VectorPath
	if vecPath == "" {
		vecPath = filepath.Join(filepath.Dir(cfg.Storage.SQLitePath), "vector")
	}
	if err := os.MkdirAll(vecPath, 0o755); err != nil {
		return nil, err
	}
	st, err := storage.OpenSQLite(cfg.Storage.SQLitePath)
	if err != nil {
		return nil, fmt.Errorf("sqlite: %w", err)
	}
	vi, err := vector.Open(vecPath)
	if err != nil {
		_ = st.Close()
		return nil, fmt.Errorf("vector: %w", err)
	}
	srcs, err := config.NormalizeSources(cfg.Sources)
	if err != nil {
		_ = vi.Close()
		_ = st.Close()
		return nil, err
	}
	reg, err := app.NewSourceRegistry(srcs)
	if err != nil {
		_ = vi.Close()
		_ = st.Close()
		return nil, err
	}
	fact := connectors.NewFactory(connectors.FactoryOpts{
		HTTPReuseUpstreamSession: cfg.Connectors.HTTPReuseUpstreamSession,
		HTTPReuseIdleTimeout:     time.Duration(cfg.Connectors.HTTPReuseIdleTimeoutSeconds) * time.Second,
	})
	sum := summarizer.New(cfg)
	emb := embeddings.New(cfg)
	svc := search.NewService(st, vi, emb, search.ScoreWeights{
		ExactCanonical:   cfg.Search.Scoring.ExactCanonical,
		ExactName:        cfg.Search.Scoring.ExactName,
		Substring:        cfg.Search.Scoring.Substring,
		VectorMultiplier: cfg.Search.Scoring.VectorMultiplier,
		UserSummary:      cfg.Search.Scoring.UserSummary,
		Favorite:         cfg.Search.Scoring.Favorite,
	}, cfg.Search.LexicalOnly)
	svc.FullCatalogSubstring = !cfg.Search.DisableFullCatalogSubstring
	svc.EmptyQueryIDBatch = cfg.Search.EmptyQueryIDBatch
	svc.EmptyQueryMaxCatalogIDs = cfg.Search.EmptyQueryMaxCatalogIDs
	return &Stack{
		Cfg:        cfg,
		Store:      st,
		Vec:        vi,
		Registry:   reg,
		Factory:    fact,
		Search:     svc,
		Summarizer: sum,
		Embedder:   emb,
	}, nil
}

func (s *Stack) Close() error {
	var first error
	if s.Factory != nil {
		if err := s.Factory.Close(); err != nil {
			first = err
		}
	}
	if s.Vec != nil {
		if err := s.Vec.Close(); err != nil && first == nil {
			first = err
		}
	}
	if s.Store != nil {
		if err := s.Store.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}
