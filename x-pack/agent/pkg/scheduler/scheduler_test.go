// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package scheduler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type e struct {
	count int
	at    time.Time
}

type tickRecorder struct {
	scheduler Scheduler
	count     int
	done      chan struct{}
	recorder  chan e
}

func (m *tickRecorder) Start() {
	for {
		select {
		case t := <-m.scheduler.WaitTick():
			m.count = m.count + 1
			m.recorder <- e{count: m.count, at: t}
		case <-m.done:
			return
		}
	}
}

func (m *tickRecorder) Stop() {
	close(m.done)
}

func TestScheduler(t *testing.T) {
	t.Run("Step scheduler", testStepScheduler)
	t.Run("Periodic scheduler", testPeriodic)
	t.Run("PeriodicJitter scheduler", testPeriodicJitter)
}

func newTickRecorder(scheduler Scheduler) *tickRecorder {
	return &tickRecorder{
		scheduler: scheduler,
		done:      make(chan struct{}),
		recorder:  make(chan e),
	}
}

func testStepScheduler(t *testing.T) {
	t.Run("Trigger the Tick manually", func(t *testing.T) {
		scheduler := NewStepper()
		defer scheduler.Stop()

		recorder := newTickRecorder(scheduler)
		go recorder.Start()
		defer recorder.Stop()

		scheduler.Next()
		nE := <-recorder.recorder
		require.Equal(t, 1, nE.count)
		scheduler.Next()
		nE = <-recorder.recorder
		require.Equal(t, 2, nE.count)
		scheduler.Next()
		nE = <-recorder.recorder
		require.Equal(t, 3, nE.count)
	})
}

func testPeriodic(t *testing.T) {
	t.Run("tick than wait", func(t *testing.T) {
		duration := 1 * time.Minute
		scheduler := NewPeriodic(duration)
		defer scheduler.Stop()

		startedAt := time.Now()
		recorder := newTickRecorder(scheduler)
		go recorder.Start()
		defer recorder.Stop()

		nE := <-recorder.recorder

		require.True(t, nE.at.Sub(startedAt) < duration)
	})

	t.Run("multiple ticks", func(t *testing.T) {
		duration := 1 * time.Millisecond
		scheduler := NewPeriodic(duration)
		defer scheduler.Stop()

		recorder := newTickRecorder(scheduler)
		go recorder.Start()
		defer recorder.Stop()

		nE := <-recorder.recorder
		require.Equal(t, 1, nE.count)
		nE = <-recorder.recorder
		require.Equal(t, 2, nE.count)
		nE = <-recorder.recorder
		require.Equal(t, 3, nE.count)
	})

}

func testPeriodicJitter(t *testing.T) {
	t.Run("tick than wait", func(t *testing.T) {
		duration := 5 * time.Second
		variance := 2 * time.Second
		scheduler := NewPeriodicJitter(duration, variance)
		defer scheduler.Stop()

		startedAt := time.Now()
		recorder := newTickRecorder(scheduler)
		go recorder.Start()
		defer recorder.Stop()

		nE := <-recorder.recorder

		diff := nE.at.Sub(startedAt)
		require.True(
			t,
			diff < duration,
		)

		startedAt = time.Now()
		nE = <-recorder.recorder
		diff = nE.at.Sub(startedAt)
		require.True(
			t,
			diff >= duration,
		)
	})

	t.Run("multiple ticks", func(t *testing.T) {
		duration := 1 * time.Millisecond
		variance := 100 * time.Millisecond
		scheduler := NewPeriodicJitter(duration, variance)
		defer scheduler.Stop()

		recorder := newTickRecorder(scheduler)
		go recorder.Start()
		defer recorder.Stop()

		nE := <-recorder.recorder
		require.Equal(t, 1, nE.count)
		nE = <-recorder.recorder
		require.Equal(t, 2, nE.count)
		nE = <-recorder.recorder
		require.Equal(t, 3, nE.count)
	})

	t.Run("unblock on first tick", func(t *testing.T) {
		duration := 30 * time.Minute
		variance := 30 * time.Minute
		scheduler := NewPeriodicJitter(duration, variance)

		go func() {
			// Not a fan of introducing sync-timing-code but
			// give us a chance to be waiting.
			<-time.After(500 * time.Millisecond)
			scheduler.Stop()
		}()

		<-scheduler.WaitTick()
	})

	t.Run("unblock on any tick", func(t *testing.T) {
		duration := 1 * time.Millisecond
		variance := 2 * time.Second
		scheduler := NewPeriodicJitter(duration, variance)

		<-scheduler.WaitTick()

		// Increase time between next tick
		scheduler.d = 20 * time.Minute
		scheduler.variance = 20 * time.Minute

		go func() {
			// Not a fan of introducing sync-timing-code but
			// give us a chance to be waiting.
			<-time.After(500 * time.Millisecond)
			scheduler.Stop()
		}()

		<-scheduler.WaitTick()
	})
}
