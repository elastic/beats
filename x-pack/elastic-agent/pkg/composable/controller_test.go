// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package composable_test

import (
	"context"
	"sync"
	"testing"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/transpiler"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"

	_ "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable/providers/env"
	_ "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable/providers/host"
	_ "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable/providers/local"
	_ "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable/providers/localdynamic"
)

func TestController(t *testing.T) {
	cfg, err := config.NewConfigFrom(map[string]interface{}{
		"providers": map[string]interface{}{
			"env": map[string]interface{}{
				"enabled": "false",
			},
			"local": map[string]interface{}{
				"vars": map[string]interface{}{
					"key1": "value1",
				},
			},
			"local_dynamic": map[string]interface{}{
				"items": []map[string]interface{}{
					{
						"vars": map[string]interface{}{
							"key1": "value1",
						},
						"processors": []map[string]interface{}{
							{
								"add_fields": map[string]interface{}{
									"fields": map[string]interface{}{
										"add": "value1",
									},
									"to": "dynamic",
								},
							},
						},
					},
					{
						"vars": map[string]interface{}{
							"key1": "value2",
						},
						"processors": []map[string]interface{}{
							{
								"add_fields": map[string]interface{}{
									"fields": map[string]interface{}{
										"add": "value2",
									},
									"to": "dynamic",
								},
							},
						},
					},
				},
			},
		},
	})
	require.NoError(t, err)

	c, err := composable.New(cfg)
	require.NoError(t, err)

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg.Add(1)
	var setVars []transpiler.Vars
	err = c.Run(ctx, func(vars []transpiler.Vars) {
		setVars = vars
		wg.Done()
	})
	require.NoError(t, err)
	wg.Wait()

	assert.Len(t, setVars, 3)

	_, hostExists := setVars[0].Mapping["host"]
	assert.True(t, hostExists)
	_, envExists := setVars[0].Mapping["env"]
	assert.False(t, envExists)
	localMap := setVars[0].Mapping["local"].(map[string]interface{})
	assert.Equal(t, "value1", localMap["key1"])
	assert.Equal(t, "", setVars[0].ProcessorsKey)
	assert.Nil(t, setVars[0].Processors)

	localMap = setVars[1].Mapping["local_dynamic"].(map[string]interface{})
	assert.Equal(t, "value1", localMap["key1"])
	assert.Equal(t, "local_dynamic", setVars[1].ProcessorsKey)
	assert.Len(t, setVars[1].Processors, 1)

	localMap = setVars[2].Mapping["local_dynamic"].(map[string]interface{})
	assert.Equal(t, "value2", localMap["key1"])
	assert.Equal(t, "local_dynamic", setVars[2].ProcessorsKey)
	assert.Len(t, setVars[2].Processors, 1)
}
