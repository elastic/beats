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

package queue

// Observer is an interface for queues to send state updates to a metrics
// or test listener.
type Observer interface {
	MaxEvents(int)
	MaxBytes(int)

	// Restore queue state on startup. Used by the disk queue to report events
	// that are already in the queue from a previous run.
	Restore(eventCount int, byteCount int)

	// All reported byte counts are zero if the output doesn't support
	// early encoding.
	AddEvent(byteCount int)
	ConsumeEvents(eventCount int, byteCount int)
	RemoveEvents(eventCount int, byteCount int)
}

func NewQueueObserver() Observer {
	//queueACKed:     monitoring.NewUint(reg, "queue.acked"),
	//queueMaxEvents: monitoring.NewUint(reg, "queue.max_events"),

	return nil
}
