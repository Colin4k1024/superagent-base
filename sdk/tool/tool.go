package tool

import (
	"context"
	"fmt"
)

type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, args map[string]any) (map[string]any, error)
}

type ToolFunc func(ctx context.Context, args map[string]any) (map[string]any, error)

type funcTool struct {
	name        string
	description string
	fn          ToolFunc
}

func New(name, description string, fn ToolFunc) Tool {
	return &funcTool{name: name, description: description, fn: fn}
}

func (t *funcTool) Name() string        { return t.name }
func (t *funcTool) Description() string { return t.description }
func (t *funcTool) Execute(ctx context.Context, args map[string]any) (map[string]any, error) {
	return t.fn(ctx, args)
}

type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

func (r *Registry) List() []Tool {
	result := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		result = append(result, t)
	}
	return result
}

func (r *Registry) Invoke(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	t, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool %q not found", name)
	}
	return t.Execute(ctx, args)
}
