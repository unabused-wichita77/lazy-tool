package connectors

import (
	"context"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"lazy-tool/pkg/models"
)

type baseConnector struct {
	src        models.Source
	httpClient *http.Client
	httpReuse  httpSessionRunner // HTTP session reuse (non-nil only when factory opt-in)
}

func (b *baseConnector) SourceID() string { return b.src.ID }

// Close is a no-op for current transports; Indexer defers it for a future connector-local cleanup hook.
func (b *baseConnector) Close() error { return nil }

func (b *baseConnector) ListTools(ctx context.Context) ([]ToolMeta, error) {
	var meta []ToolMeta
	err := withRetries(ctx, b.src.ID, func() error {
		return withSession(ctx, b.src, b.httpClient, b.httpReuse, func(ses *mcp.ClientSession) error {
			tools, err := listToolsPaginated(ctx, ses)
			if err != nil {
				return err
			}
			meta = toolsToMeta(tools)
			return nil
		})
	})
	return meta, err
}

func (b *baseConnector) ListPrompts(ctx context.Context) ([]PromptMeta, error) {
	var meta []PromptMeta
	err := withRetries(ctx, b.src.ID, func() error {
		return withSession(ctx, b.src, b.httpClient, b.httpReuse, func(ses *mcp.ClientSession) error {
			prompts, err := listPromptsPaginated(ctx, ses)
			if err != nil {
				return err
			}
			meta = promptsToMeta(prompts)
			return nil
		})
	})
	return meta, err
}

func (b *baseConnector) ListResources(ctx context.Context) ([]ResourceMeta, error) {
	var meta []ResourceMeta
	err := withRetries(ctx, b.src.ID, func() error {
		return withSession(ctx, b.src, b.httpClient, b.httpReuse, func(ses *mcp.ClientSession) error {
			res, err := listResourcesPaginated(ctx, ses)
			if err != nil {
				return err
			}
			meta = resourcesToMeta(res)
			return nil
		})
	})
	return meta, err
}

func (b *baseConnector) ListResourceTemplates(ctx context.Context) ([]ResourceTemplateMeta, error) {
	var meta []ResourceTemplateMeta
	err := withRetries(ctx, b.src.ID, func() error {
		return withSession(ctx, b.src, b.httpClient, b.httpReuse, func(ses *mcp.ClientSession) error {
			tpl, err := listResourceTemplatesPaginated(ctx, ses)
			if err != nil {
				return err
			}
			meta = resourceTemplatesToMeta(tpl)
			return nil
		})
	})
	return meta, err
}

func (b *baseConnector) ListForIndex(ctx context.Context) (*IndexSnapshot, error) {
	var snap IndexSnapshot
	err := withRetries(ctx, b.src.ID, func() error {
		return withSession(ctx, b.src, b.httpClient, b.httpReuse, func(ses *mcp.ClientSession) error {
			tools, err := listToolsPaginated(ctx, ses)
			if err != nil {
				return err
			}
			snap.Tools = toolsToMeta(tools)

			prompts, perr := listPromptsPaginated(ctx, ses)
			if perr != nil {
				snap.PromptsErr = perr
			} else {
				snap.Prompts = promptsToMeta(prompts)
			}

			resources, rerr := listResourcesPaginated(ctx, ses)
			if rerr != nil {
				snap.ResourcesErr = rerr
			} else {
				snap.Resources = resourcesToMeta(resources)
			}

			templates, terr := listResourceTemplatesPaginated(ctx, ses)
			if terr != nil {
				snap.ResourceTemplatesErr = terr
			} else {
				snap.ResourceTemplates = resourceTemplatesToMeta(templates)
			}
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return &snap, nil
}

func (b *baseConnector) Health(ctx context.Context) error {
	return withRetries(ctx, b.src.ID, func() error {
		return withSession(ctx, b.src, b.httpClient, b.httpReuse, func(ses *mcp.ClientSession) error {
			_, err := ses.ListTools(ctx, &mcp.ListToolsParams{})
			return err
		})
	})
}

func (b *baseConnector) CallTool(ctx context.Context, toolName string, arguments map[string]any) (*mcp.CallToolResult, error) {
	var res *mcp.CallToolResult
	err := withRetries(ctx, b.src.ID, func() error {
		return withSession(ctx, b.src, b.httpClient, b.httpReuse, func(ses *mcp.ClientSession) error {
			r, err := ses.CallTool(ctx, &mcp.CallToolParams{
				Name:      toolName,
				Arguments: arguments,
			})
			if err != nil {
				return err
			}
			res = r
			return nil
		})
	})
	return res, err
}

func (b *baseConnector) GetPrompt(ctx context.Context, name string, arguments map[string]string) (*mcp.GetPromptResult, error) {
	var res *mcp.GetPromptResult
	err := withRetries(ctx, b.src.ID, func() error {
		return withSession(ctx, b.src, b.httpClient, b.httpReuse, func(ses *mcp.ClientSession) error {
			r, err := ses.GetPrompt(ctx, &mcp.GetPromptParams{
				Name:      name,
				Arguments: arguments,
			})
			if err != nil {
				return err
			}
			res = r
			return nil
		})
	})
	return res, err
}

func (b *baseConnector) ReadResource(ctx context.Context, uri string) (*mcp.ReadResourceResult, error) {
	var res *mcp.ReadResourceResult
	err := withRetries(ctx, b.src.ID, func() error {
		return withSession(ctx, b.src, b.httpClient, b.httpReuse, func(ses *mcp.ClientSession) error {
			r, err := ses.ReadResource(ctx, &mcp.ReadResourceParams{URI: uri})
			if err != nil {
				return err
			}
			res = r
			return nil
		})
	})
	return res, err
}
