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

package logstash

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestDeadlockListener(t *testing.T) {
	const timeout = time.Second
	var currentTime time.Time
	getTime := func() time.Time { return currentTime }

	logger := logptest.NewTestingLogger(t, "")
	dl := idleDeadlockListener(logger, timeout, getTime)

	// Channels get a buffer so we can trigger them deterministically in
	// one goroutine.
	tickerChan := make(chan time.Time, 1)
	dl.tickerChan = tickerChan
	dl.ackChan = make(chan int, 1)

	// Verify that the listener doesn't trigger when receiving regular acks
	for i := 0; i < 5; i++ {
		// Advance the "current time" and ping the ticker channel to refresh
		// the timeout check, then send an ack and confirm that it hasn't timed
		// out yet.
		currentTime = currentTime.Add(timeout - 1)
		tickerChan <- currentTime
		dl.runIteration()

		dl.ack(1)
		dl.runIteration()
		assert.Equal(t, currentTime, dl.lastTime)
		assert.Nil(t, dl.ctx.Err(), "Deadlock listener context shouldn't expire until the timeout is reached")
	}

	// Verify that the listener does trigger when the acks stop
	currentTime = currentTime.Add(timeout)
	tickerChan <- currentTime
	dl.runIteration()

	select {
	case <-dl.ctx.Done():
	default:
		require.Fail(t, "Deadlock listener should trigger when there is no progress for the configured time interval")
	}
}
