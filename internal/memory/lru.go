package memory

import "container/list"

// LRU tracks session access order for eviction.
// It maintains a doubly-linked list (most-recent at front) and a map for O(1) lookups.
type LRU struct {
	order    *list.List
	elements map[string]*list.Element
}

// NewLRU creates a new LRU tracker.
func NewLRU() *LRU {
	return &LRU{
		order:    list.New(),
		elements: make(map[string]*list.Element),
	}
}

// Promote moves a session to the most-recently-used position.
// If the session is new, it is added to the front.
func (l *LRU) Promote(sessionID string) {
	if elem, ok := l.elements[sessionID]; ok {
		l.order.MoveToFront(elem)
		return
	}
	elem := l.order.PushFront(sessionID)
	l.elements[sessionID] = elem
}

// Evict removes and returns the least-recently-used session ID.
// Returns empty string if the tracker is empty.
func (l *LRU) Evict() string {
	back := l.order.Back()
	if back == nil {
		return ""
	}
	sessionID, _ := back.Value.(string)
	l.order.Remove(back)
	delete(l.elements, sessionID)
	return sessionID
}

// Remove removes a specific session from the tracker.
func (l *LRU) Remove(sessionID string) {
	if elem, ok := l.elements[sessionID]; ok {
		l.order.Remove(elem)
		delete(l.elements, sessionID)
	}
}

// Len returns the number of tracked sessions.
func (l *LRU) Len() int {
	return l.order.Len()
}
