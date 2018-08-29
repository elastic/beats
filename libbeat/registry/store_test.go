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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreBeginTx(t *testing.T) {
	t.Run("closed", testStoreBeginTxClosed)
	t.Run("run tx", testStoreWithTx)
}

func testStoreBeginTxClosed(t *testing.T) {
	makeTestStore := func(t *testing.T, ms *mockStore) *Store {
		mr := newMockRegistry()
		mr.OnAccess("test").Once().Return(ms, nil)

		reg := NewRegistry(mr)

		s, err := reg.Get("test")
		require.NoError(t, err)

		ms.OnClose().Return(nil)
		require.NoError(t, s.Close())
		return s
	}

	t.Run("begin readonly", func(t *testing.T) {
		ms := newMockStore()
		store := makeTestStore(t, ms)

		_, err := store.Begin(true)
		assert.Error(t, err)

		ms.AssertExpectations(t)
		ms.AssertNotCalled(t, "Begin")
	})

	t.Run("begin rw", func(t *testing.T) {
		ms := newMockStore()
		store := makeTestStore(t, ms)

		_, err := store.Begin(false)
		assert.Error(t, err)

		ms.AssertExpectations(t)
		ms.AssertNotCalled(t, "Begin")
	})

	t.Run("update fails", func(t *testing.T) {
		ms := newMockStore()
		store := makeTestStore(t, ms)

		err := store.Update(func(tx *Tx) error { return nil })
		assert.Error(t, err)

		ms.AssertExpectations(t)
		ms.AssertNotCalled(t, "Begin")
	})

	t.Run("view fails", func(t *testing.T) {
		ms := newMockStore()
		store := makeTestStore(t, ms)

		err := store.View(func(tx *Tx) error { return nil })
		assert.Error(t, err)

		ms.AssertExpectations(t)
		ms.AssertNotCalled(t, "Begin")
	})
}

func testStoreWithTx(t *testing.T) {
	makeTestStore := func(t *testing.T, ms *mockStore) *Store {
		mr := newMockRegistry()
		mr.OnAccess("test").Once().Return(ms, nil)

		reg := NewRegistry(mr)
		s, err := reg.Get("test")
		require.NoError(t, err)

		ms.OnClose().Return(nil)
		return s
	}

	t.Run("begin readonly", func(t *testing.T) {
		ms := newMockStore()
		mtx := newMockTx()
		ms.OnBegin(true).Once().Return(mtx, nil)
		mtx.OnClose().Once().Return(nil)

		s := makeTestStore(t, ms)
		tx, err := s.Begin(true)
		require.NoError(t, err)
		require.NoError(t, tx.Close())
		require.NoError(t, s.Close())

		ms.AssertExpectations(t)
		mtx.AssertExpectations(t)
	})

	t.Run("begin rw", func(t *testing.T) {
		ms := newMockStore()
		mtx := newMockTx()
		ms.OnBegin(false).Once().Return(mtx, nil)
		mtx.OnClose().Once().Return(nil)
		mtx.OnRollback().Once().Return(nil)

		s := makeTestStore(t, ms)
		tx, err := s.Begin(false)
		require.NoError(t, err)
		require.NoError(t, tx.Close())
		require.NoError(t, s.Close())

		ms.AssertExpectations(t)
		mtx.AssertExpectations(t)
	})

	t.Run("fail close if internal rollback fails", func(t *testing.T) {
		errRollbackFail := errors.New("rollback test error")

		ms := newMockStore()
		mtx := newMockTx()
		ms.OnBegin(false).Once().Return(mtx, nil)
		mtx.OnClose().Once().Return(nil)
		mtx.OnRollback().Once().Return(errRollbackFail)

		s := makeTestStore(t, ms)
		tx, err := s.Begin(false)
		require.NoError(t, err)
		require.Error(t, tx.Close())
		require.NoError(t, s.Close())

		ms.AssertExpectations(t)
		mtx.AssertExpectations(t)
	})

	t.Run("fail close if internal close fails", func(t *testing.T) {
		errCloseFail := errors.New("close test error")

		ms := newMockStore()
		mtx := newMockTx()
		ms.OnBegin(false).Once().Return(mtx, nil)
		mtx.OnClose().Once().Return(errCloseFail)
		mtx.OnRollback().Once().Return(nil)

		s := makeTestStore(t, ms)
		tx, err := s.Begin(false)
		require.NoError(t, err)
		require.Equal(t, errCloseFail, tx.Close())
		require.NoError(t, s.Close())

		ms.AssertExpectations(t)
		mtx.AssertExpectations(t)
	})

	t.Run("update rolls back if tx fails", func(t *testing.T) {
		errTxFail := errors.New("test tx fail")

		ms := newMockStore()
		mtx := newMockTx()
		ms.OnBegin(false).Once().Return(mtx, nil) // check update uses rw transaction
		ms.OnClose().Once().Return(nil)
		mtx.OnRollback().Once().Return(nil)
		mtx.OnClose().Once().Return(nil)

		s := makeTestStore(t, ms)
		err := s.Update(func(tx *Tx) error { return errTxFail })
		assert.Error(t, err)
		require.NoError(t, s.Close())

		ms.AssertExpectations(t)
		mtx.AssertExpectations(t)
	})

	t.Run("update commits if tx succeeds", func(t *testing.T) {
		ms := newMockStore()
		mtx := newMockTx()
		ms.OnBegin(false).Once().Return(mtx, nil) // check update uses rw transaction
		ms.OnClose().Once().Return(nil)
		mtx.OnCommit().Once().Return(nil)
		mtx.OnClose().Once().Return(nil)

		s := makeTestStore(t, ms)
		err := s.Update(func(tx *Tx) error { return nil })
		assert.NoError(t, err, "update did not succeed")
		assert.NoError(t, s.Close(), "failure on close")

		ms.AssertExpectations(t)
		mtx.AssertExpectations(t)
	})

	t.Run("view uses readonly tx", func(t *testing.T) {
		ms := newMockStore()
		mtx := newMockTx()
		ms.OnBegin(true).Once().Return(mtx, nil) // check update uses rw transaction
		ms.OnClose().Once().Return(nil)
		mtx.OnClose().Once().Return(nil)

		s := makeTestStore(t, ms)
		err := s.View(func(tx *Tx) error { return nil })
		assert.NoError(t, err, "update did not succeed")
		assert.NoError(t, s.Close(), "failure on close")

		ms.AssertExpectations(t)
		mtx.AssertExpectations(t)
	})
}
