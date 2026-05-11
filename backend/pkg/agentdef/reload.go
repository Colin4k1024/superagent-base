package agentdef

import (
	"context"
	"sync"

	"github.com/superagent-ai/superagent-base/backend/pkg/logs"
)

type ChangeType int

const (
	ChangeAdded ChangeType = iota
	ChangeUpdated
	ChangeDeleted
)

type ChangeEvent struct {
	Type      ChangeType
	AgentName string
	Def       *AgentDefinition
}

type ChangeHandler func(event ChangeEvent)

type Reloader struct {
	mu       sync.RWMutex
	registry *Loader
	handlers []ChangeHandler
}

func NewReloader(ctx context.Context, dir string) (*Reloader, error) {
	reg := NewLoader(dir)
	_, err := reg.LoadAll()
	if err != nil {
		logs.Warnf("agentdef reloader: initial load warnings: %v", err)
	}
	r := &Reloader{registry: reg}
	logs.Infof("agentdef reloader: loaded %d agent(s) from %s", len(reg.List()), dir)
	_ = ctx
	return r, nil
}

func (r *Reloader) OnChange(h ChangeHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers = append(r.handlers, h)
}

func (r *Reloader) Get(name string) (*AgentDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.registry.Get(name)
}

func (r *Reloader) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.registry.List()
}

func (r *Reloader) ReloadDir(ctx context.Context, dir string) error {
	fresh := NewLoader(dir)
	_, err := fresh.LoadAll()
	if err != nil {
		logs.Warnf("agentdef reloader: reload warnings: %v", err)
	}

	r.mu.Lock()
	old := r.registry
	r.registry = fresh
	r.mu.Unlock()

	r.diffAndNotify(ctx, old, fresh)
	logs.Infof("agentdef reloader: reload complete (%d agents)", len(fresh.List()))
	return nil
}

func (r *Reloader) diffAndNotify(_ context.Context, old, fresh *Loader) {
	r.mu.RLock()
	handlers := append([]ChangeHandler(nil), r.handlers...)
	r.mu.RUnlock()

	if len(handlers) == 0 {
		return
	}

	oldNames := toSet(old.List())
	freshNames := toSet(fresh.List())

	for name := range freshNames {
		def, _ := fresh.Get(name)
		if _, existed := oldNames[name]; !existed {
			emit(handlers, ChangeEvent{Type: ChangeAdded, AgentName: name, Def: def})
		} else {
			emit(handlers, ChangeEvent{Type: ChangeUpdated, AgentName: name, Def: def})
		}
	}

	for name := range oldNames {
		if _, stillExists := freshNames[name]; !stillExists {
			emit(handlers, ChangeEvent{Type: ChangeDeleted, AgentName: name})
		}
	}
}

func emit(handlers []ChangeHandler, evt ChangeEvent) {
	for _, h := range handlers {
		h(evt)
	}
}

func toSet(names []string) map[string]struct{} {
	s := make(map[string]struct{}, len(names))
	for _, n := range names {
		s[n] = struct{}{}
	}
	return s
}
