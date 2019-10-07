// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"time"

	"github.com/elastic/beats/libbeat/common/cfgwarn"

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
	ID      []string       `config:"resource_id"`
	Group   []string       `config:"resource_group"`
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

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	Client *Client
}

// NewMetricSet will instantiate a new azure metricset
func NewMetricSet(base mb.BaseMetricSet) (*MetricSet, error) {
	metricsetName := base.Name()
	cfgwarn.Beta("The azure %s metricset is beta.", metricsetName)
	var config Config
	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, errors.Wrap(err, "error unpack raw module config using UnpackConfig")
	}

	//validate config based on metricset
	switch base.Name() {
	case nativeMetricset:
		// resources must be configured for the monitor metricset
		if len(config.Resources) == 0 {
			return nil, errors.Errorf("no resource options defined: module azure - %s metricset", metricsetName)
		}
	default:
		// validate config resource options entered, no resource queries allowed for the compute_vm and compute_vm_scaleset metricsets
		for _, resource := range config.Resources {
			if resource.Query != "" {
				return nil, errors.Errorf("error initializing the monitor client: module azure - %s metricset. No queries allowed, please select one of the allowed options", metricsetName)
			}
		}

	}
	// instantiate monitor client
	monitorClient, err := NewClient(config)
	if err != nil {
		return nil, errors.Wrapf(err, "error initializing the monitor client: module azure - %s metricset", metricsetName)
	}
	return &MetricSet{
		BaseMetricSet: base,
		Client:        monitorClient,
	}, nil
}

// FetchValues methods implements the data gathering and data conversion to the right metricset
func (m *MetricSet) FetchValues(fn mapMetric, report mb.ReporterV2) error {
	err := m.Client.InitResources(fn, report)
	if err != nil {
		return err
	}
	// retrieve metrics
	err = m.Client.GetMetricValues(report)
	if err != nil {
		return err
	}
	return EventsMapping(report, m.Client.Resources.Metrics, m.BaseMetricSet.Name())
}
