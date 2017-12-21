package docker

var defaultConfig = config{
	Containers: containers{
		IDs:  []string{},
		Path: "/var/lib/docker/containers",
	},
}

type config struct {
	Containers containers `config:"containers"`
}

type containers struct {
	IDs  []string `config:"ids"`
	Path string   `config:"path"`
}
