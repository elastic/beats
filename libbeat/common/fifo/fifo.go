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

package fifo

// FIFO is a minimal first-in first-out queue based on a singly linked list.
type FIFO[T any] struct {
	first *node[T]
	last  *node[T]
}

type node[T any] struct {
	next  *node[T]
	value T
}

func (f *FIFO[T]) Add(value T) {
	newNode := &node[T]{value: value}
	if f.first == nil {
		f.first = newNode
	} else {
		f.last.next = newNode
	}
	f.last = newNode
}

func (f *FIFO[T]) Empty() bool {
	return f.first == nil
}

// Return the first value (if present) without removing it from the queue.
// If the queue is empty a default value is returned, to detect this case
// use f.Empty().
func (f *FIFO[T]) First() T {
	if f.first == nil {
		var none T
		return none
	}
	return f.first.value
}

// Return the first value (if present) and remove it from the queue.
// If the queue is empty a default value is returned, to detect this case
// use f.Empty().
func (f *FIFO[T]) ConsumeFirst() T {
	result := f.First()
	f.Remove()
	return result
}

// Remove the first entry in the queue. Does nothing if the FIFO is empty.
func (f *FIFO[T]) Remove() {
	if f.first != nil {
		f.first = f.first.next
		if f.first == nil {
			f.last = nil
		}
	}
}
