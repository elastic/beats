package beater

import (
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/logp"
)

type signalWait struct {
	count   int // number of potential 'alive' signals
	signals chan struct{}
}

type signaler func()

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

func (s *signalWait) Add(fn signaler) {
	s.count++
	go func() {
		fn()
		var v struct{}
		s.signals <- v
	}()
}

func (s *signalWait) AddChan(c <-chan struct{}) {
	s.Add(waitChannel(c))
}

func (s *signalWait) AddTimer(t *time.Timer) {
	s.Add(waitTimer(t))
}

func (s *signalWait) AddTimeout(d time.Duration) {
	s.Add(waitDuration(d))
}

func (s *signalWait) Signal() {
	s.Add(func() {})
}

func waitGroup(wg *sync.WaitGroup) signaler {
	return wg.Wait
}

func waitChannel(c <-chan struct{}) signaler {
	return func() { <-c }
}

func waitTimer(t *time.Timer) signaler {
	return func() { <-t.C }
}

func waitDuration(d time.Duration) signaler {
	return waitTimer(time.NewTimer(d))
}

func withLog(s signaler, msg string) signaler {
	return func() {
		s()
		logp.Info("%v", msg)
	}
}
