package summarizer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"lazy-tool/pkg/models"
)

type OpenAICompatible struct {
	BaseURL   string
	APIKey    string
	Model     string
	Client    *http.Client
	UserAgent string
}

func (o *OpenAICompatible) Summarize(ctx context.Context, rec models.CapabilityRecord) (string, error) {
	if o.Client == nil {
		o.Client = &http.Client{Timeout: 60 * time.Second}
	}
	base := strings.TrimSuffix(o.BaseURL, "/")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	body := map[string]any{
		"model": o.Model,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "Follow the user instructions exactly. Output plain text only.",
			},
			{
				"role":    "user",
				"content": specSummaryPrompt(rec),
			},
		},
		"temperature": 0.2,
		"max_tokens":  100,
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/chat/completions", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if o.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+o.APIKey)
	}
	if o.UserAgent != "" {
		req.Header.Set("User-Agent", o.UserAgent)
	}
	resp, err := o.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("summary API %s: %s", resp.Status, truncate(string(raw), 200))
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}
	out := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if out == "" {
		return "", fmt.Errorf("empty summary")
	}
	return enforceSummaryRules(out), nil
}

func specSummaryPrompt(rec models.CapabilityRecord) string {
	meta := rec.MetadataJSON
	if meta == "" {
		meta = "{}"
	}
	return strings.TrimSpace(`You summarize MCP capabilities for LLM tool discovery.

Return exactly one sentence.
Target length: 12-30 words.
Must include:
- primary action
- target object
- important inputs only if useful

Avoid:
- hype
- repetition
- examples
- generic filler
- implementation details unless critical

Capability kind: ` + string(rec.Kind) + `
Capability name: ` + rec.OriginalName + `
Original description: ` + rec.OriginalDescription + `
Input schema: ` + rec.InputSchemaJSON + `
Additional metadata: ` + meta)
}

func enforceSummaryRules(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	// First sentence only
	for _, sep := range []string{". ", ".\n", "! ", "? "} {
		if i := strings.Index(s, sep); i >= 0 {
			s = s[:i+1]
			break
		}
	}
	words := strings.Fields(s)
	if len(words) > 30 {
		s = strings.Join(words[:30], " ")
		if !strings.HasSuffix(s, ".") && !strings.HasSuffix(s, "!") && !strings.HasSuffix(s, "?") {
			s += "."
		}
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func APIKeyFromEnv(name string) string {
	if name == "" {
		return ""
	}
	return os.Getenv(name)
}
