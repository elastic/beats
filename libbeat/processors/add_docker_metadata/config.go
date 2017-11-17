package add_docker_metadata

import (
	"time"

	"github.com/elastic/beats/libbeat/common/docker"
)

// Config for docker processor
type Config struct {
	Host        string            `config:"host"`
	TLS         *docker.TLSConfig `config:"ssl"`
	Fields      []string          `config:"match_fields"`
	MatchSource bool              `config:"match_source"`
	SourceIndex int               `config:"match_source_index"`

	// Annotations are kept after container is killled, until they haven't been accessed
	// for a full `cleanup_timeout`:
	CleanupTimeout time.Duration `config:"cleanup_timeout"`
}

func defaultConfig() Config {
	return Config{
		Host:        "unix:///var/run/docker.sock",
		MatchSource: true,
		SourceIndex: 4,
	}
}
