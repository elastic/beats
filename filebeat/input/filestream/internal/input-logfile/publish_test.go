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

package input_logfile

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v8/libbeat/beat"
	pubtest "github.com/elastic/beats/v8/libbeat/publisher/testing"
)

func TestPublish(t *testing.T) {
	t.Run("event with cursor state creates update operation", func(t *testing.T) {
		store := testOpenStore(t, "test", createSampleStore(t, nil))
		defer store.Release()
		cursor := makeCursor(store.Get("test::key"))

		var actual beat.Event
		client := &pubtest.FakeClient{
			PublishFunc: func(event beat.Event) { actual = event },
		}
		publisher := cursorPublisher{nil, client, &cursor}
		publisher.Publish(beat.Event{}, "test")

		require.NotNil(t, actual.Private)
	})

	t.Run("event without cursor creates no update operation", func(t *testing.T) {
		store := testOpenStore(t, "test", createSampleStore(t, nil))
		defer store.Release()
		cursor := makeCursor(store.Get("test::key"))

		var actual beat.Event
		client := &pubtest.FakeClient{
			PublishFunc: func(event beat.Event) { actual = event },
		}
		publisher := cursorPublisher{nil, client, &cursor}
		publisher.Publish(beat.Event{}, nil)
		require.Nil(t, actual.Private)
	})

	t.Run("publish returns error if context has been cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.TODO())
		cancel()

		store := testOpenStore(t, "test", createSampleStore(t, nil))
		defer store.Release()
		cursor := makeCursor(store.Get("test::key"))

		publisher := cursorPublisher{ctx, &pubtest.FakeClient{}, &cursor}
		err := publisher.Publish(beat.Event{}, nil)
		require.Equal(t, context.Canceled, err)
	})
}

func TestOp_Execute(t *testing.T) {
	t.Run("applying final op marks the key as finished", func(t *testing.T) {
		store := testOpenStore(t, "test", createSampleStore(t, nil))
		defer store.Release()
		res := store.Get("test::key")

		// create op and release resource. The 'resource' must still be active
		op := mustCreateUpdateOp(t, res, "test-updated-cursor-state")
		res.Release()
		require.False(t, res.Finished())

		// this was the last op, the resource should become inactive
		op.Execute(store, 1)
		require.True(t, res.Finished())

		// validate state:
		inSyncCursor := storeInSyncSnapshot(store)["test::key"].Cursor
		inMemCursor := storeMemorySnapshot(store)["test::key"].Cursor
		want := "test-updated-cursor-state"
		assert.Equal(t, want, inSyncCursor)
		assert.Equal(t, want, inMemCursor)
	})

	t.Run("acking multiple ops applies the latest update and marks key as finished", func(t *testing.T) {
		// when acking N events, intermediate updates are dropped in favor of the latest update operation.
		// This test checks that the resource is correctly marked as finished.

		store := testOpenStore(t, "test", createSampleStore(t, nil))
		defer store.Release()
		res := store.Get("test::key")

		// create update operations and release resource. The 'resource' must still be active
		mustCreateUpdateOp(t, res, "test-updated-cursor-state-dropped")
		op := mustCreateUpdateOp(t, res, "test-updated-cursor-state-final")
		res.Release()
		require.False(t, res.Finished())

		// this was the last op, the resource should become inactive
		op.Execute(store, 2)
		require.True(t, res.Finished())

		// validate state:
		inSyncCursor := storeInSyncSnapshot(store)["test::key"].Cursor
		inMemCursor := storeMemorySnapshot(store)["test::key"].Cursor
		want := "test-updated-cursor-state-final"
		assert.Equal(t, want, inSyncCursor)
		assert.Equal(t, want, inMemCursor)
	})

	t.Run("ACK only subset of pending ops will only update up to ACKed state", func(t *testing.T) {
		// when acking N events, intermediate updates are dropped in favor of the latest update operation.
		// This test checks that the resource is correctly marked as finished.

		store := testOpenStore(t, "test", createSampleStore(t, nil))
		defer store.Release()
		res := store.Get("test::key")

		// create update operations and release resource. The 'resource' must still be active
		op1 := mustCreateUpdateOp(t, res, "test-updated-cursor-state-intermediate")
		op2 := mustCreateUpdateOp(t, res, "test-updated-cursor-state-final")
		res.Release()
		require.False(t, res.Finished())

		defer op2.done(1) // cleanup after test

		// this was the intermediate op, the resource should still be active
		op1.Execute(store, 1)
		require.False(t, res.Finished())

		// validate state (in memory state is always up to data to most recent update):
		inSyncCursor := storeInSyncSnapshot(store)["test::key"].Cursor
		inMemCursor := storeMemorySnapshot(store)["test::key"].Cursor
		assert.Equal(t, "test-updated-cursor-state-intermediate", inSyncCursor)
		assert.Equal(t, "test-updated-cursor-state-final", inMemCursor)
	})
}

func mustCreateUpdateOp(t *testing.T, resource *resource, updates interface{}) *updateOp {
	op, err := createUpdateOp(resource, updates)
	if err != nil {
		t.Fatalf("Failed to create update op: %v", err)
	}
	return op
}
