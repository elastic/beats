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
	"github.com/stretchr/testify/mock"

	"github.com/elastic/beats/libbeat/registry/backend"
)

type mockRegistry struct {
	mock.Mock
}

type mockStore struct {
	mock.Mock
}

type mockTx struct {
	mock.Mock
}

func newMockRegistry() *mockRegistry { return &mockRegistry{} }

func (m *mockRegistry) OnAccess(name string) *mock.Call { return m.On("Access", name) }
func (m *mockRegistry) Access(name string) (backend.Store, error) {
	args := m.Called(name)

	var store backend.Store
	if ifc := args.Get(0); ifc != nil {
		store = ifc.(backend.Store)
	}

	return store, args.Error(1)
}

func (m *mockRegistry) OnClose() *mock.Call { return m.On("Close") }
func (m *mockRegistry) Close() error {
	args := m.Called()
	return args.Error(0)
}

func newMockStore() *mockStore { return &mockStore{} }

func (m *mockStore) OnClose() *mock.Call { return m.On("Close") }
func (m *mockStore) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockStore) OnBegin(readonly bool) *mock.Call { return m.On("Begin", readonly) }
func (m *mockStore) Begin(readonly bool) (backend.Tx, error) {
	args := m.Called(readonly)
	return args.Get(0).(backend.Tx), args.Error(1)
}

func newMockTx() *mockTx {
	return &mockTx{}
}

func (m *mockTx) OnClose() *mock.Call { return m.On("Close") }
func (m *mockTx) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockTx) OnRollback() *mock.Call { return m.On("Rollback") }
func (m *mockTx) Rollback() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockTx) OnCommit() *mock.Call { return m.On("Commit") }
func (m *mockTx) Commit() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockTx) OnHas(key backend.Key) *mock.Call { return m.On("Has", key) }
func (m *mockTx) Has(key backend.Key) (bool, error) {
	args := m.Called(key)
	return args.Bool(0), args.Error(1)
}

func (m *mockTx) OnGet(key backend.Key) *mock.Call { return m.On("Get", key) }
func (m *mockTx) Get(key backend.Key) (backend.ValueDecoder, error) {
	args := m.Called(key)
	return args.Get(0).(backend.ValueDecoder), args.Error(1)
}

func (m *mockTx) Set(key backend.Key, from interface{}) error {
	args := m.Called(key, from)
	return args.Error(0)
}

func (m *mockTx) Update(key backend.Key, fields interface{}) error {
	args := m.Called(key, fields)
	return args.Error(0)
}

func (m *mockTx) Remove(key backend.Key) error {
	args := m.Called(key)
	return args.Error(0)
}

func (m *mockTx) EachKey(internal bool, fn func(backend.Key) (bool, error)) error {
	args := m.Called(internal, fn)
	return args.Error(0)
}

func (m *mockTx) Each(internal bool, fn func(backend.Key, backend.ValueDecoder) (bool, error)) error {
	args := m.Called(internal, fn)
	return args.Error(0)
}
