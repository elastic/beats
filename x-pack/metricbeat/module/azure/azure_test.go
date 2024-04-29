// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGroupMetricsDefinitionsByResourceId(t *testing.T) {

	t.Run("Group metrics definitions by resource ID", func(t *testing.T) {
		metrics := []Metric{
			{
				ResourceId: "resource-1",
				Namespace:  "namespace-1",
				Names:      []string{"metric-1"},
			},
			{
				ResourceId: "resource-1",
				Namespace:  "namespace-1",
				Names:      []string{"metric-2"},
			},
			{
				ResourceId: "resource-1",
				Namespace:  "namespace-1",
				Names:      []string{"metric-3"},
			},
		}

		metricsByResourceId := groupMetricsDefinitionsByResourceId(metrics)

		assert.Equal(t, 1, len(metricsByResourceId))
		assert.Equal(t, 3, len(metricsByResourceId["resource-1"]))
	})
}
