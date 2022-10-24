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
// mb.BaseMetricSet because it implements all the required mb.MetricSet
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

// TimeIntervalOptions represents the options used to retrieve the billing data.
type TimeIntervalOptions struct {
	// Usage details start time (UTC).
	usageStart time.Time
	// Usage details end time (UTC).
	usageEnd time.Time
	// Forecast start time (UTC).
	forecastStart time.Time
	// Forecast end time (UTC).
	forecastEnd time.Time
}

// Fetch methods implements the data gathering and data conversion to the right metricset
// It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	// reference time used to calculate usage and forecast time intervals.
	referenceTime := time.Now()

	usageStart, usageEnd := usageIntervalFrom(referenceTime)
	forecastStart, forecastEnd := forecastIntervalFrom(referenceTime)

	timeIntervalOptions := TimeIntervalOptions{
		usageStart:    usageStart,
		usageEnd:      usageEnd,
		forecastStart: forecastStart,
		forecastEnd:   forecastEnd,
	}

	m.log.
		With("billing.reference_time", referenceTime).
		Infow("Fetching billing data")

	results, err := m.client.GetMetrics(timeIntervalOptions)
	if err != nil {
		return fmt.Errorf("error retrieving usage information: %w", err)
	}

	events, err := EventsMapping(m.client.Config.SubscriptionId, results, timeIntervalOptions, m.log)
	if err != nil {
		return fmt.Errorf("error mapping events: %w", err)
	}

	for _, event := range events {
		isOpen := report.Event(event)
		if !isOpen {
			break
		}
	}

	return nil
}

// usageIntervalFrom returns the start/end times (UTC) of the usage period given the `reference` time.
//
// Currently, the usage period is the start/end time (00:00:00->23:59:59 UTC) of the day before the reference time.
//
// For example, if the reference time is 2007-01-09 09:41:00Z, the usage period is:
//
//	2007-01-08 00:00:00Z -> 2007-01-08 23:59:59Z
func usageIntervalFrom(reference time.Time) (time.Time, time.Time) {
	beginningOfDay := reference.UTC().Truncate(24 * time.Hour).Add((-24) * time.Hour)
	endOfDay := beginningOfDay.Add(time.Hour * 24).Add(time.Second * (-1))
	return beginningOfDay, endOfDay
}

// forecastIntervalFrom returns the start/end times (UTC) of the forecast period, given the `reference` time.
//
// Currently, the forecast period is the start/end times (00:00:00->23:59:59 UTC) of the current month relative to the
// reference time.
//
// For example, if the reference time is 2007-01-09 09:41:00Z, the forecast period is:
//
//	2007-01-01T00:00:00Z -> 2007-01-31:59:59Z
func forecastIntervalFrom(reference time.Time) (time.Time, time.Time) {
	referenceUTC := reference.UTC()
	beginningOfMonth := time.Date(referenceUTC.Year(), referenceUTC.Month(), 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := beginningOfMonth.AddDate(0, 1, 0).Add(-1 * time.Second)
	return beginningOfMonth, endOfMonth
}
