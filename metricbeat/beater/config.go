package beater

import "github.com/urso/ucfg"

type MetricbeatConfig struct {
	Metricbeat struct {
		Modules map[string]*ucfg.Config
	}
}
