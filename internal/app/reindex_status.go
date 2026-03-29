package app

import (
	"sync"
	"time"
)

// ReindexStatus holds the last completed reindex for Web/TUI (spec §12.3 reindex visibility).
var ReindexStatus struct {
	mu      sync.RWMutex
	At      time.Time
	OK      bool
	Message string
}

func SetReindexStatus(ok bool, message string) {
	ReindexStatus.mu.Lock()
	defer ReindexStatus.mu.Unlock()
	ReindexStatus.At = time.Now()
	ReindexStatus.OK = ok
	ReindexStatus.Message = message
}

func GetReindexStatus() (at time.Time, ok bool, message string) {
	ReindexStatus.mu.RLock()
	defer ReindexStatus.mu.RUnlock()
	return ReindexStatus.At, ReindexStatus.OK, ReindexStatus.Message
}
