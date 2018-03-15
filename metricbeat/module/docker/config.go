package docker

type Config struct {
	TLS   *TLSConfig `config:"ssl"`
	DeDot bool       `config:"labels.dedot"`
}

// DefaultConfig returns default module config
func DefaultConfig() Config {
	return Config{
		DeDot: true,
	}
}

type TLSConfig struct {
	Enabled     *bool  `config:"enabled"`
	CA          string `config:"certificate_authority"`
	Certificate string `config:"certificate"`
	Key         string `config:"key"`
}

func (c *TLSConfig) IsEnabled() bool {
	return c != nil && (c.Enabled == nil || *c.Enabled)
}
