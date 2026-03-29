package summarizer

import (
	"os"
	"strings"

	"lazy-tool/internal/config"
)

func FromConfig(c *config.Config) Summarizer {
	if !c.Summary.Enabled || strings.EqualFold(c.Summary.Provider, "noop") || c.Summary.Provider == "" {
		return Noop{}
	}
	if strings.Contains(strings.ToLower(c.Summary.Provider), "openai") || c.Summary.Provider == "openai-compatible" {
		key := ""
		if c.Summary.APIKeyEnv != "" {
			key = os.Getenv(c.Summary.APIKeyEnv)
		}
		return &OpenAICompatible{
			BaseURL:   c.Summary.BaseURL,
			APIKey:    key,
			Model:     c.Summary.Model,
			UserAgent: "lazy-tool/0.1",
		}
	}
	return Noop{}
}
