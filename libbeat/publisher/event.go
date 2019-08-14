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
	"github.com/elastic/beats/libbeat/beat"
)

// Batch is used to pass a batch of events to the outputs and asynchronously listening
// for signals from these outpts. After a batch is processed (completed or
// errors), one of the signal methods must be called.
type Batch interface {
	Events() []Event

	// signals
	ACK()
	Drop()
	Retry()
	RetryEvents(events []Event)
	Cancelled()
	CancelledEvents(events []Event)
}

// Event is used by the publisher pipeline and broker to pass additional
// meta-data to the consumers/outputs.
type Event struct {
	Content beat.Event
	Flags   EventFlags
}

// EventFlags provides additional flags/option types  for used with the outputs.
type EventFlags uint8

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
