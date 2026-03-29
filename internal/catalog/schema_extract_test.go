package catalog

import (
	"slices"
	"testing"
)

func TestSchemaArgNames_properties(t *testing.T) {
	schema := `{"type":"object","properties":{"owner":{"type":"string"},"repo":{"type":"string"}}}`
	got := SchemaArgNames(schema)
	want := []string{"owner", "repo"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestSchemaArgNames_nested(t *testing.T) {
	schema := `{"properties":{"opts":{"type":"object","properties":{"dry_run":{"type":"boolean"}}}}}`
	got := SchemaArgNames(schema)
	if !slices.Contains(got, "opts") || !slices.Contains(got, "dry_run") {
		t.Fatalf("got %v", got)
	}
}
