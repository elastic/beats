// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigBlocksEqual(t *testing.T) {
	tests := []struct {
		name  string
		a, b  ConfigBlocks
		equal bool
	}{
		{
			name:  "empty lists or nil",
			a:     nil,
			b:     ConfigBlocks{},
			equal: true,
		},
		{
			name: "single element",
			a: ConfigBlocks{
				ConfigBlocksWithType{
					Type: "metricbeat.modules",
					Blocks: []*ConfigBlock{
						&ConfigBlock{
							Raw: map[string]interface{}{
								"foo": "bar",
							},
						},
					},
				},
			},
			b: ConfigBlocks{
				ConfigBlocksWithType{
					Type: "metricbeat.modules",
					Blocks: []*ConfigBlock{
						&ConfigBlock{
							Raw: map[string]interface{}{
								"foo": "bar",
							},
						},
					},
				},
			},
			equal: true,
		},
		{
			name: "single element with slices",
			a: ConfigBlocks{
				ConfigBlocksWithType{
					Type: "metricbeat.modules",
					Blocks: []*ConfigBlock{
						&ConfigBlock{
							Raw: map[string]interface{}{
								"foo": []string{"foo", "bar"},
							},
						},
					},
				},
			},
			b: ConfigBlocks{
				ConfigBlocksWithType{
					Type: "metricbeat.modules",
					Blocks: []*ConfigBlock{
						&ConfigBlock{
							Raw: map[string]interface{}{
								"foo": []string{"foo", "bar"},
							},
						},
					},
				},
			},
			equal: true,
		},
		{
			name: "different number of blocks",
			a: ConfigBlocks{
				ConfigBlocksWithType{
					Type: "metricbeat.modules",
					Blocks: []*ConfigBlock{
						&ConfigBlock{
							Raw: map[string]interface{}{
								"foo": "bar",
							},
						},
						&ConfigBlock{
							Raw: map[string]interface{}{
								"baz": "buzz",
							},
						},
					},
				},
			},
			b: ConfigBlocks{
				ConfigBlocksWithType{
					Type: "metricbeat.modules",
					Blocks: []*ConfigBlock{
						&ConfigBlock{
							Raw: map[string]interface{}{
								"foo": "bar",
							},
						},
					},
				},
			},
			equal: false,
		},
		{
			name: "different block",
			a: ConfigBlocks{
				ConfigBlocksWithType{
					Type: "metricbeat.modules",
					Blocks: []*ConfigBlock{
						&ConfigBlock{
							Raw: map[string]interface{}{
								"baz": "buzz",
							},
						},
					},
				},
			},
			b: ConfigBlocks{
				ConfigBlocksWithType{
					Type: "metricbeat.modules",
					Blocks: []*ConfigBlock{
						&ConfigBlock{
							Raw: map[string]interface{}{
								"foo": "bar",
							},
						},
					},
				},
			},
			equal: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			check, err := ConfigBlocksEqual(test.a, test.b)
			if !assert.NoError(t, err) {
				return
			}
			assert.Equal(t, test.equal, check)
		})
	}
}
