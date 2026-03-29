package catalog

import (
	"strings"
	"testing"
	"time"

	"lazy-tool/internal/connectors"
	"lazy-tool/pkg/models"
)

func TestNormalizeTool_tagsAndSearchText(t *testing.T) {
	src := models.Source{ID: "gh", Type: models.SourceTypeGateway, Transport: models.TransportHTTP}
	meta := connectors.ToolMeta{
		Name:        "create_issue",
		Description: "open issue",
		InputSchema: []byte(`{"type":"object","properties":{"title":{"type":"string"},"body":{"type":"string"}}}`),
	}
	rec := NormalizeTool(src, meta, time.Now())
	if len(rec.Tags) < 2 {
		t.Fatalf("tags: %v", rec.Tags)
	}
	if !strings.Contains(rec.SearchText, "title") || !strings.Contains(rec.SearchText, "gh") {
		t.Fatalf("search text missing expected tokens: %q", rec.SearchText)
	}
}

func TestNormalizePrompt_kindAndCanonical(t *testing.T) {
	src := models.Source{ID: "gw", Type: models.SourceTypeGateway, Transport: models.TransportHTTP}
	meta := connectors.PromptMeta{
		Name:          "review",
		Description:   "Code review prompt",
		ArgumentsJSON: []byte(`[{"name":"path","required":true}]`),
	}
	rec := NormalizePrompt(src, meta, time.Now())
	if rec.Kind != models.CapabilityKindPrompt {
		t.Fatalf("kind %q", rec.Kind)
	}
	if !strings.Contains(rec.CanonicalName, "__p_") || !strings.Contains(rec.CanonicalName, "review") {
		t.Fatalf("canonical: %q", rec.CanonicalName)
	}
}
