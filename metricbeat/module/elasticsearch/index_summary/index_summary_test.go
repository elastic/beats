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
