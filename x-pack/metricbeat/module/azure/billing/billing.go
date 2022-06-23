// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"fmt"
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure"
	"github.com/elastic/elastic-agent-libs/logp"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("azure", "billing", New, mb.WithHostParser(parse.EmptyHostParser))
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	client *Client
	log    *logp.Logger
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	var config azure.Config
	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, fmt.Errorf("error unpack raw module config using UnpackConfig: %w", err)
	}
	if err != nil {
		return nil, err
	}
	// instantiate monitor client
	billingClient, err := NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("error initializing the billing client: module azure - billing metricset: %w", err)
	}
	return &MetricSet{
		BaseMetricSet: base,
		client:        billingClient,
		log:           logp.NewLogger("azure billing"),
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right metricset
// It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	// The time interval for the Usage Details data is yesterday (00:00:00->23:59:59) in UTC.
	startTime := time.Now().UTC().Truncate(24 * time.Hour).Add((-24) * time.Hour)
	endTime := startTime.Add(time.Hour * 24).Add(time.Second * (-1))

	m.log.Infof("Fetching billing data for period: %s to %s", startTime, endTime)

	results, err := m.client.GetMetrics(startTime, endTime)
	if err != nil {
		return fmt.Errorf("error retrieving usage information: %w", err)
	}

	events := EventsMapping(m.client.Config.SubscriptionId, results, startTime, endTime)
	for _, event := range events {
		isOpen := report.Event(event)
		if !isOpen {
			break
		}
	}

	return nil
}
