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

package index

import (
	"strings"
	"testing"
	"testing/quick"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/version"
)

func TestGetServiceURIExpectedPath(t *testing.T) {
	path770 := strings.Replace(statsPath, expandWildcards, expandWildcards+hiddenSuffix, 1)
	path800 := strings.Replace(path770, statsMetrics, statsMetrics+bulkSuffix, 1) + allowClosedIndices

	tests := map[string]struct {
		esVersion    *version.V
		expectedPath string
	}{
		"bulk_stats_unavailable": {
			esVersion:    version.MustNew("7.6.0"),
			expectedPath: statsPath,
		},
		"bulk_stats_available": {
			esVersion:    version.MustNew("8.0.0"),
			expectedPath: path800,
		},
		"expand_wildcards_hidden_unavailable": {
			esVersion:    version.MustNew("7.6.0"),
			expectedPath: statsPath,
		},
		"expand_wildcards_hidden_available": {
			esVersion:    version.MustNew("7.7.0"),
			expectedPath: path770,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			newURI, err := getServicePath(*test.esVersion)
			require.NoError(t, err)
			require.Equal(t, test.expectedPath, newURI)
		})
	}
}

func TestGetServiceURIMultipleCalls(t *testing.T) {
	path := strings.Replace(statsPath, expandWildcards, expandWildcards+hiddenSuffix, 1)
	path = strings.Replace(path, statsMetrics, statsMetrics+bulkSuffix, 1)
	path += allowClosedIndices

	err := quick.Check(func(r uint) bool {
		numCalls := 2 + (r % 10) // between 2 and 11

		var uri string
		var err error
		for i := uint(0); i < numCalls; i++ {
			uri, err = getServicePath(*version.MustNew("8.0.0"))
			if err != nil {
				return false
			}
		}

		return err == nil && uri == path
	}, nil)
	require.NoError(t, err)
}
