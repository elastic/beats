// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package monitor

import (
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure"
)

// concurrentMapMetrics fetches concurrently metric definitions and writes them in MetricDefinitionsChan channel
func concurrentMapMetrics(client *azure.BatchClient, resources []*armresources.GenericResourceExpanded, resourceConfig azure.ResourceConfig, wg *sync.WaitGroup) {
	go func() {
		defer wg.Done()
		for _, resource := range resources {
			// Call the shared mapping function, passing the batch client.
			// The shared function is now located in client_helper.go.
			res, err := getMappedResourceDefinitions(client, resource, resourceConfig)
			if err != nil {
				client.ResourceConfigurations.ErrorChan <- err // Send error and stop processing
				return
			}
			client.ResourceConfigurations.MetricDefinitionsChan <- res
		}
	}()
}