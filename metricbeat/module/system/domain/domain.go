package domain

import (
	"github.com/StackExchange/wmi"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("system", "domain", New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	registeredDomain []queryKey
}

type queryKey struct {
	Domain string
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The system domain metricset is beta.")

	config := struct{}{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	var newQuery = []queryKey{}

	return &MetricSet{
		BaseMetricSet:    base,
		registeredDomain: newQuery,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	var dst []queryKey
	wmi.Query("Select * from Win32_ComputerSystem", &dst)
	for _, v := range dst {
		m.registeredDomain = append(m.registeredDomain, queryKey{Domain: v.Domain})
	}
	rootFields := common.MapStr{}
	for _, item := range m.registeredDomain {
		rootFields["registeredDomain"] = item.Domain
	}

	var event mb.Event
	event.MetricSetFields = rootFields
	report.Event(event)
	return nil
}
