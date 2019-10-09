// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package compute_vm

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/metricbeat/module/azure"
)

const defaultVMNamespace = "Microsoft.Compute/virtualMachines"

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("azure", "compute_vm", New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	client *azure.Client
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The azure compute_vm metricset is beta.")
	var config azure.Config
	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, errors.Wrap(err, "error unpack raw module config using UnpackConfig")
	}
	if len(config.Resources) == 0 {
		config.Resources = []azure.ResourceConfig{
			{
				Query: fmt.Sprintf("resourceType eq '%s'", defaultVMNamespace),
			},
		}
	}
	for index := range config.Resources {
		// if any resource groups were configured the resource type should be added
		if len(config.Resources[index].Group) > 0 {
			config.Resources[index].Type = defaultVMNamespace
		}
		// one metric configuration will be added containing all metrics names
		config.Resources[index].Metrics = []azure.MetricConfig{
			{
				Name: []string{"*"},
			},
		}
	}
	monitorClient, err := azure.NewClient(config)
	if err != nil {
		return nil, errors.Wrap(err, "error initializing the monitor client: module azure - compute_vm metricset")
	}
	return &MetricSet{
		BaseMetricSet: base,
		client:        monitorClient,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	err := m.client.InitResources(mapMetric, report)
	if err != nil {
		return err
	}
	// retrieve metrics
	err = m.client.GetMetricValues(report)
	if err != nil {
		return err
	}
	return azure.EventsMapping(report, m.client.Resources.Metrics, m.BaseMetricSet.Name())
}
