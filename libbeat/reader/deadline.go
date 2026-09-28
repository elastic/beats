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

import "time"

// Deadline implements DeadlineSetter for a reader that actually blocks waiting
// for data, and bounds that wait with a reusable timer. Embed it and use:
//
//   - Arm/Disarm when the wait is a select on the data source (a channel
//     receive), or
//   - WaitBackoff when the wait is a backoff sleep between polls of the source.
//
// A Deadline is owned by the goroutine performing the read: SetReadDeadline is
// called by the wrapping reader before Next/Read on that same goroutine, so it
// needs no synchronization.
//
// The timer is created on first use and reused afterwards, so a read path that
// repeatedly arms a deadline (the multiline timeout) stays allocation-free.
type Deadline struct {
	deadline time.Time
	timer    *time.Timer
}

// SetReadDeadline bounds how long the next blocking read may wait for data. A
// zero time clears the deadline. It always returns true: a reader embedding
// Deadline honors deadlines, which lets the multiline timeout reader enforce
// its timeout synchronously instead of spawning a read-ahead goroutine.
func (d *Deadline) SetReadDeadline(t time.Time) bool {
	d.deadline = t
	return true
}

// ReadDeadline returns the current deadline, zero if none is set. It is used by
// readers that forward the deadline to a source they may recreate.
func (d *Deadline) ReadDeadline() time.Time {
	return d.deadline
}

// Arm returns a channel that fires once the deadline elapses, for use in a
// select alongside the data source. It returns nil when no deadline is set: a
// receive from a nil channel blocks forever, so a single select handles both
// cases. Call Disarm once the select returns.
func (d *Deadline) Arm() <-chan time.Time {
	if d.deadline.IsZero() {
		return nil
	}
	return d.arm(time.Until(d.deadline))
}

// Disarm stops the timer armed by Arm or WaitBackoff.
func (d *Deadline) Disarm() {
	if d.timer != nil {
		d.timer.Stop()
	}
}

// WaitBackoff blocks for backoff, but never past the deadline, returning early
// if done is closed. It reports whether the full backoff interval elapsed, so
// the caller can grow its backoff, and whether reading may continue: ok is
// false once the deadline has elapsed, telling the caller to return
// ErrReadDeadline.
//
// It must only be called while a deadline is set (see ReadDeadline); without
// one the caller performs its own unbounded backoff wait.
func (d *Deadline) WaitBackoff(done <-chan struct{}, backoff time.Duration) (completed, ok bool) {
	remaining := time.Until(d.deadline)
	if remaining <= 0 {
		return false, false
	}

	// Never sleep past the deadline: if the backoff would overshoot it (or is
	// unset), wait exactly the remaining time and report the deadline as hit.
	wait, atDeadline := backoff, false
	if wait <= 0 || wait >= remaining {
		wait, atDeadline = remaining, true
	}

	timer := d.arm(wait)
	select {
	case <-done:
		d.Disarm()
		return false, true
	case <-timer:
		return !atDeadline, !atDeadline
	}
}

// arm creates the timer on first use and resets it afterwards. Go 1.23+ (see
// go.mod) makes Reset/Stop safe without draining the channel.
func (d *Deadline) arm(wait time.Duration) <-chan time.Time {
	if d.timer == nil {
		d.timer = time.NewTimer(wait)
	} else {
		d.timer.Reset(wait)
	}
	return d.timer.C
}
