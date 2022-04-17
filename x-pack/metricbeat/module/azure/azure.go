// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/metricbeat/mb"
)

func init() {
	// Register the ModuleFactory function for the "azure" module.
	if err := mb.Registry.AddModule("azure", newModule); err != nil {
		panic(err)
	}
}

// newModule adds validation that hosts is non-empty, a requirement to use the
// azure module.
func newModule(base mb.BaseModule) (mb.Module, error) {
	return &base, nil
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	Client     *Client
	MapMetrics mapResourceMetrics
}

// NewMetricSet will instantiate a new azure metricset
func NewMetricSet(base mb.BaseMetricSet) (*MetricSet, error) {
	metricsetName := base.Name()
	var config Config
	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, errors.Wrap(err, "error unpack raw module config using UnpackConfig")
	}

	//validate config based on metricset
	switch metricsetName {
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
		// check for lightweight resources if no groups or ids have been entered, if not a new resource is created to check the entire subscription
		var resources []ResourceConfig
		for _, resource := range config.Resources {
			if hasConfigOptions(resource.Group) || hasConfigOptions(resource.Id) {
				resources = append(resources, resource)
			}
		}
		// check if this is a light metricset or not and no resources have been configured
		if len(resources) == 0 && len(config.Resources) != 0 {
			resources = append(resources, ResourceConfig{
				Query:   fmt.Sprintf("resourceType eq '%s'", config.DefaultResourceType),
				Metrics: config.Resources[0].Metrics,
			})
		}
		config.Resources = resources
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

// Fetch methods implements the data gathering and data conversion to the right metricset
// It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	err := m.Client.InitResources(m.MapMetrics)
	if err != nil {
		return err
	}
	if len(m.Client.ResourceConfigurations.Metrics) == 0 {
		// error message is previously logged in the InitResources, no error event should be created
		return nil
	}
	// retrieve metrics
	groupedMetrics := groupMetricsByResource(m.Client.ResourceConfigurations.Metrics)

	for _, metrics := range groupedMetrics {
		results := m.Client.GetMetricValues(metrics, report)
		err := EventsMapping(results, m.Client, report)
		if err != nil {
			return errors.Wrap(err, "error running EventsMapping")
		}
	}
	return nil
}

// hasConfigOptions func will check if any resource id or resource group options have been entered in the light metricsets
func hasConfigOptions(config []string) bool {
	if config == nil {
		return false
	}
	for _, group := range config {
		if group == "" {
			return false
		}
	}
	return true
}
