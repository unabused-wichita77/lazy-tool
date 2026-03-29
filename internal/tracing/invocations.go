package tracing

import (
	"context"
	"log/slog"
	"time"
)

func LogInvocation(ctx context.Context, log *slog.Logger, proxyName, sourceID, tool string, err error) {
	_ = ctx
	if log == nil {
		log = slog.Default()
	}
	inv := Invocation{
		Time:      time.Now(),
		ProxyName: proxyName,
		SourceID:  sourceID,
		Tool:      tool,
		OK:        err == nil,
	}
	if err != nil {
		inv.Error = err.Error()
		log.Info("tool_invoke", "proxy", proxyName, "source", sourceID, "tool", tool, "err", err)
	} else {
		log.Info("tool_invoke", "proxy", proxyName, "source", sourceID, "tool", tool, "ok", true)
	}
	AppendInvocation(inv)
}
