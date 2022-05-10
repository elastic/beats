// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package app_insights

import (
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"github.com/Azure/azure-sdk-for-go/services/preview/appinsights/v1/insights"
	"github.com/pkg/errors"

	"github.com/elastic/elastic-agent-libs/logp"
)

// Client represents the azure client which will make use of the azure sdk go metrics related clients
type Client struct {
	Service Service
	Config  Config
	Log     *logp.Logger
}

// NewClient instantiates the an Azure monitoring client
func NewClient(config Config) (*Client, error) {
	service, err := NewService(config)
	if err != nil {
		return nil, err
	}
	client := &Client{
		Service: service,
		Config:  config,
	}
	return client, nil
}

// GetMetricValues returns the specified app insights metric data points.
func (client *Client) GetMetricValues() (insights.ListMetricsResultsItem, error) {
	var bodyMetrics []insights.MetricsPostBodySchema
	var result insights.ListMetricsResultsItem
	for _, metrics := range client.Config.Metrics {
		var aggregations []insights.MetricsAggregation
		var segments []insights.MetricsSegment
		for _, agg := range metrics.Aggregation {
			aggregations = append(aggregations, insights.MetricsAggregation(agg))
		}
		for _, seg := range metrics.Segment {
			segments = append(segments, insights.MetricsSegment(seg))
		}
		for _, metric := range metrics.ID {
			bodyMetric := insights.MetricsPostBodySchemaParameters{
				MetricID:    insights.MetricID(metric),
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
				return result, errors.Wrap(err, "could not generate identifier in client")
			}
			strId := id.String()
			bodyMetrics = append(bodyMetrics, insights.MetricsPostBodySchema{ID: &strId, Parameters: &bodyMetric})
		}
	}
	result, err := client.Service.GetMetricValues(client.Config.ApplicationId, bodyMetrics)
	if err == nil {
		return result, nil
	}
	return result, errors.Wrap(err, "could not retrieve app insights metrics from service")
}

func calculateTimespan(duration time.Duration) *string {
	timespan := fmt.Sprintf("PT%fM", duration.Minutes())
	return &timespan
}
