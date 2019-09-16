// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package compute_vm_scaleset

import (
	"errors"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-03-01/resources"
	"testing"

	"github.com/elastic/beats/x-pack/metricbeat/module/azure"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	resourceQueryConfig = azure.Config{
		Resources: []azure.ResourceConfig{
			{
				Query: "query",
				Metrics: []azure.MetricConfig{
					{
						Name: []string{"hello", "test"},
					},
				}}},
	}
)

func TestInitResources(t *testing.T) {
	client := azure.NewMockClient()
	t.Run("return error when no resource options were configured", func(t *testing.T) {
		mr := azure.MockReporterV2{}
		err := InitResources(client, &mr)
		assert.Error(t, err, "no resource options were configured")
	})
	t.Run("return error no resources were found", func(t *testing.T) {
		client.Config = resourceQueryConfig
		m := &azure.AzureMockService{}
		m.On("GetResourceDefinitions", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(resources.ListResultPage{}, errors.New("invalid resource query"))
		client.AzureMonitorService = m
		mr := azure.MockReporterV2{}
		mr.On("Error", mock.Anything).Return(true)
		err := InitResources(client, &mr)
		assert.Error(t, err, "no resources were found based on all the configurations options entered")
		assert.Equal(t, len(client.Resources.Metrics), 0)
		m.AssertExpectations(t)
	})
}
