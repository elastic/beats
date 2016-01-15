package flows

import (
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"
)

type worker struct {
	wg   sync.WaitGroup
	done chan struct{}
	run  func(*worker)
}

type spool struct {
	pub    publisher.Client
	events []common.MapStr
}

func newWorker(fn func(w *worker)) *worker {
	return &worker{
		done: make(chan struct{}),
		run:  fn,
	}
}

func (w *worker) Start() {
	w.wg.Add(1)
	go w.run(w)
}

func (w *worker) Stop() {
	close(w.done)
}

func (w *worker) finished() {
	w.wg.Done()
	logp.Info("flows worker loop stopped")
}

func (w *worker) sleep(d time.Duration) bool {
	select {
	case <-w.done:
		return false
	case <-time.After(d):
		return true
	}
}

func (w *worker) tick(t *time.Ticker) bool {
	select {
	case <-w.done:
		return false
	case <-t.C:
		return true
	}
}

func (w *worker) periodicaly(tick time.Duration, fn func() error) {
	ticker := time.NewTicker(tick)
	for {
		cont := w.tick(ticker)
		if !cont {
			return
		}

		err := fn()
		if err != nil {
			return
		}
	}
}

func (s *spool) init(pub publisher.Client, sz int) {
	s.pub = pub
	s.events = make([]common.MapStr, 0, sz)
}

func (s *spool) publish(event common.MapStr) {
	s.events = append(s.events, event)
	if len(s.events) == cap(s.events) {
		s.flush()
	}
}

func (s *spool) flush() {
	if len(s.events) == 0 {
		return
	}

	s.pub.PublishEvents(s.events)
	s.events = make([]common.MapStr, 0, cap(s.events))
}

func gcd(a, b int64) int64 {
	if a < 0 || b < 0 {
		return 0
	}

	switch {
	case a == b:
		return a
	case a == 0:
		return b
	case b == 0:
		return a
	}

	shift := uint(0)
	for (a&1) == 0 && (b&1) == 0 {
		shift++
		a /= 2
		b /= 2
	}

	for (a & 1) == 0 {
		a = a / 2
	}

	// a is always odd
	for {
		for (b & 1) == 0 {
			b = b / 2
		}

		// both a and b are odd. guaranteed b >= a
		if a > b {
			a, b = b, a
		}
		b -= a

		if b == 0 {
			break
		}
	}

	// restore common factors of 2
	return a << shift
}
