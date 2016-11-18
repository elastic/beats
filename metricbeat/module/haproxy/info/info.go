package info

import (
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/haproxy"
)

const (
	// defaultSocket is the default path to the unix socket tfor stats on haproxy.
	statsMethod = "info"
	defaultAddr = "unix:///var/lib/haproxy/stats"
)

var (
	debugf = logp.MakeDebug("haproxy-info")
)

// init haproxy info metricset.
func init() {
	if err := mb.Registry.AddMetricSet("haproxy", "info", New); err != nil {
		panic(err)
	}
}

// MetricSet for haproxy info.
type MetricSet struct {
	mb.BaseMetricSet
	statsAddr string
}

// New creates a info metricset.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	logp.Warn("EXPERIMENTAL: The haproxy info metricset is experimental")

	return &MetricSet{
		BaseMetricSet: base,
		statsAddr:     base.Host(),
	}, nil
}

// Fetch fetches info stats from the haproxy service.
func (m *MetricSet) Fetch() (common.MapStr, error) {

	hapc, err := haproxy.NewHaproxyClient(m.statsAddr)
	if err != nil {
		return nil, fmt.Errorf("HAProxy Client error: %s", err)
	}

	res, err := hapc.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("HAProxy Client error fetching %s: %s", statsMethod, err)
	}

	mappedEvent, err := eventMapping(res)
	if err != nil {
		return nil, err
	}
	return mappedEvent, nil
}
