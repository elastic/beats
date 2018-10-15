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
