package scheduler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type tickRecorder struct {
	scheduler Scheduler
	count     int
	done      chan struct{}
	recorder  chan int
}

func (m *tickRecorder) Start() {
	for {
		select {
		case <-m.scheduler.WaitTick():
			m.count = m.count + 1
			m.recorder <- m.count
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
}

func newTickRecorder(scheduler Scheduler) *tickRecorder {
	return &tickRecorder{
		scheduler: scheduler,
		done:      make(chan struct{}),
		recorder:  make(chan int),
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
		require.Equal(t, 1, <-recorder.recorder)
		scheduler.Next()
		require.Equal(t, 2, <-recorder.recorder)
		scheduler.Next()
		require.Equal(t, 3, <-recorder.recorder)
	})
}

func testPeriodic(t *testing.T) {
	duration := 1 * time.Millisecond
	scheduler := NewPeriodic(duration)
	defer scheduler.Stop()

	recorder := newTickRecorder(scheduler)
	go recorder.Start()
	defer recorder.Stop()

	require.Equal(t, 1, <-recorder.recorder)
	require.Equal(t, 2, <-recorder.recorder)
	require.Equal(t, 3, <-recorder.recorder)
}
