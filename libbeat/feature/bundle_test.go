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

package feature

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBundle(t *testing.T) {
	factory := func() {}
	features := []Featurable{
		New("libbeat.outputs", "elasticsearch", factory, &Details{stability: Stable}),
		New("libbeat.outputs", "edge", factory, &Details{stability: Experimental}),
		New("libbeat.input", "tcp", factory, &Details{stability: Beta}),
	}

	t.Run("Creates a new Bundle", func(t *testing.T) {
		b := NewBundle(features...)
		assert.Equal(t, 3, len(b.Features()))
	})

	t.Run("Filters feature based on stability", func(t *testing.T) {
		b := NewBundle(features...)
		new := b.Filter(Experimental)
		assert.Equal(t, 1, len(new.Features()))
	})

	t.Run("Filters feature based on multiple different stability", func(t *testing.T) {
		b := NewBundle(features...)
		new := b.Filter(Experimental, Stable)
		assert.Equal(t, 2, len(new.Features()))
	})

	t.Run("Creates a new Bundle from specified feature", func(t *testing.T) {
		f1 := New("libbeat.outputs", "elasticsearch", factory, &Details{stability: Stable})
		b := MustBundle(f1)
		assert.Equal(t, 1, len(b.Features()))
	})

	t.Run("Creates a new Bundle with grouped features", func(t *testing.T) {
		f1 := New("libbeat.outputs", "elasticsearch", factory, &Details{stability: Stable})
		f2 := New("libbeat.outputs", "edge", factory, &Details{stability: Experimental})
		f3 := New("libbeat.input", "tcp", factory, &Details{stability: Beta})
		f4 := New("libbeat.input", "udp", factory, &Details{stability: Beta})

		b := MustBundle(
			MustBundle(f1),
			MustBundle(f2),
			MustBundle(f3, f4),
		)

		assert.Equal(t, 4, len(b.Features()))
	})
}
