// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package container

import (
	"fmt"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/cloudfoundry"

	"github.com/elastic/beats/v7/metricbeat/mb"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	mb.Registry.MustAddMetricSet("cloudfoundry", "container", New, mb.DefaultMetricSet())
}

// MetricSet type defines all fields of the MetricSet
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type MetricSet struct {
	mb.BaseMetricSet

	mod *cloudfoundry.Module
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	mod, ok := base.Module().(*cloudfoundry.Module)
	if !ok {
		return nil, fmt.Errorf("must be child of cloudfoundry module")
	}
	return &MetricSet{base, mod}, nil
}

// Run method provides the module with a reporter with which events can be reported.
func (m *MetricSet) Run(reporter mb.PushReporterV2) {
	m.mod.RunContainerReporter(reporter)
}
