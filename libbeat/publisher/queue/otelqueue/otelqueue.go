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

// Package otelqueue implements a multi-pipeline queue used by Beats
// receivers. Storage is separated from FIFO ordering:
//
//   - A Pool owns a fixed-size backing array of slots and a free list.
//     Publish acquires a slot; batch.Done returns it. The free list also
//     serves as a counting semaphore: total live events across all
//     pipelines is capped by Settings.Events. There is no per-pipeline cap
//     — one pipeline may use the full budget while others are quiet.
//   - Each connected pipeline gets its own Queue (implementing
//     queue.Queue[T]) with its own FIFO over the shared array. A slow or
//     stalled consumer on one pipeline only holds its own in-flight slots;
//     other pipelines flow independently.
//
// Slot release and ACK ordering are decoupled. Slots return to the pool as
// soon as a batch is Done so other producers can make progress. Producer
// ACK callbacks fire in publish order: a later batch's ACK is held until
// every earlier in-flight batch has also been Done, matching memqueue's
// ackLoop guarantee — required by order-sensitive consumers such as
// filestream's registry tracker. Queue.Done() therefore waits for the
// FIFO to drain *and* every batch handed out by Get to be Done.
//
// Unlike memqueue, Get returns whatever is available immediately; there is
// no FlushTimeout / MaxGetRequest. Batch consolidation is left to the
// output (e.g. the exporter's own batching).
package otelqueue

// Settings configures a Pool's capacity.
type Settings struct {
	// Events is the total number of slots in the pool's backing array. It
	// is the upper bound on events live (published but not yet ack'd) across
	// every pipeline connected to the pool.
	Events int
}
