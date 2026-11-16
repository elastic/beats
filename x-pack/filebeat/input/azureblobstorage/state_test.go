// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azureblobstorage

import (
	"testing"
	"time"

	azcontainer "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/stretchr/testify/assert"
)

func TestCheckpointMixedCaseBlobNames(t *testing.T) {
	// In Azure listing order. HOST-Z sorts last only case-insensitively.
	names := []string{"HOST-Z.json", "host-a.json", "host-b.json"}
	ts := time.Now()
	var jobs []*job
	for _, name := range names {
		jobs = append(jobs, &job{blob: &azcontainer.BlobItem{
			Name:       &name,
			Properties: &azcontainer.BlobProperties{LastModified: &ts},
		}})
	}

	st := newState()
	for _, j := range jobs {
		_, done := st.saveForTx(j.name(), *j.timestamp())
		done()
	}

	var again []string
	for _, j := range (&scheduler{state: st}).moveToLastSeenJob(jobs) {
		again = append(again, j.name())
	}
	assert.Empty(t, again, "processed blobs were selected again")
}
