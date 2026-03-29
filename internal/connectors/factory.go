package connectors

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"lazy-tool/pkg/models"
)

type FactoryOpts struct {
	Timeout time.Duration
	// HTTPReuseUpstreamSession keeps one MCP session per source id for streamable HTTP transports (stdio unchanged).
	HTTPReuseUpstreamSession bool
	// HTTPReuseIdleTimeout closes the session after this duration elapses since the last successful request (sliding window). Zero disables.
	HTTPReuseIdleTimeout time.Duration
}

func NewFactory(opts FactoryOpts) Factory {
	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Second
	}
	return &factoryImpl{opts: opts}
}

type factoryImpl struct {
	opts      FactoryOpts
	holdersMu sync.Mutex
	holders   map[string]*httpSessionHolder
}

func (f *factoryImpl) New(ctx context.Context, src models.Source) (Connector, error) {
	_ = ctx
	if src.Adapter != "" && src.Adapter != "default" {
		return nil, fmt.Errorf("unsupported source adapter %q", src.Adapter)
	}
	hc := &http.Client{Timeout: f.opts.Timeout}
	var reuse httpSessionRunner
	if f.opts.HTTPReuseUpstreamSession && src.Transport == models.TransportHTTP {
		reuse = f.runnerFor(src, hc)
	}
	bc := baseConnector{src: src, httpClient: hc, httpReuse: reuse}
	switch src.Type {
	case models.SourceTypeGateway:
		return &GatewayConnector{baseConnector: bc}, nil
	case models.SourceTypeServer:
		return &ServerConnector{baseConnector: bc}, nil
	default:
		return nil, fmt.Errorf("unknown source type %q", src.Type)
	}
}

func (f *factoryImpl) runnerFor(src models.Source, hc *http.Client) httpSessionRunner {
	f.holdersMu.Lock()
	defer f.holdersMu.Unlock()
	if f.holders == nil {
		f.holders = make(map[string]*httpSessionHolder)
	}
	if h, ok := f.holders[src.ID]; ok {
		return h
	}
	h := &httpSessionHolder{src: src, hc: hc, idleTTL: f.opts.HTTPReuseIdleTimeout}
	f.holders[src.ID] = h
	return h
}

func (f *factoryImpl) Close() error {
	f.holdersMu.Lock()
	defer f.holdersMu.Unlock()
	for _, h := range f.holders {
		h.close()
	}
	f.holders = nil
	return nil
}
