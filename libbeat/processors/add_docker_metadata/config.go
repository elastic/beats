package add_docker_metadata

import (
	"time"

	"github.com/elastic/beats/libbeat/common/docker"
)

// Config for docker processor.
type Config struct {
	Host         string            `config:"host"`               // Docker socket (UNIX or TCP socket).
	TLS          *docker.TLSConfig `config:"ssl"`                // TLS settings for connecting to Docker.
	Fields       []string          `config:"match_fields"`       // A list of fields to match a container ID.
	MatchSource  bool              `config:"match_source"`       // Match container ID from a log path present in source field.
	MatchShortID bool              `config:"match_short_id"`     // Match to container short ID from a log path present in source field.
	SourceIndex  int               `config:"match_source_index"` // Index in the source path split by / to look for container ID.
	MatchPIDs    []string          `config:"match_pids"`         // A list of fields containing process IDs (PIDs).
	HostFS       string            `config:"system.hostfs"`      // Specifies the mount point of the host’s filesystem for use in monitoring a host from within a container.

	// Annotations are kept after container is killed, until they haven't been
	// accessed for a full `cleanup_timeout`:
	CleanupTimeout time.Duration `config:"cleanup_timeout"`
}

func defaultConfig() Config {
	return Config{
		Host:        "unix:///var/run/docker.sock",
		MatchSource: true,
		SourceIndex: 4, // Use 4 to match the CID in /var/lib/docker/containers/<container_id>/*.log.
		MatchPIDs:   []string{"process.pid", "process.ppid"},
	}
}
