// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package interfaces

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/helper/snmp"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("cisco", "interfaces", New, mb.DefaultMetricSet())
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	log  *logp.Logger
	snmp *snmp.SNMP
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	snmp, err := snmp.NewSNMP(base)
	if err != nil {
		return nil, err
	}

	log := logp.NewLogger("ciscointerfaces")

	return &MetricSet{
		BaseMetricSet: base,
		snmp:          snmp,
		log:           log,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {
	ifaces, err := m.getIfaces()
	if err != nil {
		return errors.Wrap(err, "error in SNMP request")
	}

	fields := common.MapStr{
		"message": ifaces,
	}

	reporter.Event(mb.Event{MetricSetFields: fields})

	return nil
}

func (m *MetricSet) getIfaces() (string, error) {
	m.snmp.Client.Target = m.Host()
	content, err := m.snmp.BulkWalkAll("1.3.6.1.2.1.2.2.1")
	if err != nil {
		return "", errors.Wrap(err, "error in SNMP request")
	}

	m.log.Info(content)

	return content, nil
}
