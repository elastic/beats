// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/x-pack/libbeat/management/api"
)

func TestHasConfig(t *testing.T) {
	tests := map[string]struct {
		configs  api.ConfigBlocks
		expected bool
	}{
		"with config": {
			configs: api.ConfigBlocks{
				api.ConfigBlocksWithType{Type: "metricbeat "},
			},
			expected: true,
		},
		"without config": {
			configs:  api.ConfigBlocks{},
			expected: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cache := Cache{Configs: test.configs}
			assert.Equal(t, test.expected, cache.HasConfig())
		})
	}
}
