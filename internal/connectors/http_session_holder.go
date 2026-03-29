package connectors

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"lazy-tool/internal/metrics"
	"lazy-tool/pkg/models"
)

// httpSessionRunner reuses one MCP session per upstream HTTP source for sequential calls.
// Not used for stdio (process isolation). Safe for concurrent use across goroutines (per-source mutex).
type httpSessionRunner interface {
	withSession(ctx context.Context, fn func(*mcp.ClientSession) error) error
	close()
}

type httpSessionHolder struct {
	src     models.Source
	hc      *http.Client
	idleTTL time.Duration

	mu           sync.Mutex
	ses          *mcp.ClientSession
	idleDeadline time.Time // next expiry after last success; zero if idleTTL==0 or no session yet
}

func (h *httpSessionHolder) withSession(ctx context.Context, fn func(*mcp.ClientSession) error) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.ses != nil && h.idleTTL > 0 && !h.idleDeadline.IsZero() && time.Now().After(h.idleDeadline) {
		upstreamDebugf("idle timeout recycle source_id=%s", h.src.ID)
		metrics.UpstreamMCPIdleSessionRecycled(h.src.ID)
		metrics.UpstreamMCPSessionClosed(h.src.ID, h.ses.Close())
		h.ses = nil
		h.idleDeadline = time.Time{}
	}

	if h.ses != nil {
		err := fn(h.ses)
		if err != nil {
			metrics.UpstreamMCPSessionClosed(h.src.ID, h.ses.Close())
			h.ses = nil
			h.idleDeadline = time.Time{}
		} else if h.idleTTL > 0 {
			h.idleDeadline = time.Now().Add(h.idleTTL)
		}
		return err
	}

	t, err := transportFor(h.src, h.hc)
	if err != nil {
		return err
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "lazy-tool", Version: "0.1.0"}, nil)
	ses, err := client.Connect(ctx, t, nil)
	metrics.UpstreamMCPConnect(h.src.ID, err)
	if err != nil {
		return err
	}
	h.ses = ses
	upstreamDebugf("session start source_id=%s transport=http reuse=true", h.src.ID)
	err = fn(ses)
	if err != nil {
		metrics.UpstreamMCPSessionClosed(h.src.ID, h.ses.Close())
		h.ses = nil
		h.idleDeadline = time.Time{}
	} else if h.idleTTL > 0 {
		h.idleDeadline = time.Now().Add(h.idleTTL)
	}
	return err
}

func (h *httpSessionHolder) close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.ses != nil {
		metrics.UpstreamMCPSessionClosed(h.src.ID, h.ses.Close())
		h.ses = nil
		h.idleDeadline = time.Time{}
	}
}
