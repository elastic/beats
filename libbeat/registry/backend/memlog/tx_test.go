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
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/registry/backend"
)

func TestTx(t *testing.T) {
	t.Run("iter active hash line", func(t *testing.T) {

		store := bin{
			{
				key:   backend.Key("a"),
				value: common.MapStr{"a": 1},
			},
		}
		txCache := txCacheLine{
			{
				key:      backend.Key("b"),
				value:    common.MapStr{"b": 2},
				exists:   true,
				modified: true,
			},
		}

		visited := map[string]bool{}
		err := iterActiveHashLine(
			func(ref cacheEntryRef) error {
				visited[string(ref.Access().key)] = true
				return nil
			},
			func(ref valueRef) error {
				visited[string(ref.Access().key)] = true
				return nil
			},
			1,
			store,
			txCache,
		)
		require.NoError(t, err)

		expected := map[string]bool{"a": true, "b": true}
		assert.Equal(t, expected, visited)
	})
}
