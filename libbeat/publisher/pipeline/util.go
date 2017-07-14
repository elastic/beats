package pipeline

import "sync"

type sema struct {
	// simulate cancellable counting semaphore using counter + mutex + cond
	mutex      sync.Mutex
	cond       sync.Cond
	count, max int
}

func newSema(max int) *sema {
	s := &sema{max: max}
	s.cond.L = &s.mutex
	return s
}

func (s *sema) inc() {
	s.mutex.Lock()
	for s.count == s.max {
		s.cond.Wait()
	}
	s.mutex.Unlock()
}

func (s *sema) release(n int) {
	s.mutex.Lock()
	old := s.count
	s.count -= n
	if old == s.max {
		if n == 1 {
			s.cond.Signal()
		} else {
			s.cond.Broadcast()
		}
	}
	s.mutex.Unlock()
}
