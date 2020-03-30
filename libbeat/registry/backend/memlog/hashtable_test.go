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

package memlog

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/registry/backend"
)

func TestHashtable(t *testing.T) {
	t.Run("find is Nil on empty table", func(t *testing.T) {
		tbl := newHashtable()
		ref := tbl.find(keyPair{1, backend.Key("a")})
		assert.True(t, ref.IsNil())
	})

	t.Run("overwrite key", func(t *testing.T) {
		key := backend.Key("k")
		pos := keyPair{1, key}
		v1 := common.MapStr{"a": 1}
		v2 := common.MapStr{"a": 2}

		tbl := newHashtable()
		tbl.set(pos.hash, key, v1)
		assert.Equal(t, tbl.find(pos).Access().value, v1)

		tbl.set(pos.hash, key, v2)
		assert.Equal(t, tbl.find(pos).Access().value, v2)
	})

	t.Run("bin operations", func(t *testing.T) {
		t.Run("find key", func(t *testing.T) {
			b := bin{
				{
					key:   backend.Key("a"),
					value: common.MapStr{"v": 1},
				},
				{
					key:   backend.Key("b"),
					value: common.MapStr{"v": 2},
				},
			}

			assert.Equal(t, 0, b.index(backend.Key("a")))
			assert.Equal(t, 1, b.index(backend.Key("b")))
			assert.Equal(t, -1, b.index(backend.Key("c")))
		})

		t.Run("remove", func(t *testing.T) {
			b := bin{
				{
					key:   backend.Key("a"),
					value: common.MapStr{"v": 1},
				},
				{
					key:   backend.Key("b"),
					value: common.MapStr{"v": 2},
				},
				{
					key:   backend.Key("c"),
					value: common.MapStr{"v": 2},
				},
			}

			idx := b.index(backend.Key("b"))
			b.remove(idx)

			assert.Equal(t, 2, len(b))
			assert.Equal(t, 0, b.index(backend.Key("a")))
			assert.Equal(t, -1, b.index(backend.Key("b")))
			assert.Equal(t, 1, b.index(backend.Key("c")))
		})
	})
}
