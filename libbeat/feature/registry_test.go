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

func TestRegister(t *testing.T) {
	f := func() {}

	t.Run("when the factory is nil", func(t *testing.T) {
		r := newRegistry()
		err := r.Register(New("outputs", "null", nil, Stable))
		if !assert.Error(t, err) {
			return
		}
	})

	t.Run("namespace and feature doesn't exist", func(t *testing.T) {
		r := newRegistry()
		err := r.Register(New("outputs", "null", f, Stable))
		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(t, 1, r.Size())
	})

	t.Run("namespace exists and feature doesn't exist", func(t *testing.T) {
		r := newRegistry()
		r.Register(New("processor", "bar", f, Stable))
		err := r.Register(New("processor", "foo", f, Stable))
		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(t, 2, r.Size())
	})

	t.Run("namespace exists and feature exists and not the same factory", func(t *testing.T) {
		r := newRegistry()
		r.Register(New("processor", "foo", func() {}, Stable))
		err := r.Register(New("processor", "foo", f, Stable))
		if !assert.Error(t, err) {
			return
		}
		assert.Equal(t, 1, r.Size())
	})

	t.Run("when the exact feature is already registered", func(t *testing.T) {
		feature := New("processor", "foo", f, Stable)
		r := newRegistry()
		r.Register(feature)
		err := r.Register(feature)
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, 1, r.Size())
	})
}

func TestFeature(t *testing.T) {
	f := func() {}

	r := newRegistry()
	r.Register(New("processor", "foo", f, Stable))
	r.Register(New("HOLA", "fOO", f, Stable))

	t.Run("when namespace and feature are present", func(t *testing.T) {
		feature, err := r.Lookup("processor", "foo")
		if !assert.NotNil(t, feature.Factory()) {
			return
		}
		assert.NoError(t, err)
	})

	t.Run("when namespace doesn't exist", func(t *testing.T) {
		_, err := r.Lookup("hello", "foo")
		if !assert.Error(t, err) {
			return
		}
	})

	t.Run("when namespace and key are normalized", func(t *testing.T) {
		_, err := r.Lookup("HOLA", "foo")
		if !assert.NoError(t, err) {
			return
		}
	})
}

func TestLookup(t *testing.T) {
	f := func() {}

	r := newRegistry()
	r.Register(New("processor", "foo", f, Stable))
	r.Register(New("processor", "foo2", f, Stable))
	r.Register(New("HELLO", "fOO", f, Stable))

	t.Run("when namespace and feature are present", func(t *testing.T) {
		features, err := r.LookupAll("processor")
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, 2, len(features))
	})

	t.Run("when namespace is not present", func(t *testing.T) {
		_, err := r.LookupAll("foobar")
		if !assert.Error(t, err) {
			return
		}
	})

	t.Run("when namespace and name are normalized", func(t *testing.T) {
		features, err := r.LookupAll("hello")
		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(t, 1, len(features))
	})
}

func TestUnregister(t *testing.T) {
	f := func() {}

	t.Run("when the namespace and the feature exists", func(t *testing.T) {
		r := newRegistry()
		r.Register(New("processor", "foo", f, Stable))
		assert.Equal(t, 1, r.Size())
		err := r.Unregister("processor", "foo")
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, 0, r.Size())
	})

	t.Run("when the namespace exist and the feature doesn't", func(t *testing.T) {
		r := newRegistry()
		r.Register(New("processor", "foo", f, Stable))
		assert.Equal(t, 1, r.Size())
		err := r.Unregister("processor", "bar")
		if assert.Error(t, err) {
			return
		}
		assert.Equal(t, 0, r.Size())
	})

	t.Run("when the namespace doesn't exists", func(t *testing.T) {
		r := newRegistry()
		r.Register(New("processor", "foo", f, Stable))
		assert.Equal(t, 1, r.Size())
		err := r.Unregister("outputs", "bar")
		if assert.Error(t, err) {
			return
		}
		assert.Equal(t, 0, r.Size())
	})
}
