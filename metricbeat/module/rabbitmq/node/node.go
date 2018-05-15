package node

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/rabbitmq"
)

func init() {
	mb.Registry.MustAddMetricSet("rabbitmq", "node", New,
		mb.WithHostParser(rabbitmq.HostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet for fetching RabbitMQ node metrics
type MetricSet struct {
	*rabbitmq.MetricSet
}

// New creates new instance of MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The rabbitmq node metricset is beta")

	ms, err := rabbitmq.NewMetricSet(base, rabbitmq.NodesPath)
	if err != nil {
		return nil, err
	}
	return &MetricSet{ms}, nil
}

func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	content, err := m.HTTP.FetchContent()

	if err != nil {
		return nil, err
	}

	events, _ := eventsMapping(content)
	return events, nil
}
