package connectors

import (
	"testing"

	"lazy-tool/pkg/models"
)

func TestTransportFor_unsupported(t *testing.T) {
	_, err := transportFor(models.Source{
		ID: "x", Type: models.SourceTypeServer,
		Transport: models.Transport("ws"),
		Command:   "",
	}, nil)
	if err == nil {
		t.Fatal("expected error for unsupported transport")
	}
}

func TestTransportFor_http(t *testing.T) {
	tr, err := transportFor(models.Source{
		ID: "h", Type: models.SourceTypeGateway,
		Transport: models.TransportHTTP,
		URL:       "http://127.0.0.1:9/mcp",
	}, nil)
	if err != nil || tr == nil {
		t.Fatalf("got %v err %v", tr, err)
	}
}
