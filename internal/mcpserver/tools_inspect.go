package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"lazy-tool/internal/catalog"
	"lazy-tool/internal/metrics"
	"lazy-tool/internal/runtime"
)

type inspectCapabilityArgs struct {
	CanonicalName string `json:"canonical_name" jsonschema:"exact proxy_tool_name from search_tools"`
}

func registerInspectCapability(server *mcp.Server, stack *runtime.Stack) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "inspect_capability",
		Description: "Return catalog record, configured source, and last reindex health for one capability. Use the exact proxy_tool_name (canonical name) from search_tools.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in inspectCapabilityArgs) (*mcp.CallToolResult, catalog.InspectView, error) {
		name := strings.TrimSpace(in.CanonicalName)
		if name == "" {
			err := fmt.Errorf("canonical_name is required (use proxy_tool_name from search_tools)")
			metrics.McpToolCall("inspect_capability", err)
			return nil, catalog.InspectView{}, err
		}
		v, err := catalog.BuildInspectView(ctx, stack.Store, stack.Registry, name)
		metrics.McpToolCall("inspect_capability", err)
		if err != nil {
			return nil, catalog.InspectView{}, err
		}
		return nil, v, nil
	})
}

// InspectCapabilityJSON is for CLI/tests; same payload as inspect_capability tool.
func InspectCapabilityJSON(ctx context.Context, stack *runtime.Stack, canonicalName string) ([]byte, error) {
	v, err := catalog.BuildInspectView(ctx, stack.Store, stack.Registry, strings.TrimSpace(canonicalName))
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(v, "", "  ")
}
