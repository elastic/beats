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
