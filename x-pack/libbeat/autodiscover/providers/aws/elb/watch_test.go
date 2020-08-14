// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elb

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestWatchTicks(t *testing.T) {
	lbls := []*lbListener{fakeLbl()}

	lock := sync.Mutex{}
	var startUUIDs []string
	var startLbls []*lbListener
	var stopUUIDs []string

	fetcher := newMockFetcher(lbls, nil)
	watcher := newWatcher(
		fetcher,
		time.Millisecond,
		func(uuid string, lbListener *lbListener) {
			lock.Lock()
			defer lock.Unlock()

			startUUIDs = append(startUUIDs, uuid)
			startLbls = append(startLbls, lbListener)
		},
		func(uuid string) {
			lock.Lock()
			defer lock.Unlock()

			stopUUIDs = append(stopUUIDs, uuid)
		})
	defer watcher.stop() // unnecessary, but good hygiene

	// Run through 10 ticks
	for i := 0; i < 10; i++ {
		err := watcher.once()
		require.NoError(t, err)
	}

	// The listener ARN is the unique identifier used.
	uuids := []string{*lbls[0].listener.ListenerArn}

	// Test that we've seen one lbl start, but none stop
	assert.Equal(t, uuids, startUUIDs)
	assert.Len(t, stopUUIDs, 0)
	assert.Equal(t, lbls, startLbls)

	// Stop the lbl and test that we see a single stop
	// and no change to starts
	fetcher.setLbls(nil)
	watcher.once()

	assert.Equal(t, uuids, startUUIDs)
	assert.Equal(t, uuids, stopUUIDs)
}
