package metric_annotations

type config struct {
	Prefix string `config:"prefix"`
}

func defaultConfig() config {
	return config{
		Prefix: "co.elastic.metrics",
	}
}
