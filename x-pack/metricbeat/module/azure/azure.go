// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/metricbeat/mb"
)

// Config options
type Config struct {
	ClientID            string           `config:"client_id"    validate:"required"`
	ClientSecret        string           `config:"client_secret" validate:"required"`
	TenantID            string           `config:"tenant_id" validate:"required"`
	SubscriptionID      string           `config:"subscription_id" validate:"required"`
	Period              time.Duration    `config:"period" validate:"nonzero,required"`
	Resources           []ResourceConfig `config:"resources"`
	RefreshListInterval time.Duration    `config:"refresh_list_interval"`
}

// ResourceConfig contains resource and metric list specific configuration.
type ResourceConfig struct {
	ID      string         `config:"resource_id"`
	Group   string         `config:"resource_group"`
	Metrics []MetricConfig `config:"metrics"`
	Type    string         `config:"resource_type"`
	Query   string         `config:"resource_query"`
}

// MetricConfig contains metric specific configuration.
type MetricConfig struct {
	Name         []string          `config:"name"`
	Namespace    string            `config:"namespace"`
	Aggregations []string          `config:"aggregations"`
	Dimensions   []DimensionConfig `config:"dimensions"`
	Timegrain    string            `config:"timegrain"`
}

// DimensionConfig contains dimensions specific configuration.
type DimensionConfig struct {
	Name  string `config:"name"`
	Value string `config:"value"`
}

func init() {
	// Register the ModuleFactory function for the "azure" module.
	if err := mb.Registry.AddModule("azure", newModule); err != nil {
		panic(err)
	}
}

// newModule adds validation that hosts is non-empty, a requirement to use the
// azure module.
func newModule(base mb.BaseModule) (mb.Module, error) {
	var config Config
	if err := base.UnpackConfig(&config); err != nil {
		return nil, errors.Wrap(err, "error unpack raw module config using UnpackConfig")
	}
	return &base, nil
}
