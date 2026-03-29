package mcpserver

import (
	"context"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"lazy-tool/internal/runtime"
)

func NewServer(stack *runtime.Stack, log *slog.Logger) *mcp.Server {
	srv := mcp.NewServer(&mcp.Implementation{Name: "lazy-tool", Version: "0.1.0"}, nil)
	registerSearchTools(srv, stack)
	registerInspectCapability(srv, stack)
	registerInvokeTool(srv, stack, log)
	registerGetProxyPrompt(srv, stack, log)
	registerReadProxyResource(srv, stack, log)
	return srv
}

func RunStdio(ctx context.Context, stack *runtime.Stack, log *slog.Logger) error {
	srv := NewServer(stack, log)
	return srv.Run(ctx, &mcp.StdioTransport{})
}
