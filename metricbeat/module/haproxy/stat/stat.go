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

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	if err := mb.Registry.AddMetricSet("haproxy", statsMethod, New); err != nil {
		panic(err)
	}
}

// MetricSet type defines all fields of the MetricSet
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type MetricSet struct {
	mb.BaseMetricSet
	statsAddr string
	counter   int
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	logp.Warn("EXPERIMENTAL: The haproxy stat metricset is experimental")

	config := struct {
		StatsAddr string `config:"stats_addr"`
	}{
		StatsAddr: defaultAddr,
	}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		statsAddr:     config.StatsAddr,
		counter:       1,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {

	hapc, err := haproxy.NewHaproxyClient(m.statsAddr)
	if err != nil {
		return nil, fmt.Errorf("HAProxy Client error: %s", err)
	}

	res, err := hapc.GetStat()

	if err != nil {
		return nil, fmt.Errorf("HAProxy Client error fetching %s: %s", statsMethod, err)
	}
	m.counter++

	return eventMapping(res), nil

}
