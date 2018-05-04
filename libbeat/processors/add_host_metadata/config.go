package add_host_metadata

// Config for add_host_metadata processor.
type Config struct {
	NetInfoEnabled bool `config:"netinfo.enabled"` // Add IP and MAC to event
}

func defaultConfig() Config {
	return Config{
		NetInfoEnabled: false,
	}
}
