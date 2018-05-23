package shard

import (
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/elasticsearch"
)

func init() {
	mb.Registry.MustAddMetricSet("elasticsearch", "shard", New,
		mb.WithHostParser(elasticsearch.HostParser),
		mb.DefaultMetricSet(),
		mb.WithNamespace("elasticsearch.shard"),
	)
}

const (
	// Get the stats from the local node
	statePath = "/_cluster/state/version,master_node,routing_table"
)

// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	*elasticsearch.MetricSet
}

// New create a new instance of the MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The elasticsearch shard metricset is beta")

	// Get the stats from the local node
	ms, err := elasticsearch.NewMetricSet(base, statePath)
	if err != nil {
		return nil, err
	}
	return &MetricSet{MetricSet: ms}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
func (m *MetricSet) Fetch(r mb.ReporterV2) {
	content, err := m.HTTP.FetchContent()
	if err != nil {
		r.Error(err)
		return
	}

	if m.XPack {
		eventsMappingXPack(r, m, content)
	} else {
		eventsMapping(r, content)
	}
}
