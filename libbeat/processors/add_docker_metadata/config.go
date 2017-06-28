package add_docker_metadata

// Config for docker processor
type Config struct {
	Host        string     `config:"host"`
	TLS         *TLSConfig `config:"ssl"`
	Fields      []string   `config:"match_fields"`
	MatchSource bool       `config:"match_source"`
	SourceIndex int        `config:"match_source_index"`
}

// TLSConfig for docker socket connection
type TLSConfig struct {
	CA          string `config:"certificate_authority"`
	Certificate string `config:"certificate"`
	Key         string `config:"key"`
}

func defaultConfig() Config {
	return Config{
		Host:        "unix:///var/run/docker.sock",
		MatchSource: true,
		SourceIndex: 4,
	}
}
