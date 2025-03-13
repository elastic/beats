package jetstream

type ModuleConfig struct {
	Jetstream MetricsetConfig `config:"jetstream"`
}

type MetricsetConfig struct {
	Stream   StreamConfig   `config:"stream"`
	Consumer ConsumerConfig `config:"consumer"`
}

type StreamConfig struct {
	Names []string `config:"names"`
}

type ConsumerConfig struct {
	Names []string `config:"names"`
}
