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

package timerqueue

// timerHeap is the internal type that implements container/heap
// pointers are faster to swap than bare structs here.
type timerHeap []*timerTask

// Less computes the order of the heap. We want the earliest time to Pop().
func (th timerHeap) Less(i, j int) bool {
	// We want pop to give us the earliest (lowest) time so use before
	return th[i].runAt.Before(th[j].runAt)
}

// Swap switches two elements.
func (th timerHeap) Swap(i, j int) {
	th[i], th[j] = th[j], th[i]
}

// Push adds a new timerTask to the heap
func (th *timerHeap) Push(tt interface{}) {
	*th = append(*th, tt.(*timerTask))
}

// Pop returns the timerTask scheduled soonest.
func (th *timerHeap) Pop() interface{} {
	old := *th
	n := len(old)
	tt := old[n-1]
	*th = old[0 : n-1]
	return tt
}

// Len returns the length.
func (th timerHeap) Len() int {
	return len(th)
}
