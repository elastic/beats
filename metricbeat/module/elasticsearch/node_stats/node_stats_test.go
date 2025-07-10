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
	"strconv"
	"testing"

	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"

	"github.com/stretchr/testify/require"
)

func TestGetServiceURI(t *testing.T) {
	scopes := []struct {
		name  string
		scope elasticsearch.Scope
	}{
		{name: "scope_node", scope: elasticsearch.ScopeNode},
		{name: "scope_cluster", scope: elasticsearch.ScopeCluster},
	}

	latestScopedURIs := map[elasticsearch.Scope]string{
		elasticsearch.ScopeNode:    "/_nodes/_local/stats/jvm,indices,fs,os,process,transport,thread_pool,indexing_pressure,ingest/bulk,docs,get,merge,translog,fielddata,indexing,query_cache,request_cache,search,shard_stats,store,segments,refresh,flush",
		elasticsearch.ScopeCluster: "/_nodes/_all/stats/jvm,indices,fs,os,process,transport,thread_pool,indexing_pressure,ingest/bulk,docs,get,merge,translog,fielddata,indexing,query_cache,request_cache,search,shard_stats,store,segments,refresh,flush",
	}

	legacyScopedURIs := map[elasticsearch.Scope]string{
		elasticsearch.ScopeNode:    "/_nodes/_local/stats",
		elasticsearch.ScopeCluster: "/_nodes/_all/stats",
	}

	tests := []struct {
		majorVersion int
		legacy       bool
	}{
		{majorVersion: 10, legacy: false},
		{majorVersion: 9, legacy: false},
		{majorVersion: 8, legacy: false},
		{majorVersion: 7, legacy: true},
		{majorVersion: 6, legacy: true},
		{majorVersion: 5, legacy: true},
		{majorVersion: 2, legacy: true},
	}

	for _, scope := range scopes {
		for _, test := range tests {
			t.Run("scope_"+scope.name+"_v"+strconv.Itoa(test.majorVersion), func(t *testing.T) {
				newURI, err := getServiceURI("/foo/bar", scope.scope, test.majorVersion)
				require.NoError(t, err)

				scopedURIs := latestScopedURIs

				if test.legacy {
					scopedURIs = legacyScopedURIs
				}

				require.Equal(t, scopedURIs[scope.scope], newURI)
			})
		}
	}
}
