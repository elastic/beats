// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build (linux && 386) || (linux && amd64)

package helper

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type intElement struct {
	value int
	prev  LinkedElement
	next  LinkedElement
}

func (i *intElement) Prev() LinkedElement {
	return i.prev
}

func (i *intElement) Next() LinkedElement {
	return i.next
}

func (i *intElement) SetPrev(element LinkedElement) {
	i.prev = element
}

func (i *intElement) SetNext(element LinkedElement) {
	i.next = element
}

func (i *intElement) Timestamp() time.Time {
	return time.Unix(int64(i.value), 0)
}

func benchmarkLinkedListAdd(b *testing.B) {
	var input, output LinkedList
	for i := 0; i < b.N; i++ {
		input.Add(&intElement{})
	}
	// Enable allocations reporting
	b.ReportAllocs()
	// Reset timer and allocation counters
	b.ResetTimer()
	// Consuming a linked list and constructing a new linked list from the
	// original elements must result in zero allocations.
	for elem := input.Get(); elem != nil; elem = input.Get() {
		output.Add(elem)
	}
}

func benchmarkLinkedListAppend(b *testing.B) {
	var dst LinkedList
	elems := make([]intElement, b.N)
	getList := func() (ret LinkedList) {
		if n := len(elems) - 1; n >= 0 {
			ret.Add(&elems[n])
			elems = elems[:n]
		}
		return ret
	}
	// Enable allocations reporting
	b.ReportAllocs()
	// Reset timer and allocation counters
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lst := getList()
		dst.Append(&lst)
	}
}

func benchmarkLinkedListRemoveOlder(b *testing.B) {
	var ll LinkedList
	for i := 0; i < b.N; i++ {
		ll.Add(&intElement{})
	}
	// Enable allocations reporting
	b.ReportAllocs()
	// Reset timer and allocation counters
	b.ResetTimer()
	remove := true
	ll.RemoveOlder(time.Now(), func(e LinkedElement) bool {
		time.Sleep(time.Millisecond)
		if remove = !remove; remove {
			ll.Remove(e)
		}
		return remove
	})
}

func TestLinkedListAddNoAllocs(t *testing.T) {
	result := testing.Benchmark(benchmarkLinkedListAdd)
	assert.Zero(t, result.AllocsPerOp())
	assert.Zero(t, result.AllocedBytesPerOp())
}

func TestLinkedListAppendNoAllocs(t *testing.T) {
	result := testing.Benchmark(benchmarkLinkedListAppend)
	assert.Zero(t, result.AllocsPerOp())
	assert.Zero(t, result.AllocedBytesPerOp())
}

func TestLinkedListRemoveOlderNoAllocs(t *testing.T) {
	result := testing.Benchmark(benchmarkLinkedListRemoveOlder)
	assert.Zero(t, result.AllocsPerOp())
	assert.Zero(t, result.AllocedBytesPerOp())
}

func TestLinkedListRemoveOlder(t *testing.T) {
	callbacks := map[string]func(*LinkedList) func(LinkedElement) bool{
		"self_removed": func(ll *LinkedList) func(LinkedElement) bool {
			return func(e LinkedElement) bool {
				ll.Remove(e)
				return true
			}
		},
		"caller_removed": func(*LinkedList) func(LinkedElement) bool {
			return func(LinkedElement) bool {
				return false
			}
		},
	}

	deadline := time.Unix(10, 0)
	for idx, testCase := range []struct {
		input    []int
		expected []int
	}{
		{
			input:    nil,
			expected: nil,
		},
		{
			input:    []int{1, 2, 3, 4, 5, 6, 7, 8, 9},
			expected: nil,
		},
		{
			input:    []int{1, 1, 10},
			expected: []int{10},
		},
		{
			input:    []int{11, 12, 13},
			expected: []int{11, 12, 13},
		},
	} {
		for name, callback := range callbacks {
			t.Run(fmt.Sprintf("case#%d-%s", idx+1, name), func(t *testing.T) {
				var input LinkedList
				for _, v := range testCase.input {
					input.Add(&intElement{value: v})
				}
				input.RemoveOlder(deadline, callback(&input))
				var remaining []int
				for item := input.Get(); item != nil; item = input.Get() {
					remaining = append(remaining, item.(*intElement).value)
				}
				assert.Equal(t, testCase.expected, remaining)
			})
		}
	}
}
