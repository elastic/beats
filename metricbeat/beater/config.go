package beater

import "github.com/elastic/beats/libbeat/common"

type Config struct {
	Metricbeat MetricbeatConfig
}

type MetricbeatConfig struct {
	Modules []*common.Config
}
