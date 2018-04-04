package elasticsearch

type Config struct {
	Source bool `config:"pending_tasks.source"`
}

// DefaultConfig returns default module config
func DefaultConfig() Config {
	return Config{}
}
