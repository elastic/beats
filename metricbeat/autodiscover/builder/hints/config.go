package hints

import "github.com/elastic/beats/metricbeat/mb"

type config struct {
	Key      string `config:"key"`
	Registry *mb.Register
}

func defaultConfig() config {
	return config{
		Key:      "metrics",
		Registry: mb.Registry,
	}
}
