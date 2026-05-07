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

const (
	metricsetName = "app_insights"

	// AuthTypeAPIKey uses API key authentication (default for backwards compatibility).
	AuthTypeAPIKey string = "api_key"
	// AuthTypeClientSecret uses client secret credentials (Microsoft Entra ID).
	AuthTypeClientSecret string = "client_secret"
)

// Config options
type Config struct {
	ApplicationId string        `config:"application_id" validate:"required"`
	Period        time.Duration `config:"period" validate:"nonzero,required"`
	Metrics       []Metric      `config:"metrics" validate:"required"`
	Namespace     string        `config:"namespace"`

	// AuthType specifies the authentication method.
	// Valid values: api_key (default), client_secret.
	AuthType string `config:"auth_type"`

	// API key authentication
	ApiKey string `config:"api_key"`

	// Client secret authentication (Microsoft Entra ID)
	TenantId     string `config:"tenant_id"`
	ClientId     string `config:"client_id"`
	ClientSecret string `config:"client_secret"`
}

// Validate checks that the authentication configuration is complete.
func (c *Config) Validate() error {
	if c.AuthType == "" {
		c.AuthType = AuthTypeAPIKey
	}

	switch c.AuthType {
	case AuthTypeAPIKey:
		return c.validateAPIKeyAuth()
	case AuthTypeClientSecret:
		return c.validateClientSecretAuth()
	default:
		return fmt.Errorf("unknown auth_type: %s (valid values: %s, %s)", c.AuthType, AuthTypeAPIKey, AuthTypeClientSecret)
	}
}

func (c *Config) validateAPIKeyAuth() error {
	if c.ApiKey == "" {
		return fmt.Errorf("api_key is required when auth_type is %s", AuthTypeAPIKey)
	}
	return nil
}

func (c *Config) validateClientSecretAuth() error {
	if c.TenantId == "" {
		return fmt.Errorf("tenant_id is required when auth_type is %s", AuthTypeClientSecret)
	}
	if c.ClientId == "" {
		return fmt.Errorf("client_id is required when auth_type is %s", AuthTypeClientSecret)
	}
	if c.ClientSecret == "" {
		return fmt.Errorf("client_secret is required when auth_type is %s", AuthTypeClientSecret)
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
