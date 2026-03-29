package mcpserver

import (
	"encoding/json"
	"testing"

	"lazy-tool/internal/config"
	"lazy-tool/pkg/models"
)

func TestAnthropicHint_tool(t *testing.T) {
	b := anthropicHint(string(models.CapabilityKindTool), "src__echo")
	if b == nil || b.SuggestedToolUse == nil {
		t.Fatal("expected tool_use block")
	}
	if b.SuggestedToolUse.Name != "invoke_proxy_tool" {
		t.Fatal(b.SuggestedToolUse.Name)
	}
}

func TestAnthropicHint_promptResource(t *testing.T) {
	p := anthropicHint(string(models.CapabilityKindPrompt), "src__p_x")
	if p == nil || p.SuggestedToolUse.Name != "get_proxy_prompt" {
		t.Fatalf("prompt: %+v", p)
	}
	r := anthropicHint(string(models.CapabilityKindResource), "src__r_y")
	if r == nil || r.SuggestedToolUse.Name != "read_proxy_resource" {
		t.Fatalf("resource: %+v", r)
	}
}

func TestApplySearchAliases(t *testing.T) {
	cfg := &config.Config{}
	cfg.Search.Aliases = map[string]string{
		"short": "longer query text",
	}
	if got := applySearchAliases(cfg, "  short  "); got != "longer query text" {
		t.Fatalf("got %q", got)
	}
	if got := applySearchAliases(cfg, "unmapped"); got != "unmapped" {
		t.Fatalf("got %q", got)
	}
	if got := applySearchAliases(nil, "x"); got != "x" {
		t.Fatalf("got %q", got)
	}
}

func TestSearchToolsOut_JSONStableShape(t *testing.T) {
	out := searchToolsOut{Results: []searchHit{{
		Kind:          "tool",
		ProxyToolName: "src__echo",
		SourceID:      "src",
		Summary:       "Echo",
		Score:         0.99,
		WhyMatched:    []string{"exact:canonical"},
		NextToolName:  "invoke_proxy_tool",
		NextStep:      "Call invoke_proxy_tool",
	}}}
	b, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["results"]; !ok {
		t.Fatal("missing results")
	}
	var arr []map[string]json.RawMessage
	if err := json.Unmarshal(raw["results"], &arr); err != nil || len(arr) != 1 {
		t.Fatal(err)
	}
	row := arr[0]
	for _, k := range []string{"proxy_tool_name", "source_id", "summary", "score", "why_matched"} {
		if _, ok := row[k]; !ok {
			t.Fatalf("missing key %q", k)
		}
	}
}
