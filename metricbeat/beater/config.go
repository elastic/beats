package beater

import (
	"github.com/elastic/beats/metricbeat/helper"
)

type Config struct {
	Metricbeat MetricbeatConfig
}

type MetricbeatConfig struct {
	Modules []helper.ModuleConfig
}
