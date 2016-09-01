// +build darwin freebsd linux openbsd

package load

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/system"
	"github.com/pkg/errors"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	if err := mb.Registry.AddMetricSet("system", "load", New); err != nil {
		panic(err)
	}
}

// MetricSet type defines all fields of the MetricSet
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type MetricSet struct {
	mb.BaseMetricSet
	counter int
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	config := struct{}{}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		counter:       1,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch() (common.MapStr, error) {

	loadStat, err := GetSystemLoad()
	if err != nil {
		return nil, errors.Wrap(err, "load statistics")
	}

	event := common.MapStr{
		"1":  system.Round(loadStat.Load1, .5, 4),
		"5":  system.Round(loadStat.Load5, .5, 4),
		"15": system.Round(loadStat.Load15, .5, 4),
		"norm": common.MapStr{
			"1":  system.Round(loadStat.LoadNorm1, .5, 4),
			"5":  system.Round(loadStat.LoadNorm5, .5, 4),
			"15": system.Round(loadStat.LoadNorm15, .5, 4),
		},
	}

	return event, nil
}
