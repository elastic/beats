// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package server

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
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
	ReservoirSize int           `config:"reservoir_size"`
	TTL           time.Duration `config:"ttl"`
}

func defaultConfig() Config {
	return Config{
		ReservoirSize: 1000,
		TTL:           time.Second * 30,
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

	processor := newMetricProcessor(config.ReservoirSize, config.TTL)
	return &MetricSet{
		BaseMetricSet: base,
		server:        svc,
		processor:     processor,
	}, nil
}

func (m *MetricSet) getEvents() []*mb.Event {
	groups := m.processor.GetAll()
	events := make([]*mb.Event, len(groups))

	for idx, tagGroup := range groups {

		mapstrTags := common.MapStr{}
		for k, v := range tagGroup.tags {
			mapstrTags[k] = v
		}

		sanitizedMetrics := common.MapStr{}
		for k, v := range tagGroup.metrics {
			sanitizedMetrics[common.DeDot(k)] = v
		}

		events[idx] = &mb.Event{
			MetricSetFields: sanitizedMetrics,
			RootFields:      common.MapStr{"labels": mapstrTags},
			Namespace:       "statsd",
		}
	}
	return events
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
			for _, e := range m.getEvents() {
				reporter.Event(*e)
			}
		case msg := <-m.server.GetEvents():
			err := m.processor.Process(msg)
			if err != nil {
				reporter.Error(err)
			}
		}
	}
}
