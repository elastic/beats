package util

import (
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/helper/prometheus"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
)

// NewStateMetricsClient gets `state_metrics` settings from the module and returns a Prometheus client for them
func NewStateMetricsClient(base mb.BaseMetricSet) (prometheus.Prometheus, error) {
	stateMetricsConfig := struct {
		StateMetrics struct {
			helper.HTTPConfig
			Host string `config:"host"`
		} `config:"state_metrics"`
	}{}

	err := base.Module().UnpackConfig(&stateMetricsConfig)
	if err != nil {
		return nil, err
	}

	var state prometheus.Prometheus
	if stateMetricsConfig.StateMetrics.Host != "" {
		stateHostConf := struct {
			StateMetrics map[string]interface{} `config:"state_metrics"`
		}{
			StateMetrics: map[string]interface{}{},
		}
		err = base.Module().UnpackConfig(&stateHostConf)
		if err != nil {
			return nil, err
		}

		hostData, err := parse.URLHostParserBuilder{
			DefaultScheme: "http",
			DefaultPath:   "/metrics",
		}.BuildFromConfig(stateHostConf.StateMetrics, base.Module().Name(), stateMetricsConfig.StateMetrics.Host)
		if err != nil {
			return nil, err
		}

		stateHTTP, err := helper.NewHTTPFromConfig(&stateMetricsConfig.StateMetrics.HTTPConfig, hostData, base.Name())
		if err != nil {
			return nil, err
		}
		state = prometheus.NewPrometheusClientWithHTTP(stateHTTP)
	}

	return state, nil
}
