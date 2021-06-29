// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

func TestCursorUpdate(t *testing.T) {
	testCases := []struct {
		name          string
		baseConfig    map[string]interface{}
		trCtx         *transformContext
		initialState  common.MapStr
		expectedState common.MapStr
	}{
		{
			name: "update an unexisting value",
			baseConfig: map[string]interface{}{
				"entry1": map[string]interface{}{
					"value": "v1",
				},
			},
			trCtx:        emptyTransformContext(),
			initialState: common.MapStr{},
			expectedState: common.MapStr{
				"entry1": "v1",
			},
		},
		{
			name: "update an existing value with a template",
			baseConfig: map[string]interface{}{
				"entry1": map[string]interface{}{
					"value": "[[ .last_response.body.foo ]]",
				},
			},
			trCtx: func() *transformContext {
				trCtx := emptyTransformContext()
				trCtx.lastResponse.body = common.MapStr{
					"foo": "v2",
				}
				return trCtx
			}(),
			initialState: common.MapStr{
				"entry1": "v1",
			},
			expectedState: common.MapStr{
				"entry1": "v2",
			},
		},
		{
			name: "don't update an existing value if template result is empty",
			baseConfig: map[string]interface{}{
				"entry1": map[string]interface{}{
					"value": "[[ .last_response.body.unknown ]]",
				},
				"entry2": map[string]interface{}{
					"value":              "[[ .last_response.body.unknown ]]",
					"ignore_empty_value": true,
				},
				"entry3": map[string]interface{}{
					"value":              "[[ .last_response.body.unknown ]]",
					"ignore_empty_value": nil,
				},
			},
			trCtx: emptyTransformContext(),
			initialState: common.MapStr{
				"entry1": "v1",
				"entry2": "v2",
				"entry3": "v3",
			},
			expectedState: common.MapStr{
				"entry1": "v1",
				"entry2": "v2",
				"entry3": "v3",
			},
		},
		{
			name: "update an existing value if template result is empty and ignore_empty_value is false",
			baseConfig: map[string]interface{}{
				"entry1": map[string]interface{}{
					"value":              "[[ .last_response.body.unknown ]]",
					"ignore_empty_value": false,
				},
			},
			trCtx: emptyTransformContext(),
			initialState: common.MapStr{
				"entry1": "v1",
			},
			expectedState: common.MapStr{
				"entry1": "",
			},
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			cfg := common.MustNewConfigFrom(tc.baseConfig)

			conf := cursorConfig{}
			require.NoError(t, cfg.Unpack(&conf))

			c := newCursor(conf, logp.NewLogger("cursor-test"))
			c.state = tc.initialState
			c.update(tc.trCtx)
			assert.Equal(t, tc.expectedState, c.state)
		})
	}
}
