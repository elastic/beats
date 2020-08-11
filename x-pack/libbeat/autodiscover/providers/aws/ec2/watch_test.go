// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ec2

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestWatchTicks(t *testing.T) {
	instances := []*ec2Instance{fakeEC2Instance()}

	lock := sync.Mutex{}
	var startUUIDs []string
	var startEC2s []*ec2Instance
	var stopUUIDs []string

	fetcher := newMockFetcher(instances, nil)
	watcher := newWatcher(
		fetcher,
		time.Millisecond,
		func(uuid string, lbListener *ec2Instance) {
			lock.Lock()
			defer lock.Unlock()

			startUUIDs = append(startUUIDs, uuid)
			startEC2s = append(startEC2s, lbListener)
		},
		func(uuid string) {
			lock.Lock()
			defer lock.Unlock()

			stopUUIDs = append(stopUUIDs, uuid)
		})
	defer watcher.stop()

	// Run through 10 ticks
	for i := 0; i < 10; i++ {
		err := watcher.once()
		require.NoError(t, err)
	}

	// The instanceID is the unique identifier used.
	instanceIDs := []string{*instances[0].ec2Instance.InstanceId}

	// Test that we've seen one ec2 start, but none stop
	assert.Equal(t, instanceIDs, startUUIDs)
	assert.Len(t, stopUUIDs, 0)
	assert.Equal(t, instances, startEC2s)

	// Stop the ec2 and test that we see a single stop
	// and no change to starts
	fetcher.setEC2s(nil)
	watcher.once()

	assert.Equal(t, instanceIDs, startUUIDs)
	assert.Equal(t, instanceIDs, stopUUIDs)
}
