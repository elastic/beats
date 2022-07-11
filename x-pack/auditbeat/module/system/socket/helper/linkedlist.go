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

package helper

import "time"

// LinkedList represents a linked list that can be used
// to construct an LRU.
type LinkedList struct {
	head, tail LinkedElement
	size       uint
}

// LinkedElement is the interface that must be implemented
// by types stored to a LinkedList.
type LinkedElement interface {
	// SetPrev links this element to the previous.
	SetPrev(LinkedElement)

	// SetNext links this element to the next.
	SetNext(LinkedElement)

	// Prev returns the LinkedElement set by SetPrev.
	Prev() LinkedElement

	// Next returns the LinkedElement set by SetNext.
	Next() LinkedElement

	// Timestamp returns the last-used time for this element.
	Timestamp() time.Time
}

// Size returns the number of elements in the LinkedList.
func (l *LinkedList) Size() uint {
	return l.size
}

// Append removes all elements from b and adds them
// to the end (tail) of the linked list.
func (l *LinkedList) Append(b *LinkedList) {
	if b.size == 0 {
		return
	}
	if l.size == 0 {
		*l = *b
		*b = LinkedList{}
		return
	}
	l.tail.SetNext(b.head)
	b.head.SetPrev(l.tail)
	l.tail = b.tail
	l.size += b.size
	*b = LinkedList{}
}

// Add adds the given element at the back (tail) of the
// linked list.
func (l *LinkedList) Add(f LinkedElement) {
	if f == nil || f.Next() != nil || f.Prev() != nil {
		panic("bad flow in Linked list")
	}
	l.size++
	if l.tail == nil {
		l.head = f
		l.tail = f
		f.SetNext(nil)
		f.SetPrev(nil)
		return
	}
	l.tail.SetNext(f)
	f.SetPrev(l.tail)
	l.tail = f
	f.SetNext(nil)
}

// Get removes and returns the first element in the LinkedList.
// If the list is empty, returns nil.
func (l *LinkedList) Get() LinkedElement {
	f := l.head
	if f != nil {
		l.Remove(f)
	}
	return f
}

// Remove removes the given LinkedElement from the LinkedList.
func (l *LinkedList) Remove(e LinkedElement) {
	l.size--
	if e.Prev() != nil {
		e.Prev().SetNext(e.Next())
	} else {
		l.head = e.Next()
	}
	if e.Next() != nil {
		e.Next().SetPrev(e.Prev())
	} else {
		l.tail = e.Prev()
	}
	e.SetPrev(nil)
	e.SetNext(nil)
}

// RemoveOlder sequentially scans the head of the Linked list for elements
// with a Timestamp() before the given deadline and calls the provided callback
// on them. The LinkedList must be sorted by incremental Timestamp() (LRU).
//
// This callback is expected to return true if it removed the element from
// the Linked list. Otherwise, it will be removed by this function.
func (l *LinkedList) RemoveOlder(deadline time.Time, callback func(LinkedElement) bool) {
	for l.head != nil && l.head.Timestamp().Before(deadline) {
		if !callback(l.head) {
			l.Get()
		}
	}
}
