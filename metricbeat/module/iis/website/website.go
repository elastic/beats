package website

import (
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/iis"
	"github.com/elastic/beats/metricbeat/module/windows/perfmon"
	"github.com/pkg/errors"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("iis", "website", New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	reader *perfmon.PerfmonReader
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Experimental("The iis website metricset is experimental.")

	config := iis.InitConfig("website")

	reader, err := perfmon.NewPerfmonReader(config)
	if err != nil {
		return nil, errors.Wrap(err, "initialization of reader failed")
	}

	return &MetricSet{
		BaseMetricSet: base,
		reader : reader,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) {
	events, err := m.reader.Read()
	if err != nil {
		err = errors.Wrap(err, "failed reading counters")
		report.Error(err)
	}
	eventsMapping(report, events)
}
