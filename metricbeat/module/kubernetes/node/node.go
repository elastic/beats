package node

import (
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/helper/prometheus"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
	"github.com/elastic/beats/metricbeat/module/kubernetes/state_node"
	"github.com/elastic/beats/metricbeat/module/kubernetes/util"
)

const (
	defaultScheme = "http"
	defaultPath   = "/stats/summary"
)

var (
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		DefaultPath:   defaultPath,
	}.Build()
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	mb.Registry.MustAddMetricSet("kubernetes", "node", New,
		mb.WithHostParser(hostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet type defines all fields of the MetricSet
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type MetricSet struct {
	mb.BaseMetricSet
	http *helper.HTTP

	// kube-state-metrics client
	state prometheus.Prometheus
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	http, err := helper.NewHTTP(base)
	if err != nil {
		return nil, err
	}

	state, err := util.NewStateMetricsClient(base)
	if err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		http:          http,
		state:         state,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	body, err := m.http.FetchContent()
	if err != nil {
		return nil, err
	}

	var stateMetrics []common.MapStr
	if m.state != nil {
		stateMetrics, err = m.state.GetProcessedMetrics(&state_node.Mapping)
		if err != nil {
			logp.Warn(fmt.Sprintf("Error fetching kube-state-metrics for %s: %s", m.Name(), err))
		}
	}

	events, err := eventMapping(body, stateMetrics)
	if err != nil {
		return nil, err
	}

	return events, nil
}
