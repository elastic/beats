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
	"sync"

	"github.com/elastic/go-concert/unison"
)

// updateWriter asynchronously writes all updates to the persistent store.
// All updates are tracked as key value pairs. In case of back-pressure due to the local disk
// we overwrite the pending states that have not been written in memory,
// until the disk is ready for more updates.
type updateWriter struct {
	store *store
	tg    unison.TaskGroup
	ch    *updateChan
}

type updateChan struct {
	// pending update operations for key value pairs. The new state always
	// overwrites the old state.
	pending map[string]int
	updates []scheduledUpdate

	// we use a chan as conditional, so we can break on context cancellation.
	// `waiter` is set if the writer is waiting for new entries to be reported to the store.
	mutex  sync.Mutex
	waiter chan struct{}
}

type scheduledUpdate struct {
	op scheduledOp
	n  uint
}

type scheduledOp interface {
	Key() string
	Execute(store *store, n uint)
}

func newUpdateWriter(store *store, ch *updateChan) *updateWriter {
	w := &updateWriter{
		store: store,
		ch:    ch,
	}
	w.tg.Go(func(ctx context.Context) error {
		w.run(ctx)
		return nil
	})

	return w
}

// Close stops the background writing provess and attempts to serialize
// all pending operations.
func (w *updateWriter) Close() {
	w.tg.Stop()
	w.syncStates(w.ch.TryRecv())
}

func (w *updateWriter) run(ctx context.Context) {
	for ctx.Err() == nil {
		updates, err := w.ch.Recv(ctx)
		if err != nil {
			return
		}

		w.syncStates(updates)
	}
}

func (w *updateWriter) syncStates(updates []scheduledUpdate) {
	for _, upd := range updates {
		upd.op.Execute(w.store, upd.n)
	}
}

func newUpdateChan() *updateChan {
	return &updateChan{
		pending: map[string]int{},
	}
}

// Send adds a new update to the channel. Update operations
// for the same resource key will be merged by dropping the old operation.
func (ch *updateChan) Send(upd scheduledUpdate) {
	ch.mutex.Lock()
	defer ch.mutex.Unlock()

	key := upd.op.Key()

	idx, exists := ch.pending[key]
	if !exists {
		idx = len(ch.updates)
		ch.updates = append(ch.updates, upd)
		ch.pending[key] = idx
	} else {
		ch.updates[idx].op = upd.op
		ch.updates[idx].n += upd.n
	}

	// notify pending Read that new updates are available.
	if ch.waiter != nil {
		close(ch.waiter)
		ch.waiter = nil
	}
}

// Recv waits until at least one entry is available and returns a table of key
// value pairs with pending updates that need to be written to the registry.
func (ch *updateChan) Recv(ctx context.Context) ([]scheduledUpdate, error) {
	ch.mutex.Lock()

	for ctx.Err() == nil {
		updates := ch.updates
		if len(updates) > 0 {
			ch.pending = map[string]int{}
			ch.updates = nil
			ch.mutex.Unlock()
			return updates, nil
		}

		waiter := make(chan struct{})
		ch.waiter = waiter
		ch.mutex.Unlock()

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-waiter:
			ch.mutex.Lock()
		}
	}

	return nil, ctx.Err()
}

// TryRecv returns available update operations or nil if there
// are no pending operations. The channel is cleared if there had been pending
// operations.
func (ch *updateChan) TryRecv() []scheduledUpdate {
	ch.mutex.Lock()
	defer ch.mutex.Unlock()

	updates := ch.updates
	if len(updates) > 0 {
		ch.pending = map[string]int{}
		ch.updates = nil
	}

	return updates
}
