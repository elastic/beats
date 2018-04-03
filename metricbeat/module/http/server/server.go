package server

import (
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	serverhelper "github.com/elastic/beats/metricbeat/helper/server"
	"github.com/elastic/beats/metricbeat/helper/server/http"
	"github.com/elastic/beats/metricbeat/mb"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	if err := mb.Registry.AddMetricSet("http", "server", New); err != nil {
		panic(err)
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
	cfgwarn.Beta("The http server metricset is beta")

	config := defaultHttpServerConfig()
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	svc, err := http.NewHttpServer(base)
	if err != nil {
		return nil, err
	}

	processor := NewMetricProcessor(config.Paths, config.DefaultPath)
	return &MetricSet{
		BaseMetricSet: base,
		server:        svc,
		processor:     processor,
	}, nil
}

// Run method provides the Graphite server with a reporter with which events can be reported.
func (m *MetricSet) Run(reporter mb.PushReporter) {
	// Start event watcher
	m.server.Start()

	for {
		select {
		case <-reporter.Done():
			m.server.Stop()
			return
		case msg := <-m.server.GetEvents():
			event, err := m.processor.Process(msg)
			if err != nil {
				reporter.Error(err)
			} else {
				reporter.Event(event)
			}

		}
	}
}
