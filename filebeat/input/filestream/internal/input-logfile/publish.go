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
	"time"

	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/transform/typeconv"
	"github.com/elastic/beats/v7/libbeat/statestore"
)

// Publisher is used to publish an event and update the cursor in a single call to Publish.
// Inputs are allowed to pass `nil` as cursor state. In this case the state is not updated, but the
// event will still be published as is.
type Publisher interface {
	Publish(event beat.Event, cursor interface{}) error
}

// cursorPublisher implements the Publisher interface and used internally by the managedInput.
// When publishing an event with cursor state updates, the cursorPublisher
// updates the in memory state and create an updateOp that is used to schedule
// an update for the persistent store. The updateOp is run by the inputs ACK
// handler, persisting the pending update.
type cursorPublisher struct {
	canceler input.Canceler
	client   beat.Client
	cursor   *Cursor
}

// updateOp keeps track of pending updates that are not written to the persistent store yet.
// Update operations are ordered. The input manager guarantees that only one
// input can create update operation for a source, such that new input
// instances can add update operations to be executed after already pending
// update operations from older inputs instances that have been shutdown.
type updateOp struct {
	resource *resource

	// state updates to persist
	timestamp time.Time
	delta     interface{}
}

func newUpdateOp(resource *resource, ts time.Time, delta interface{}) *updateOp {
	return &updateOp{
		resource:  resource,
		timestamp: ts,
		delta:     delta,
	}
}

func (op *updateOp) Key() string {
	return op.resource.key
}

// Publish publishes an event. Publish returns false if the inputs cancellation context has been marked as done.
// If cursorUpdate is not nil, Publish updates the in memory state and create and updateOp for the pending update.
// It overwrite event.Private with the update operation, before finally sending the event.
// The ACK ordering in the publisher pipeline guarantees that update operations
// will be ACKed and executed in the correct order.
func (c *cursorPublisher) Publish(event beat.Event, cursorUpdate interface{}) error {
	if cursorUpdate == nil {
		return c.forward(event)
	}

	op, err := createUpdateOp(c.cursor.resource, cursorUpdate)
	if err != nil {
		return err
	}

	event.Private = op
	return c.forward(event)
}

func (c *cursorPublisher) forward(event beat.Event) error {
	c.client.Publish(event)
	if c.canceler == nil {
		return nil
	}
	return c.canceler.Err()
}

func createUpdateOp(resource *resource, updates interface{}) (*updateOp, error) {
	ts := time.Now()

	resource.stateMutex.Lock()
	defer resource.stateMutex.Unlock()

	resource.pendingUpdate = updates

	resource.Retain()
	resource.activeCursorOperations++
	return newUpdateOp(resource, ts, updates), nil
}

// done releases resources held by the last N updateOps.
func (op *updateOp) done(n uint) {
	op.resource.UpdatesReleaseN(n)
	op.resource = nil
}

// Execute updates the persistent store with the scheduled changes and releases the resource.
func (op *updateOp) Execute(store *store, n uint) {
	resource := op.resource

	resource.stateMutex.Lock()
	defer resource.stateMutex.Unlock()

	if resource.lockedVersion != op.resource.version || resource.isDeleted() {
		return
	}

	defer op.done(n)
	resource.activeCursorOperations -= n
	if resource.activeCursorOperations == 0 {
		resource.cursor = resource.pendingCursor()
		resource.pendingCursorValue = nil
	} else {
		typeconv.Convert(&resource.cursor, op.delta)
	}

	if resource.internalState.Updated.Before(op.timestamp) {
		resource.internalState.Updated = op.timestamp
	}

	err := store.persistentStore.Set(resource.key, resource.inSyncStateSnapshot())
	if err != nil {
		if !statestore.IsClosed(err) {
			store.log.Errorf("Failed to update state in the registry for '%v'", resource.key)
		}
	} else {
		resource.stored = true
	}
}
