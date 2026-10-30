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
	"errors"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/ebpfevents"
)

const allEvents = EventMask(math.MaxUint64)

func TestWatcherStartStop(t *testing.T) {
	w := GetWatcher()
	assert.Equal(t, stopped, gWatcher.status)
	assert.Equal(t, 0, gWatcher.nclients())

	_, err := w.Subscribe("test-1", allEvents)
	if err != nil {
		t.Skipf("skipping ebpf watcher test: %v", err)
	}
	assert.Equal(t, started, gWatcher.status)
	assert.Equal(t, 1, gWatcher.nclients())

	_, err = w.Subscribe("test-2", allEvents)
	require.NoError(t, err)
	assert.Equal(t, 2, gWatcher.nclients())

	w.Unsubscribe("test-2")
	assert.Equal(t, 1, gWatcher.nclients())

	w.Unsubscribe("dummy")
	assert.Equal(t, 1, gWatcher.nclients())

	assert.Equal(t, started, gWatcher.status)
	w.Unsubscribe("test-1")
	assert.Equal(t, 0, gWatcher.nclients())
	assert.Equal(t, stopped, gWatcher.status)

	_, err = w.Subscribe("new", allEvents)
	require.NoError(t, err)
	assert.Equal(t, 1, gWatcher.nclients())
	assert.Equal(t, started, gWatcher.status)
	w.Unsubscribe("new")
}

// TestSubscribeLoaderStartFails covers the loader failing to start in Subscribe: Subscribe must
// return the error rather than use the nil loader, and the next Subscribe must try again.
func TestSubscribeLoaderStartFails(t *testing.T) {
	errStart := errors.New("loader start failed")
	loads := 0
	newLoader := func() (*ebpfevents.Loader, error) {
		loads++
		return nil, errStart
	}

	var w Watcher
	for i := 1; i <= 2; i++ {
		records, err := w.subscribeWith("test", allEvents, newLoader)
		require.ErrorIs(t, err, errStart)
		assert.Nil(t, records, "a failed Subscribe must not return a channel")
		assert.Equal(t, i, loads, "each Subscribe on a stopped watcher must load the ebpf programs once")
		assert.Equal(t, stopped, w.status, "a failed start must leave the watcher stopped")
		assert.Equal(t, 0, w.nclients(), "a failed Subscribe must not register the client")
	}
}
