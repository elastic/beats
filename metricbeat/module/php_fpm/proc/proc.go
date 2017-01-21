package proc

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"

	"github.com/elastic/beats/metricbeat/module/php_fpm"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	if err := mb.Registry.AddMetricSet("php_fpm", "proc", New, php_fpm.HostParser); err != nil {
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
	logp.Warn("EXPERIMENTAL: The php-fpm proc metricset is experimental")
	return &MetricSet{
		BaseMetricSet: base,
		client:        php_fpm.NewStatsClient(base, true),
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	body, err := m.client.Fetch()

	if err != nil {
		return nil, err
	}

	defer body.Close()

	stats := &fullStats{}
	err = json.NewDecoder(body).Decode(stats)
	if err != nil {
		return nil, fmt.Errorf("error parsing json: %v", err)
	}

	events := []common.MapStr{}
	for _, proc := range stats.Processes {
		events = append(events, common.MapStr{
			"hostname": m.Host(),

			"pid":                 strconv.Itoa(proc.Pid),
			"state":               proc.State,
			"start_time":          proc.StartTime,
			"start_since":         proc.StartSince,
			"requests":            proc.Requests,
			"request_duration":    proc.RequestDuration,
			"request_method":      proc.RequestMethod,
			"request_uri":         proc.RequestURI,
			"content_length":      proc.ContentLength,
			"user":                proc.User,
			"script":              proc.Script,
			"last_request_cpu":    proc.LastRequestCPU,
			"last_request_memory": proc.LastRequestMemory,
		})
	}

	return events, nil
}
