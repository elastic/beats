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

//go:build !integration

package index_summary

import (
	"testing"
)

func TestGetServicePath(t *testing.T) {
	expectedPath := "/_nodes/stats?level=node&filter_path=nodes.*.indices.docs,nodes.*.indices.indexing.index_total,nodes.*.indices.indexing.index_time_in_millis,nodes.*.indices.search.query_total,nodes.*.indices.search.query_time_in_millis,nodes.*.indices.segments.count,nodes.*.indices.segments.memory_in_bytes,nodes.*.indices.store.size_in_bytes,nodes.*.indices.store.total_data_set_size_in_bytes"
	path, err := getServicePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, path)
	}
}
