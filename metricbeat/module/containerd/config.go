package containerd

// Config contains the config needed for containerd
type Config struct {
	CalculatePct bool `config:"calcpct"`
}

// DefaultConfig returns default module config
func DefaultConfig() Config {
	return Config{
		CalculatePct: true,
	}
}
