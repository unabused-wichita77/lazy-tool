package mcpserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"lazy-tool/internal/runtime"
	"lazy-tool/internal/tracing"
	"lazy-tool/pkg/models"
)

func getCapabilityByCanonicalName(ctx context.Context, stack *runtime.Stack, proxyName string) (models.CapabilityRecord, error) {
	rec, err := stack.Store.GetByCanonicalName(ctx, proxyName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return rec, fmt.Errorf(
				"unknown proxy_tool_name %q; call search_tools first and reuse an exact proxy_tool_name from results",
				proxyName,
			)
		}
		return rec, err
	}
	return rec, nil
}

// ExecuteProxy routes a proxy tool name to the correct upstream MCP server and returns the raw result.
func ExecuteProxy(ctx context.Context, stack *runtime.Stack, log *slog.Logger, proxyName string, input map[string]any) (*mcp.CallToolResult, []byte, error) {
	rec, err := getCapabilityByCanonicalName(ctx, stack, proxyName)
	if err != nil {
		tracing.LogInvocation(ctx, log, proxyName, "", "", err)
		return nil, nil, err
	}
	src, ok := stack.Registry.Get(rec.SourceID)
	if !ok {
		e := fmt.Errorf("unknown source %q", rec.SourceID)
		tracing.LogInvocation(ctx, log, proxyName, rec.SourceID, rec.OriginalName, e)
		return nil, nil, e
	}
	if rec.Kind != models.CapabilityKindTool {
		e := fmt.Errorf("invoke_proxy_tool only supports tools (kind=%s); use search hits with kind=tool", rec.Kind)
		tracing.LogInvocation(ctx, log, proxyName, rec.SourceID, rec.OriginalName, e)
		return nil, nil, e
	}
	if input == nil {
		input = map[string]any{}
	}
	conn, err := stack.Factory.New(ctx, src)
	if err != nil {
		tracing.LogInvocation(ctx, log, proxyName, rec.SourceID, rec.OriginalName, err)
		return nil, nil, err
	}
	defer func() { _ = conn.Close() }()
	res, err := conn.CallTool(ctx, rec.OriginalName, input)
	tracing.LogInvocation(ctx, log, proxyName, rec.SourceID, rec.OriginalName, err)
	if err != nil {
		return nil, nil, err
	}
	raw, _ := json.Marshal(res)
	return res, raw, nil
}

func stringArgumentsFromAny(m map[string]any) map[string]string {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		if v == nil {
			continue
		}
		switch t := v.(type) {
		case string:
			out[k] = t
		default:
			out[k] = fmt.Sprint(t)
		}
	}
	return out
}

// resourceReadURI returns the MCP resource URI for a catalog row from resources/list (not templates).
func resourceReadURI(rec models.CapabilityRecord) (string, error) {
	var meta struct {
		URI              string `json:"uri"`
		ResourceTemplate bool   `json:"resource_template"`
	}
	_ = json.Unmarshal([]byte(rec.MetadataJSON), &meta)
	if meta.ResourceTemplate {
		return "", fmt.Errorf("read_proxy_resource does not support resource templates (canonical %q); use a concrete resource from search", rec.CanonicalName)
	}
	if meta.URI != "" {
		return meta.URI, nil
	}
	if strings.Contains(rec.OriginalName, "://") {
		return rec.OriginalName, nil
	}
	return "", fmt.Errorf("catalog record has no resource URI (canonical %q)", rec.CanonicalName)
}

// ExecuteGetPrompt loads a prompt from the upstream MCP server named in the catalog record.
func ExecuteGetPrompt(ctx context.Context, stack *runtime.Stack, log *slog.Logger, proxyName string, arguments map[string]any) (*mcp.GetPromptResult, []byte, error) {
	rec, err := getCapabilityByCanonicalName(ctx, stack, proxyName)
	if err != nil {
		tracing.LogInvocation(ctx, log, proxyName, "", "", err)
		return nil, nil, err
	}
	src, ok := stack.Registry.Get(rec.SourceID)
	if !ok {
		e := fmt.Errorf("unknown source %q", rec.SourceID)
		tracing.LogInvocation(ctx, log, proxyName, rec.SourceID, rec.OriginalName, e)
		return nil, nil, e
	}
	if rec.Kind != models.CapabilityKindPrompt {
		e := fmt.Errorf("get_proxy_prompt only supports prompts (kind=%s); use search hits with kind=prompt", rec.Kind)
		tracing.LogInvocation(ctx, log, proxyName, rec.SourceID, rec.OriginalName, e)
		return nil, nil, e
	}
	conn, err := stack.Factory.New(ctx, src)
	if err != nil {
		tracing.LogInvocation(ctx, log, proxyName, rec.SourceID, rec.OriginalName, err)
		return nil, nil, err
	}
	defer func() { _ = conn.Close() }()
	strArgs := stringArgumentsFromAny(arguments)
	res, err := conn.GetPrompt(ctx, rec.OriginalName, strArgs)
	tracing.LogInvocation(ctx, log, proxyName, rec.SourceID, rec.OriginalName, err)
	if err != nil {
		return nil, nil, err
	}
	raw, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return res, nil, err
	}
	return res, raw, nil
}

// ExecuteReadResource reads a concrete resource URI from the upstream MCP server.
func ExecuteReadResource(ctx context.Context, stack *runtime.Stack, log *slog.Logger, proxyName string) (*mcp.ReadResourceResult, []byte, error) {
	rec, err := stack.Store.GetByCanonicalName(ctx, proxyName)
	if err != nil {
		tracing.LogInvocation(ctx, log, proxyName, "", "", err)
		return nil, nil, err
	}
	src, ok := stack.Registry.Get(rec.SourceID)
	if !ok {
		e := fmt.Errorf("unknown source %q", rec.SourceID)
		tracing.LogInvocation(ctx, log, proxyName, rec.SourceID, rec.OriginalName, e)
		return nil, nil, e
	}
	if rec.Kind != models.CapabilityKindResource {
		e := fmt.Errorf("read_proxy_resource only supports resources (kind=%s); use search hits with kind=resource (not templates)", rec.Kind)
		tracing.LogInvocation(ctx, log, proxyName, rec.SourceID, rec.OriginalName, e)
		return nil, nil, e
	}
	uri, err := resourceReadURI(rec)
	if err != nil {
		tracing.LogInvocation(ctx, log, proxyName, rec.SourceID, rec.OriginalName, err)
		return nil, nil, err
	}
	conn, err := stack.Factory.New(ctx, src)
	if err != nil {
		tracing.LogInvocation(ctx, log, proxyName, rec.SourceID, rec.OriginalName, err)
		return nil, nil, err
	}
	defer func() { _ = conn.Close() }()
	res, err := conn.ReadResource(ctx, uri)
	tracing.LogInvocation(ctx, log, proxyName, rec.SourceID, uri, err)
	if err != nil {
		return nil, nil, err
	}
	raw, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return res, nil, err
	}
	return res, raw, nil
}
