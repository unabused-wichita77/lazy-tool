package main

import (
	"context"
	"errors"
	"log/slog"

	"github.com/spf13/cobra"

	"lazy-tool/internal/catalog"
	"lazy-tool/internal/runtime"
)

func newReindexCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reindex",
		Short: "Fetch upstream tools and rebuild local catalog + vector index",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := resolveConfigPath()
			if path == "" {
				return errors.New("config path required: use --config or LAZY_TOOL_CONFIG")
			}
			stack, err := runtime.OpenStack(path)
			if err != nil {
				return err
			}
			defer func() { _ = stack.Close() }()
			ix := &catalog.Indexer{
				Registry: stack.Registry,
				Factory:  stack.Factory,
				Summary:  stack.Summarizer,
				Embed:    stack.Embedder,
				Store:    stack.Store,
				Vec:      stack.Vec,
				Log:      slog.Default(),
			}
			return ix.Run(context.Background())
		},
	}
}
