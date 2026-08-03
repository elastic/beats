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

package reader

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDeadlineSetAndClear(t *testing.T) {
	var d Deadline
	require.True(t, d.ReadDeadline().IsZero(), "a fresh Deadline must have none set")

	deadline := time.Now().Add(time.Hour)
	require.True(t, d.SetReadDeadline(deadline))
	require.Equal(t, deadline, d.ReadDeadline())

	require.True(t, d.SetReadDeadline(time.Time{}))
	require.True(t, d.ReadDeadline().IsZero(), "a zero time must clear the deadline")
}

func TestDeadlineArm(t *testing.T) {
	t.Run("no deadline never fires", func(t *testing.T) {
		var d Deadline
		require.Nil(t, d.Arm(),
			"Arm must return a nil channel so a select falls through to the data source")
	})

	t.Run("fires once the deadline elapses", func(t *testing.T) {
		var d Deadline
		d.SetReadDeadline(time.Now().Add(50 * time.Millisecond))

		start := time.Now()
		<-d.Arm()
		require.GreaterOrEqual(t, time.Since(start), 40*time.Millisecond,
			"the timer fired before the deadline")
		d.Disarm()
	})

	t.Run("reuses a single timer across arms", func(t *testing.T) {
		var d Deadline
		d.SetReadDeadline(time.Now().Add(time.Hour))
		first := d.Arm()
		d.Disarm()

		// A second, shorter deadline must re-arm the same timer and still fire.
		d.SetReadDeadline(time.Now().Add(20 * time.Millisecond))
		second := d.Arm()
		require.Equal(t, first, second, "the timer channel must be reused, not reallocated")
		select {
		case <-second:
		case <-time.After(time.Second):
			t.Fatal("the re-armed timer never fired")
		}
		d.Disarm()
	})

	t.Run("a stale firing does not leak into the next arm", func(t *testing.T) {
		var d Deadline
		d.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
		<-d.Arm() // let the timer fire but leave it disarmed by the caller

		d.SetReadDeadline(time.Now().Add(time.Hour))
		select {
		case <-d.Arm():
			t.Fatal("a previous firing was observed by the next read")
		case <-time.After(50 * time.Millisecond):
		}
		d.Disarm()
	})

	t.Run("an unread firing does not leak into the next arm", func(t *testing.T) {
		// The race a reader actually hits: the deadline elapses at the same moment
		// data arrives, so the select takes the data branch and the firing is never
		// received. Disarm plus the next Reset must discard it.
		var d Deadline
		d.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
		d.Arm()
		time.Sleep(30 * time.Millisecond)
		d.Disarm()

		d.SetReadDeadline(time.Now().Add(time.Hour))
		select {
		case <-d.Arm():
			t.Fatal("an unread firing was observed by the next read")
		case <-time.After(50 * time.Millisecond):
		}
		d.Disarm()
	})
}

func TestDeadlineWaitBackoff(t *testing.T) {
	t.Run("already elapsed", func(t *testing.T) {
		var d Deadline
		d.SetReadDeadline(time.Now().Add(-time.Second))

		completed, ok := d.WaitBackoff(nil, time.Hour)
		require.False(t, ok, "an elapsed deadline must stop the read")
		require.False(t, completed)
	})

	t.Run("backoff shorter than the deadline", func(t *testing.T) {
		var d Deadline
		d.SetReadDeadline(time.Now().Add(time.Hour))

		start := time.Now()
		completed, ok := d.WaitBackoff(nil, 20*time.Millisecond)
		require.True(t, ok, "the deadline has not elapsed, the read must continue")
		require.True(t, completed, "the full backoff elapsed, the caller may grow it")
		require.GreaterOrEqual(t, time.Since(start), 15*time.Millisecond)
	})

	t.Run("backoff clamped to the deadline", func(t *testing.T) {
		var d Deadline
		d.SetReadDeadline(time.Now().Add(30 * time.Millisecond))

		start := time.Now()
		completed, ok := d.WaitBackoff(nil, time.Hour)
		elapsed := time.Since(start)
		require.False(t, ok, "the wait must end at the deadline, not at the backoff")
		require.False(t, completed)
		require.GreaterOrEqual(t, elapsed, 20*time.Millisecond)
		require.Less(t, elapsed, 5*time.Second, "the wait overshot the deadline")
	})

	t.Run("unset backoff waits out the deadline", func(t *testing.T) {
		var d Deadline
		d.SetReadDeadline(time.Now().Add(20 * time.Millisecond))

		completed, ok := d.WaitBackoff(nil, 0)
		require.False(t, ok)
		require.False(t, completed)
	})

	t.Run("done cancels the wait", func(t *testing.T) {
		var d Deadline
		d.SetReadDeadline(time.Now().Add(time.Hour))

		done := make(chan struct{})
		close(done)

		start := time.Now()
		completed, ok := d.WaitBackoff(done, time.Hour)
		require.True(t, ok, "cancellation is the caller's business, not a deadline error")
		require.False(t, completed, "the backoff did not elapse, so it must not grow")
		require.Less(t, time.Since(start), 5*time.Second, "WaitBackoff ignored done")
	})
}

// TestDeadlineNoAllocs guards the property the reusable timer exists for: a read
// path that repeatedly arms a deadline must not allocate per read.
func TestDeadlineNoAllocs(t *testing.T) {
	var d Deadline

	// Prime the timer so the one-time allocation is not counted.
	d.SetReadDeadline(time.Now().Add(time.Hour))
	d.Arm()
	d.Disarm()

	allocs := testing.AllocsPerRun(100, func() {
		d.SetReadDeadline(time.Now().Add(time.Hour))
		d.Arm()
		d.Disarm()
	})
	require.Zero(t, allocs, "arming a deadline must not allocate")
}
