// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package app_insights

import (
	"fmt"
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb/parse"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/logp"
)

const metricsetName = "app_insights"

// Config options
type Config struct {
	ApplicationId string        `config:"application_id" validate:"required"`
	Period        time.Duration `config:"period" validate:"nonzero,required"`
	Metrics       []Metric      `config:"metrics" validate:"required"`
	Namespace     string        `config:"namespace"`

	// API Key authentication
	ApiKey string `config:"api_key"`

	// OAuth2 authentication
	TenantId                string `config:"tenant_id"`
	ClientId                string `config:"client_id"`
	ClientSecret            string `config:"client_secret"`
	ActiveDirectoryEndpoint string `config:"active_directory_endpoint"`
}

// Validate checks that exactly one authentication method is configured.
func (c *Config) Validate() error {
	hasOAuth2 := c.TenantId != "" && c.ClientId != "" && c.ClientSecret != ""
	hasPartialOAuth2 := (c.TenantId != "" || c.ClientId != "" || c.ClientSecret != "") && !hasOAuth2
	hasAPIKey := c.ApiKey != ""

	if hasPartialOAuth2 {
		return fmt.Errorf("incomplete MSI/MSEntra authentication configuration: tenant_id, client_id, and client_secret must all be provided")
	}

	if hasOAuth2 && hasAPIKey {
		return fmt.Errorf("only one authentication method can be configured: use either OAuth2 (tenant_id, client_id, client_secret) or api_key")
	}

	if !hasOAuth2 && !hasAPIKey {
		return fmt.Errorf("no MSI/MSEntra authentication configuration or api_key was provided")
	}

	return nil
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
	client, err := NewClient(config, base.Logger())
	if err != nil {
		return nil, fmt.Errorf("error initializing the monitor client: module azure - %s metricset: %w", metricsetName, err)
	}
	return &MetricSet{
		BaseMetricSet: base,
		log:           base.Logger().Named(metricsetName),
		client:        client,
	}, nil
}

// Fetch fetches events and reports them upstream
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	results, err := m.client.GetMetricValues()
	if err != nil {
		return fmt.Errorf("error retrieving metric values: %w", err)
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
