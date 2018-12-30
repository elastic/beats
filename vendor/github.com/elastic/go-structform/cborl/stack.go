// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

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
