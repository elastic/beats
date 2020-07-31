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

	"github.com/elastic/beats/v7/libbeat/common"

	"github.com/stretchr/testify/require"
)

func TestGetServiceURI(t *testing.T) {
	tests := map[string]struct {
		esVersion    *common.Version
		expectedPath string
	}{
		"bulk_stats_unavailable": {
			esVersion:    common.MustNewVersion("7.7.0"),
			expectedPath: statsPath,
		},
		"bulk_stats_available": {
			esVersion:    common.MustNewVersion("8.0.0"),
			expectedPath: strings.Replace(statsPath, statsMetrics, statsMetrics+",bulk", 1),
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
	err := quick.Check(func(r uint) bool {
		numCalls := 2 + (r % 10) // between 2 and 11

		var uri string
		var err error
		for i := uint(0); i < numCalls; i++ {
			uri, err = getServicePath(*common.MustNewVersion("8.0.0"))
			if err != nil {
				return false
			}
		}

		return err == nil && uri == strings.Replace(statsPath, statsMetrics, statsMetrics+",bulk", 1)
	}, nil)
	require.NoError(t, err)
}
