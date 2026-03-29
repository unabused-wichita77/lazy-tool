package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Ollama struct {
	BaseURL string
	Model   string
	Client  *http.Client
}

func (o *Ollama) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if o.Client == nil {
		o.Client = &http.Client{Timeout: 120 * time.Second}
	}
	base := strings.TrimSuffix(o.BaseURL, "/")
	if base == "" {
		base = "http://127.0.0.1:11434"
	}
	out := make([][]float32, len(texts))
	for i, text := range texts {
		body, _ := json.Marshal(map[string]any{"model": o.Model, "prompt": text})
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/embeddings", bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := o.Client.Do(req)
		if err != nil {
			return nil, err
		}
		defer func() { _ = resp.Body.Close() }()
		raw, _ := io.ReadAll(resp.Body)
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("ollama embeddings %s: %s", resp.Status, shortMsg(string(raw), 120))
		}
		var parsed struct {
			Embedding []float64 `json:"embedding"`
		}
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return nil, err
		}
		vec := make([]float32, len(parsed.Embedding))
		for j, v := range parsed.Embedding {
			vec[j] = float32(v)
		}
		out[i] = vec
	}
	return out, nil
}

func (o *Ollama) ModelName() string { return o.Model }
