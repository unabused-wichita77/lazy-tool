package embeddings

import (
	"os"
	"strings"

	"lazy-tool/internal/config"
)

func FromConfig(c *config.Config) Embedder {
	p := strings.ToLower(c.Embeddings.Provider)
	switch {
	case p == "" || p == "noop":
		return Noop{}
	case strings.Contains(p, "ollama"):
		return &Ollama{
			BaseURL: c.Embeddings.BaseURL,
			Model:   c.Embeddings.Model,
		}
	case strings.Contains(p, "openai") || p == "openai-compatible":
		key := ""
		if c.Embeddings.APIKeyEnv != "" {
			key = os.Getenv(c.Embeddings.APIKeyEnv)
		}
		return &OpenAICompatible{
			BaseURL:   c.Embeddings.BaseURL,
			APIKey:    key,
			Model:     c.Embeddings.Model,
			UserAgent: "lazy-tool/0.1",
		}
	default:
		return Noop{}
	}
}
