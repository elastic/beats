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

package apm // import "go.elastic.co/apm"

import (
	"bytes"
	"context"
	"io"
	"runtime/pprof"
	"time"

	"github.com/pkg/errors"
)

type profilingState struct {
	profileType  string
	profileStart func(io.Writer) error
	profileStop  func()
	sender       profileSender

	interval time.Duration
	duration time.Duration // not relevant to all profiles

	timer      *time.Timer
	timerStart time.Time
	buf        bytes.Buffer
	finished   chan struct{}
}

// newCPUProfilingState calls newProfilingState with the
// profiler type set to "cpu", and using pprof.StartCPUProfile
// and pprof.StopCPUProfile.
func newCPUProfilingState(sender profileSender) *profilingState {
	return newProfilingState("cpu", pprof.StartCPUProfile, pprof.StopCPUProfile, sender)
}

// newHeapProfilingState calls newProfilingState with the
// profiler type set to "heap", and using pprof.Lookup("heap").WriteTo(writer, 0).
func newHeapProfilingState(sender profileSender) *profilingState {
	return newLookupProfilingState("heap", sender)
}

func newLookupProfilingState(name string, sender profileSender) *profilingState {
	profileStart := func(w io.Writer) error {
		profile := pprof.Lookup(name)
		if profile == nil {
			return errors.Errorf("no profile called %q", name)
		}
		return profile.WriteTo(w, 0)
	}
	return newProfilingState("heap", profileStart, func() {}, sender)
}

// newProfilingState returns a new profilingState,
// with its timer stopped. The timer may be started
// by calling profilingState.updateConfig.
func newProfilingState(
	profileType string,
	profileStart func(io.Writer) error,
	profileStop func(),
	sender profileSender,
) *profilingState {
	state := &profilingState{
		profileType:  profileType,
		profileStart: profileStart,
		profileStop:  profileStop,
		sender:       sender,
		timer:        time.NewTimer(0),
		finished:     make(chan struct{}, 1),
	}
	if !state.timer.Stop() {
		<-state.timer.C
	}
	return state
}

func (state *profilingState) updateConfig(interval, duration time.Duration) {
	if state.sender == nil {
		// No profile sender, no point in starting a timer.
		return
	}
	state.duration = duration
	if state.interval == interval {
		return
	}
	if state.timerStart.IsZero() {
		state.interval = interval
		state.resetTimer()
	}
	// TODO(axw) handle extending/cutting short running timers once
	// it is possible to dynamically control profiling configuration.
}

func (state *profilingState) resetTimer() {
	if state.interval != 0 {
		state.timer.Reset(state.interval)
		state.timerStart = time.Now()
	} else {
		state.timerStart = time.Time{}
	}
}

// start spawns a goroutine that will capture a profile, send it using state.sender,
// and finally signal state.finished.
//
// start will return immediately after spawning the goroutine.
func (state *profilingState) start(ctx context.Context, logger Logger, metadata io.Reader) {
	// The state.duration field may be updated after the goroutine starts,
	// by the caller, so it must be read outside the goroutine.
	duration := state.duration
	go func() {
		defer func() { state.finished <- struct{}{} }()
		if err := state.profile(ctx, duration); err != nil {
			if logger != nil && ctx.Err() == nil {
				logger.Errorf("%s", err)
			}
			return
		}
		// TODO(axw) backoff like SendStream requests
		if err := state.sender.SendProfile(ctx, metadata, &state.buf); err != nil {
			if logger != nil && ctx.Err() == nil {
				logger.Errorf("failed to send %s profile: %s", state.profileType, err)
			}
			return
		}
		if logger != nil {
			logger.Debugf("sent %s profile", state.profileType)
		}
	}()
}

func (state *profilingState) profile(ctx context.Context, duration time.Duration) error {
	state.buf.Reset()
	if err := state.profileStart(&state.buf); err != nil {
		return errors.Wrapf(err, "failed to start %s profile", state.profileType)
	}
	defer state.profileStop()

	if duration > 0 {
		timer := time.NewTimer(duration)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}
	return nil
}

type profileSender interface {
	SendProfile(ctx context.Context, metadata io.Reader, profile ...io.Reader) error
}
