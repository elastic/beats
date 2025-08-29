// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPartitionFailureTracker(t *testing.T) {
	const testWindow = 100 * time.Millisecond
	const testThreshold = 3
	partitionID := "part-1"

	t.Run("failures below threshold do not report", func(t *testing.T) {
		tracker := newPartitionFailureTracker(testWindow, testThreshold)
		shouldReport, _ := tracker.TrackFailure(partitionID)
		assert.False(t, shouldReport)
		assert.False(t, tracker.HasFailingPartitions())
		shouldReport, _ = tracker.TrackFailure(partitionID)
		assert.False(t, shouldReport)
		assert.False(t, tracker.HasFailingPartitions())
	})

	t.Run("failures within window but below threshold do not report", func(t *testing.T) {
		tracker := newPartitionFailureTracker(testWindow, testThreshold)
		tracker.TrackFailure(partitionID)
		time.Sleep(testWindow / 2)
		shouldReport, _ := tracker.TrackFailure(partitionID)
		assert.False(t, shouldReport)
		assert.False(t, tracker.HasFailingPartitions())
	})

	t.Run("failures meeting threshold but within window do not report", func(t *testing.T) {
		tracker := newPartitionFailureTracker(testWindow, testThreshold)
		tracker.TrackFailure(partitionID)
		tracker.TrackFailure(partitionID)
		shouldReport, _ := tracker.TrackFailure(partitionID)
		assert.False(t, shouldReport)
		assert.False(t, tracker.HasFailingPartitions())
	})

	t.Run("failures meeting threshold outside window report", func(t *testing.T) {
		tracker := newPartitionFailureTracker(testWindow, testThreshold)
		tracker.TrackFailure(partitionID)
		tracker.TrackFailure(partitionID)
		time.Sleep(testWindow + 1*time.Millisecond)
		shouldReport, count := tracker.TrackFailure(partitionID)
		assert.True(t, shouldReport)
		assert.Equal(t, 3, count)
		assert.True(t, tracker.HasFailingPartitions())
	})

	t.Run("success resets failure tracker", func(t *testing.T) {
		tracker := newPartitionFailureTracker(testWindow, testThreshold)
		tracker.TrackFailure(partitionID)
		tracker.TrackFailure(partitionID)
		time.Sleep(testWindow + 1*time.Millisecond)
		shouldReport, count := tracker.TrackFailure(partitionID) // partition is now failing
		assert.True(t, shouldReport)
		assert.Equal(t, 3, count)
		assert.True(t, tracker.HasFailingPartitions())

		tracker.TrackSuccess(partitionID)
		assert.False(t, tracker.HasFailingPartitions())

		// Fail again, should not report immediately
		shouldReport, _ = tracker.TrackFailure(partitionID)
		assert.False(t, shouldReport)
		assert.False(t, tracker.HasFailingPartitions())
	})

	t.Run("has failing partitions", func(t *testing.T) {
		tracker := newPartitionFailureTracker(testWindow, testThreshold)
		partition2ID := "part-2"

		// Fail partition 1 to threshold
		tracker.TrackFailure(partitionID)
		tracker.TrackFailure(partitionID)
		time.Sleep(testWindow + 1*time.Millisecond)
		tracker.TrackFailure(partitionID)
		assert.True(t, tracker.HasFailingPartitions())

		// Fail partition 2 but not to threshold
		tracker.TrackFailure(partition2ID)
		assert.True(t, tracker.HasFailingPartitions())

		// Resolve partition 1
		tracker.TrackSuccess(partitionID)
		assert.False(t, tracker.HasFailingPartitions())
	})
}
