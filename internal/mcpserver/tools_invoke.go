package mcpserver

import (
	"context"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"lazy-tool/internal/metrics"
	"lazy-tool/internal/runtime"
)

type invokeArgs struct {
	ProxyToolName string         `json:"proxy_tool_name" jsonschema:"canonical proxy name from search_tools"`
	Input         map[string]any `json:"input" jsonschema:"arguments for the upstream tool"`
}

type invokeOut struct {
	RawJSON string `json:"raw"`
}

func registerInvokeTool(server *mcp.Server, stack *runtime.Stack, log *slog.Logger) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "invoke_proxy_tool",
		Description: "Call an upstream MCP tool only (search_tools hit with kind=tool). For prompts use get_proxy_prompt; for concrete resources use read_proxy_resource.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in invokeArgs) (*mcp.CallToolResult, invokeOut, error) {
		_, raw, err := ExecuteProxy(ctx, stack, log, in.ProxyToolName, in.Input)
		metrics.McpToolCall("invoke_proxy_tool", err)
		if err != nil {
			return nil, invokeOut{}, err
		}
		return nil, invokeOut{RawJSON: string(raw)}, nil
	})
}
