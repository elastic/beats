package docker

var defaultConfig = config{
	Partial: true,
	Containers: containers{
		IDs:    []string{},
		Path:   "/var/lib/docker/containers",
		Stream: "all",
	},
}

type config struct {
	Containers containers `config:"containers"`

	// Partial configures the prospector to join partial lines
	Partial bool `config:"combine_partials"`
}

type containers struct {
	IDs  []string `config:"ids"`
	Path string   `config:"path"`

	// Stream can be all,stdout or stderr
	Stream string `config:"stream"`
}
