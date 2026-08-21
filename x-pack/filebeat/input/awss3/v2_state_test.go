// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestStateRegistryV2_NormalMode(t *testing.T) {
	store := openTestStatestore()
	reg, err := newStateRegistryV2(stateRegistryV2Config{
		Log:   logptest.NewTestingLogger(t, t.Name()),
		Store: store,
	})
	require.NoError(t, err)
	defer reg.Close()

	obj := s3EventV2{}
	obj.S3.Object.LastModified = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	err = reg.MarkProcessed("bucket", "key", "etag", obj)
	require.NoError(t, err)

	id := stateID("bucket", "key", "etag", obj.S3.Object.LastModified, false)
	assert.True(t, reg.IsProcessed(id), "marked object must be processed")

	// Normal mode has no tail.
	assert.Empty(t, reg.GetStartAfterKey())
}

func TestStateRegistryV2_NormalMode_Persistence(t *testing.T) {
	store := openTestStatestore()
	log := logptest.NewTestingLogger(t, t.Name())

	obj := s3EventV2{}
	obj.S3.Object.LastModified = time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

	reg1, err := newStateRegistryV2(stateRegistryV2Config{Log: log, Store: store})
	require.NoError(t, err)
	require.NoError(t, reg1.MarkProcessed("bucket", "key", "etag", obj))
	reg1.Close()

	reg2, err := newStateRegistryV2(stateRegistryV2Config{Log: log, Store: store})
	require.NoError(t, err)
	defer reg2.Close()

	id := stateID("bucket", "key", "etag", obj.S3.Object.LastModified, false)
	assert.True(t, reg2.IsProcessed(id), "state must survive reload")
}

func TestStateRegistryV2_FailedIsPermanent(t *testing.T) {
	store := openTestStatestore()
	reg, err := newStateRegistryV2(stateRegistryV2Config{
		Log:   logptest.NewTestingLogger(t, t.Name()),
		Store: store,
	})
	require.NoError(t, err)
	defer reg.Close()

	obj := s3EventV2{}
	obj.S3.Object.LastModified = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	err = reg.MarkFailed("bucket", "key", "etag", obj)
	require.NoError(t, err)

	id := stateID("bucket", "key", "etag", obj.S3.Object.LastModified, false)
	assert.True(t, reg.IsProcessed(id), "failed state must count as processed")
}

func TestStateRegistryV2_LexicographicalMode(t *testing.T) {
	store := openTestStatestore()
	reg, err := newStateRegistryV2(stateRegistryV2Config{
		Log:      logptest.NewTestingLogger(t, t.Name()),
		Store:    store,
		Capacity: 10,
	})
	require.NoError(t, err)
	defer reg.Close()

	// Mark in-flight, verify tail tracking.
	require.NoError(t, reg.MarkObjectInFlight("a/001.log"))
	assert.Equal(t, "a/001.log", reg.GetStartAfterKey())

	require.NoError(t, reg.MarkObjectInFlight("a/002.log"))
	assert.Equal(t, "a/001.log", reg.GetStartAfterKey(), "tail stays at smallest in-flight")

	// Complete the first object.
	obj := s3EventV2{}
	obj.S3.Object.LastModified = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, reg.MarkProcessed("bucket", "a/001.log", "e1", obj))

	id := stateID("bucket", "a/001.log", "e1", obj.S3.Object.LastModified, true)
	assert.True(t, reg.IsProcessed(id))
}

func TestStateRegistryV2_LexicographicalMode_Capacity(t *testing.T) {
	store := openTestStatestore()
	reg, err := newStateRegistryV2(stateRegistryV2Config{
		Log:      logptest.NewTestingLogger(t, t.Name()),
		Store:    store,
		Capacity: 2,
	})
	require.NoError(t, err)
	defer reg.Close()

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Add 3 states with capacity 2 — smallest should be evicted.
	obj := s3EventV2{}
	obj.S3.Object.LastModified = base
	require.NoError(t, reg.MarkProcessed("bucket", "a", "e1", obj))
	require.NoError(t, reg.MarkProcessed("bucket", "b", "e2", obj))
	require.NoError(t, reg.MarkProcessed("bucket", "c", "e3", obj))

	// "a" should have been evicted (smallest key).
	idA := stateID("bucket", "a", "e1", base, true)
	assert.False(t, reg.IsProcessed(idA), "smallest key evicted at capacity")

	idB := stateID("bucket", "b", "e2", base, true)
	assert.True(t, reg.IsProcessed(idB))

	idC := stateID("bucket", "c", "e3", base, true)
	assert.True(t, reg.IsProcessed(idC))
}

func TestStateRegistryV2_LexicographicalMode_Persistence(t *testing.T) {
	store := openTestStatestore()
	log := logptest.NewTestingLogger(t, t.Name())

	reg1, err := newStateRegistryV2(stateRegistryV2Config{
		Log:      log,
		Store:    store,
		Capacity: 10,
	})
	require.NoError(t, err)

	require.NoError(t, reg1.MarkObjectInFlight("z/file.log"))
	assert.Equal(t, "z/file.log", reg1.GetStartAfterKey())
	reg1.Close()

	// Reload — tail must persist.
	reg2, err := newStateRegistryV2(stateRegistryV2Config{
		Log:      log,
		Store:    store,
		Capacity: 10,
	})
	require.NoError(t, err)
	defer reg2.Close()

	assert.Equal(t, "z/file.log", reg2.GetStartAfterKey(), "tail must survive reload")
}
