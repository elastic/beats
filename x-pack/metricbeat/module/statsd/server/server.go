// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package server

import (
	"time"

	serverhelper "github.com/elastic/beats/metricbeat/helper/server"
	"github.com/elastic/beats/metricbeat/helper/server/udp"
	"github.com/elastic/beats/metricbeat/mb"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	mb.Registry.MustAddMetricSet("statsd", "server", New, mb.DefaultMetricSet())
}

// Config for the statsd server metricset.
type Config struct {
	ReservoirSize int `config:"reservoir_size"`
}

func defaultConfig() Config {
	return Config{
		ReservoirSize: 1000,
	}
}

// MetricSet type defines all fields of the MetricSet
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type MetricSet struct {
	mb.BaseMetricSet
	server    serverhelper.Server
	processor *metricProcessor
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := defaultConfig()
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	svc, err := udp.NewUdpServer(base)
	if err != nil {
		return nil, err
	}

	processor := newMetricProcessor(config.ReservoirSize)
	return &MetricSet{
		BaseMetricSet: base,
		server:        svc,
		processor:     processor,
	}, nil
}

// Run method provides the module with a reporter with which events can be reported.
func (m *MetricSet) Run(reporter mb.PushReporterV2) {
	period := m.Module().Config().Period

	// Start event watcher
	m.server.Start()
	reportPeriod := time.After(period)
	for {
		select {
		case <-reporter.Done():
			m.server.Stop()
			return
		case <-reportPeriod:
			reportPeriod = time.After(period)
			event := mb.Event{
				MetricSetFields: m.processor.GetAll(),
				Namespace:       "statsd",
			}
			reporter.Event(event)
		case msg := <-m.server.GetEvents():
			err := m.processor.Process(msg)
			if err != nil {
				reporter.Error(err)
			}
		}
	}
}
