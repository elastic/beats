// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package filters

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/transpiler"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

const semiLongString = ""

func TestStreamCheck(t *testing.T) {
	type testCase struct {
		name      string
		configMap map[string]interface{}
		result    error
	}

	h := hex.EncodeToString(sha512.New().Sum(nil))
	semiLongString := h[:86]
	longString := fmt.Sprintf("%s%s", h, h)

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
		{
			name: "type invalid name 1 - compact",
			configMap: map[string]interface{}{
				"inputs": []map[string]interface{}{
					{"dataset.type": "-invalidstart"},
				},
			},
			result: ErrInvalidIndex,
		},
		{
			name: "type invalid combined length 1 - compact",
			configMap: map[string]interface{}{
				"inputs": []map[string]interface{}{
					{
						"dataset.type":      semiLongString,
						"dataset.namespace": semiLongString,
						"streams":           []map[string]interface{}{{"dataset.name": semiLongString}},
					},
				},
			},
			result: ErrInvalidIndex,
		},
		{
			name: "type invalid type length 1 - compact",
			configMap: map[string]interface{}{
				"inputs": []map[string]interface{}{
					{"dataset.type": longString},
				},
			},
			result: ErrInvalidIndex,
		},

		{
			name: "type invalid namespace length 1 - compact",
			configMap: map[string]interface{}{
				"inputs": []map[string]interface{}{
					{"dataset.namespace": longString},
				},
			},
			result: ErrInvalidNamespace,
		},

		{
			name: "type invalid dataset.name length 1 - compact",
			configMap: map[string]interface{}{
				"inputs": []map[string]interface{}{
					{"streams": []map[string]interface{}{{"dataset.name": longString}}},
				},
			},
			result: ErrInvalidDataset,
		},

		{
			name: "type empty streams - compact",
			configMap: map[string]interface{}{
				"inputs": []map[string]interface{}{
					{"streams": []map[string]interface{}{}},
				},
			},
			result: nil,
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
