// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/elastic/elastic-agent-libs/logp"
)

func TestLiveProfileStoreEvictsAndDeletesFiles(t *testing.T) {
	dir := t.TempDir()
	store, err := newLiveProfileStore(logp.NewLogger("test"), dir, 1)
	if err != nil {
		t.Fatalf("newLiveProfileStore error: %v", err)
	}

	query1 := "select * from uptime"
	query2 := "select * from osquery_info"

	store.Record(query1, map[string]interface{}{"source": "live", "query": query1})
	file1 := filepath.Join(dir, liveProfileFilename(liveProfileKey(query1)))
	if _, err := os.Stat(file1); err != nil {
		t.Fatalf("expected profile file for query1, got error: %v", err)
	}

	store.Record(query2, map[string]interface{}{"source": "live", "query": query2})
	file2 := filepath.Join(dir, liveProfileFilename(liveProfileKey(query2)))
	if _, err := os.Stat(file2); err != nil {
		t.Fatalf("expected profile file for query2, got error: %v", err)
	}

	if _, err := os.Stat(file1); !os.IsNotExist(err) {
		t.Fatalf("expected query1 profile to be evicted, stat error: %v", err)
	}
}
