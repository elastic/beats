// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package app_insights

import (
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/preview/appinsights/v1/insights"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	config = Config{
		ApplicationId: "",
		ApiKey:        "",
		Metrics: []Metric{
			{
				ID: []string{"requests/count"},
			},
		},
	}
)

func TestClient(t *testing.T) {
	t.Run("return error not valid query", func(t *testing.T) {
		client := NewMockClient()
		client.Config = config
		m := &MockService{}
		m.On("GetMetricValues", mock.Anything, mock.Anything).Return(insights.ListMetricsResultsItem{}, errors.New("invalid query"))
		client.Service = m
		results, err := client.GetMetricValues()
		assert.Error(t, err)
		assert.Nil(t, results.Value)
		m.AssertExpectations(t)
	})
	t.Run("return results", func(t *testing.T) {
		client := NewMockClient()
		client.Config = config
		m := &MockService{}
		metrics := []insights.MetricsResultsItem{{}, {}}
		m.On("GetMetricValues", mock.Anything, mock.Anything).Return(insights.ListMetricsResultsItem{Value: &metrics}, nil)
		client.Service = m
		results, err := client.GetMetricValues()
		assert.NoError(t, err)
		assert.Equal(t, len(*results.Value), 2)
		m.AssertExpectations(t)
	})
}
