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

package prometheus

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestOpLabelKeyPrefixRemover(t *testing.T) {
	tests := []struct {
		name           string
		prefix         string
		labels         mapstr.M
		expectedLabels mapstr.M
	}{
		{
			name:   "prefix removed from matching key",
			prefix: "exported_",
			labels: mapstr.M{
				"exported_job": "myjob",
				"instance":     "localhost:9090",
			},
			expectedLabels: mapstr.M{
				"job":      "myjob",
				"instance": "localhost:9090",
			},
		},
		{
			name:   "key shorter than prefix is left untouched",
			prefix: "exported_",
			labels: mapstr.M{
				"job": "myjob",
			},
			expectedLabels: mapstr.M{
				"job": "myjob",
			},
		},
		{
			name:   "prefix shorter than 6 chars with key longer than prefix but shorter than 6",
			prefix: "ex_",
			labels: mapstr.M{
				"ex_j":        "myjob",
				"ex_instance": "localhost",
			},
			expectedLabels: mapstr.M{
				"j":        "myjob",
				"instance": "localhost",
			},
		},
		{
			name:   "no matching keys leaves labels unchanged",
			prefix: "exported_",
			labels: mapstr.M{
				"job":      "myjob",
				"instance": "localhost:9090",
			},
			expectedLabels: mapstr.M{
				"job":      "myjob",
				"instance": "localhost:9090",
			},
		},
		{
			name:           "empty labels",
			prefix:         "exported_",
			labels:         mapstr.M{},
			expectedLabels: mapstr.M{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			option := OpLabelKeyPrefixRemover(tt.prefix)
			require.NotPanics(t, func() {
				_, _, result := option.Process("", nil, tt.labels)
				assert.Equal(t, tt.expectedLabels, result)
			})
		})
	}
}
