package stress

import "github.com/elastic/beats/libbeat/common/atomic"

type closeSignaler struct {
	active atomic.Bool
	done   chan struct{}
}

func newCloseSignaler() *closeSignaler {
	return &closeSignaler{
		active: atomic.MakeBool(true),
		done:   make(chan struct{}),
	}
}

func (s *closeSignaler) Close() {
	if act := s.active.Swap(false); act {
		close(s.done)
	}
}

func (s *closeSignaler) Active() bool {
	return s.active.Load()
}

func (s *closeSignaler) C() <-chan struct{} {
	return s.done
}
