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

	"github.com/elastic/go-concert/ctxtool"
	"github.com/elastic/go-concert/unison"
)

// updateWriter asynchronously writes all updates to the persistent store.
// All updates are tracked as key value pairs. In case of back-pressure due to the local disk
// we overwrite the pending states that have not been written in memory,
// until the disk is ready for more updates.
type updateWriter struct {
	store *store
	tg    unison.TaskGroup

	// we use a chan as conditional, so we can break on context cancellation.
	// `waiter` is set if the writer is waiting for new entries to be reported to the store.
	mutex  sync.Mutex
	waiter chan struct{}

	// pending update operations for key value pairs. The new state always
	// overwrites the old state.
	pending map[string]int
	updates []scheduledUpdate
}

type scheduledUpdate struct {
	op *updateOp
	n  uint
}

func newUpdateWriter(store *store) *updateWriter {
	w := &updateWriter{
		store:   store,
		pending: map[string]int{},
	}
	w.tg.Go(func(ctx unison.Canceler) error {
		w.run(ctxtool.FromCanceller(ctx))
		return nil
	})

	return w
}

// Close stops the background writing provess and attempts to serialize
// all pending operations.
func (w *updateWriter) Close() {
	w.tg.Stop()
	w.syncStates(w.updates)
}

// Set overwrites key value pair in the pending update operations.
func (w *updateWriter) Schedule(op *updateOp, n uint) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	key := op.resource.key

	idx, exists := w.pending[key]
	if !exists {
		idx = len(w.updates)
		w.updates = append(w.updates, scheduledUpdate{op: op, n: n})
		w.pending[key] = idx
	} else {
		w.updates[idx].op = op
		w.updates[idx].n += n
	}

	if w.waiter != nil {
		close(w.waiter)
		w.waiter = nil
	}
}

func (w *updateWriter) run(ctx context.Context) {
	for ctx.Err() == nil {
		updates, err := w.fetchPending(ctx)
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

// pending waits until at least one entry is available and returns
// a table of key value pairs with pending updates that need to be written to the registry.
func (w *updateWriter) fetchPending(ctx context.Context) ([]scheduledUpdate, error) {
	w.mutex.Lock()

	for ctx.Err() == nil {
		updates := w.updates
		if len(updates) > 0 {
			w.pending = map[string]int{}
			w.updates = nil
			w.mutex.Unlock()
			return updates, nil
		}

		waiter := make(chan struct{})
		w.waiter = waiter
		w.mutex.Unlock()

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-waiter:
			w.mutex.Lock()
		}
	}

	return nil, ctx.Err()
}
