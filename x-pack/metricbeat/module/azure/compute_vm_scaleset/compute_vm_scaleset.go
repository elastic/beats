// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package compute_vm_scaleset

import (
	"fmt"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("azure", "compute_vm_scaleset", New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	*azure.MetricSet
}

const (
	defaultVMScalesetNamespace = "Microsoft.Compute/virtualMachineScaleSets"
	customVMNamespace          = "Azure.VM.Windows.GuestMetrics"
)

var memoryMetrics = []string{"Memory\\Commit Limit", "Memory\\Committed Bytes", "Memory\\% Committed Bytes In Use", "Memory\\Available Bytes"}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := azure.NewMetricSet(base)
	if err != nil {
		return nil, err
	}
	// if no options are entered we will retrieve all the vm's from the entire subscription
	if len(ms.Client.Config.Resources) == 0 {
		ms.Client.Config.Resources = []azure.ResourceConfig{
			{
				Query: fmt.Sprintf("resourceType eq '%s'", defaultVMScalesetNamespace),
			},
		}
	}
	for index := range ms.Client.Config.Resources {
		// add the default vm scaleset type if groups are defined
		if len(ms.Client.Config.Resources[index].Group) > 0 {
			ms.Client.Config.Resources[index].Type = defaultVMScalesetNamespace
		}
		// add the default metrics for each resource option
		ms.Client.Config.Resources[index].Metrics = []azure.MetricConfig{
			{
				Name:      []string{"*"},
				Namespace: defaultVMScalesetNamespace,
			},
			{
				Name:      memoryMetrics,
				Namespace: customVMNamespace,
			},
		}
	}
	ms.MapMetrics = mapMetrics
	return &MetricSet{
		MetricSet: ms,
	}, nil
}
