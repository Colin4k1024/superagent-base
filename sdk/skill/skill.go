package skill

import (
	"context"
	"fmt"
)

type SkillMeta struct {
	Name        string
	Version     string
	Description string
}

type SkillInstance struct {
	Meta   SkillMeta
	Status string
}

type SkillFunc func(ctx context.Context, input map[string]any) (map[string]any, error)

type Invoker interface {
	Invoke(ctx context.Context, name string, input map[string]any) (map[string]any, error)
}

type LocalInvoker struct {
	skills map[string]SkillFunc
}

func NewLocalInvoker() *LocalInvoker {
	return &LocalInvoker{skills: make(map[string]SkillFunc)}
}

func (l *LocalInvoker) Register(name string, fn SkillFunc) {
	l.skills[name] = fn
}

func (l *LocalInvoker) Invoke(ctx context.Context, name string, input map[string]any) (map[string]any, error) {
	fn, ok := l.skills[name]
	if !ok {
		return nil, fmt.Errorf("skill %q not registered", name)
	}
	return fn(ctx, input)
}

type Manager struct {
	invoker   Invoker
	installed map[string]*SkillInstance
}

func NewManager(invoker Invoker) *Manager {
	return &Manager{
		invoker:   invoker,
		installed: make(map[string]*SkillInstance),
	}
}

func (m *Manager) RegisterLocal(meta SkillMeta, fn SkillFunc) {
	if li, ok := m.invoker.(*LocalInvoker); ok {
		li.Register(meta.Name, fn)
	}
	m.installed[meta.Name] = &SkillInstance{Meta: meta, Status: "builtin"}
}

func (m *Manager) Invoke(ctx context.Context, name string, input map[string]any) (map[string]any, error) {
	return m.invoker.Invoke(ctx, name, input)
}

func (m *Manager) ListInstalled() []SkillInstance {
	result := make([]SkillInstance, 0, len(m.installed))
	for _, inst := range m.installed {
		result = append(result, *inst)
	}
	return result
}
