package vector

import (
	"context"
	"errors"
	"testing"
)

func TestIndex_Close_idempotent_and_query_safe(t *testing.T) {
	v, err := NewInMemory()
	if err != nil {
		t.Fatal(err)
	}
	if err := v.Close(); err != nil {
		t.Fatal(err)
	}
	if err := v.Close(); err != nil {
		t.Fatal(err)
	}
	res, err := v.Query(context.Background(), []float32{1, 0, 0}, 5, "")
	if !errors.Is(err, ErrClosed) {
		t.Fatalf("Query: %v %v", res, err)
	}
	if err := v.Reset(context.Background()); !errors.Is(err, ErrClosed) {
		t.Fatalf("Reset: %v", err)
	}
}

func TestIndex_Query_emptyEmbedding_noError(t *testing.T) {
	v, err := NewInMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = v.Close() }()
	res, err := v.Query(context.Background(), nil, 5, "")
	if err != nil || res != nil {
		t.Fatalf("got res=%v err=%v", res, err)
	}
}
