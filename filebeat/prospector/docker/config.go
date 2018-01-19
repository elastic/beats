package docker

var defaultConfig = config{
	Containers: containers{
		IDs:    []string{},
		Path:   "/var/lib/docker/containers",
		Stream: "all",
	},
}

type config struct {
	Containers containers `config:"containers"`
}

type containers struct {
	IDs  []string `config:"ids"`
	Path string   `config:"path"`

	// Stream can be all,stdout or stderr
	Stream string `config:"stream"`
}
