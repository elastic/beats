package osdpoolstats

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/ceph"
)

func init() {
	if err := mb.Registry.AddMetricSet("ceph", "osdpoolstats", New); err != nil {
		panic(err)
	}
}

type MetricSet struct {
	mb.BaseMetricSet
	cfg *ceph.CephConfig
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	config := ceph.CheckConfig()

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		cfg:           &config,
	}, nil
}

func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	output, err := ceph.Execute(m.cfg, "osd pool stats")
	if err != nil {
		return nil, err
	}

	return eventsMapping(output), nil
}
