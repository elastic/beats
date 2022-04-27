// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package monitor

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/config"
)

var (
	missingResourcesConfig = common.MapStr{
		"module":          "azure",
		"period":          "60s",
		"metricsets":      []string{"monitor"},
		"client_secret":   "unique identifier",
		"client_id":       "unique identifier",
		"subscription_id": "unique identifier",
		"tenant_id":       "unique identifier",
	}

	resourceConfig = common.MapStr{
		"module":          "azure",
		"period":          "60s",
		"metricsets":      []string{"monitor"},
		"client_secret":   "unique identifier",
		"client_id":       "unique identifier",
		"subscription_id": "unique identifier",
		"tenant_id":       "unique identifier",
		"resources": []common.MapStr{
			{
				"resource_id": "test",
				"metrics": []map[string]interface{}{
					{
						"namespace": "namespace_name",
						"name":      []string{"metric_name"},
					}},
			}},
	}
)

func TestFetch(t *testing.T) {
	c, err := config.NewConfigFrom(missingResourcesConfig)
	if err != nil {
		t.Fatal(err)
	}
	module, metricsets, err := mb.NewModule(c, mb.Registry)
	assert.Nil(t, module)
	assert.Nil(t, metricsets)
	assert.Error(t, err, "no resource options defined: module azure - monitor metricset")
	c, err = config.NewConfigFrom(resourceConfig)
	if err != nil {
		t.Fatal(err)
	}
	module, metricsets, err = mb.NewModule(c, mb.Registry)
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, module)
	assert.NotNil(t, metricsets)
	ms, ok := metricsets[0].(*MetricSet)
	require.True(t, ok, "metricset must be MetricSet")
	assert.NotNil(t, ms)
}
