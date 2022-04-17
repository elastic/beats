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

package publisher

import (
	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
)

// Batch is used to pass a batch of events to the outputs and asynchronously listening
// for signals from these outpts. After a batch is processed (completed or
// errors), one of the signal methods must be called. In normal operation
// every batch will eventually receive an ACK() or a Drop().
type Batch interface {
	Events() []Event

	// All events have been acknowledged by the output.
	ACK()

	// Give up on these events permanently without sending.
	Drop()

	// Try sending this batch again
	Retry()

	// Try sending the events in this list again; all others are acknowledged.
	RetryEvents(events []Event)

	// Send was aborted, try again but don't decrease the batch's TTL counter.
	Cancelled()
}

// Event is used by the publisher pipeline and broker to pass additional
// meta-data to the consumers/outputs.
type Event struct {
	Content beat.Event
	Flags   EventFlags
	Cache   EventCache
}

// EventFlags provides additional flags/option types  for used with the outputs.
type EventFlags uint8

// EventCache provides a space for outputs to define per-event metadata
// that's intended to be used only within the scope of an output
type EventCache struct {
	m common.MapStr
}

// Put lets outputs put key-value pairs into the event cache
func (ec *EventCache) Put(key string, value interface{}) (interface{}, error) {
	if ec.m == nil {
		// uninitialized map
		ec.m = common.MapStr{}
	}

	return ec.m.Put(key, value)
}

// GetValue lets outputs retrieve values from the event cache by key
func (ec *EventCache) GetValue(key string) (interface{}, error) {
	if ec.m == nil {
		// uninitialized map
		return nil, common.ErrKeyNotFound
	}

	return ec.m.GetValue(key)
}

const (
	// GuaranteedSend requires an output to not drop the event on failure, but
	// retry until ACK.
	GuaranteedSend EventFlags = 0x01
)

// Guaranteed checks if the event must not be dropped by the output or the
// publisher pipeline.
func (e *Event) Guaranteed() bool {
	return (e.Flags & GuaranteedSend) == GuaranteedSend
}
