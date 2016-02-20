package prometheus

import (
	"github.com/elastic/beats/metricbeat/helper"
	"os"
)

func init() {
	Module.Register()
}

var Module = helper.NewModule("prometheus", Prometheus{})

var Config = &PrometheusModuleConfig{}

type PrometheusModuleConfig struct {
	Name string
	Metrics map[string]interface{}
	Hosts   []string
}

type Prometheus struct {
	Name   string
	Config PrometheusModuleConfig
}

func (r Prometheus) Setup() error {

// Loads module config
// This is module specific config object
	Module.LoadConfig(&Config)
	Module.Name = Config.Name
	return nil
}

///*** Helper functions for testing ***///

func GetPrometheusEnvHostPort() string {
	host := os.Getenv("PROMETHEUS_EXPORTER_HOST") + ":"
		+ os.Getenv("PROMETHEUS_EXPORTER_PORT")
	if len(host) == 0 {
		host = "127.0.0.1:8080/"
	}

return host
}
