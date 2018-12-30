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

// strbld package provides a string Builder that can be used with older go
// versions as well. The builder provided by strings.Builder will be used for
// go versions 1.10+.
// The Builder interface is fully compatible to strings.Builder. Only
// additional methods available are Pad and Fmt. If go versions < 1.10 are
// used, no additional methods of the underlying buffer can be used.
package strbld

import "fmt"

// Pad writes str to the buffer, only if the buffer is not empty.
func (b *Builder) Pad(str string) {
	if b.Len() > 0 {
		b.WriteString(str)
	}
}

// Fmt writes the formatted string to the buffer.
func (b *Builder) Fmt(s string, vs ...interface{}) {
	b.WriteString(fmt.Sprintf(s, vs...))
}

// Grow increases the buffer its capacity if required. After grow, at least n
// bytes can be written without further allocations.
func (b *Builder) Grow(n int) {
	b.buf.Grow(n)
}

// Len returns the number of bytes written to the buffer.
func (b *Builder) Len() int {
	return b.buf.Len()
}

// Write appends p to the buffer. Write always returns len(p), nil.
func (b *Builder) Write(p []byte) (int, error) {
	return b.buf.Write(p)
}

// WriteByte appends c to the buffer. WriteByte always returns nil.
func (b *Builder) WriteByte(c byte) error {
	return b.buf.WriteByte(c)
}

// WriteRune appends r to the buffer. WriteRune always returns the length or r
// (UTF-8 encoded) and nil.
func (b *Builder) WriteRune(r rune) (int, error) {
	return b.buf.WriteRune(r)
}

// WriteString appends s to the buffer. WriteString always returns len(s), nil.
func (b *Builder) WriteString(s string) (int, error) {
	return b.buf.WriteString(s)
}
