// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package azure

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

		assert.Len(t, metricsByResourceId, 1)
		assert.Len(t, metricsByResourceId["resource-1"], 3)
	})
}

func TestCalculateTimespan(t *testing.T) {
	t.Run("Collection period greater than the time grain (PT1M metric every 5 minutes)", func(t *testing.T) {
		referenceTime, _ := time.Parse(time.RFC3339, "2024-07-30T18:56:00Z")
		timeGrain := "PT1M"
		cfg := Config{
			Period: 5 * time.Minute,
		}

		startTime, endTime := calculateTimespan(referenceTime, timeGrain, cfg, nil)

		require.Equal(t, "2024-07-30T18:51:00Z", startTime.Format(time.RFC3339))
		require.Equal(t, "2024-07-30T18:56:00Z", endTime.Format(time.RFC3339))
	})

	t.Run("Collection period equal to time grain (PT1M metric every 1 minutes)", func(t *testing.T) {
		referenceTime, _ := time.Parse(time.RFC3339, "2024-07-30T18:56:00Z")
		timeGrain := "PT1M"
		cfg := Config{
			Period: 1 * time.Minute,
		}

		startTime, endTime := calculateTimespan(referenceTime, timeGrain, cfg, nil)

		require.Equal(t, "2024-07-30T18:55:00Z", startTime.Format(time.RFC3339))
		require.Equal(t, "2024-07-30T18:56:00Z", endTime.Format(time.RFC3339))
	})

	t.Run("Collection period equal to time grain (PT5M metric every 5 minutes)", func(t *testing.T) {
		referenceTime, _ := time.Parse(time.RFC3339, "2024-07-30T18:56:00Z")
		timeGrain := "PT5M"
		cfg := Config{
			Period: 5 * time.Minute,
		}

		startTime, endTime := calculateTimespan(referenceTime, timeGrain, cfg, nil)

		require.Equal(t, "2024-07-30T18:51:00Z", startTime.Format(time.RFC3339))
		require.Equal(t, "2024-07-30T18:56:00Z", endTime.Format(time.RFC3339))
	})

	t.Run("Collection period equal to time grain (PT1H metric every 60 minutes)", func(t *testing.T) {
		referenceTime, _ := time.Parse(time.RFC3339, "2024-07-30T18:56:00Z")
		timeGrain := "PT1H"
		cfg := Config{
			Period: 60 * time.Minute,
		}

		startTime, endTime := calculateTimespan(referenceTime, timeGrain, cfg, nil)

		require.Equal(t, "2024-07-30T17:56:00Z", startTime.Format(time.RFC3339))
		require.Equal(t, "2024-07-30T18:56:00Z", endTime.Format(time.RFC3339))
	})

	t.Run("Collection period is less that time grain (PT1H metric every 5 minutes)", func(t *testing.T) {
		referenceTime, _ := time.Parse(time.RFC3339, "2024-07-30T18:56:00Z")
		timeGrain := "PT1H"
		cfg := Config{
			Period: 5 * time.Minute,
		}
		startTime, endTime := calculateTimespan(referenceTime, timeGrain, cfg, nil)

		require.Equal(t, "2024-07-30T17:56:00Z", startTime.Format(time.RFC3339))
		require.Equal(t, "2024-07-30T18:56:00Z", endTime.Format(time.RFC3339))
	})

}

func TestCalculateTimespanWithLatency(t *testing.T) {
	t.Run("Collection period greater than the time grain (PT1M metric every 5 minutes)", func(t *testing.T) {
		referenceTime, _ := time.Parse(time.RFC3339, "2024-07-30T18:56:00Z")
		timeGrain := "PT1M"
		cfg := Config{
			Period:  5 * time.Minute,
			Latency: 1 * time.Minute,
		}

		startTime, endTime := calculateTimespan(referenceTime, timeGrain, cfg, nil)

		require.Equal(t, "2024-07-30T18:50:00Z", startTime.Format(time.RFC3339))
		require.Equal(t, "2024-07-30T18:55:00Z", endTime.Format(time.RFC3339))
	})

	t.Run("Collection period equal to time grain (PT1M metric every 1 minutes)", func(t *testing.T) {
		referenceTime, _ := time.Parse(time.RFC3339, "2024-07-30T18:56:00Z")
		timeGrain := "PT1M"
		cfg := Config{
			Period:  1 * time.Minute,
			Latency: 1 * time.Minute,
		}

		startTime, endTime := calculateTimespan(referenceTime, timeGrain, cfg, nil)

		require.Equal(t, "2024-07-30T18:54:00Z", startTime.Format(time.RFC3339))
		require.Equal(t, "2024-07-30T18:55:00Z", endTime.Format(time.RFC3339))
	})
}

