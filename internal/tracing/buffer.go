package tracing

import (
	"encoding/json"
	"sync"
	"time"
)

const defaultCapacity = 200

// Invocation is one proxy tool call (spec §34 / §12.3 traces).
type Invocation struct {
	Time      time.Time `json:"time"`
	ProxyName string    `json:"proxy_tool_name"`
	SourceID  string    `json:"source_id"`
	Tool      string    `json:"upstream_tool"`
	Error     string    `json:"error,omitempty"`
	OK        bool      `json:"ok"`
}

var (
	buf  = make([]Invocation, 0, defaultCapacity)
	capN = defaultCapacity
	mu   sync.RWMutex
)

// SetCapacity changes ring buffer size (call before traffic).
func SetCapacity(n int) {
	if n <= 0 {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	capN = n
	if len(buf) > capN {
		buf = buf[len(buf)-capN:]
	}
}

// AppendInvocation records an invocation for /traces and TUI.
func AppendInvocation(inv Invocation) {
	mu.Lock()
	defer mu.Unlock()
	if len(buf) >= capN {
		buf = buf[1:]
	}
	buf = append(buf, inv)
}

// Snapshot returns newest-first invocations.
func Snapshot() []Invocation {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]Invocation, len(buf))
	copy(out, buf)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// SnapshotJSON returns JSON for HTTP handlers.
func SnapshotJSON() ([]byte, error) {
	return json.MarshalIndent(Snapshot(), "", "  ")
}
