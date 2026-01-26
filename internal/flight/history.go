package flight

import (
	"sync"
)

type RequestHistory struct {
	mu    sync.Mutex
	items []string
	limit int
}

func NewRequestHistory(limit int) *RequestHistory {
	return &RequestHistory{
		items: make([]string, 0, limit),
		limit: limit,
	}
}

func (h *RequestHistory) Add(url string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	// Deduplicate: remove if exists
	for i, item := range h.items {
		if item == url {
			h.items = append(h.items[:i], h.items[i+1:]...)
			break
		}
	}
	h.items = append(h.items, url)
	if len(h.items) > h.limit {
		h.items = h.items[1:]
	}
}

func (h *RequestHistory) GetRecent() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	// Return copy in reverse order
	res := make([]string, len(h.items))
	for i, item := range h.items {
		res[len(h.items)-1-i] = item
	}
	return res
}
