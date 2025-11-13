// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestCursorUpdate(t *testing.T) {
	testCases := []struct {
		name          string
		baseConfig    map[string]interface{}
		trCtx         *transformContext
		initialState  mapstr.M
		expectedState mapstr.M
		wantStatus    []string
	}{
		{
			name: "update an unexisting value",
			baseConfig: map[string]interface{}{
				"entry1": map[string]interface{}{
					"value": "v1",
				},
			},
			trCtx:        emptyTransformContext(),
			initialState: mapstr.M{},
			expectedState: mapstr.M{
				"entry1": "v1",
			},
			wantStatus: nil,
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
				trCtx.lastResponse.body = mapstr.M{
					"foo": "v2",
				}
				return trCtx
			}(),
			initialState: mapstr.M{
				"entry1": "v1",
			},
			expectedState: mapstr.M{
				"entry1": "v2",
			},
			wantStatus: nil,
		},
		{
			name: "don't update an existing value if template result is empty",
			baseConfig: map[string]interface{}{
				"entry1": map[string]interface{}{
					"value":              ``,
					"do_not_log_failure": true,
				},
				"entry2": map[string]interface{}{
					"value":              ``,
					"ignore_empty_value": true,
				},
				"entry3": map[string]interface{}{
					"value":              ``,
					"ignore_empty_value": nil,
				},
				"entry4": map[string]interface{}{
					"value":              ``,
					"ignore_empty_value": false,
					"do_not_log_failure": true,
				},
				"entry5": map[string]interface{}{
					"value":              ``,
					"ignore_empty_value": false,
					"do_not_log_failure": false,
				},
				"entry6": map[string]interface{}{
					"value":              ``,
					"ignore_empty_value": false,
				},
			},
			trCtx: emptyTransformContext(),
			initialState: mapstr.M{
				"entry1": "v1",
				"entry2": "v2",
				"entry3": "v3",
				"entry4": "v4",
				"entry5": "v5",
				"entry6": "v6",
			},
			expectedState: mapstr.M{
				"entry1": "v1",
				"entry2": "v2",
				"entry3": "v3",
				"entry4": "",
				"entry5": "",
				"entry6": "",
			},
			wantStatus: []string{
				"Degraded: failed to execute template entry5: the template result is empty",
			},
		},
		{
			name: "update an existing value if template result is empty and ignore_empty_value is false",
			baseConfig: map[string]interface{}{
				"entry1": map[string]interface{}{
					"value":              ``,
					"ignore_empty_value": false,
					"do_not_log_failure": true,
				},
			},
			trCtx: emptyTransformContext(),
			initialState: mapstr.M{
				"entry1": "v1",
			},
			expectedState: mapstr.M{
				"entry1": "",
			},
			wantStatus: nil,
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			cfg := conf.MustNewConfigFrom(tc.baseConfig)

			conf := cursorConfig{}
			require.NoError(t, cfg.Unpack(&conf))

			var stat testStatus
			c := newCursor(conf, &stat, logp.NewLogger("cursor-test"))
			c.state = tc.initialState
			c.update(tc.trCtx)
			assert.Equal(t, tc.expectedState, c.state)
			sort.Strings(stat.updates) // Can happen out of order.
			assert.Equal(t, tc.wantStatus, stat.updates)
		})
	}
}
