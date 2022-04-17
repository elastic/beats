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

package meta

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/libbeat/common"
)

func TestStoreNil(t *testing.T) {
	m := NewMap()
	assert.Equal(t, common.MapStrPointer{}, m.Store(0, nil))
}

func TestStore(t *testing.T) {
	m := NewMap()

	// Store meta
	res := m.Store(0, common.MapStr{"foo": "bar"})
	assert.Equal(t, res.Get(), common.MapStr{"foo": "bar"})

	// Update it
	res = m.Store(0, common.MapStr{"foo": "baz"})
	assert.Equal(t, res.Get(), common.MapStr{"foo": "baz"})

	m.Remove(0)
	assert.Equal(t, len(m.meta), 0)
}
