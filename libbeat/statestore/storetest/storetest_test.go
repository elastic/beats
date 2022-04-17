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

package storetest

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/libbeat/statestore/backend"
	"github.com/menderesk/beats/v7/libbeat/statestore/internal/storecompliance"
)

func init() {
	logp.DevelopmentSetup()
}

func TestCompliance(t *testing.T) {
	storecompliance.TestBackendCompliance(t, func(testPath string) (backend.Registry, error) {
		return NewMemoryStoreBackend(), nil
	})
}

func TestStore_IsClosed(t *testing.T) {
	t.Run("false by default", func(t *testing.T) {
		store := &MapStore{}
		assert.False(t, store.IsClosed())
	})
	t.Run("true after close", func(t *testing.T) {
		store := &MapStore{}
		store.Close()
		assert.True(t, store.IsClosed())
	})
	t.Run("true after reopen", func(t *testing.T) {
		store := &MapStore{}
		store.Close()
		store.Reopen()
		assert.False(t, store.IsClosed())
	})
}
