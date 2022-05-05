package shovel

import (
	"github.com/pkg/errors"
	"github.com/elastic/beats/v7/metricbeat/module/rabbitmq"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

func init() {
	mb.Registry.MustAddMetricSet("rabbitmq", "shovel", New,
		mb.WithHostParser(rabbitmq.HostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet for fetching RabbitMQ shovels metrics.
type MetricSet struct {
	*rabbitmq.MetricSet
}

// New creates new instance of MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := rabbitmq.NewMetricSet(base, rabbitmq.ShovelsPath)
	if err != nil {
		return nil, err
	}
	return &MetricSet{ms}, nil
}

// Fetch fetches shovel data
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	content, err := m.HTTP.FetchContent()

	if err != nil {
		return errors.Wrap(err, "error in fetch")
	}

	return eventsMapping(content, report)
}