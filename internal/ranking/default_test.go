package ranking

import (
	"context"
	"testing"

	"lazy-tool/pkg/models"
)

func TestDefault_NormalizeScores(t *testing.T) {
	var d Default
	in := []models.SearchResult{
		{ProxyToolName: "a", Score: 10},
		{ProxyToolName: "b", Score: 20},
	}
	out, err := d.Rank(context.Background(), models.SearchQuery{Limit: 10}, in)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Results) != 2 {
		t.Fatal(len(out.Results))
	}
	if out.Results[0].Score < out.Results[1].Score {
		t.Fatal("order")
	}
	if out.Results[0].Score > 1.01 || out.Results[0].Score < 0.99 {
		t.Fatalf("top score want ~1 got %v", out.Results[0].Score)
	}
}
