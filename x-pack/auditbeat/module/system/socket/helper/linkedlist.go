// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build (linux && 386) || (linux && amd64)

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
// The element `e` must be in `l` before this call.
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
func (l *LinkedList) RemoveOlder(deadline time.Time, callback func(LinkedElement) (removed bool)) {
	for l.head != nil && l.head.Timestamp().Before(deadline) {
		if !callback(l.head) {
			l.Get()
		}
	}
}
