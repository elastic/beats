// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

func TestSplit(t *testing.T) {
	cases := []struct {
		name             string
		config           *splitConfig
		ctx              transformContext
		resp             *transformable
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
			resp: &transformable{
				body: common.MapStr{
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
			resp: &transformable{
				body: common.MapStr{
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
