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

package common

import "time"

// A Backoff waits on errors with exponential backoff (limited by maximum
// backoff). Resetting Backoff will reset the next sleep timer to the initial
// backoff duration.
type Backoff struct {
	duration time.Duration
	done     <-chan struct{}

	init time.Duration
	max  time.Duration

	last time.Time
}

func NewBackoff(done <-chan struct{}, init, max time.Duration) *Backoff {
	return &Backoff{
		duration: init,
		done:     done,
		init:     init,
		max:      max,
	}
}

func (b *Backoff) Reset() {
	b.duration = b.init
}

func (b *Backoff) Wait() bool {
	backoff := b.duration
	b.duration *= 2
	if b.duration > b.max {
		b.duration = b.max
	}

	select {
	case <-b.done:
		return false
	case <-time.After(backoff):
		b.last = time.Now()
		return true
	}
}

func (b *Backoff) WaitOnError(err error) bool {
	if err == nil {
		b.Reset()
		return true
	}
	return b.Wait()
}

func (b *Backoff) TryWaitOnError(failTS time.Time, err error) bool {
	if err == nil {
		b.Reset()
		return true
	}

	if failTS.Before(b.last) {
		return true
	}

	return b.Wait()
}
