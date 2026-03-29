package connectors

import (
	"context"
	"testing"

	"lazy-tool/pkg/models"
)

func TestFactory_HTTPReuseSharesRunner(t *testing.T) {
	f := NewFactory(FactoryOpts{HTTPReuseUpstreamSession: true}).(*factoryImpl)
	defer func() { _ = f.Close() }()
	ctx := context.Background()
	src := models.Source{
		ID: "g", Type: models.SourceTypeGateway, Transport: models.TransportHTTP,
		URL: "http://127.0.0.1:9/mcp",
	}
	if _, err := f.New(ctx, src); err != nil {
		t.Fatal(err)
	}
	if _, err := f.New(ctx, src); err != nil {
		t.Fatal(err)
	}
	if len(f.holders) != 1 {
		t.Fatalf("holders: %d", len(f.holders))
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if f.holders != nil {
		t.Fatalf("holders should clear: %v", f.holders)
	}
}

func TestFactory_CloseNoReuse(t *testing.T) {
	f := NewFactory(FactoryOpts{})
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}
