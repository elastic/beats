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

package outest

import (
	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/publisher"
)

type Batch struct {
	events   []publisher.Event
	Signals  []BatchSignal
	OnSignal func(sig BatchSignal)
}

type BatchSignal struct {
	Tag    BatchSignalTag
	Events []publisher.Event
}

type BatchSignalTag uint8

const (
	BatchACK BatchSignalTag = iota
	BatchDrop
	BatchRetry
	BatchRetryEvents
	BatchCancelled
)

func NewBatch(in ...beat.Event) *Batch {
	events := make([]publisher.Event, len(in))
	for i, c := range in {
		events[i] = publisher.Event{Content: c}
	}
	return &Batch{events: events}
}

func (b *Batch) Events() []publisher.Event {
	return b.events
}

func (b *Batch) ACK() {
	b.doSignal(BatchSignal{Tag: BatchACK})
}

func (b *Batch) Drop() {
	b.doSignal(BatchSignal{Tag: BatchDrop})
}

func (b *Batch) Retry() {
	b.doSignal(BatchSignal{Tag: BatchRetry})
}

func (b *Batch) RetryEvents(events []publisher.Event) {
	b.doSignal(BatchSignal{Tag: BatchRetryEvents, Events: events})
}

func (b *Batch) Cancelled() {
	b.doSignal(BatchSignal{Tag: BatchCancelled})
}

func (b *Batch) doSignal(sig BatchSignal) {
	b.Signals = append(b.Signals, sig)
	if b.OnSignal != nil {
		b.OnSignal(sig)
	}
}
