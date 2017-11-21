package node_stats

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	if err := mb.Registry.AddMetricSet("elasticsearch", "node_stats", New, hostParser); err != nil {
		panic(err)
	}
}

var (
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: "http",
		PathConfigKey: "path",
		// Get the stats from the local node
		DefaultPath: "_nodes/_local/stats",
	}.Build()
)

// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	mb.BaseMetricSet
	http *helper.HTTP
}

// New create a new instance of the MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The elasticsearch node_stats metricset is beta")

	return &MetricSet{
		base,
		helper.NewHTTP(base),
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	content, err := m.http.FetchContent()
	if err != nil {
		return nil, err
	}

	events, _ := eventsMapping(content)
	return events, nil
}
