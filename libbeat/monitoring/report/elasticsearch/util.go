package elasticsearch

import "sync"

type stopper struct {
	once sync.Once
	done chan struct{}
}

func newStopper() *stopper {
	return &stopper{done: make(chan struct{})}
}

func (s *stopper) Stop() {
	s.once.Do(func() { close(s.done) })
}

func (s *stopper) C() <-chan struct{} {
	return s.done
}

func (s *stopper) Wait() {
	<-s.done
}

func (s *stopper) DoWait(f func()) {
	s.Wait()
	f()
}
