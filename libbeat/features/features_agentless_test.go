// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package features

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_initFromEnv(t *testing.T) {
	const envName = "TEST_AGENTLESS_ENV"

	t.Run("Without setting env", func(t *testing.T) {
		// default init
		assert.False(t, IsElasticsearchStateStoreEnabled())
		assert.Empty(t, esTypesEnabled)
		assert.False(t, IsElasticsearchStateStoreEnabledForInput("xxx"))

		// init from env
		initFromEnv(envName)
		assert.False(t, IsElasticsearchStateStoreEnabled())
		assert.Empty(t, esTypesEnabled)
		assert.False(t, IsElasticsearchStateStoreEnabledForInput("xxx"))
	})

	tests := []struct {
		name         string
		value        string
		wantEnabled  bool
		wantContains []string
	}{
		{
			name:         "Empty",
			value:        "",
			wantEnabled:  false,
			wantContains: nil,
		},
		{
			name:         "Single value",
			value:        "xxx",
			wantEnabled:  true,
			wantContains: []string{"xxx"},
		},
		{
			name:         "Multiple values",
			value:        "xxx,yyy",
			wantEnabled:  true,
			wantContains: []string{"xxx", "yyy"},
		},
		{
			name:         "Multiple values with spaces",
			value:        ",,, ,  xxx  , yyy, ,,,,",
			wantEnabled:  true,
			wantContains: []string{"xxx", "yyy"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(envName, tt.value)
			initFromEnv(envName)

			assert.Equal(t, tt.wantEnabled, IsElasticsearchStateStoreEnabled())
			for _, contain := range tt.wantContains {
				assert.Contains(t, esTypesEnabled, contain)
				assert.True(t, IsElasticsearchStateStoreEnabledForInput(contain))
			}
			assert.Len(t, esTypesEnabled, len(tt.wantContains))
		})
	}
}
