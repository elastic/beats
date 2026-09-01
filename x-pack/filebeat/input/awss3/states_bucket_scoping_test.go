// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/backend/memlog"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

// diskBackedStatestore mimics the production wiring: a memlog store persisted
// on disk and shared by every aws-s3 input of the process. Reopening it over
// the same home directory simulates a beat restart.
type diskBackedStatestore struct {
	registry *statestore.Registry
}

func (s *diskBackedStatestore) StoreFor(string) (*statestore.Store, error) {
	return s.registry.Get("filebeat")
}
func (s *diskBackedStatestore) CleanupInterval() time.Duration { return 24 * time.Hour }
func (s *diskBackedStatestore) Close()                         { _ = s.registry.Close() }

func openDiskStatestore(t *testing.T, home string) *diskBackedStatestore {
	t.Helper()
	backend, err := memlog.New(logptest.NewTestingLogger(t, "memlog"), memlog.Settings{Root: home})
	require.NoError(t, err)
	return &diskBackedStatestore{registry: statestore.NewRegistry(backend)}
}

// TestStateRegistryPersistsAcrossRestart verifies that a state written to the
// on-disk store is still recognized as processed after the store is closed
// and reopened. The lookup uses a state rebuilt from the values a fresh S3
// listing would return, so it also proves the state ID survives the
// serialization round trip.
func TestStateRegistryPersistsAcrossRestart(t *testing.T) {
	home := t.TempDir()
	log := logptest.NewTestingLogger(t, t.Name())

	lastModified, err := time.Parse(time.RFC3339, "2026-08-11T03:52:33Z")
	require.NoError(t, err)
	st := newState("bucket-a", "logs/object-1.gz", `"etag-1"`, lastModified)
	st.Stored = true

	store := openDiskStatestore(t, home)
	reg, err := newStates(log, store, "bucket-a", "")
	require.NoError(t, err)
	require.NoError(t, reg.AddState(st))
	require.True(t, reg.IsProcessed(st))
	reg.Close()
	store.Close()

	// Restart: reopen from disk and check against a freshly listed state.
	store2 := openDiskStatestore(t, home)
	defer store2.Close()
	reg2, err := newStates(log, store2, "bucket-a", "")
	require.NoError(t, err)
	defer reg2.Close()

	freshLastModified, err := time.Parse(time.RFC3339, "2026-08-11T03:52:33Z")
	require.NoError(t, err)
	fresh := newState("bucket-a", "logs/object-1.gz", `"etag-1"`, freshLastModified)
	assert.True(t, reg2.IsProcessed(fresh),
		"state persisted before restart must be recognized as processed after reopen")
}

// TestStateRegistryCleanUpDoesNotAffectOtherInputs is a regression test for
// the shared store scoping. Two polling inputs (different buckets) share one
// on-disk store. The cleanup of one input after its listing pass must not
// remove the other input's persisted states, or the other input re-ingests
// its whole bucket after the next restart.
func TestStateRegistryCleanUpDoesNotAffectOtherInputs(t *testing.T) {
	home := t.TempDir()
	log := logptest.NewTestingLogger(t, t.Name())

	lastModified, err := time.Parse(time.RFC3339, "2026-08-11T03:52:33Z")
	require.NoError(t, err)

	stateA := newState("bucket-a", "AWSLogs/111/elb/us-east-1/a.log.gz", `"etag-a"`, lastModified)
	stateA.Stored = true
	stateB := newState("bucket-b", "AWSLogs/111/elb/eu-west-1/b.log.gz", `"etag-b"`, lastModified)
	stateB.Stored = true

	// Run 1: each input processes one object and persists its state.
	store1 := openDiskStatestore(t, home)
	regA1, err := newStates(log, store1, "bucket-a", "")
	require.NoError(t, err)
	regB1, err := newStates(log, store1, "bucket-b", "")
	require.NoError(t, err)
	require.NoError(t, regA1.AddState(stateA))
	require.NoError(t, regB1.AddState(stateB))
	regA1.Close()
	regB1.Close()
	store1.Close()

	// Run 2 (restart): input A completes a listing of its own bucket and
	// cleans up. It must neither see nor touch input B's states.
	store2 := openDiskStatestore(t, home)
	regA2, err := newStates(log, store2, "bucket-a", "")
	require.NoError(t, err)
	regB2, err := newStates(log, store2, "bucket-b", "")
	require.NoError(t, err)

	assert.True(t, regA2.IsProcessed(stateA))
	assert.False(t, regA2.IsProcessed(stateB),
		"input A must not load input B's states from the shared store")
	assert.True(t, regB2.IsProcessed(stateB))

	require.NoError(t, regA2.CleanUp([]string{stateA.ID()}))

	regA2.Close()
	regB2.Close()
	store2.Close()

	// Run 3 (restart): input B must still know its object was processed.
	store3 := openDiskStatestore(t, home)
	defer store3.Close()
	regB3, err := newStates(log, store3, "bucket-b", "")
	require.NoError(t, err)
	defer regB3.Close()

	fresh := newState("bucket-b", "AWSLogs/111/elb/eu-west-1/b.log.gz", `"etag-b"`, lastModified)
	assert.True(t, regB3.IsProcessed(fresh),
		"input A's cleanup must not delete input B's persisted state")

	regA3, err := newStates(log, store3, "bucket-a", "")
	require.NoError(t, err)
	defer regA3.Close()
	assert.True(t, regA3.IsProcessed(stateA),
		"input A's own state must survive its cleanup")
}

