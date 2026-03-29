package config

import (
	"os"
	"testing"

	"lazy-tool/pkg/models"
)

func TestNormalizeSources_empty(t *testing.T) {
	got, err := NormalizeSources(nil)
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v err %v", got, err)
	}
}

func TestNormalizeSources_gatewayHTTP(t *testing.T) {
	got, err := NormalizeSources([]SourceYAML{
		{ID: "g", Type: "gateway", Transport: "http", URL: "http://localhost:8811/mcp"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "g" || got[0].Type != models.SourceTypeGateway {
		t.Fatalf("unexpected %+v", got)
	}
}

func TestNormalizeSources_serverStdio(t *testing.T) {
	got, err := NormalizeSources([]SourceYAML{
		{ID: "s", Type: "server", Transport: "stdio", Command: "npx", Args: []string{"-y", "pkg"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Command != "npx" || len(got[0].Args) != 2 {
		t.Fatalf("unexpected %+v", got)
	}
}

func TestNormalizeSources_duplicateID(t *testing.T) {
	_, err := NormalizeSources([]SourceYAML{
		{ID: "x", Type: "gateway", Transport: "http", URL: "http://a/mcp"},
		{ID: "x", Type: "gateway", Transport: "http", URL: "http://b/mcp"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNormalizeSources_httpMissingURL(t *testing.T) {
	_, err := NormalizeSources([]SourceYAML{
		{ID: "x", Type: "server", Transport: "http"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNormalizeSources_stdioMissingCommand(t *testing.T) {
	_, err := NormalizeSources([]SourceYAML{
		{ID: "x", Type: "server", Transport: "stdio"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNormalizeSources_adapterDefault(t *testing.T) {
	got, err := NormalizeSources([]SourceYAML{
		{ID: "g", Type: "gateway", Transport: "http", URL: "http://localhost/mcp", Adapter: "default"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Adapter != "default" {
		t.Fatalf("got %q", got[0].Adapter)
	}
	got2, err := NormalizeSources([]SourceYAML{
		{ID: "g2", Type: "gateway", Transport: "http", URL: "http://localhost/mcp"},
	})
	if err != nil || got2[0].Adapter != "default" {
		t.Fatalf("unexpected %+v err %v", got2, err)
	}
}

func TestNormalizeSources_adapterUnknown(t *testing.T) {
	_, err := NormalizeSources([]SourceYAML{
		{ID: "g", Type: "gateway", Transport: "http", URL: "http://localhost/mcp", Adapter: "acme"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNormalizeSources_disabled(t *testing.T) {
	got, err := NormalizeSources([]SourceYAML{
		{ID: "on", Type: "gateway", Transport: "http", URL: "http://127.0.0.1:1/mcp"},
		{ID: "off", Type: "gateway", Transport: "http", URL: "http://127.0.0.1:2/mcp", Disabled: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || !got[1].Disabled {
		t.Fatalf("got %+v", got)
	}
}

func TestNormalizeSources_stdioCwd(t *testing.T) {
	dir := t.TempDir()
	got, err := NormalizeSources([]SourceYAML{
		{ID: "s", Type: "server", Transport: "stdio", Command: "echo", Args: []string{"x"}, Cwd: dir},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Cwd != dir {
		t.Fatalf("cwd: got %q", got[0].Cwd)
	}
}

func TestNormalizeSources_stdioCwdBadPath(t *testing.T) {
	_, err := NormalizeSources([]SourceYAML{
		{ID: "s", Type: "server", Transport: "stdio", Command: "echo", Cwd: "/no/such/dir/for/lazy-tool-test"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNormalizeSources_httpRequiresSchemeAndHost(t *testing.T) {
	_, err := NormalizeSources([]SourceYAML{
		{ID: "x", Type: "gateway", Transport: "http", URL: "localhost:8811/mcp"},
	})
	if err == nil {
		t.Fatal("expected error for missing scheme")
	}
}

func TestLoad_rejectsBadSource(t *testing.T) {
	f := t.TempDir() + "/c.yaml"
	if err := os.WriteFile(f, []byte("sources:\n  - id: bad\n    type: gateway\n    transport: http\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(f)
	if err == nil {
		t.Fatal("expected error")
	}
}
