// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package info

import (
	"errors"

	"github.com/elastic/beats/v9/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v9/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host is defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("benchmark", "info", New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	counter        uint
	eventsPerFetch uint
	degrade        bool
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	base.Logger().Warn(cfgwarn.Beta("The benchmark info metricset is beta."))

	config := defaultConfig
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet:  base,
		counter:        1,
		eventsPerFetch: config.Count,
		degrade:        config.Degrade,
	}, nil
}

// Fetch method implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	for i := uint(0); i < m.eventsPerFetch; i++ {
		report.Event(mb.Event{
			MetricSetFields: mapstr.M{
				"counter": m.counter,
			},
		})
		m.counter++
	}

	if m.degrade {
		return errors.New("benchmark input degraded")
	}

	return nil
}
