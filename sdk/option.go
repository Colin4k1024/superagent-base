package sdk

type Option func(*Config)

type Config struct {
	AgentsDir    string
	ModelConfig  ModelRuntimeConfig
	RedisAddr    string
	RedisPwd     string
	EnableEvolution bool
}

type ModelRuntimeConfig struct {
	BaseURL string
	APIKey  string
	ModelID string
}

func WithAgentsDir(dir string) Option {
	return func(c *Config) { c.AgentsDir = dir }
}

func WithModel(cfg ModelRuntimeConfig) Option {
	return func(c *Config) { c.ModelConfig = cfg }
}

func WithRedis(addr, password string) Option {
	return func(c *Config) {
		c.RedisAddr = addr
		c.RedisPwd = password
	}
}

func WithEvolution(enabled bool) Option {
	return func(c *Config) { c.EnableEvolution = enabled }
}
