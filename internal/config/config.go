package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the top-level application configuration (subset grows in later phases).
type Config struct {
	App struct {
		Name        string `yaml:"name"`
		Environment string `yaml:"environment"`
	} `yaml:"app"`
	Server struct {
		MCP struct {
			Transport string `yaml:"transport"`
			Host      string `yaml:"host"`
			Port      int    `yaml:"port"`
		} `yaml:"mcp"`
	} `yaml:"server"`
	Storage struct {
		SQLitePath  string `yaml:"sqlite_path"`
		VectorPath  string `yaml:"vector_path"`
		HistoryPath string `yaml:"history_path"` // optional: append-only search query log (P2.3)
	} `yaml:"storage"`
	Summary struct {
		Provider  string `yaml:"provider"`
		Model     string `yaml:"model"`
		Enabled   bool   `yaml:"enabled"`
		BaseURL   string `yaml:"base_url"`
		APIKeyEnv string `yaml:"api_key_env"`
	} `yaml:"summary"`
	Embeddings struct {
		Provider  string `yaml:"provider"`
		Model     string `yaml:"model"`
		BaseURL   string `yaml:"base_url"`
		APIKeyEnv string `yaml:"api_key_env"`
	} `yaml:"embeddings"`
	Sources []SourceYAML `yaml:"sources"`
	// Connectors controls upstream MCP client behavior (HTTP session reuse, etc.).
	Connectors struct {
		// HTTPReuseUpstreamSession keeps one MCP session per HTTP source between connector calls until error or shutdown.
		// Stdio sources always spawn a new process per call.
		HTTPReuseUpstreamSession bool `yaml:"http_reuse_upstream_session"`
		// HTTPReuseIdleTimeoutSeconds closes a reused HTTP MCP session after this many seconds with no successful handler completion.
		// 0 disables idle close (session lives until error or Factory.Close). Only applies when HTTPReuseUpstreamSession is true.
		HTTPReuseIdleTimeoutSeconds int `yaml:"http_reuse_idle_timeout_seconds"`
	} `yaml:"connectors"`
	Search struct {
		AnthropicToolRefs bool `yaml:"anthropic_tool_refs"`
		// LexicalOnly disables embedding + vector retrieval for all searches (P2.4).
		LexicalOnly bool `yaml:"lexical_only"`
		// DisableFullCatalogSubstring skips the SQL substring fallback on search_text when FTS returned no hits (part-3 degraded mode).
		// Faster; may miss hits only reachable via substring when BM25 returns zero rows.
		DisableFullCatalogSubstring bool `yaml:"disable_full_catalog_substring"`
		// EmptyQueryIDBatch is the SQLite batch size for '' query catalog walks (0 = default 128).
		EmptyQueryIDBatch int `yaml:"empty_query_id_batch"`
		// EmptyQueryMaxCatalogIDs caps how many capability IDs are considered for '' query (0 = no cap). Truncates in stable ListAll order.
		EmptyQueryMaxCatalogIDs int `yaml:"empty_query_max_catalog_ids"`
		// Aliases map exact query strings to a replacement query (e.g. shortcut -> canonical search) (P2.3).
		Aliases map[string]string `yaml:"aliases"`
		Scoring struct {
			ExactCanonical   float64 `yaml:"exact_canonical"`
			ExactName        float64 `yaml:"exact_name"`
			Substring        float64 `yaml:"substring"`
			VectorMultiplier float64 `yaml:"vector_multiplier"`
			UserSummary      float64 `yaml:"user_summary"`
			Favorite         float64 `yaml:"favorite"`
		} `yaml:"scoring"`
	} `yaml:"search"`
}

// Load reads and parses a YAML config file from path.
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}
	if _, err := NormalizeSources(c.Sources); err != nil {
		return nil, err
	}
	return &c, nil
}
