// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package billing

import (
	"fmt"
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	// defaultUsageLookback is the usage query window when billing_usage_lookback is unset.
	// It matches the original hardcoded behaviour: query the single previous full UTC day.
	defaultUsageLookback = 24 * time.Hour

	// defaultForecastWindow is the forecast query window when billing_forecast_window is unset.
	// It matches the original hardcoded behaviour: 30 days forward from the forecast start date.
	defaultForecastWindow = 30 * 24 * time.Hour
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
	applyBillingDefaults(&config)
	// instantiate monitor client
	billingClient, err := NewClient(config, base.Logger())
	if err != nil {
		return nil, fmt.Errorf("error initializing the billing client: module azure - billing metricset: %w", err)
	}
	return &MetricSet{
		BaseMetricSet: base,
		client:        billingClient,
		log:           base.Logger().Named("azure billing"),
	}, nil
}

// applyBillingDefaults fills in default values for billing-specific duration
// config fields that were not explicitly set by the user.
func applyBillingDefaults(cfg *azure.Config) {
	if cfg.BillingUsageLookback == 0 {
		cfg.BillingUsageLookback = defaultUsageLookback
	}
	if cfg.BillingForecastWindow == 0 {
		cfg.BillingForecastWindow = defaultForecastWindow
	}
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

	usageStart, usageEnd := usageIntervalFrom(referenceTime, m.client.Config.BillingUsageLookback)
	forecastStart, forecastEnd := forecastIntervalFrom(referenceTime, m.client.Config.BillingForecastWindow)

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

// usageIntervalFrom returns the start/end times (UTC) of the usage period given the
// reference time and a lookback duration.
//
// The usage period ends at start-of-today minus one second (i.e. yesterday 23:59:59 UTC)
// and starts lookback duration before that.
//
// For example, with reference 2007-01-09 09:41:00Z and a 24h lookback, the usage period is:
//
//	2007-01-08 00:00:00Z -> 2007-01-08 23:59:59Z
//
// With a 72h lookback it covers the three previous full days:
//
//	2007-01-06 00:00:00Z -> 2007-01-08 23:59:59Z
func usageIntervalFrom(reference time.Time, lookback time.Duration) (time.Time, time.Time) {
	startOfToday := reference.UTC().Truncate(24 * time.Hour)
	return startOfToday.Add(-lookback), startOfToday.Add(-time.Second)
}

// forecastIntervalFrom returns the start/end times (UTC) of the forecast period given the
// reference time and a window duration.
//
// The forecast period always starts at reference minus 2 days (00:00:00 UTC) and extends
// forward for the given window duration.
//
// For example, with reference 2007-01-09 09:41:00Z and a 30-day (720h) window:
//
//	2007-01-07T00:00:00Z -> 2007-02-05T23:59:59Z
func forecastIntervalFrom(reference time.Time, window time.Duration) (time.Time, time.Time) {
	forecastStart := reference.UTC().Truncate(24 * time.Hour).Add(-48 * time.Hour)
	forecastEnd := forecastStart.Add(window).Add(-time.Second)
	return forecastStart, forecastEnd
}
