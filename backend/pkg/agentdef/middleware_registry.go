package agentdef

import (
	"context"
	"fmt"
	"sync"

	"github.com/cloudwego/eino/adk"
)

// MiddlewareFactory constructs an ADK ChatModelAgentMiddleware from YAML config.
type MiddlewareFactory func(ctx context.Context, config map[string]any) (adk.ChatModelAgentMiddleware, error)

var (
	mwRegistryMu sync.RWMutex
	mwRegistry   = make(map[string]MiddlewareFactory)
)

// RegisterMiddleware registers a named middleware factory for use in agent YAML definitions.
// Built-in middlewares register themselves in init(); user-defined middlewares can register at startup.
func RegisterMiddleware(name string, factory MiddlewareFactory) {
	if name == "" {
		panic("agentdef: RegisterMiddleware called with empty name")
	}
	if factory == nil {
		panic(fmt.Sprintf("agentdef: RegisterMiddleware %q called with nil factory", name))
	}
	mwRegistryMu.Lock()
	defer mwRegistryMu.Unlock()
	mwRegistry[name] = factory
}

// GetMiddlewareFactory retrieves a registered middleware factory by name.
func GetMiddlewareFactory(name string) (MiddlewareFactory, bool) {
	mwRegistryMu.RLock()
	defer mwRegistryMu.RUnlock()
	f, ok := mwRegistry[name]
	return f, ok
}

// ListMiddleware returns all registered middleware names.
func ListMiddleware() []string {
	mwRegistryMu.RLock()
	defer mwRegistryMu.RUnlock()
	names := make([]string, 0, len(mwRegistry))
	for name := range mwRegistry {
		names = append(names, name)
	}
	return names
}
