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
						"dataset.namespace": "somenamespace",
						"streams":           []map[string]interface{}{{"dataset.name": "somedatasetname"}},
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
							"namespace": "somenamespace",
						},
						"streams": []map[string]interface{}{
							{
								"dataset": map[string]interface{}{
									"name": "somedatasetname",
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
			name: "dataset.name invalid dot - compact",
			configMap: map[string]interface{}{
				"inputs": []map[string]interface{}{
					{"streams": []map[string]interface{}{{"dataset.name": "."}}},
				},
			},
			result: ErrInvalidDataset,
		},
		{
			name: "dataset.name invalid dotdot- compact",
			configMap: map[string]interface{}{
				"inputs": []map[string]interface{}{
					{"streams": []map[string]interface{}{{"dataset.name": ".."}}},
				},
			},
			result: ErrInvalidDataset,
		},
		{
			name: "dataset.name invalid uppercase - compact",
			configMap: map[string]interface{}{
				"inputs": []map[string]interface{}{
					{"streams": []map[string]interface{}{{"dataset.name": "myNameIs"}}},
				},
			},
			result: ErrInvalidDataset,
		},
		{
			name: "dataset.name invalid space- compact",
			configMap: map[string]interface{}{
				"inputs": []map[string]interface{}{
					{"streams": []map[string]interface{}{{"dataset.name": "outer space"}}},
				},
			},
			result: ErrInvalidDataset,
		},
		{
			name: "dataset.name invalid invalid char- compact",
			configMap: map[string]interface{}{
				"inputs": []map[string]interface{}{
					{"streams": []map[string]interface{}{{"dataset.name": "is\\thisvalid"}}},
				},
			},
			result: ErrInvalidDataset,
		},
		{
			name: "dataset.name invalid invalid prefix- compact",
			configMap: map[string]interface{}{
				"inputs": []map[string]interface{}{
					{"streams": []map[string]interface{}{{"dataset.name": "_isthisvalid"}}},
				},
			},
			result: ErrInvalidDataset,
		},

		{
			name: "namespace invalid - compact",
			configMap: map[string]interface{}{
				"inputs": []map[string]interface{}{{"dataset.namespace": ""}},
			},
			result: ErrInvalidNamespace,
		},
		{
			name: "namespace invalid name 1 - compact",
			configMap: map[string]interface{}{
				"inputs": []map[string]interface{}{
					{"dataset.namespace": "."},
				},
			},
			result: ErrInvalidNamespace,
		},
		{
			name: "namespace invalid name 2 - compact",
			configMap: map[string]interface{}{
				"inputs": []map[string]interface{}{{"dataset.namespace": ".."}},
			},
			result: ErrInvalidNamespace,
		},
		{
			name: "namespace invalid name uppercase - compact",
			configMap: map[string]interface{}{
				"inputs": []map[string]interface{}{{"dataset.namespace": "someUpper"}},
			},
			result: ErrInvalidNamespace,
		},
		{
			name: "namespace invalid name space - compact",
			configMap: map[string]interface{}{
				"inputs": []map[string]interface{}{{"dataset.namespace": "some space"}},
			},
			result: ErrInvalidNamespace,
		},
		{
			name: "namespace invalid name invalid char - compact",
			configMap: map[string]interface{}{
				"inputs": []map[string]interface{}{{"dataset.namespace": "isitok?"}},
			},
			result: ErrInvalidNamespace,
		},
		{
			name: "namespace invalid name invalid prefix - compact",
			configMap: map[string]interface{}{
				"inputs": []map[string]interface{}{{"dataset.namespace": "+isitok"}},
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
