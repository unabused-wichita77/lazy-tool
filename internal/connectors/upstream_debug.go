package connectors

import (
	"log"
	"os"
	"strings"
)

func upstreamDebugEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("LAZY_TOOL_UPSTREAM_MCP_DEBUG")))
	return v == "1" || v == "true" || v == "yes"
}

func upstreamDebugf(format string, args ...any) {
	if !upstreamDebugEnabled() {
		return
	}
	log.Printf("[lazy-tool:upstream-mcp] "+format, args...)
}
