// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package app_insights

import (
	"fmt"
	"time"

	"github.com/gofrs/uuid/v5"

	"github.com/elastic/elastic-agent-libs/logp"
)

// Client represents the azure client which calls the Application Insights
// metrics endpoint via the configured Service.
type Client struct {
	Service Service
	Config  Config
	Log     *logp.Logger
}

// NewClient instantiates an Application Insights client.
func NewClient(config Config, logger *logp.Logger) (*Client, error) {
	service, err := NewService(config, logger)
	if err != nil {
		return nil, err
	}
	return &Client{
		Service: service,
		Config:  config,
	}, nil
}

// GetMetricValues returns the configured Application Insights metric data points.
func (client *Client) GetMetricValues() (ListMetricsResultsItem, error) {
	var bodyMetrics []MetricsBatchRequestItem
	var result ListMetricsResultsItem
	for _, metrics := range client.Config.Metrics {
		metrics := metrics
		// Copy the slices so each batch entry holds its own pointer; sharing
		// the same backing array across requests would couple unrelated metric
		// configs together.
		aggregations := append([]string(nil), metrics.Aggregation...)
		segments := append([]string(nil), metrics.Segment...)
		for _, metric := range metrics.ID {
			params := MetricsBatchParameters{
				MetricID:    metric,
				Timespan:    calculateTimespan(client.Config.Period),
				Aggregation: &aggregations,
				Interval:    &metrics.Interval,
				Segment:     &segments,
				Top:         &metrics.Top,
				Orderby:     &metrics.OrderBy,
				Filter:      &metrics.Filter,
			}
			id, err := uuid.NewV4()
			if err != nil {
				return result, fmt.Errorf("could not generate identifier in client: %w", err)
			}
			strID := id.String()
			bodyMetrics = append(bodyMetrics, MetricsBatchRequestItem{ID: &strID, Parameters: &params})
		}
	}
	result, err := client.Service.GetMetricValues(client.Config.ApplicationId, bodyMetrics)
	if err != nil {
		return result, fmt.Errorf("could not retrieve app insights metrics from service: %w", err)
	}
	return result, nil
}

func calculateTimespan(duration time.Duration) *string {
	timespan := fmt.Sprintf("PT%fM", duration.Minutes())
	return &timespan
}
