package app

import (
	"testing"

	"lazy-tool/internal/config"
)

func TestSourceRegistry_disabledNotRoutable(t *testing.T) {
	srcs, err := config.NormalizeSources([]config.SourceYAML{
		{ID: "a", Type: "gateway", Transport: "http", URL: "http://x/mcp"},
		{ID: "b", Type: "server", Transport: "stdio", Command: "echo", Disabled: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	r, err := NewSourceRegistry(srcs)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := r.Get("b"); ok {
		t.Fatal("disabled source should not be routable")
	}
	if s, ok := r.GetConfigured("b"); !ok || !s.Disabled {
		t.Fatalf("expected disabled entry in registry: %+v", s)
	}
	if len(r.All()) != 1 || r.All()[0].ID != "a" {
		t.Fatalf("All: %+v", r.All())
	}
	if len(r.AllConfigured()) != 2 {
		t.Fatal(len(r.AllConfigured()))
	}
}

func TestSourceRegistry_filter(t *testing.T) {
	srcs, err := config.NormalizeSources([]config.SourceYAML{
		{ID: "a", Type: "gateway", Transport: "http", URL: "http://x/mcp"},
		{ID: "b", Type: "server", Transport: "stdio", Command: "npx"},
	})
	if err != nil {
		t.Fatal(err)
	}
	r, err := NewSourceRegistry(srcs)
	if err != nil {
		t.Fatal(err)
	}
	f := r.Filter([]string{"b"})
	if len(f) != 1 || f[0].ID != "b" {
		t.Fatalf("got %+v", f)
	}
	all := r.Filter(nil)
	if len(all) != 2 {
		t.Fatalf("got %+v", all)
	}
	if _, ok := r.Get("missing"); ok {
		t.Fatal("expected miss")
	}
}
