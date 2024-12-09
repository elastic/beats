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

	"github.com/elastic/beats/v7/libbeat/publisher"
)

type resendListener struct {
	timeout time.Duration
	batch   publisher.Batch
	ticker  *time.Ticker

	ackChan chan int
	acked   int

	doneChan chan struct{}
}

func newResendListener(timeout time.Duration, batch publisher.Batch) *resendListener {
	if timeout <= 0 {
		return nil
	}
	r := &resendListener{
		timeout: timeout,
		batch:   batch,
		ticker:  time.NewTicker(timeout),

		ackChan:  make(chan int),
		doneChan: make(chan struct{}),
	}
	go r.run()
	return r
}

func (r *resendListener) run() {
	defer r.ticker.Stop()
	defer close(r.doneChan)
	for {
		select {
		case n, ok := <-r.ackChan:
			if !ok {
				// Listener has been closed
				return
			}
			r.acked += n
			if n > 0 {
				// If progress was made, reset the countdown.
				r.ticker.Reset(r.timeout)
			}
		case <-r.ticker.C:
			// No progress was made within the timeout, hand unacknowledged events
			// back to the pipeline and close the listener.
			events := r.batch.Events()
			r.batch.LogstashParallelRetry(events[r.acked:])
			return
		}
	}
}

func (r *resendListener) ack(n int) {
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

func (r *resendListener) close() {
	if r == nil {
		return
	}
	// Signal the run loop to shut down
	close(r.ackChan)
}
