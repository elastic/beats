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
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
)

type deadlockListener struct {
	log     *logp.Logger
	timeout time.Duration
	ticker  *time.Ticker

	ackChan chan int

	doneChan chan struct{}
}

const logstashDeadlockTimeout = 5 * time.Minute

func newDeadlockListener(log *logp.Logger, timeout time.Duration) *deadlockListener {
	if timeout <= 0 {
		return nil
	}
	r := &deadlockListener{
		log:     log,
		timeout: timeout,
		ticker:  time.NewTicker(timeout),

		ackChan:  make(chan int),
		doneChan: make(chan struct{}),
	}
	go r.run()
	return r
}

func (r *deadlockListener) run() {
	defer r.ticker.Stop()
	defer close(r.doneChan)
	for {
		select {
		case n, ok := <-r.ackChan:
			if !ok {
				// Listener has been closed
				return
			}
			if n > 0 {
				// If progress was made, reset the countdown.
				r.ticker.Reset(r.timeout)
			}
		case <-r.ticker.C:
			// No progress was made within the timeout, log error so users
			// know there is likely a problem with the upstream host
			r.log.Errorf("Logstash batch hasn't reported progress in the last %v, the Logstash host may be stalled. This problem can be prevented by configuring Logstash to use PipelineBusV1 or by upgrading Logstash to 8.17+, for details see https://github.com/elastic/logstash/issues/16657", r.timeout)
			return
		}
	}
}

func (r *deadlockListener) ack(n int) {
	if r == nil {
		return
	}
	// Send the new ack to the run loop, unless it has already shut down in
	// which case it can be safely ignored.
	select {
	case r.ackChan <- n:
	case <-r.doneChan:
	}
}

func (r *deadlockListener) close() {
	if r == nil {
		return
	}
	// Signal the run loop to shut down
	close(r.ackChan)
}
