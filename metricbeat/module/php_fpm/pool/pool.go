package pool

import (
	"encoding/json"
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"

	"github.com/elastic/beats/metricbeat/module/php_fpm"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	if err := mb.Registry.AddMetricSet("php_fpm", "pool", New, php_fpm.HostParser); err != nil {
		panic(err)
	}
}

// MetricSet type defines all fields of the MetricSet
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type MetricSet struct {
	mb.BaseMetricSet
	client *php_fpm.StatsClient // StatsClient that is reused across requests.
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	logp.Warn("EXPERIMENTAL: The php-fpm pool metricset is experimental")
	return &MetricSet{
		BaseMetricSet: base,
		client:        php_fpm.NewStatsClient(base),
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch() (common.MapStr, error) {
	body, err := m.client.Fetch()

	if err != nil {
		return nil, err
	}

	defer body.Close()

	stats := &poolStats{}
	err = json.NewDecoder(body).Decode(stats)
	if err != nil {
		return nil, fmt.Errorf("error parsing json: %v", err)
	}

	return common.MapStr{
		"hostname": m.Host(),

		"pool": stats.Pool,
		"connections": common.MapStr{
			"queue":    stats.ListenQueue,
			"accepted": stats.AcceptedConn,
		},
		"processes": common.MapStr{
			"idle":   stats.IdleProcesses,
			"active": stats.ActiveProcesses,
		},
		"slow_requests": stats.SlowRequests,
	}, nil
}
