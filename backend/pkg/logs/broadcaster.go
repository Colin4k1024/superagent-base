package logs

import (
	"encoding/json"
	"sync"
	"time"
)

// LogEntry represents a structured log entry for streaming.
type LogEntry struct {
	Level     string         `json:"level"`
	Message   string         `json:"msg"`
	Timestamp string         `json:"ts"`
	Fields    map[string]any `json:"fields,omitempty"`
}

// Broadcaster fans out log entries to subscribed SSE clients.
// It maintains a bounded ring buffer so new subscribers see recent history.
type Broadcaster struct {
	mu          sync.RWMutex
	subscribers map[chan LogEntry]struct{}
	ring        []LogEntry
	ringSize    int
	ringIdx     int
	ringFull    bool
}

// NewBroadcaster creates a Broadcaster with the given ring buffer capacity.
func NewBroadcaster(ringSize int) *Broadcaster {
	return &Broadcaster{
		subscribers: make(map[chan LogEntry]struct{}),
		ring:        make([]LogEntry, ringSize),
		ringSize:    ringSize,
	}
}

// Publish sends a log entry to all subscribers and stores it in the ring buffer.
func (b *Broadcaster) Publish(level, msg string, fields map[string]any) {
	entry := LogEntry{
		Level:     level,
		Message:   msg,
		Timestamp: time.Now().Format(time.RFC3339Nano),
		Fields:    fields,
	}

	b.mu.Lock()
	// Store in ring buffer.
	b.ring[b.ringIdx] = entry
	b.ringIdx = (b.ringIdx + 1) % b.ringSize
	if b.ringIdx == 0 {
		b.ringFull = true
	}

	// Fan-out to subscribers (non-blocking).
	for ch := range b.subscribers {
		select {
		case ch <- entry:
		default:
			// Slow consumer — drop entry rather than blocking.
		}
	}
	b.mu.Unlock()
}

// Subscribe returns a channel that receives log entries and an unsubscribe function.
// The channel buffer is 100 entries; slow consumers will miss entries.
func (b *Broadcaster) Subscribe() (<-chan LogEntry, func()) {
	ch := make(chan LogEntry, 100)

	b.mu.Lock()
	b.subscribers[ch] = struct{}{}

	// Send recent history from ring buffer.
	history := b.getHistoryLocked()
	b.mu.Unlock()

	go func() {
		for _, entry := range history {
			ch <- entry
		}
	}()

	unsubscribe := func() {
		b.mu.Lock()
		delete(b.subscribers, ch)
		b.mu.Unlock()
		// Drain channel to prevent goroutine leaks.
		go func() {
			for range ch {
			}
		}()
		close(ch)
	}

	return ch, unsubscribe
}

// getHistoryLocked returns ring buffer entries in chronological order.
// Must be called with b.mu held.
func (b *Broadcaster) getHistoryLocked() []LogEntry {
	if !b.ringFull {
		result := make([]LogEntry, b.ringIdx)
		copy(result, b.ring[:b.ringIdx])
		return result
	}
	// Ring is full: entries from ringIdx..end, then 0..ringIdx-1
	result := make([]LogEntry, b.ringSize)
	copy(result, b.ring[b.ringIdx:])
	copy(result[b.ringSize-b.ringIdx:], b.ring[:b.ringIdx])
	return result
}

// SubscriberCount returns the current number of active subscribers.
func (b *Broadcaster) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}

// MarshalEntry serializes a LogEntry to JSON bytes.
func MarshalEntry(e LogEntry) []byte {
	data, _ := json.Marshal(e)
	return data
}

// Global broadcaster instance.
var globalBroadcaster = NewBroadcaster(1000)

// GetBroadcaster returns the global log broadcaster.
func GetBroadcaster() *Broadcaster {
	return globalBroadcaster
}

// BroadcastLog publishes a log entry to the global broadcaster.
// Call this from the logging infrastructure to feed the SSE stream.
func BroadcastLog(level, msg string, fields map[string]any) {
	globalBroadcaster.Publish(level, msg, fields)
}
