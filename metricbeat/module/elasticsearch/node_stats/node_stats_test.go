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
	"testing"

	"github.com/elastic/beats/v8/metricbeat/module/elasticsearch"

	"github.com/stretchr/testify/require"
)

func TestGetServiceURI(t *testing.T) {
	tests := map[string]struct {
		scope       elasticsearch.Scope
		expectedURI string
	}{
		"scope_node": {
			scope:       elasticsearch.ScopeNode,
			expectedURI: "/_nodes/_local/stats",
		},
		"scope_cluster": {
			scope:       elasticsearch.ScopeCluster,
			expectedURI: "/_nodes/_all/stats",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			newURI, err := getServiceURI("/foo/bar", test.scope)
			require.NoError(t, err)
			require.Equal(t, test.expectedURI, newURI)
		})
	}
}
