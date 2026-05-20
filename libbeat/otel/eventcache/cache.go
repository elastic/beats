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

// Package eventcache holds in-flight beat events so they can be passed
// through the OTel pipeline without any serialization. The producer
// (otelconsumer) stores an event and receives a numeric token; the consumer
// (beatprocessor) retrieves the event by token once it needs it.
package eventcache

import (
	"sync"
	"sync/atomic"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/publisher"
)

// TokenAttribute is the pdata log-record attribute key used to carry the cache
// token through the OTel pipeline.
const TokenAttribute = "elastic.beat.cache_token"

// Entry is what the cache stores for each in-flight event.
// Both fields are pointers: Event points into the batch's event slice (valid
// until batch ACK/Drop/Retry) and BeatInfo points to the receiver-lifetime
// struct. No copies are made.
type Entry struct {
	Event    *publisher.Event
	BeatInfo *beat.Info
}

var (
	counter atomic.Int64
	mu      sync.Mutex
	store   = make(map[int64]*Entry)
)

// Put stores entry in the cache and returns a unique token that can be used
// to retrieve it later via Take. Each token is written exactly once and read
// exactly once, so a plain mutex-protected map outperforms sync.Map here.
func Put(entry *Entry) int64 {
	token := counter.Add(1)
	mu.Lock()
	store[token] = entry
	mu.Unlock()
	return token
}

// Take retrieves and removes the entry associated with token. The second return
// value is false when no entry exists for the given token.
func Take(token int64) (*Entry, bool) {
	mu.Lock()
	entry, ok := store[token]
	if ok {
		delete(store, token)
	}
	mu.Unlock()
	return entry, ok
}
