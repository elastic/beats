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

package dissect

import (
	"fmt"
	"strings"
)

//delimiter represents a text section after or before a key, it keeps track of the needle and allows
// to retrieve the position where it starts from a haystack.
type delimiter interface {
	// IndexOf receives the haystack and a offset position and will return the absolute position where
	// the needle is found.
	IndexOf(haystack string, offset int) int

	// Len returns the length of the needle used to calculate boundaries.
	Len() int

	// String displays debugging information.
	String() string

	// Delimiter returns the actual delimiter string.
	Delimiter() string

	// IsGreedy return true if the next key should be greedy (end of string) or when explicitly
	// configured.
	IsGreedy() bool

	// MarkGreedy marks this delimiter as greedy.
	MarkGreedy()

	// Next returns the next delimiter in the chain.
	Next() delimiter

	//SetNext sets the next delimiter or nil if current delimiter is the last.
	SetNext(d delimiter)
}

// zeroByte represents a zero string delimiter its usually start of the line.
type zeroByte struct {
	needle string
	greedy bool
	next   delimiter
}

func (z *zeroByte) IndexOf(haystack string, offset int) int {
	return offset
}

func (z *zeroByte) Len() int {
	return 0
}

func (z *zeroByte) String() string {
	return "delimiter: zerobyte"
}

func (z *zeroByte) Delimiter() string {
	return z.needle
}

func (z *zeroByte) IsGreedy() bool {
	return z.greedy
}

func (z *zeroByte) MarkGreedy() {
	z.greedy = true
}

func (z *zeroByte) Next() delimiter {
	return z.next
}

func (z *zeroByte) SetNext(d delimiter) {
	z.next = d
}

// multiByte represents a delimiter with at least one byte.
type multiByte struct {
	needle string
	greedy bool
	next   delimiter
}

func (m *multiByte) IndexOf(haystack string, offset int) int {
	i := strings.Index(haystack[offset:], m.needle)
	if i != -1 {
		return i + offset
	}
	return -1
}

func (m *multiByte) Len() int {
	return len(m.needle)
}

func (m *multiByte) IsGreedy() bool {
	return m.greedy
}

func (m *multiByte) MarkGreedy() {
	m.greedy = true
}

func (m *multiByte) String() string {
	return fmt.Sprintf("delimiter: multibyte (match: '%s', len: %d)", string(m.needle), m.Len())
}

func (m *multiByte) Delimiter() string {
	return m.needle
}

func (m *multiByte) Next() delimiter {
	return m.next
}

func (m *multiByte) SetNext(d delimiter) {
	m.next = d
}

func newDelimiter(needle string) delimiter {
	if len(needle) == 0 {
		return &zeroByte{}
	}
	return &multiByte{needle: needle}
}