// TestStateRegistryScopedLoad verifies that a registry only loads states of
// its own bucket, while an unscoped registry (empty bucket) still loads
// everything.
func TestStateRegistryScopedLoad(t *testing.T) {
	lastModified := time.Unix(1733221244, 0)
	store := openTestStatestore()

	stA := newState("bucket-a", "objA", "etagA", lastModified)
	stA.Stored = true
	stB := newState("bucket-b", "objB", "etagB", lastModified)
	stB.Stored = true

	regA, err := newStates(nil, store, "bucket-a", "")
	require.NoError(t, err)
	regB, err := newStates(nil, store, "bucket-b", "")
	require.NoError(t, err)
	require.NoError(t, regA.AddState(stA))
	require.NoError(t, regB.AddState(stB))
	regA.Close()
	regB.Close()

	scoped, err := newStates(nil, store, "bucket-a", "")
	require.NoError(t, err)
	defer scoped.Close()
	assert.True(t, scoped.IsProcessed(stA))
	assert.False(t, scoped.IsProcessed(stB), "must not load another bucket's state")

	unscoped, err := newStates(nil, store, "", "")
	require.NoError(t, err)
	defer unscoped.Close()
	assert.True(t, unscoped.IsProcessed(stA))
	assert.True(t, unscoped.IsProcessed(stB), "empty bucket loads states of any bucket")
}

// TestStateRegistryCleanUpRespectsKeyPrefix covers two inputs polling the
// same bucket with disjoint key prefixes: the cleanup of one must not remove
// the other's states.
func TestStateRegistryCleanUpRespectsKeyPrefix(t *testing.T) {
	lastModified := time.Unix(1733221244, 0)
	store := openTestStatestore()

	stA := newState("bucket", "prefix-a/obj", "etagA", lastModified)
	stA.Stored = true
	stB := newState("bucket", "prefix-b/obj", "etagB", lastModified)
	stB.Stored = true

	regA, err := newStates(nil, store, "bucket", "prefix-a/")
	require.NoError(t, err)
	regB, err := newStates(nil, store, "bucket", "prefix-b/")
	require.NoError(t, err)
	require.NoError(t, regA.AddState(stA))
	require.NoError(t, regB.AddState(stB))
	regB.Close()

	// Input A's listing returned nothing: its cleanup removes everything it
	// knows, which must not include input B's states.
	require.NoError(t, regA.CleanUp(nil))
	assert.False(t, regA.IsProcessed(stA))
	regA.Close()

	regB2, err := newStates(nil, store, "bucket", "prefix-b/")
	require.NoError(t, err)
	defer regB2.Close()
	assert.True(t, regB2.IsProcessed(stB),
		"cleanup of a same-bucket input with a different prefix must not delete this input's state")
}
