package index

import (
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
	"github.com/elastic/beats/metricbeat/module/elasticsearch"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	mb.Registry.MustAddMetricSet("elasticsearch", "index", New,
		mb.WithHostParser(hostParser),
	)
}

var (
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: "http",
		PathConfigKey: "path",
		// TODO: This currently gets index data for all indices. Make it configurable.
		DefaultPath: "_stats",
	}.Build()
)

// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	mb.BaseMetricSet
	http *helper.HTTP
}

// New create a new instance of the MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Experimental("The elasticsearch index metricset is experimental")

	http, err := helper.NewHTTP(base)
	if err != nil {
		return nil, err
	}

	return &MetricSet{
		base,
		http,
	}, nil
}

// Fetch gathers stats for each index from the _stats API
func (m *MetricSet) Fetch(r mb.ReporterV2) {

	isMaster, err := elasticsearch.IsMaster(m.http, m.HostData().SanitizedURI)
	if err != nil {
		r.Error(err)
		return
	}

	// Not master, no event sent
	if !isMaster {
		logp.Debug("elasticsearch", "Trying to fetch index stats from a none master node.")
		return
	}

	content, err := m.http.FetchContent()
	if err != nil {
		r.Error(err)
		return
	}

	info, err := elasticsearch.GetInfo(m.http, m.HostData().SanitizedURI)
	if err != nil {
		r.Error(err)
		return
	}

	eventsMapping(r, *info, content)
}
