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

package periodic

import (
	"sync"
	"sync/atomic"
	"time"
)

// Doer limits an action to be executed at most once within a specified period.
// It is intended for managing events that occur frequently, but instead of an
// action being taken for every event, the action should be executed at most
// once within a given period of time.
//
// Doer takes a function to execute, doFn, which is called every time
// the specified period has elapsed with the number of events and the period.
type Doer struct {
	count atomic.Uint64

	period time.Duration

	// doFn is called for executing the action every period if at least one
	// event happened. It receives the count of events and the period.
	doFn     func(count uint64, d time.Duration)
	lastDone time.Time
	done     chan struct{}

	// nowFn is used to acquire the current time instead of time.Now so it can
	// be mocked for tests.
	nowFn func() time.Time
	// newTickerFn is used to acquire a *time.Ticker instead of time.NewTicker
	// so it can be mocked for tests.
	newTickerFn func(duration time.Duration) *time.Ticker

	started atomic.Bool
	wg      sync.WaitGroup
	ticker  *time.Ticker
}

// NewDoer returns a new Doer. It takes a doFn, which is
// called with the current count of events and the period.
func NewDoer(period time.Duration, doFn func(count uint64, d time.Duration)) *Doer {
	return &Doer{
		period: period,
		doFn:   doFn,

		nowFn:       time.Now,
		newTickerFn: time.NewTicker,
	}
}

func (r *Doer) Add() {
	r.count.Add(1)
}

func (r *Doer) AddN(n uint64) {
	r.count.Add(n)
}

func (r *Doer) Start() {
	if r.started.Swap(true) {
		return
	}

	r.done = make(chan struct{})
	r.lastDone = r.nowFn()
	r.ticker = r.newTickerFn(r.period)

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer r.ticker.Stop()

		for {
			select {
			case <-r.ticker.C:
				r.do()
			case <-r.done:
				r.do()
				return
			}
		}
	}()
}

func (r *Doer) Stop() {
	if !r.started.Swap(false) {
		return
	}

	close(r.done)
	r.wg.Wait()
}

func (r *Doer) do() {
	count := r.count.Swap(0)
	if count == 0 {
		return
	}

	r.lastDone = r.nowFn()
	r.doFn(count, r.period)

}
