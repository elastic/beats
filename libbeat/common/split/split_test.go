// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package split

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestSplit(t *testing.T) {
	cases := []struct {
		name             string
		config           *SplitConfig
		json             mapstr.M
		expectedMessages []mapstr.M
		expectedErr      error
	}{
		{
			name: "Single Split with keep_parent",
			config: &SplitConfig{
				Target:     "Records",
				KeepParent: true,
			},
			json: mapstr.M{
				"Records": []interface{}{
					map[string]interface{}{
						"this_is": "also kept",
					},
					map[string]interface{}{
						"this_is": "also kept 2",
					},
				},
			},
			expectedMessages: []mapstr.M{
				{
					"Records.this_is": "also kept",
				},
				{
					"Records.this_is": "also kept 2",
				},
			},
			expectedErr: nil,
		},
		{
			name: "Two nested Split Arrays with keep_parent",
			config: &SplitConfig{
				Target:     "Records",
				KeepParent: true,
				Split: &SplitConfig{
					Target:     "Records.sub_array",
					KeepParent: true,
				},
			},
			json: mapstr.M{
				"this": "is kept",
				"Records": []interface{}{
					map[string]interface{}{
						"this_is": "also kept",
						"sub_array": []interface{}{
							map[string]interface{}{
								"something_else": "also kept",
							},
						},
					},
					map[string]interface{}{
						"this_is": "also kept 2",
						"sub_array": []interface{}{
							map[string]interface{}{
								"something_else": "also kept 2",
							},
						},
					},
				},
			},
			expectedMessages: []mapstr.M{
				{
					"this":                             "is kept",
					"Records.this_is":                  "also kept",
					"Records.sub_array.something_else": "also kept",
				},
				{
					"this":                             "is kept",
					"Records.this_is":                  "also kept 2",
					"Records.sub_array.something_else": "also kept 2",
				},
			},
			expectedErr: nil,
		},
		{
			name: "Single Split without keep_parent",
			config: &SplitConfig{
				Target:     "Records",
				KeepParent: false,
			},
			json: mapstr.M{
				"Records": []interface{}{
					map[string]interface{}{
						"this_is": "also kept",
					},
					map[string]interface{}{
						"this_is": "also kept 2",
					},
				},
			},
			expectedMessages: []mapstr.M{
				{
					"this_is": "also kept",
				},
				{
					"this_is": "also kept 2",
				},
			},
			expectedErr: nil,
		},
		{
			name: "Two nested Split Arrays without keep_parent",
			config: &SplitConfig{
				Target:     "Records",
				KeepParent: false,
				Split: &SplitConfig{
					Target:     "sub_array",
					KeepParent: false,
				},
			},
			json: mapstr.M{
				"this": "is kept",
				"Records": []interface{}{
					map[string]interface{}{
						"this_is": "also kept",
						"sub_array": []interface{}{
							map[string]interface{}{
								"something_else": "also kept",
							},
						},
					},
					map[string]interface{}{
						"this_is": "also kept 2",
						"sub_array": []interface{}{
							map[string]interface{}{
								"something_else": "also kept 2",
							},
						},
					},
				},
			},
			expectedMessages: []mapstr.M{
				{
					"something_else": "also kept",
				},
				{
					"something_else": "also kept 2",
				},
			},
			expectedErr: nil,
		},
		{
			name: "Two nested Split Arrays without first keep_parent",
			config: &SplitConfig{
				Target:     "Records",
				KeepParent: false,
				Split: &SplitConfig{
					Target:     "sub_array",
					KeepParent: true,
				},
			},
			json: mapstr.M{
				"this": "is kept",
				"Records": []interface{}{
					map[string]interface{}{
						"this_is": "also kept",
						"sub_array": []interface{}{
							map[string]interface{}{
								"something_else": "also kept",
							},
						},
					},
					map[string]interface{}{
						"this_is": "also kept 2",
						"sub_array": []interface{}{
							map[string]interface{}{
								"something_else": "also kept 2",
							},
						},
					},
				},
			},
			expectedMessages: []mapstr.M{
				{
					"this_is":                  "also kept",
					"sub_array.something_else": "also kept",
				},
				{
					"this_is":                  "also kept 2",
					"sub_array.something_else": "also kept 2",
				},
			},
			expectedErr: nil,
		},
		{
			name: "Two nested Split Arrays without second keep_parent",
			config: &SplitConfig{
				Target:     "Records",
				KeepParent: true,
				Split: &SplitConfig{
					Target:     "Records.sub_array",
					KeepParent: false,
				},
			},
			json: mapstr.M{
				"this": "is kept",
				"Records": []interface{}{
					map[string]interface{}{
						"this_is": "also kept",
						"sub_array": []interface{}{
							map[string]interface{}{
								"something_else": "also kept",
							},
						},
					},
					map[string]interface{}{
						"this_is": "also kept 2",
						"sub_array": []interface{}{
							map[string]interface{}{
								"something_else": "also kept 2",
							},
						},
					},
				},
			},
			expectedMessages: []mapstr.M{
				{
					"something_else": "also kept",
				},
				{
					"something_else": "also kept 2",
				},
			},
			expectedErr: nil,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ch := make(chan maybeMsg, len(tc.expectedMessages))
			split, err := NewSplit(tc.config, logp.NewLogger(""))
			assert.NoError(t, err)
			err = split.run(tc.json, ch)
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
				assert.Equal(t, msg.Flatten(), e.Msg.Flatten())
			}
		})
	}
}
