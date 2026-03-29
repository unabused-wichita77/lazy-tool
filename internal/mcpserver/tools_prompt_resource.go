package mcpserver

import (
	"context"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"lazy-tool/internal/metrics"
	"lazy-tool/internal/runtime"
)

type getProxyPromptArgs struct {
	ProxyToolName string         `json:"proxy_tool_name" jsonschema:"canonical name from search_tools where kind=prompt"`
	Arguments     map[string]any `json:"arguments,omitempty" jsonschema:"prompt arguments (coerced to strings for upstream prompts/get)"`
}

type readProxyResourceArgs struct {
	ProxyToolName string `json:"proxy_tool_name" jsonschema:"canonical name from search_tools where kind=resource (concrete resource, not template)"`
}

type proxyJSONOut struct {
	RawJSON string `json:"raw"`
}

func registerGetProxyPrompt(server *mcp.Server, stack *runtime.Stack, log *slog.Logger) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_proxy_prompt",
		Description: "Fetch an upstream MCP prompt (prompts/get). Use proxy_tool_name from search_tools hits with kind=prompt.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in getProxyPromptArgs) (*mcp.CallToolResult, proxyJSONOut, error) {
		_, raw, err := ExecuteGetPrompt(ctx, stack, log, in.ProxyToolName, in.Arguments)
		metrics.McpToolCall("get_proxy_prompt", err)
		if err != nil {
			return nil, proxyJSONOut{}, err
		}
		return nil, proxyJSONOut{RawJSON: string(raw)}, nil
	})
}

func registerReadProxyResource(server *mcp.Server, stack *runtime.Stack, log *slog.Logger) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "read_proxy_resource",
		Description: "Read an upstream MCP resource by URI (resources/read). Use search hits with kind=resource from resources/list; resource templates are not supported here.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in readProxyResourceArgs) (*mcp.CallToolResult, proxyJSONOut, error) {
		_, raw, err := ExecuteReadResource(ctx, stack, log, in.ProxyToolName)
		metrics.McpToolCall("read_proxy_resource", err)
		if err != nil {
			return nil, proxyJSONOut{}, err
		}
		return nil, proxyJSONOut{RawJSON: string(raw)}, nil
	})
}
