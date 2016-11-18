package stat

import (
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/haproxy"
)

const (
	// defaultSocket is the default path to the unix socket for stats on haproxy.
	statsMethod = "stat"
	defaultAddr = "unix:///var/lib/haproxy/stats"
)

var (
	debugf = logp.MakeDebug("haproxy-stat")
)

// init adds stat metricset.
func init() {
	if err := mb.Registry.AddMetricSet("haproxy", statsMethod, New); err != nil {
		panic(err)
	}
}

// MetricSet defines stat metricset.
type MetricSet struct {
	mb.BaseMetricSet
	statsAddr string
}

// New creates a new instance of haproxy stat metricset.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	logp.Warn("EXPERIMENTAL: The haproxy stat metricset is experimental")

	return &MetricSet{
		BaseMetricSet: base,
		statsAddr:     base.Host(),
	}, nil
}

// Fetch methods returns a list of stats metrics.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {

	hapc, err := haproxy.NewHaproxyClient(m.statsAddr)
	if err != nil {
		return nil, fmt.Errorf("HAProxy Client error: %s", err)
	}

	res, err := hapc.GetStat()
	if err != nil {
		return nil, fmt.Errorf("HAProxy Client error fetching %s: %s", statsMethod, err)
	}

	return eventMapping(res), nil

}