func TestCalculateTimespanWithLookback(t *testing.T) {
	referenceTime, _ := time.Parse(time.RFC3339, "2024-07-30T19:00:00Z")
	cfg := Config{Period: 5 * time.Minute}

	t.Run("nil lookbackStart uses normal window", func(t *testing.T) {
		startTime, endTime := calculateTimespan(referenceTime, "PT5M", cfg, nil)
		require.Equal(t, "2024-07-30T18:55:00Z", startTime.Format(time.RFC3339))
		require.Equal(t, "2024-07-30T19:00:00Z", endTime.Format(time.RFC3339))
	})

	t.Run("lookbackStart before normalStart expands window", func(t *testing.T) {
		// 7 minutes ago — older than the 5-minute normal window
		lookback, _ := time.Parse(time.RFC3339, "2024-07-30T18:53:00Z")
		startTime, endTime := calculateTimespan(referenceTime, "PT5M", cfg, &lookback)
		require.Equal(t, "2024-07-30T18:53:00Z", startTime.Format(time.RFC3339))
		require.Equal(t, "2024-07-30T19:00:00Z", endTime.Format(time.RFC3339))
	})

	t.Run("lookbackStart after normalStart does not expand window", func(t *testing.T) {
		// 3 minutes ago — within the 5-minute normal window, no expansion needed
		lookback, _ := time.Parse(time.RFC3339, "2024-07-30T18:57:00Z")
		startTime, endTime := calculateTimespan(referenceTime, "PT5M", cfg, &lookback)
		require.Equal(t, "2024-07-30T18:55:00Z", startTime.Format(time.RFC3339))
		require.Equal(t, "2024-07-30T19:00:00Z", endTime.Format(time.RFC3339))
	})

	t.Run("lookbackStart equal to normalStart does not expand window", func(t *testing.T) {
		lookback, _ := time.Parse(time.RFC3339, "2024-07-30T18:55:00Z")
		startTime, endTime := calculateTimespan(referenceTime, "PT5M", cfg, &lookback)
		require.Equal(t, "2024-07-30T18:55:00Z", startTime.Format(time.RFC3339))
		require.Equal(t, "2024-07-30T19:00:00Z", endTime.Format(time.RFC3339))
	})
}

func TestResourcesFingerprint(t *testing.T) {
	vmNS := "Microsoft.Compute/virtualMachines"
	stNS := "Microsoft.Storage/storageAccounts"

	vm := func(ns string) ResourceConfig {
		return ResourceConfig{Metrics: []MetricConfig{{Namespace: ns}}}
	}
	vmWithGroup := func(ns, group string) ResourceConfig {
		return ResourceConfig{Group: []string{group}, Metrics: []MetricConfig{{Namespace: ns}}}
	}
	vmWithType := func(ns, rtype string) ResourceConfig {
		return ResourceConfig{Type: rtype, Metrics: []MetricConfig{{Namespace: ns}}}
	}
	vmWithID := func(ns, id string) ResourceConfig {
		return ResourceConfig{Id: []string{id}, Metrics: []MetricConfig{{Namespace: ns}}}
	}
	vmWithQuery := func(ns, query string) ResourceConfig {
		return ResourceConfig{Query: query, Metrics: []MetricConfig{{Namespace: ns}}}
	}

	t.Run("same config produces same fingerprint", func(t *testing.T) {
		a := resourcesFingerprint([]ResourceConfig{vm(vmNS)})
		b := resourcesFingerprint([]ResourceConfig{vm(vmNS)})
		assert.Equal(t, a, b)
	})

	t.Run("order of resource entries does not matter", func(t *testing.T) {
		a := resourcesFingerprint([]ResourceConfig{vm(vmNS), vm(stNS)})
		b := resourcesFingerprint([]ResourceConfig{vm(stNS), vm(vmNS)})
		assert.Equal(t, a, b)
	})

	t.Run("different namespace produces different fingerprint", func(t *testing.T) {
		assert.NotEqual(t,
			resourcesFingerprint([]ResourceConfig{vm(vmNS)}),
			resourcesFingerprint([]ResourceConfig{vm(stNS)}))
	})

	t.Run("different resource group produces different fingerprint", func(t *testing.T) {
		assert.NotEqual(t,
			resourcesFingerprint([]ResourceConfig{vmWithGroup(vmNS, "prod")}),
			resourcesFingerprint([]ResourceConfig{vmWithGroup(vmNS, "staging")}))
	})

	t.Run("different resource type produces different fingerprint", func(t *testing.T) {
		assert.NotEqual(t,
			resourcesFingerprint([]ResourceConfig{vmWithType(vmNS, "Microsoft.Compute/virtualMachines")}),
			resourcesFingerprint([]ResourceConfig{vmWithType(vmNS, "Microsoft.Storage/storageAccounts")}))
	})

	t.Run("different resource id produces different fingerprint", func(t *testing.T) {
		assert.NotEqual(t,
			resourcesFingerprint([]ResourceConfig{vmWithID(vmNS, "/subscriptions/sub1/resourceGroups/rg/providers/Microsoft.Compute/virtualMachines/vm1")}),
			resourcesFingerprint([]ResourceConfig{vmWithID(vmNS, "/subscriptions/sub1/resourceGroups/rg/providers/Microsoft.Compute/virtualMachines/vm2")}))
	})

	t.Run("different resource query produces different fingerprint", func(t *testing.T) {
		assert.NotEqual(t,
			resourcesFingerprint([]ResourceConfig{vmWithQuery(vmNS, "resourceType eq 'Microsoft.Compute/virtualMachines'")}),
			resourcesFingerprint([]ResourceConfig{vmWithQuery(vmNS, "resourceType eq 'Microsoft.Storage/storageAccounts'")}))
	})

	t.Run("service_type does not affect fingerprint", func(t *testing.T) {
		withSvc := ResourceConfig{ServiceType: []string{"blob"}, Metrics: []MetricConfig{{Namespace: vmNS}}}
		withoutSvc := ResourceConfig{Metrics: []MetricConfig{{Namespace: vmNS}}}
		assert.Equal(t,
			resourcesFingerprint([]ResourceConfig{withSvc}),
			resourcesFingerprint([]ResourceConfig{withoutSvc}))
	})
}
