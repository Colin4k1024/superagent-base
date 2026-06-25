package model

type Config struct {
	Name     string
	Provider string
	BaseURL  string
	APIKey   string
	ModelID  string
}

type Registry struct {
	configs map[string]Config
}

func NewRegistry() *Registry {
	return &Registry{configs: make(map[string]Config)}
}

func (r *Registry) Register(cfg Config) {
	r.configs[cfg.Name] = cfg
}

func (r *Registry) Get(name string) (Config, bool) {
	cfg, ok := r.configs[name]
	return cfg, ok
}

func (r *Registry) List() []Config {
	result := make([]Config, 0, len(r.configs))
	for _, cfg := range r.configs {
		result = append(result, cfg)
	}
	return result
}

type Router struct {
	registry *Registry
}

func NewRouter(registry *Registry) *Router {
	return &Router{registry: registry}
}

func (r *Router) Resolve(modelID string) (Config, bool) {
	if cfg, ok := r.registry.Get(modelID); ok {
		return cfg, true
	}
	for _, cfg := range r.registry.List() {
		if cfg.ModelID == modelID {
			return cfg, true
		}
	}
	return Config{}, false
}
