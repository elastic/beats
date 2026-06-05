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

// Package slabqueue implements a multi-pipeline in-memory queue used
// both by standalone Beats (selectable via queue.slab) and by Beat
// receivers (always, when running with an in-memory queue config).
// Storage is separated from FIFO ordering:
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
package slabqueue

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	c "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

// QueueType is the user-facing queue type selector. It mirrors
// memqueue.QueueType / diskqueue.QueueType so slabqueue can be selected
// from a pipeline config (queue.slab) just like the other implementations.
const QueueType = "slab"

// Settings configures a Pool's capacity.
type Settings struct {
	// Events is the total number of slots in the pool's backing array. It
	// is the upper bound on events live (published but not yet ack'd) across
	// every pipeline connected to the pool.
	Events int
}

// userConfig is the YAML-facing shape of slabqueue settings. Kept separate
// from Settings so we can attach struct tags without exposing them as part
// of the public Settings type.
type userConfig struct {
	Events int `config:"events" validate:"min=32"`
}

var defaultUserConfig = userConfig{
	Events: 3200, // matches memqueue's DefaultEvents
}

// SettingsForUserConfig unpacks a ucfg config from a Beats queue
// configuration and returns the equivalent slabqueue.Settings.
func SettingsForUserConfig(cfg *c.C) (Settings, error) {
	parsed := defaultUserConfig
	if cfg != nil {
		if err := cfg.Unpack(&parsed); err != nil {
			return Settings{}, fmt.Errorf("couldn't unpack slabqueue config: %w", err)
		}
	}
	return Settings(parsed), nil
}

// FactoryForSettings returns a queue.QueueFactory[T] that gives each
// pipeline its own private slabqueue.Pool sized to settings.Events. The
// returned Queue is wired so closing it also shuts down the underlying
// pool — matching the lifecycle the queue factory contract assumes (one
// queue, one owner, Close releases all resources).
//
// For multi-receiver shared-budget scenarios use the controller-level
// acquire/release path in the OTel output controller directly; this
// factory is for the standalone pipeline path where each queue is owned
// by one pipeline.
func FactoryForSettings[T any](settings Settings) queue.QueueFactory[T] {
	return func(
		_ *logp.Logger,
		observer queue.Observer,
		_ int,
		_ queue.EncoderFactory[T],
	) (queue.Queue[T], error) {
		pool := NewPool[T](settings, observer)
		return &slabBackedQueue[T]{Queue: pool.Connect(), pool: pool}, nil
	}
}

// slabBackedQueue is a Queue whose Close also shuts down the pool that
// created it. Used by FactoryForSettings to give the standalone pipeline
// path a queue.Queue[T] with single-owner lifecycle semantics.
type slabBackedQueue[T any] struct {
	*Queue[T]
	pool *Pool[T]
}

func (q *slabBackedQueue[T]) Close(force bool) error {
	err := q.Queue.Close(force)
	q.pool.Shutdown()
	return err
}
