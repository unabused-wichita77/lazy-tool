package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"

	"github.com/spf13/cobra"

	"lazy-tool/internal/mcpserver"
	"lazy-tool/internal/runtime"
)

func newServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Run lazy-tool as an MCP server (stdio) exposing search_tools",
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
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()
			err = mcpserver.RunStdio(ctx, stack, slog.Default())
			if err != nil {
				if errors.Is(err, io.EOF) || strings.Contains(err.Error(), "server is closing: EOF") {
					return nil
				}
			}
			return err
		},
	}
}
