package beater

import "time"

type signalWait struct {
	count   int // number of potential 'alive' signals
	signals chan struct{}
}

func newSignalWait() *signalWait {
	return &signalWait{
		signals: make(chan struct{}, 1),
	}
}

func (s *signalWait) Wait() {
	if s.count == 0 {
		return
	}

	<-s.signals
	s.count--
}

func (s *signalWait) Add(fn func()) {
	s.count++
	go func() {
		fn()
		var v struct{}
		s.signals <- v
	}()
}

func (s *signalWait) AddChan(c <-chan struct{}) {
	s.Add(func() { <-c })
}

func (s *signalWait) AddTimer(t *time.Timer) {
	s.Add(func() { <-t.C })
}

func (s *signalWait) AddTimeout(d time.Duration) {
	s.AddTimer(time.NewTimer(d))
}

func (s *signalWait) Signal() {
	s.Add(func() {})
}
