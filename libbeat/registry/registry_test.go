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

package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAccessStore(t *testing.T) {
	t.Run("single access", func(t *testing.T) {
		mr := newMockRegistry()
		ms := newMockStore()
		mr.OnClose().Once().Return(nil)
		mr.OnAccess("test").Once().Return(ms, nil)
		ms.OnClose().Once().Return(nil)

		reg := NewRegistry(mr)
		store, _ := reg.Get("test")
		assert.NoError(t, store.Close())
		assert.NoError(t, reg.Close())

		mr.AssertExpectations(t)
		ms.AssertExpectations(t)
	})

	t.Run("shared store instance", func(t *testing.T) {
		mr := newMockRegistry()
		ms := newMockStore()
		mr.OnClose().Once().Return(nil)

		// test instance sharing. Store must be opened and closed only once
		mr.OnAccess("test").Once().Return(ms, nil)
		ms.OnClose().Once().Return(nil)

		reg := NewRegistry(mr)
		s1, _ := reg.Get("test")
		s2, _ := reg.Get("test")
		assert.NoError(t, s1.Close())
		assert.NoError(t, s2.Close())
		assert.NoError(t, reg.Close())

		mr.AssertExpectations(t)
		ms.AssertExpectations(t)
	})

	t.Run("close non-shared store needs open", func(t *testing.T) {
		mr := newMockRegistry()
		ms := newMockStore()
		mr.OnClose().Once().Return(nil)

		// test instance sharing. Store must be opened and closed only once
		mr.OnAccess("test").Twice().Return(ms, nil)
		ms.OnClose().Twice().Return(nil)

		reg := NewRegistry(mr)

		store, err := reg.Get("test")
		assert.NoError(t, err)
		assert.NoError(t, store.Close())

		store, err = reg.Get("test")
		assert.NoError(t, err)
		assert.NoError(t, store.Close())

		assert.NoError(t, reg.Close())

		mr.AssertExpectations(t)
		ms.AssertExpectations(t)
	})

	t.Run("separate stores are not shared", func(t *testing.T) {
		mr := newMockRegistry()
		mr.OnClose().Once().Return(nil)

		ms1 := newMockStore()
		ms1.OnClose().Once().Return(nil)
		mr.OnAccess("s1").Once().Return(ms1, nil)

		ms2 := newMockStore()
		ms2.OnClose().Once().Return(nil)
		mr.OnAccess("s2").Once().Return(ms2, nil)

		reg := NewRegistry(mr)
		s1, err := reg.Get("s1")
		assert.NoError(t, err)
		s2, err := reg.Get("s2")
		assert.NoError(t, err)
		assert.NoError(t, s1.Close())
		assert.NoError(t, s2.Close())
		assert.NoError(t, reg.Close())

		mr.AssertExpectations(t)
		ms1.AssertExpectations(t)
		ms2.AssertExpectations(t)
	})
}
