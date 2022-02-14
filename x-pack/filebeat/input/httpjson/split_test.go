// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

func TestSplit(t *testing.T) {
	registerResponseTransforms()
	t.Cleanup(func() { registeredTransforms = newRegistry() })
	cases := []struct {
		name             string
		config           *splitConfig
		ctx              *transformContext
		resp             transformable
		expectedMessages []common.MapStr
		expectedErr      error
	}{
		{
			name: "Two nested Split Arrays with keep_parent",
			config: &splitConfig{
				Target:     "body.alerts",
				Type:       "array",
				KeepParent: true,
				Split: &splitConfig{
					Target:     "body.alerts.entities",
					Type:       "array",
					KeepParent: true,
				},
			},
			ctx: emptyTransformContext(),
			resp: transformable{
				"body": common.MapStr{
					"this": "is kept",
					"alerts": []interface{}{
						map[string]interface{}{
							"this_is": "also kept",
							"entities": []interface{}{
								map[string]interface{}{
									"something": "something",
								},
								map[string]interface{}{
									"else": "else",
								},
							},
						},
						map[string]interface{}{
							"this_is": "also kept 2",
							"entities": []interface{}{
								map[string]interface{}{
									"something": "something 2",
								},
								map[string]interface{}{
									"else": "else 2",
								},
							},
						},
					},
				},
			},
			expectedMessages: []common.MapStr{
				{
					"this":                      "is kept",
					"alerts.this_is":            "also kept",
					"alerts.entities.something": "something",
				},
				{
					"this":                 "is kept",
					"alerts.this_is":       "also kept",
					"alerts.entities.else": "else",
				},
				{
					"this":                      "is kept",
					"alerts.this_is":            "also kept 2",
					"alerts.entities.something": "something 2",
				},
				{
					"this":                 "is kept",
					"alerts.this_is":       "also kept 2",
					"alerts.entities.else": "else 2",
				},
			},
			expectedErr: nil,
		},
		{
			name: "A nested array with a nested map",
			config: &splitConfig{
				Target:     "body.alerts",
				Type:       "array",
				KeepParent: false,
				Split: &splitConfig{
					Target:     "body.entities",
					Type:       "map",
					KeepParent: true,
					KeyField:   "id",
				},
			},
			ctx: emptyTransformContext(),
			resp: transformable{
				"body": common.MapStr{
					"this": "is not kept",
					"alerts": []interface{}{
						map[string]interface{}{
							"this_is": "kept",
							"entities": map[string]interface{}{
								"id1": map[string]interface{}{
									"something": "else",
								},
							},
						},
						map[string]interface{}{
							"this_is": "also kept",
							"entities": map[string]interface{}{
								"id2": map[string]interface{}{
									"something": "else 2",
								},
							},
						},
					},
				},
			},
			expectedMessages: []common.MapStr{
				{
					"this_is":            "kept",
					"entities.id":        "id1",
					"entities.something": "else",
				},
				{
					"this_is":            "also kept",
					"entities.id":        "id2",
					"entities.something": "else 2",
				},
			},
			expectedErr: nil,
		},
		{
			name: "A nested array with a nested map with transforms",
			config: &splitConfig{
				Target: "body.alerts",
				Type:   "array",
				Split: &splitConfig{
					Target: "body.entities",
					Type:   "map",
					Transforms: transformsConfig{
						common.MustNewConfigFrom(map[string]interface{}{
							"set": map[string]interface{}{
								"target": "body.foo",
								"value":  "set for each",
							},
						}),
					},
				},
			},
			ctx: emptyTransformContext(),
			resp: transformable{
				"body": common.MapStr{
					"this": "is not kept",
					"alerts": []interface{}{
						map[string]interface{}{
							"this_is": "kept",
							"entities": map[string]interface{}{
								"id1": map[string]interface{}{
									"something": "else",
								},
							},
						},
						map[string]interface{}{
							"this_is": "also not kept",
							"entities": map[string]interface{}{
								"id2": map[string]interface{}{
									"something": "else 2",
								},
							},
						},
					},
				},
			},
			expectedMessages: []common.MapStr{
				{
					"something": "else",
					"foo":       "set for each",
				},
				{
					"something": "else 2",
					"foo":       "set for each",
				},
			},
			expectedErr: nil,
		},
		{
			name: "A nested array with a nested array in an object",
			config: &splitConfig{
				Target: "body.response",
				Type:   "array",
				Split: &splitConfig{
					Target:     "body.Event.Attributes",
					KeepParent: true,
				},
			},
			ctx: emptyTransformContext(),
			resp: transformable{
				"body": common.MapStr{
					"response": []interface{}{
						map[string]interface{}{
							"Event": map[string]interface{}{
								"timestamp": "1606324417",
								"Attributes": []interface{}{
									map[string]interface{}{
										"key": "value",
									},
									map[string]interface{}{
										"key2": "value2",
									},
								},
							},
						},
					},
				},
			},
			expectedMessages: []common.MapStr{
				{
					"Event": common.MapStr{
						"timestamp": "1606324417",
						"Attributes": common.MapStr{
							"key": "value",
						},
					},
				},
				{
					"Event": common.MapStr{
						"timestamp": "1606324417",
						"Attributes": common.MapStr{
							"key2": "value2",
						},
					},
				},
			},
			expectedErr: nil,
		},
		{
			name: "A nested array with an empty nested array in an object publishes without the key",
			config: &splitConfig{
				Target: "body.response",
				Type:   "array",
				Split: &splitConfig{
					Target:     "body.Event.Attributes",
					KeepParent: true,
				},
			},
			ctx: emptyTransformContext(),
			resp: transformable{
				"body": common.MapStr{
					"response": []interface{}{
						map[string]interface{}{
							"Event": map[string]interface{}{
								"timestamp": "1606324417",
							},
						},
					},
				},
			},
			expectedMessages: []common.MapStr{
				{
					"Event": common.MapStr{
						"timestamp": "1606324417",
					},
				},
			},
		},
		{
			name: "First level split skips publish if no events",
			config: &splitConfig{
				Target: "body.response",
				Type:   "array",
				Split: &splitConfig{
					Target:     "body.Event.Attributes",
					KeepParent: true,
				},
			},
			ctx: emptyTransformContext(),
			resp: transformable{
				"body": common.MapStr{
					"response": []interface{}{},
				},
			},
			expectedMessages: []common.MapStr{},
			expectedErr:      errEmptyRootField,
		},
		{
			name: "Changes must be local to parent when nested splits",
			config: &splitConfig{
				Target: "body.items",
				Type:   "array",
				Split: &splitConfig{
					Target:     "body.splitHere.splitMore",
					Type:       "array",
					KeepParent: true,
				},
			},
			ctx: emptyTransformContext(),
			resp: transformable{
				"body": common.MapStr{
					"@timestamp":    "1234567890",
					"nextPageToken": "tok",
					"items": []interface{}{
						common.MapStr{"foo": "bar"},
						common.MapStr{
							"baz": "buzz",
							"splitHere": common.MapStr{
								"splitMore": []interface{}{
									common.MapStr{
										"deepest1": "data",
									},
									common.MapStr{
										"deepest2": "data",
									},
								},
							},
						},
					},
				},
			},
			expectedMessages: []common.MapStr{
				{"foo": "bar"},
				{"baz": "buzz", "splitHere": common.MapStr{"splitMore": common.MapStr{"deepest1": "data"}}},
				{"baz": "buzz", "splitHere": common.MapStr{"splitMore": common.MapStr{"deepest2": "data"}}},
			},
		},
		{
			name: "Split string",
			config: &splitConfig{
				Target:          "body.items",
				Type:            "string",
				DelimiterString: "\n",
			},
			ctx: emptyTransformContext(),
			resp: transformable{
				"body": common.MapStr{
					"@timestamp": "1234567890",
					"items":      "Line 1\nLine 2\nLine 3",
				},
			},
			expectedMessages: []common.MapStr{
				{"@timestamp": "1234567890", "items": "Line 1"},
				{"@timestamp": "1234567890", "items": "Line 2"},
				{"@timestamp": "1234567890", "items": "Line 3"},
			},
		},
		{
			name: "An empty array in an object",
			config: &splitConfig{
				Target: "body.response",
				Type:   "array",
				Split: &splitConfig{
					Target:           "body.Event.Attributes",
					IgnoreEmptyValue: true,
					KeepParent:       true,
					Split: &splitConfig{
						Target:     "body.Event.OtherAttributes",
						KeepParent: true,
					},
				},
			},
			ctx: emptyTransformContext(),
			resp: transformable{
				"body": common.MapStr{
					"response": []interface{}{
						map[string]interface{}{
							"Event": map[string]interface{}{
								"timestamp":  "1606324417",
								"Attributes": []interface{}{},
								"OtherAttributes": []interface{}{
									map[string]interface{}{
										"key": "value",
									},
									map[string]interface{}{
										"key2": "value2",
									},
								},
							},
						},
					},
				},
			},
			expectedMessages: []common.MapStr{
				{
					"Event": common.MapStr{
						"timestamp":  "1606324417",
						"Attributes": []interface{}{},
						"OtherAttributes": common.MapStr{
							"key": "value",
						},
					},
				},
				{
					"Event": common.MapStr{
						"timestamp":  "1606324417",
						"Attributes": []interface{}{},
						"OtherAttributes": common.MapStr{
							"key2": "value2",
						},
					},
				},
			},
			expectedErr: nil,
		},
		{
			name: "A missing array in an object",
			config: &splitConfig{
				Target: "body.response",
				Type:   "array",
				Split: &splitConfig{
					Target:           "body.Event.Attributes",
					IgnoreEmptyValue: true,
					KeepParent:       true,
					Split: &splitConfig{
						Target:     "body.Event.OtherAttributes",
						KeepParent: true,
					},
				},
			},
			ctx: emptyTransformContext(),
			resp: transformable{
				"body": common.MapStr{
					"response": []interface{}{
						map[string]interface{}{
							"Event": map[string]interface{}{
								"timestamp": "1606324417",
								"OtherAttributes": []interface{}{
									map[string]interface{}{
										"key": "value",
									},
									map[string]interface{}{
										"key2": "value2",
									},
								},
							},
						},
					},
				},
			},
			expectedMessages: []common.MapStr{
				{
					"Event": common.MapStr{
						"timestamp": "1606324417",
						"OtherAttributes": common.MapStr{
							"key": "value",
						},
					},
				},
				{
					"Event": common.MapStr{
						"timestamp": "1606324417",
						"OtherAttributes": common.MapStr{
							"key2": "value2",
						},
					},
				},
			},
			expectedErr: nil,
		},
		{
			name: "An empty map in an object",
			config: &splitConfig{
				Target: "body.response",
				Type:   "array",
				Split: &splitConfig{
					Target:           "body.Event.Attributes",
					Type:             "map",
					IgnoreEmptyValue: true,
					KeepParent:       true,
					Split: &splitConfig{
						Type:       "map",
						Target:     "body.Event.OtherAttributes",
						KeepParent: true,
					},
				},
			},
			ctx: emptyTransformContext(),
			resp: transformable{
				"body": common.MapStr{
					"response": []interface{}{
						map[string]interface{}{
							"Event": map[string]interface{}{
								"timestamp":  "1606324417",
								"Attributes": map[string]interface{}{},
								"OtherAttributes": map[string]interface{}{
									// Only include a single item here to avoid
									// map iteration order flakes.
									"1": map[string]interface{}{
										"key": "value",
									},
								},
							},
						},
					},
				},
			},
			expectedMessages: []common.MapStr{
				{
					"Event": common.MapStr{
						"timestamp":  "1606324417",
						"Attributes": common.MapStr{},
						"OtherAttributes": common.MapStr{
							"key": "value",
						},
					},
				},
			},
			expectedErr: nil,
		},
		{
			name: "A missing map in an object",
			config: &splitConfig{
				Target: "body.response",
				Type:   "array",
				Split: &splitConfig{
					Target:           "body.Event.Attributes",
					Type:             "map",
					IgnoreEmptyValue: true,
					KeepParent:       true,
					Split: &splitConfig{
						Type:       "map",
						Target:     "body.Event.OtherAttributes",
						KeepParent: true,
					},
				},
			},
			ctx: emptyTransformContext(),
			resp: transformable{
				"body": common.MapStr{
					"response": []interface{}{
						map[string]interface{}{
							"Event": map[string]interface{}{
								"timestamp": "1606324417",
								"OtherAttributes": map[string]interface{}{
									// Only include a single item here to avoid
									// map iteration order flakes.
									"1": map[string]interface{}{
										"key": "value",
									},
								},
							},
						},
					},
				},
			},
			expectedMessages: []common.MapStr{
				{
					"Event": common.MapStr{
						"timestamp": "1606324417",
						"OtherAttributes": common.MapStr{
							"key": "value",
						},
					},
				},
			},
			expectedErr: nil,
		},
		{
			name: "An empty string",
			config: &splitConfig{
				Target:           "body.items",
				Type:             "string",
				DelimiterString:  "\n",
				IgnoreEmptyValue: true,
				Split: &splitConfig{
					Target:          "body.other_items",
					Type:            "string",
					DelimiterString: "\n",
				},
			},
			ctx: emptyTransformContext(),
			resp: transformable{
				"body": common.MapStr{
					"@timestamp":  "1234567890",
					"items":       "",
					"other_items": "Line 1\nLine 2\nLine 3",
				},
			},
			expectedMessages: []common.MapStr{
				{"@timestamp": "1234567890", "items": "", "other_items": "Line 1"},
				{"@timestamp": "1234567890", "items": "", "other_items": "Line 2"},
				{"@timestamp": "1234567890", "items": "", "other_items": "Line 3"},
			},
		},
		{
			name: "A missing string",
			config: &splitConfig{
				Target:           "body.items",
				Type:             "string",
				DelimiterString:  "\n",
				IgnoreEmptyValue: true,
				Split: &splitConfig{
					Target:          "body.other_items",
					Type:            "string",
					DelimiterString: "\n",
				},
			},
			ctx: emptyTransformContext(),
			resp: transformable{
				"body": common.MapStr{
					"@timestamp":  "1234567890",
					"other_items": "Line 1\nLine 2\nLine 3",
				},
			},
			expectedMessages: []common.MapStr{
				{"@timestamp": "1234567890", "other_items": "Line 1"},
				{"@timestamp": "1234567890", "other_items": "Line 2"},
				{"@timestamp": "1234567890", "other_items": "Line 3"},
			},
		},
		{
			name: "Array of Strings with keep_parent",
			config: &splitConfig{
				Target:     "body.alerts",
				Type:       "array",
				KeepParent: true,
			},
			ctx: emptyTransformContext(),
			resp: transformable{
				"body": common.MapStr{
					"this": "is kept",
					"alerts": []interface{}{
						"test1",
						"test2",
						"test3",
					},
				},
			},
			expectedMessages: []common.MapStr{
				{
					"this":        "is kept",
					"alerts.data": "test1",
				},
				{
					"this":        "is kept",
					"alerts.data": "test2",
				},
				{
					"this":        "is kept",
					"alerts.data": "test3",
				},
			},
			expectedErr: nil,
		},
		{
			name: "Array of Arrays with keep_parent",
			config: &splitConfig{
				Target:     "body.alerts",
				Type:       "array",
				KeepParent: true,
			},
			ctx: emptyTransformContext(),
			resp: transformable{
				"body": common.MapStr{
					"this": "is kept",
					"alerts": []interface{}{
						[]interface{}{"test1-1", "test1-2"},
						[]string{"test2-1", "test2-2"},
						[]int{1, 2},
					},
				},
			},
			expectedMessages: []common.MapStr{
				{
					"this": "is kept",
					"alerts.data": []interface{}{
						"test1-1",
						"test1-2",
					},
				},
				{
					"this": "is kept",
					"alerts.data": []string{
						"test2-1",
						"test2-2",
					},
				},
				{
					"this": "is kept",
					"alerts.data": []int{
						1,
						2,
					},
				},
			},
			expectedErr: nil,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ch := make(chan maybeMsg, len(tc.expectedMessages))
			split, err := newSplitResponse(tc.config, logp.NewLogger(""))
			assert.NoError(t, err)
			err = split.run(tc.ctx, tc.resp, ch)
			if tc.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedErr.Error())
			}
			close(ch)
			assert.Equal(t, len(tc.expectedMessages), len(ch))
			for _, msg := range tc.expectedMessages {
				e := <-ch
				assert.NoError(t, e.err)
				assert.Equal(t, msg.Flatten(), e.msg.Flatten())
			}
		})
	}
}
