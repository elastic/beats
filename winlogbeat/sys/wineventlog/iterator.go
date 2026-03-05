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

//go:build windows

package wineventlog

import (
	"errors"
	"fmt"
	"sync"

	"golang.org/x/sys/windows"

	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	evtNextMaxHandles     = 1024
	evtNextDefaultHandles = 512
)

// EventIterator provides an iterator to read events from a log. It takes the
// place of calling EvtNext directly.
type EventIterator struct {
	subscriptionFactory SubscriptionFactory          // Factory for producing a new subscription handle.
	subscription        EvtHandle                    // Handle from EvtQuery or EvtSubscribe.
	batchSize           uint32                       // Number of handles to request by default.
	handles             [evtNextMaxHandles]EvtHandle // Handles returned by EvtNext.
	lastErr             error                        // Last error returned by EvtNext.
	active              []EvtHandle                  // Slice of the handles array containing the valid unread handles.
	mutex               sync.Mutex                   // Mutex to enable parallel iteration.
	log                 *logp.Logger

	// For testing purposes to be able to mock EvtNext.
	evtNext func(resultSet EvtHandle, eventArraySize uint32, eventArray *EvtHandle, timeout uint32, flags uint32, numReturned *uint32) (err error)
}

// SubscriptionFactory produces a handle from EvtQuery or EvtSubscribe that
// points to the next unread event. Provide a factory to enable automatic
// recover of certain errors.
type SubscriptionFactory func() (EvtHandle, error)

// EventIteratorOption represents a configuration of for the construction of
// the EventIterator.
type EventIteratorOption func(*EventIterator)

// WithSubscriptionFactory configures a SubscriptionFactory for the iterator to
// use to create a subscription handle.
func WithSubscriptionFactory(factory SubscriptionFactory) EventIteratorOption {
	return func(itr *EventIterator) {
		itr.subscriptionFactory = factory
	}
}

// WithSubscription configures the iterator with an existing subscription handle.
func WithSubscription(subscription EvtHandle) EventIteratorOption {
	return func(itr *EventIterator) {
		itr.subscription = subscription
	}
}

// WithBatchSize configures the number of handles the iterator will request
// when calling EvtNext. Valid batch sizes range on [1, 1024].
func WithBatchSize(size int) EventIteratorOption {
	return func(itr *EventIterator) {
		if size > 0 {
			if size > int(evtNextMaxHandles) {
				itr.batchSize = evtNextMaxHandles
				return
			}
			itr.batchSize = uint32(size)
		}
	}
}

// NewEventIterator creates an iterator to read event handles from a subscription.
// The iterator is thread-safe.
func NewEventIterator(opts ...EventIteratorOption) (*EventIterator, error) {
	itr := &EventIterator{
		batchSize: evtNextDefaultHandles,
		evtNext:   _EvtNext,
		log:       logp.NewLogger("wineventlog.iterator"),
	}

	for _, opt := range opts {
		opt(itr)
	}

	if itr.subscriptionFactory == nil && itr.subscription == NilHandle {
		return nil, errors.New("either a subscription or subscription factory is required")
	}

	if itr.subscription == NilHandle {
		handle, err := itr.subscriptionFactory()
		if err != nil {
			return nil, err
		}
		itr.subscription = handle
	}

	return itr, nil
}

// Next advances the iterator to the next handle. After Next returns false, the
// Err() method will return any error that occurred during iteration, except
// that if it was windows.ERROR_NO_MORE_ITEMS, Err() will return nil and you
// may call Next() again later to check if new events are available.
func (itr *EventIterator) Next() (EvtHandle, bool) {
	itr.mutex.Lock()
	defer itr.mutex.Unlock()

	if itr.lastErr != nil {
		return NilHandle, false
	}

	if !itr.empty() {
		itr.active = itr.active[1:]
	}

	if itr.empty() && !itr.moreHandles() {
		return NilHandle, false
	}

	return itr.active[0], true
}

// empty returns true when there are no more handles left to read from memory.
func (itr *EventIterator) empty() bool {
	return len(itr.active) == 0
}

// moreHandles fetches more handles using EvtNext. It returns true if it
// successfully fetched more handles.
func (itr *EventIterator) moreHandles() bool {
	batchSize := itr.batchSize

	for batchSize > 0 {
		var numReturned uint32

		nextErr := itr.evtNext(itr.subscription, batchSize, &itr.handles[0], 0, 0, &numReturned)
		switch {
		case nextErr == nil:
			itr.lastErr = nil
			itr.active = itr.handles[:numReturned]
		case errors.Is(nextErr, windows.ERROR_NO_MORE_ITEMS):
		case errors.Is(nextErr, windows.ERROR_INVALID_OPERATION):
			// ERROR_INVALID_OPERATION can be returned during polling with zero handles.
			if numReturned == 0 {
				break
			}
			fallthrough
		case errors.Is(nextErr, windows.RPC_S_INVALID_BOUND):
			// Attempt automated recovery if we have a factory.
			if itr.subscriptionFactory != nil {
				itr.log.Warnw("EvtNext failed, recreating subscription.",
					"error", nextErr,
					"batch_size", batchSize,
					"num_returned", numReturned)
				itr.subscription.Close()
				var err error
				itr.subscription, err = itr.subscriptionFactory()
				if err != nil {
					itr.lastErr = fmt.Errorf("failed in EvtNext while trying to recover: %w", err)
					return false
				}

				// Reduce batch size only for RPC_S_INVALID_BOUND and try again.
				if errors.Is(nextErr, windows.RPC_S_INVALID_BOUND) {
					batchSize = batchSize / 2
				}
				continue
			} else {
				itr.lastErr = fmt.Errorf("failed in EvtNext (try reducing the batch size or providing a subscription factory for automatic recovery): %w", nextErr)
			}
		default:
			itr.lastErr = nextErr
		}

		break
	}

	return !itr.empty()
}

// Err returns the first non-ERROR_NO_MORE_ITEMS error encountered by the
// EventIterator.
//
// Some Windows versions will fail with windows.RPC_S_INVALID_BOUND when the
// batch size is too large. If this occurs you can recover by closing the
// iterator, creating a new subscription, seeking to the next unread event, and
// creating a new EventIterator with a smaller batch size.
func (itr *EventIterator) Err() error {
	itr.mutex.Lock()
	defer itr.mutex.Unlock()

	return itr.lastErr
}

// Close closes the subscription handle and any unread event handles.
func (itr *EventIterator) Close() error {
	if itr == nil {
		return errors.New("closing nil event iterator")
	}
	itr.mutex.Lock()
	defer itr.mutex.Unlock()

	for _, h := range itr.active {
		h.Close()
	}
	return itr.subscription.Close()
}
