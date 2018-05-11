package exchange

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/rabbitmq"
)

func init() {
	mb.Registry.MustAddMetricSet("rabbitmq", "exchange", New,
		mb.WithHostParser(rabbitmq.HostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet for fetching RabbitMQ exchanges metrics.
type MetricSet struct {
	*rabbitmq.MetricSet
}

// New creates new instance of MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The rabbitmq exchange metricset is beta")

	ms, err := rabbitmq.NewMetricSet(base, rabbitmq.ExchangesPath)
	if err != nil {
		return nil, err
	}
	return &MetricSet{ms}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	content, err := m.HTTP.FetchContent()

	if err != nil {
		return nil, err
	}

	events, _ := eventsMapping(content)
	return events, nil
}
