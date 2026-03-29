package summarizer

import (
	"strings"
	"testing"
	"time"

	"lazy-tool/pkg/models"
)

func TestSpecSummaryPrompt_containsSpecSections(t *testing.T) {
	rec := models.CapabilityRecord{
		Kind:                models.CapabilityKindTool,
		OriginalName:        "ping",
		OriginalDescription: "pong",
		InputSchemaJSON:     `{"type":"object"}`,
		MetadataJSON:        `{"k":"v"}`,
		LastSeenAt:          time.Now(),
	}
	p := specSummaryPrompt(rec)
	for _, frag := range []string{
		"You summarize MCP capabilities",
		"12-30 words",
		"Capability kind:",
		"Capability name: ping",
		"Original description:",
		"Input schema:",
		"Additional metadata:",
	} {
		if !strings.Contains(p, frag) {
			t.Fatalf("missing %q in prompt", frag)
		}
	}
}

func TestEnforceSummaryRules_truncatesWords(t *testing.T) {
	var words []string
	for i := 0; i < 40; i++ {
		words = append(words, "w")
	}
	s := strings.Join(words, " ") + "."
	out := enforceSummaryRules(s)
	fs := strings.Fields(out)
	if len(fs) > 31 {
		t.Fatalf("len %d", len(fs))
	}
}
