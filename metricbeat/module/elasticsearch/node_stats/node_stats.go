package node_stats

import (
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	mb.Registry.MustAddMetricSet("elasticsearch", "node_stats", New,
		mb.WithHostParser(hostParser),
		mb.DefaultMetricSet(),
		mb.WithNamespace("elasticsearch.node.stats"),
	)
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
	http  *helper.HTTP
	xpack bool
}

// New create a new instance of the MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The elasticsearch node_stats metricset is beta")

	config := struct {
		XPack bool `config:"xpack.enabled"`
	}{
		XPack: false,
	}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	if config.XPack {
		cfgwarn.Experimental("The experimental xpack.enabled flag in elasticsearch/node_stats metricset is enabled.")
	}

	http, err := helper.NewHTTP(base)
	if err != nil {
		return nil, err
	}
	return &MetricSet{
		BaseMetricSet: base,
		http:          http,
		xpack:         config.XPack,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
func (m *MetricSet) Fetch(r mb.ReporterV2) {
	content, err := m.http.FetchContent()
	if err != nil {
		r.Error(err)
		return
	}

	if m.xpack {
		eventsMappingXPack(r, m, content)
	} else {
		eventsMapping(r, content)
	}
}
