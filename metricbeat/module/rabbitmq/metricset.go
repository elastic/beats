package rabbitmq

import (
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/mb"
)

// MetricSet can be used to build other metric sets that query RabbitMQ
// management plugin
type MetricSet struct {
	mb.BaseMetricSet
	*helper.HTTP
}

// NewMetricSet creates an metric set that can be used to build other metric
// sets that query RabbitMQ management plugin
func NewMetricSet(base mb.BaseMetricSet, subPath string) (*MetricSet, error) {
	http, err := helper.NewHTTP(base)
	if err != nil {
		return nil, err
	}
	http.SetURI(http.GetURI() + subPath)
	http.SetHeader("Accept", "application/json")

	return &MetricSet{
		base,
		http,
	}, nil
}
