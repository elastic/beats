package docker

type Config struct {
	TLS *TLSConfig `config:"ssl"`
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
