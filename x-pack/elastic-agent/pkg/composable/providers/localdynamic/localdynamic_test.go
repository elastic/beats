// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package localdynamic

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable"
	ctesting "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable/testing"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
)

func TestContextProvider(t *testing.T) {
	mapping1 := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	processors1 := []map[string]interface{}{
		{
			"add_fields": map[string]interface{}{
				"fields": map[string]interface{}{
					"add": "value1",
				},
				"to": "dynamic",
			},
		},
	}
	mapping2 := map[string]interface{}{
		"key1": "value12",
		"key2": "value22",
	}
	processors2 := []map[string]interface{}{
		{
			"add_fields": map[string]interface{}{
				"fields": map[string]interface{}{
					"add": "value12",
				},
				"to": "dynamic",
			},
		},
	}
	mapping := []map[string]interface{}{
		{
			"vars":       mapping1,
			"processors": processors1,
		},
		{
			"vars":       mapping2,
			"processors": processors2,
		},
	}
	cfg, err := config.NewConfigFrom(map[string]interface{}{
		"items": mapping,
	})
	require.NoError(t, err)
	builder, _ := composable.Providers.GetDynamicProvider("local_dynamic")
	provider, err := builder(cfg)
	require.NoError(t, err)

	comm := ctesting.NewDynamicComm(context.Background())
	err = provider.Run(comm)
	require.NoError(t, err)

	curr1, ok1 := comm.Current("0")
	assert.True(t, ok1)
	assert.Equal(t, mapping1, curr1.Mapping)
	assert.Equal(t, processors1, curr1.Processors)

	curr2, ok2 := comm.Current("1")
	assert.True(t, ok2)
	assert.Equal(t, mapping2, curr2.Mapping)
	assert.Equal(t, processors2, curr2.Processors)
}
