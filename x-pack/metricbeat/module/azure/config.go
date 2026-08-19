// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package azure

import (
	"fmt"
	"strings"
	"time"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	// DefaultBaseURI is the default URI used for the service Insights
	DefaultBaseURI = "https://management.azure.com/"

	billingDay                   = 24 * time.Hour
	defaultBillingUsageLookback  = billingDay
	defaultBillingForecastWindow = 30 * billingDay
	defaultTimeGrain             = "PT5M"
)

var (
	AzureEnvs = mapstr.M{
		"https://management.azure.com/":         "https://login.microsoftonline.com/",
		"https://management.usgovcloudapi.net/": "https://login.microsoftonline.us/",
		"https://management.chinacloudapi.cn/":  "https://login.chinacloudapi.cn/",
		"https://management.microsoftazure.de/": "https://login.microsoftonline.de/",
	}
)

func normalizeResourceManagerEndpoint(endpoint string) string {
	if endpoint == "" {
		return ""
	}
	return strings.TrimRight(endpoint, "/") + "/"
}

// Config options
type Config struct {
	// shared config options
	ClientId       string        `config:"client_id"  validate:"required"`
	ClientSecret   string        `config:"client_secret"  validate:"required"`
	TenantId       string        `config:"tenant_id"  validate:"required"`
	SubscriptionId string        `config:"subscription_id"  validate:"required"`
	Period         time.Duration `config:"period" validate:"nonzero,required"`
	// Latency is the time it takes for the Azure service to publish the metric values.
	// This is used to compensate for the latency in the timespan.
	Latency                 time.Duration `config:"latency" validate:"positive"`
	ResourceManagerEndpoint string        `config:"resource_manager_endpoint"`
	ResourceManagerAudience string        `config:"resource_manager_audience"`
	ActiveDirectoryEndpoint string        `config:"active_directory_endpoint"`
	// specific to resource metrics
	Resources           []ResourceConfig `config:"resources"`
	RefreshListInterval time.Duration    `config:"refresh_list_interval"`
	DefaultResourceType string           `config:"default_resource_type"`
	AddCloudMetadata    bool             `config:"add_cloud_metadata"`
	// specific to billing
	BillingScopeDepartment string `config:"billing_scope_department"` // retrieve usage details from department scope
	BillingScopeAccountId  string `config:"billing_scope_account_id"` // retrieve usage details from billing account ID scope
	// BillingUsageLookback controls how far back the billing metricset queries usage data.
	// It must be a positive multiple of 24h and defaults to the previous full day.
	BillingUsageLookback time.Duration `config:"billing_usage_lookback"`
	// BillingForecastWindow controls the length of the forecast period. It must be a
	// positive multiple of 24h and defaults to 720h (30 days).
	BillingForecastWindow time.Duration `config:"billing_forecast_window"`
	// Use BatchApi for metric values collection
	EnableBatchApi bool `config:"enable_batch_api"` // defaults to false
	// DefaultTimeGrain sets the default time interval when the resource config
	// doesn't specify one. If no time grain is configured, this value will be
	// used whenever possible.
	//
	// When the metric definition doesn't support this time grain, we fall back
	// to the smallest supported interval.
	//
	// Note: currently, this is only used for the storage metricset.
	DefaultTimeGrain string `config:"default_timegrain"` // defaults to PT5M
}

// InitDefaults initializes default values before configuration is unpacked.
func (c *Config) InitDefaults() {
	c.BillingUsageLookback = defaultBillingUsageLookback
	c.BillingForecastWindow = defaultBillingForecastWindow
	c.DefaultTimeGrain = defaultTimeGrain
}

func validateBillingDuration(name string, value time.Duration) error {
	if value <= 0 || value%billingDay != 0 {
		return fmt.Errorf("%s must be a positive multiple of 24h, got %s", name, value)
	}
	return nil
}

// ResourceConfig contains resource and metric list specific configuration.
type ResourceConfig struct {
	Id          []string       `config:"resource_id"`
	Group       []string       `config:"resource_group"`
	Metrics     []MetricConfig `config:"metrics"`
	Type        string         `config:"resource_type"`
	Query       string         `config:"resource_query"`
	ServiceType []string       `config:"service_type"`
}

// MetricConfig contains metric specific configuration.
type MetricConfig struct {
	Name         []string          `config:"name"`
	Namespace    string            `config:"namespace"`
	Aggregations []string          `config:"aggregations"`
	Dimensions   []DimensionConfig `config:"dimensions"`
	Timegrain    string            `config:"timegrain"`
	// namespaces can be unsupported by some resources and supported in some, this configuration option makes sure no error messages are returned if namespace is unsupported
	// info messages will be logged instead. Same situation with metrics, some are being removed from the API, we would like to make sure that does not affect the module
	IgnoreUnsupported bool `config:"ignore_unsupported"`
}

// DimensionConfig contains dimensions specific configuration.
type DimensionConfig struct {
	Name  string `config:"name"`
	Value string `config:"value"`
}

func (conf *Config) Validate() error {
	if err := validateBillingDuration("billing_usage_lookback", conf.BillingUsageLookback); err != nil {
		return err
	}
	if err := validateBillingDuration("billing_forecast_window", conf.BillingForecastWindow); err != nil {
		return err
	}
	if conf.ResourceManagerEndpoint == "" {
		conf.ResourceManagerEndpoint = DefaultBaseURI
	}
	if conf.ActiveDirectoryEndpoint == "" {
		lookupKey := normalizeResourceManagerEndpoint(conf.ResourceManagerEndpoint)
		ok, err := AzureEnvs.HasKey(lookupKey)
		if err != nil {
			return fmt.Errorf("no active directory endpoint found for the resource manager endpoint selected: %w", err)
		}
		if ok {
			add, err := AzureEnvs.GetValue(lookupKey)
			if err != nil {
				return fmt.Errorf("no active directory endpoint found for the resource manager endpoint selected: %w", err)
			}
			conf.ActiveDirectoryEndpoint, _ = add.(string)
		}
		if conf.ActiveDirectoryEndpoint == "" {
			return fmt.Errorf("no active directory endpoint has been configured")
		}
	}
	return nil
}
