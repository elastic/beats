package jetstream

type ModuleConfig struct {
	Jetstream MetricsetConfig `config:"jetstream"`
}

type MetricsetConfig struct {
	Stream StreamConfig `config:"stream"`
}

type StreamConfig struct {
	Names []string `config:"names"`
}
