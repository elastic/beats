package prometheus

import (
	"github.com/elastic/beats/metricbeat/helper"
	"os"
)

func init() {
	helper.Registry.AddModuler("prometheus", Moduler{})
}

type Moduler struct{}

func (r Moduler) Setup() error {
	// Loads module config
	// This is module specific config object
	//	Moduler.LoadConfig(&Config)
	//	Moduler.Name = Config.Name
	return nil
}

///*** Helper functions for testing ***///

func GetPrometheusEnvHostPort() string {
	host := os.Getenv("PROMETHEUS_EXPORTER_HOST") + ":" +
		os.Getenv("PROMETHEUS_EXPORTER_PORT")

	if len(host) == 0 {
		host = "127.0.0.1:8080/"
	}

	return host
}
