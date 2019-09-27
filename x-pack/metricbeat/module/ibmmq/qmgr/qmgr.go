// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package qmgr

import (
	"os"
	"plugin"
	"encoding/json"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common/cfgwarn"

	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/metricbeat/module/ibmmq"
)

var (
	CollectQmgrMetricset func(eventType string, qmgrName string, ccPacked []byte) ([]beat.Event, error)
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	p, err := plugin.Open(os.Getenv("IBM_LIBRARY_PATH"))
	if err != nil {
		return
	}

	collectQmgrMetricset, err := p.Lookup("CollectQmgrMetricset")
	if err != nil {
		panic(err)
	}
	CollectQmgrMetricset = collectQmgrMetricset.(func(eventType string, qmgrName string, ccPacked []byte) ([]beat.Event, error))

	mb.Registry.MustAddMetricSet("ibmmq", "qmgr", New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	queueManager     string
	connectionConfig ibmmq.ConnectionConfig
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Experimental("The ibmmq qmgr metricset is experimental.")

	config := ibmmq.DefaultConfig
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet:    base,
		queueManager:     config.QueueManager,
		connectionConfig: config.ConnectionConfig,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) {

	configData, err := json.Marshal(m.connectionConfig)
	if err!=nil {
		panic(err)
	}
	events, _ := CollectQmgrMetricset("QueueManager", m.queueManager, configData)

	for _, beatEvent := range events {
		var mbEvent mb.Event
		mbEvent.MetricSetFields = beatEvent.Fields
		report.Event(mbEvent)
	}

}
