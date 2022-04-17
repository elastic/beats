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

package node_stats

import (
	"errors"
	"testing"
	"testing/quick"

	"github.com/stretchr/testify/require"
)

func TestGetServiceURI(t *testing.T) {
	tests := map[string]struct {
		currURI            string
		xpackEnabled       bool
		graphAPIsAvailable func() error
		expectedURI        string
		errExpected        bool
	}{
		"apis_unavailable": {
			currURI:            "/_node/stats",
			xpackEnabled:       true,
			graphAPIsAvailable: func() error { return errors.New("test") },
			expectedURI:        "",
			errExpected:        true,
		},
		"with_pipeline_vertices": {
			currURI:            "_node/stats",
			xpackEnabled:       true,
			graphAPIsAvailable: func() error { return nil },
			expectedURI:        "/_node/stats?vertices=true",
			errExpected:        false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			newURI, err := getServiceURI(nodeStatsPath, test.graphAPIsAvailable)
			if test.errExpected {
				require.Equal(t, "", newURI)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expectedURI, newURI)
			}
		})
	}
}

// See https://github.com/menderesk/beats/issues/15974
func TestGetServiceURIMultipleCalls(t *testing.T) {
	err := quick.Check(func(r uint) bool {
		var err error
		uri := "_node/stats"

		numCalls := 2 + (r % 10) // between 2 and 11
		for i := uint(0); i < numCalls; i++ {
			uri, err = getServiceURI(uri, func() error { return nil })
			if err != nil {
				return false
			}
		}

		return err == nil && uri == "_node/stats?vertices=true"
	}, nil)
	require.NoError(t, err)
}
