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
	"context"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
)

type deadlockListener struct {
	log        *logp.Logger
	timeout    time.Duration
	tickerChan <-chan time.Time

	// getTime returns the current time (allows overriding time.Now in the tests)
	getTime  func() time.Time
	lastTime time.Time

	ackChan chan int

	ctx       context.Context
	ctxCancel context.CancelFunc
}

const logstashDeadlockTimeout = 5 * time.Minute

func newDeadlockListener(log *logp.Logger, timeout time.Duration) *deadlockListener {
	if timeout <= 0 {
		return nil
	}

	dl := idleDeadlockListener(log, timeout, time.Now)

	// Check timeouts at a steady interval of once a second. This loses
	// sub-second granularity (which we don't need) in exchange for making
	// the API deterministically testable (using real timers in the tests
	// was causing flakiness).
	ticker := time.NewTicker(time.Second)
	dl.tickerChan = ticker.C
	go func() {
		defer ticker.Stop()
		dl.run()
	}()

	return dl
}

// Initialize the listener without an active ticker, and don't start the run()
// goroutine, so unit tests can control the execution timing.
func idleDeadlockListener(log *logp.Logger, timeout time.Duration, getTime func() time.Time) *deadlockListener {
	ctx, cancel := context.WithCancel(context.Background())
	return &deadlockListener{
		log:      log,
		timeout:  timeout,
		getTime:  getTime,
		lastTime: getTime(),

		ackChan: make(chan int),

		ctx:       ctx,
		ctxCancel: cancel,
	}
}

func (dl *deadlockListener) run() {
	for dl.ctx.Err() == nil {
		dl.runIteration()
	}
}

func (dl *deadlockListener) runIteration() {
	select {
	case <-dl.ctx.Done():
		// Listener has been closed
		return
	case n := <-dl.ackChan:
		if n > 0 {
			// If progress was made, reset the countdown.
			dl.lastTime = dl.getTime()
		}
	case <-dl.tickerChan:
		if dl.getTime().Sub(dl.lastTime) >= dl.timeout {
			// No progress was made within the timeout, log error so users
			// know there is likely a problem with the upstream host
			dl.log.Errorf("Logstash batch hasn't reported progress in the last %v, the Logstash host may be stalled. This problem can be prevented by configuring Logstash to use PipelineBusV1 or by upgrading Logstash to 8.17+, for details see https://github.com/elastic/logstash/issues/16657", dl.timeout)
			dl.ctxCancel()
		}
	}
}

func (dl *deadlockListener) ack(n int) {
	if dl == nil {
		return
	}
	// Send the new ack to the run loop, unless it has already shut down in
	// which case it can be safely ignored.
	select {
	case dl.ackChan <- n:
	case <-dl.ctx.Done():
	}
}

func (dl *deadlockListener) close() {
	if dl == nil {
		return
	}
	// Signal the run loop to shut down
	dl.ctxCancel()
}
