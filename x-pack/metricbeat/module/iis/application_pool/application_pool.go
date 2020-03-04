// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build windows

package application_pool

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("iis", "application_pool", New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	log    *logp.Logger
	reader *Reader
}

// Config for the iis website metricset.
type Config struct {
	Names []string `config:"application_pool.name"`
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The iis application_pool metricset is beta.")
	var config Config
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}
	// instantiate reader object
	reader, err := newReader()
	if err != nil {
		return nil, err
	}
	ms := &MetricSet{
		BaseMetricSet: base,
		log:           logp.NewLogger("application pool"),
		reader:        reader,
	}
	if err := ms.reader.initCounters(config.Names); err != nil {
		return ms, err
	}
	return ms, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	var config Config
	if err := m.Module().UnpackConfig(&config); err != nil {
		return nil
	}

	events, err := m.reader.fetch(config.Names)
	if err != nil {
		return errors.Wrap(err, "failed reading counters")
	}

	for _, event := range events {
		isOpen := report.Event(event)
		if !isOpen {
			break
		}
	}
	return nil
}

// Close will be called when metricbeat is stopped, should close the query.
func (m *MetricSet) Close() error {
	err := m.reader.close()
	if err != nil {
		return errors.Wrap(err, "failed to close pdh query")
	}
	return nil
}
