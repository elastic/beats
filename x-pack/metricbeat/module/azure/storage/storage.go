// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package storage

import (
	"fmt"

	"github.com/menderesk/beats/v7/metricbeat/mb"
	"github.com/menderesk/beats/v7/x-pack/metricbeat/module/azure"
)

const defaultStorageAccountNamespace = "Microsoft.Storage/storageAccounts"

var (
	storageServiceNamespaces = []string{"/blobServices", "/tableServices", "/queueServices", "/fileServices"}
	allowedDimensions        = []string{"ResponseType", "ApiName"}
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("azure", "storage", New)
	mb.Registry.MustAddMetricSet("azure", "storage_account", New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	*azure.MetricSet
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := azure.NewMetricSet(base)
	if err != nil {
		return nil, err
	}
	// set default resource type to indicate this is not the generic monitor metricset
	ms.Client.Config.DefaultResourceType = defaultStorageAccountNamespace
	// if no options are entered we will retrieve all the vm's from the entire subscription
	if len(ms.Client.Config.Resources) == 0 {
		ms.Client.Config.Resources = []azure.ResourceConfig{
			{
				Query: fmt.Sprintf("resourceType eq '%s'", defaultStorageAccountNamespace),
			},
		}
	}
	for index := range ms.Client.Config.Resources {
		// if any resource groups were configured the resource type should be added
		if len(ms.Client.Config.Resources[index].Group) > 0 {
			ms.Client.Config.Resources[index].Type = defaultStorageAccountNamespace
		}
		// one metric configuration will be added containing all metrics names
		ms.Client.Config.Resources[index].Metrics = []azure.MetricConfig{
			{
				Name: []string{"*"},
			},
		}
	}
	ms.MapMetrics = mapMetrics
	return &MetricSet{
		MetricSet: ms,
	}, nil
}
