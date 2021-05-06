package syncgateway

import (
	"github.com/elastic/beats/v7/metricbeat/helper"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	"github.com/pkg/errors"
)

const (
	defaultScheme = "http"
	defaultPath   = "/_expvar"
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
	mb.Registry.MustAddMetricSet("couchbase", "syncgateway", New,
		mb.WithHostParser(hostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	mb.BaseMetricSet
	http       *helper.HTTP
	extraStats ExtraStats
}

type ExtraStats struct {
	PerReplication bool `config:"per_replication"`
	MemStats       bool `config:"mem_stats"`
}

// New create a new instance of the MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := struct {
		Extra ExtraStats
	}{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	http, err := helper.NewHTTP(base)
	if err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		http:          http,
		extraStats:    config.Extra,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {
	content, err := m.http.FetchContent()
	if err != nil {
		return errors.Wrap(err, "error in fetch")
	}

	err = eventMapping(reporter, content, m.extraStats.PerReplication, m.extraStats.MemStats)

	return err
}
