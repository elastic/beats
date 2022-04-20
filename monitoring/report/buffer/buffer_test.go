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

package buffer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ringBuffer(t *testing.T) {
	t.Run("Len 2 buffer", func(t *testing.T) {
		r := newBuffer(2)
		assert.Equal(t, 2, len(r.entries))
		assert.False(t, r.full)
		assert.Equal(t, 0, r.i)

		assert.Empty(t, r.getAll())

		r.add("1")
		assert.False(t, r.full)
		assert.Equal(t, 1, r.i)
		assert.Equal(t, r.entries[0], "1")
		assert.ElementsMatch(t, []string{"1"}, r.getAll())

		r.add("2")
		assert.True(t, r.full)
		assert.Equal(t, 0, r.i)
		assert.Equal(t, r.entries[1], "2")
		assert.ElementsMatch(t, []string{"1", "2"}, r.getAll())

		r.add("3")
		assert.True(t, r.full)
		assert.Equal(t, 1, r.i)
		assert.Equal(t, r.entries[0], "3")
		assert.ElementsMatch(t, []string{"2", "3"}, r.getAll())

		r.add("4")
		assert.True(t, r.full)
		assert.Equal(t, 0, r.i)
		assert.Equal(t, r.entries[1], "4")
		assert.ElementsMatch(t, []string{"3", "4"}, r.getAll())
	})

	t.Run("Len 3 buffer", func(t *testing.T) {
		r := newBuffer(3)
		assert.Empty(t, r.getAll())

		r.add("1")
		assert.ElementsMatch(t, []string{"1"}, r.getAll())

		r.add("2")
		assert.ElementsMatch(t, []string{"1", "2"}, r.getAll())

		r.add("3")
		assert.ElementsMatch(t, []string{"1", "2", "3"}, r.getAll())

		r.add("4")
		assert.ElementsMatch(t, []string{"2", "3", "4"}, r.getAll())

		r.add("5")
		assert.ElementsMatch(t, []string{"3", "4", "5"}, r.getAll())

		r.add("6")
		assert.ElementsMatch(t, []string{"4", "5", "6"}, r.getAll())
	})
}

func Benchmark_ringBuffer_add(b *testing.B) {
	b.Run("size 6", func(b *testing.B) {
		r := newBuffer(6)
		for i := 0; i < b.N; i++ {
			r.add(i)
		}
	})
	b.Run("size 60", func(b *testing.B) {
		r := newBuffer(60)
		for i := 0; i < b.N; i++ {
			r.add(i)
		}
	})
	b.Run("size 600", func(b *testing.B) {
		r := newBuffer(600)
		for i := 0; i < b.N; i++ {
			r.add(i)
		}
	})
	b.Run("size 6000", func(b *testing.B) {
		r := newBuffer(6000)
		for i := 0; i < b.N; i++ {
			r.add(i)
		}
	})
}

func Benchmark_ringBuffer_add_filled(b *testing.B) {
	b.Run("size 6", func(b *testing.B) {
		r := newFullBuffer(b, 6)
		for i := 0; i < b.N; i++ {
			r.add(i)
		}
	})
	b.Run("size 60", func(b *testing.B) {
		r := newFullBuffer(b, 60)
		for i := 0; i < b.N; i++ {
			r.add(i)
		}
	})
	b.Run("size 600", func(b *testing.B) {
		r := newFullBuffer(b, 600)
		for i := 0; i < b.N; i++ {
			r.add(i)
		}
	})
	b.Run("size 6000", func(b *testing.B) {
		r := newFullBuffer(b, 6000)
		for i := 0; i < b.N; i++ {
			r.add(i)
		}
	})
}

func newFullBuffer(b *testing.B, size int) *ringBuffer {
	b.Helper()
	r := newBuffer(size)
	// fill size +1 so full flag is toggled
	for i := 0; i < size+1; i++ {
		r.add(-1)
	}
	return r
}
