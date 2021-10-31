package internal

import "sync"

type closeSignaler struct {
	done chan struct{}
	wg   sync.WaitGroup
}

func makeCloseSignaler() closeSignaler {
	return closeSignaler{
		done: make(chan struct{}),
	}
}

func (s *closeSignaler) Add(n int) {
	s.wg.Add(n)
}

func (s *closeSignaler) Sig() <-chan struct{} {
	return s.done
}

func (s *closeSignaler) Done() {
	s.wg.Done()
}

func (s *closeSignaler) Close() {
	close(s.done)
	s.wg.Wait()
}
