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

package spool

import (
	"time"
)

type timer struct {
	// flush timer
	timer    *time.Timer
	C        <-chan time.Time
	duration time.Duration
}

func newTimer(duration time.Duration) *timer {
	stdtimer := time.NewTimer(duration)
	if !stdtimer.Stop() {
		<-stdtimer.C
	}

	return &timer{
		timer:    stdtimer,
		C:        nil,
		duration: duration,
	}
}

func (t *timer) Zero() bool {
	return t.duration == 0
}

func (t *timer) Restart() {
	t.Stop(false)
	t.Start()
}

func (t *timer) Start() {
	if t.C != nil {
		return
	}

	t.timer.Reset(t.duration)
	t.C = t.timer.C
}

func (t *timer) Stop(triggered bool) {
	if t.C == nil {
		return
	}

	if !triggered && !t.timer.Stop() {
		<-t.C
	}

	t.C = nil
}
