package cborl

type stateStack struct {
	stack   []state // state stack for nested arrays/objects
	stack0  [64]state
	current state
}

type lengthStack struct {
	stack   []int64
	stack0  [32]int64
	current int64
}

func (s *stateStack) init(s0 state) {
	s.current = s0
	s.stack = s.stack0[:0]
}

func (s *stateStack) push(next state) {
	if s.current.major != stFail {
		s.stack = append(s.stack, s.current)
	}
	s.current = next
}

func (s *stateStack) pop() {
	if len(s.stack) == 0 {
		s.current = state{stFail, stStart}
	} else {
		last := len(s.stack) - 1
		s.current = s.stack[last]
		s.stack = s.stack[:last]
	}
}

func (s *lengthStack) init() {
	s.stack = s.stack0[:0]
}

func (s *lengthStack) push(l int64) {
	s.stack = append(s.stack, s.current)
	s.current = l
}

func (s *lengthStack) pop() int64 {
	if len(s.stack) == 0 {
		s.current = -1
		return -1
	} else {
		last := len(s.stack) - 1
		old := s.current
		s.current = s.stack[last]
		s.stack = s.stack[:last]
		return old
	}
}
