// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package routing

import (
	"errors"
	"strings"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/panos"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	metricsetName = "routing"
	vsys          = ""
)

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	config *panos.Config
	logger *logp.Logger
	client panos.PanosClient
}

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host is defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet(panos.ModuleName, metricsetName, New)
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The panos routing metricset is beta.")

	config, err := panos.NewConfig(base)
	if err != nil {
		return nil, err
	}

	logger := logp.NewLogger(base.FullyQualifiedName())

	client, err := panos.GetPanosClient(config)
	if err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		config:        config,
		logger:        logger,
		client:        client,
	}, nil
}

// Fetch method implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	// accumulate errs and report them all at the end so that we don't
	// stop processing events if one of the fetches fails
	var errs []string

	bgpEvents, err := getBGPEvents(m)
	if err != nil {
		m.logger.Error("Error get BGP events: %s", err)
		errs = append(errs, err.Error())
	}

	for _, event := range bgpEvents {
		report.Event(event)
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	} else {
		return nil
	}
}
