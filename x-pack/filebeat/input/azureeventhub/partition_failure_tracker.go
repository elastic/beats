// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"sync"
	"time"
)

// partitionFailureInfo holds the timestamp of the first failure
// of a partition and a count of how many failures have occurred
// without success in that sequence. A success will reset this data.
type partitionFailureInfo struct {
	firstFailureTime time.Time
	failureCount     int
}

// partitionFailureTracker tracks partition failures to avoid
// status flapping due to transient errors.
type partitionFailureTracker struct {
	partitionFailureData map[string]*partitionFailureInfo
	mux                  sync.Mutex
	minWindow            time.Duration
	threshold            int
	failingPartitions    map[string]struct{}
}

// newPartitionFailureTracker creates a new partitionFailureTracker.
func newPartitionFailureTracker(minWindow time.Duration, threshold int) *partitionFailureTracker {
	return &partitionFailureTracker{
		partitionFailureData: make(map[string]*partitionFailureInfo),
		minWindow:            minWindow,
		threshold:            threshold,
		failingPartitions:    make(map[string]struct{}),
	}
}

// TrackFailure tracks a failure for a given partition. It returns
// true if the failure should be reported as Degraded.
func (t *partitionFailureTracker) TrackFailure(partitionID string) (bool, int) {
	t.mux.Lock()
	defer t.mux.Unlock()

	info, exists := t.partitionFailureData[partitionID]
	if !exists {
		info = &partitionFailureInfo{firstFailureTime: time.Now()}
		t.partitionFailureData[partitionID] = info
	}

	info.failureCount++
	isFailing := time.Since(info.firstFailureTime) > t.minWindow && info.failureCount >= t.threshold

	if isFailing {
		t.failingPartitions[partitionID] = struct{}{}
	}

	return isFailing, info.failureCount
}

// TrackSuccess resets the failure tracking for a given partition
// on its success.
func (t *partitionFailureTracker) TrackSuccess(partitionID string) {
	t.mux.Lock()
	defer t.mux.Unlock()

	delete(t.partitionFailureData, partitionID)
	delete(t.failingPartitions, partitionID)
}

// HasFailingPartitions returns true if any partition is currently in a
// persistent failure state.
func (t *partitionFailureTracker) HasFailingPartitions() bool {
	t.mux.Lock()
	defer t.mux.Unlock()
	return len(t.failingPartitions) > 0
}
