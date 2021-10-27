// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package container

import (
	"fmt"
	"math"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/cloudfoundry"

	"github.com/elastic/beats/v7/libbeat/logp"
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

	mod cloudfoundry.Module
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	mod, ok := base.Module().(cloudfoundry.Module)
	if !ok {
		return nil, fmt.Errorf("must be child of cloudfoundry module")
	}
	return &MetricSet{base, mod}, nil
}

// Run method provides the module with a reporter with which events can be reported.
func (m *MetricSet) Run(reporter mb.PushReporterV2) {
	m.mod.RunContainerReporter(&containerReporter{reporter, m.Logger()})
}

type containerReporter struct {
	mb.PushReporterV2

	logger *logp.Logger
}

func (r *containerReporter) Event(event mb.Event) bool {
	cpuPctKey := "cloudfoundry.container.cpu.pct"
	value, err := event.RootFields.GetValue(cpuPctKey)
	if err != nil {
		r.logger.Debugf("Unexpected failure while checking for non-numeric values: %v", err)
	} else {
		if value, ok := value.(float64); ok {
			if math.IsNaN(value) || math.IsInf(value, 0) {
				event.RootFields.Delete(cpuPctKey)
			} else {
				event.RootFields.Put(cpuPctKey, value/100)
			}
		}
	}
	return r.PushReporterV2.Event(event)
}
