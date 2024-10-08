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

//go:build linux

package ebpf

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

const allEvents = EventMask(math.MaxUint64)

func TestWatcherStartStop(t *testing.T) {
	w, err := GetWatcher()
	if err != nil {
		t.Skipf("skipping ebpf watcher test: %v", err)
	}
	assert.Equal(t, gWatcher.status, stopped)
	assert.Equal(t, 0, gWatcher.nclients())

	_ = w.Subscribe("test-1", allEvents)
	assert.Equal(t, gWatcher.status, started)
	assert.Equal(t, 1, gWatcher.nclients())

	_ = w.Subscribe("test-2", allEvents)
	assert.Equal(t, 2, gWatcher.nclients())

	w.Unsubscribe("test-2")
	assert.Equal(t, 1, gWatcher.nclients())

	w.Unsubscribe("dummy")
	assert.Equal(t, 1, gWatcher.nclients())

	assert.Equal(t, gWatcher.status, started)
	w.Unsubscribe("test-1")
	assert.Equal(t, 0, gWatcher.nclients())
	assert.Equal(t, gWatcher.status, stopped)

	_ = w.Subscribe("new", allEvents)
	assert.Equal(t, 1, gWatcher.nclients())
	assert.Equal(t, gWatcher.status, started)
	w.Unsubscribe("new")
}
