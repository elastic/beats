package docker

type TlsConfig struct {
	Enabled bool `config:"enabled"`
	// TODO: Naming should be standardised with output cert configs
	CaPath   string `config:"ca_path"`
	CertPath string `config:"cert_path"`
	KeyPath  string `config:"key_path"`
}

type Config struct {
	Socket string `config:"socket"`
	Tls    TlsConfig
}

func GetDefaultConf() Config {
	return Config{
		Socket: "unix:///var/run/docker.sock",
		Tls: TlsConfig{
			Enabled: false,
		},
	}
}
