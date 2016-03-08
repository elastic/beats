package beater

import (
	"github.com/urso/ucfg"
)

type Config struct {
	Metricbeat MetricbeatConfig
}

type MetricbeatConfig struct {
	Modules []*ucfg.Config
}
