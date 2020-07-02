// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package filters

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/transpiler"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

func TestStreamCheck(t *testing.T) {
	type testCase struct {
		name      string
		configMap map[string]interface{}
		result    error
	}

	testCases := []testCase{
		{
			name:      "all missing",
			configMap: map[string]interface{}{},
			result:    nil,
		},
		{
			name: "all ok - compact",
			configMap: map[string]interface{}{
				"inputs": []map[string]interface{}{
					{
						"dataset.namespace": "someNamespace",
						"streams":           []map[string]interface{}{{"dataset.name": "someDatasetName"}},
					},
				},
			},
			result: nil,
		},
		{
			name: "all ok - long",
			configMap: map[string]interface{}{
				"inputs": []map[string]interface{}{
					{
						"dataset": map[string]interface{}{
							"namespace": "someNamespace",
						},
						"streams": []map[string]interface{}{
							{
								"dataset": map[string]interface{}{
									"name": "someDatasetName",
								},
							},
						},
					},
				},
			},
			result: nil,
		},
		{
			name: "dataset.name invalid - compact",
			configMap: map[string]interface{}{
				"inputs": []map[string]interface{}{
					{"streams": []map[string]interface{}{{"dataset.name": ""}}},
				},
			},
			result: ErrInvalidDataset,
		},
		{
			name: "dataset.name invalid - long",
			configMap: map[string]interface{}{
				"inputs": []map[string]interface{}{
					{
						"streams": []map[string]interface{}{
							{
								"dataset": map[string]interface{}{
									"name": "",
								},
							},
						},
					},
				},
			},
			result: ErrInvalidDataset,
		},
		{
			name: "namespace invalid - compact",
			configMap: map[string]interface{}{
				"inputs": []map[string]interface{}{
					{"dataset.namespace": ""},
				},
			},
			result: ErrInvalidNamespace,
		},
		{
			name: "namespace invalid - long",
			configMap: map[string]interface{}{
				"inputs": []map[string]interface{}{
					{
						"dataset": map[string]interface{}{
							"namespace": "",
						},
					},
				},
			},
			result: ErrInvalidNamespace,
		},
	}

	log, err := logger.New("")
	assert.NoError(t, err)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ast, err := transpiler.NewAST(tc.configMap)
			assert.NoError(t, err)

			result := StreamChecker(log, ast)
			assert.Equal(t, tc.result, result)
		})
	}
}
