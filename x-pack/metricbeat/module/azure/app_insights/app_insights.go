// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package app_insights

import (
	"time"

	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/metricbeat/mb/parse"

	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/metricbeat/mb"
)

const metricsetName = "app_insights"

// Config options
type Config struct {
	ApplicationId string        `config:"application_id"    validate:"required"`
	ApiKey        string        `config:"api_key" validate:"required"`
	Period        time.Duration `config:"period" validate:"nonzero,required"`
	Metrics       []Metric      `config:"metrics" validate:"required"`
	Namespace     string        `config:"namespace"`
}

// Metric struct used for configuration options
type Metric struct {
	ID          []string `config:"id" validate:"required"`
	Interval    string   `config:"interval"`
	Aggregation []string `config:"aggregation"`
	Segment     []string `config:"segment"`
	Top         int32    `config:"top"`
	OrderBy     string   `config:"order_by"`
	Filter      string   `config:"filter"`
}

func init() {
	mb.Registry.MustAddMetricSet("azure", metricsetName, New, mb.WithHostParser(parse.EmptyHostParser))
}

// MetricSet struct used for app insights.
type MetricSet struct {
	mb.BaseMetricSet
	log    *logp.Logger
	client *Client
}

// New creates a new instance of the MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	var config Config
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}
	client, err := NewClient(config)
	if err != nil {
		return nil, errors.Wrapf(err, "error initializing the monitor client: module azure - %s metricset", metricsetName)
	}
	return &MetricSet{
		BaseMetricSet: base,
		log:           logp.NewLogger(metricsetName),
		client:        client,
	}, nil
}

// Fetch fetches events and reports them upstream
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	results, err := m.client.GetMetricValues()
	if err != nil {
		return errors.Wrap(err, "error retrieving metric values")
	}
	events := EventsMapping(results, m.client.Config.ApplicationId, m.client.Config.Namespace)
	for _, event := range events {
		isOpen := report.Event(event)
		if !isOpen {
			break
		}
	}
	return nil
}
