package tunnel

import (
	"sync"
	"time"
)

type RuntimeEvent struct {
	Time   string `json:"time"`
	Role   string `json:"role"`
	Type   string `json:"type"`
	Detail string `json:"detail"`
}

var (
	eventsMu     sync.Mutex
	eventHistory []RuntimeEvent
)

const maxRuntimeEvents = 500

func recordEvent(role, eventType, detail string) {
	eventsMu.Lock()
	defer eventsMu.Unlock()
	eventHistory = append(eventHistory, RuntimeEvent{
		Time:   time.Now().UTC().Format(time.RFC3339),
		Role:   role,
		Type:   eventType,
		Detail: detail,
	})
	if len(eventHistory) > maxRuntimeEvents {
		eventHistory = append([]RuntimeEvent(nil), eventHistory[len(eventHistory)-maxRuntimeEvents:]...)
	}
}

func GetRecentEvents(limit int) []RuntimeEvent {
	eventsMu.Lock()
	defer eventsMu.Unlock()
	if limit <= 0 || limit > len(eventHistory) {
		limit = len(eventHistory)
	}
	start := len(eventHistory) - limit
	out := make([]RuntimeEvent, limit)
	copy(out, eventHistory[start:])
	return out
}
