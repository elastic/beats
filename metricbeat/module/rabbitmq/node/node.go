package node

import (
	"encoding/json"

	"github.com/pkg/errors"

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

// ClusterMetricSet is the MetricSet type used when node.collect is "all"
type ClusterMetricSet struct {
	*rabbitmq.MetricSet
}

// New creates new instance of MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The rabbitmq node metricset is beta")

	config := defaultConfig
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	switch config.Collect {
	case configCollectNode:
		ms, err := rabbitmq.NewMetricSet(base, rabbitmq.OverviewPath)
		if err != nil {
			return nil, err
		}

		return &MetricSet{ms}, nil
	case configCollectCluster:
		ms, err := rabbitmq.NewMetricSet(base, rabbitmq.NodesPath)
		if err != nil {
			return nil, err
		}

		return &ClusterMetricSet{ms}, nil
	default:
		return nil, errors.Errorf("incorrect node.collect: %s", config.Collect)
	}
}

type apiOverview struct {
	Node string `json:"node"`
}

func (m *MetricSet) fetchOverview() (*apiOverview, error) {
	d, err := m.HTTP.FetchContent()
	if err != nil {
		return nil, err
	}

	var apiOverview apiOverview
	err = json.Unmarshal(d, &apiOverview)
	if err != nil {
		return nil, errors.Wrap(err, string(d))
	}
	return &apiOverview, nil
}

// Fetch metrics from rabbitmq node
func (m *MetricSet) Fetch(r mb.ReporterV2) {
	o, err := m.fetchOverview()
	if err != nil {
		r.Error(err)
		return
	}

	node, err := rabbitmq.NewMetricSet(m.BaseMetricSet, rabbitmq.NodesPath+"/"+o.Node)
	if err != nil {
		r.Error(err)
		return
	}

	content, err := node.HTTP.FetchJSON()
	if err != nil {
		r.Error(err)
		return
	}

	eventMapping(r, content)
}

// Fetch metrics from all rabbitmq nodes in the cluster
func (m *ClusterMetricSet) Fetch(r mb.ReporterV2) {
	content, err := m.HTTP.FetchContent()
	if err != nil {
		r.Error(err)
		return
	}

	eventsMapping(r, content)
}
