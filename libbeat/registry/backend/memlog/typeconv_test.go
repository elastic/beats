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

	"github.com/elastic/beats/libbeat/common"
)

func TestTypeConv(t *testing.T) {
	t.Run("init", withTypeConv(func(t *testing.T, tc *typeConv) {
	}))

	t.Run("from MapStr", withTypeConv(func(t *testing.T, tc *typeConv) {
		type testStruct struct {
			A int
			B int
		}

		var v testStruct
		tc.Convert(&v, &common.MapStr{"a": 1})
		assert.Equal(t, testStruct{1, 0}, v)
	}))

	t.Run("to MapStr", withTypeConv(func(t *testing.T, tc *typeConv) {
		var m common.MapStr
		tc.Convert(&m, struct{ A string }{"test"})
		assert.Equal(t, common.MapStr{"a": "test"}, m)
	}))
}

func withTypeConv(fn func(t *testing.T, tc *typeConv)) func(*testing.T) {
	return func(t *testing.T) {
		tc := newTypeConv()
		defer tc.release()
		require.NotNil(t, tc)
		fn(t, tc)
	}
}
