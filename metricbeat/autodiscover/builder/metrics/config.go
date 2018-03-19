package metrics

type config struct {
	Key string `config:"key"`
}

func defaultConfig() config {
	return config{
		Key: "metrics",
	}
}
