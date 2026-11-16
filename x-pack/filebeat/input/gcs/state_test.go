// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcs

import (
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/pkg/logp"
	"github.com/elastic/beats/v7/pkg/monitoring"
)

func TestCheckpointMixedCaseObjectNames(t *testing.T) {
	// In GCS listing order. HOST-Z sorts last only case-insensitively.
	names := []string{"HOST-Z.json", "host-a.json", "host-b.json"}
	ts := time.Now()
	var jobs []*job
	for _, name := range names {
		jobs = append(jobs, &job{object: &storage.ObjectAttrs{Name: name, Updated: ts}})
	}

	st := newState()
	metrics := newInputMetrics(monitoring.NewRegistry(), logp.NewNopLogger())
	for _, j := range jobs {
		_, done := st.saveForTx(j.Name(), j.Timestamp(), metrics)
		done()
	}

	var again []string
	for _, j := range (&scheduler{state: st}).moveToLastSeenJob(jobs) {
		again = append(again, j.Name())
	}
	assert.Empty(t, again, "processed objects were selected again")
}
