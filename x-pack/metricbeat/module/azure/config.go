// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"time"

	"github.com/pkg/errors"
)

// Config options
type Config struct {
	ClientId            string           `config:"client_id"`
	ClientSecret        string           `config:"client_secret"`
	TenantId            string           `config:"tenant_id"`
	SubscriptionId      string           `config:"subscription_id"`
	Period              time.Duration    `config:"period" validate:"nonzero,required"`
	Resources           []ResourceConfig `config:"resources"`
	RefreshListInterval time.Duration    `config:"refresh_list_interval"`
	DefaultResourceType string           `config:"default_resource_type"`
	AddCloudMetadata    bool             `config:"add_cloud_metadata"`
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
	// info messages will be logged instead
	IgnoreUnsupported bool `config:"ignore_unsupported"`
}

// DimensionConfig contains dimensions specific configuration.
type DimensionConfig struct {
	Name  string `config:"name"`
	Value string `config:"value"`
}

func (conf *Config) Validate() error {
	if conf.SubscriptionId == "" {
		return errors.New("no subscription ID has been configured")
	}
	if conf.ClientSecret == "" {
		return errors.New("no client secret has been configured")
	}
	if conf.ClientId == "" {
		return errors.New("no client ID has been configured")
	}
	if conf.TenantId == "" {
		return errors.New("no tenant ID has been configured")
	}
	return nil
}
