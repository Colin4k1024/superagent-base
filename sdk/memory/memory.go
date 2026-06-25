package memory

import (
	"context"
	"sync"
	"time"
)

type Message struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
}

type Backend interface {
	GetMessages(ctx context.Context, sessionID string, limit int) ([]Message, error)
	AddMessage(ctx context.Context, sessionID string, msg Message) error
	Clear(ctx context.Context, sessionID string) error
}

type BuiltinMemory struct {
	store map[string][]Message
	mu    sync.RWMutex
}

func NewBuiltin() *BuiltinMemory {
	return &BuiltinMemory{store: make(map[string][]Message)}
}

func (m *BuiltinMemory) GetMessages(ctx context.Context, sessionID string, limit int) ([]Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	msgs := m.store[sessionID]
	if limit > 0 && limit < len(msgs) {
		msgs = msgs[len(msgs)-limit:]
	}
	return msgs, nil
}

func (m *BuiltinMemory) AddMessage(ctx context.Context, sessionID string, msg Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if msg.Timestamp == 0 {
		msg.Timestamp = time.Now().Unix()
	}
	m.store[sessionID] = append(m.store[sessionID], msg)
	return nil
}

func (m *BuiltinMemory) Clear(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.store, sessionID)
	return nil
}
