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

package typeconv

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/libbeat/common"
)

func TestConverter(t *testing.T) {
	t.Run("from MapStr", func(t *testing.T) {
		type testStruct struct {
			A int
			B int
		}

		var v testStruct
		Convert(&v, &common.MapStr{"a": 1})
		assert.Equal(t, testStruct{1, 0}, v)
	})

	t.Run("to MapStr", func(t *testing.T) {
		var m common.MapStr
		err := Convert(&m, struct{ A string }{"test"})
		require.NoError(t, err)
		assert.Equal(t, common.MapStr{"a": "test"}, m)
	})

	t.Run("timestamp to MapStr", func(t *testing.T) {
		var m common.MapStr
		ts := time.Unix(1234, 5678).UTC()

		off := int16(-1)
		expected := []uint64{uint64(5678) | uint64(uint16(off))<<32, 1234}

		err := Convert(&m, struct{ Timestamp time.Time }{ts})
		require.NoError(t, err)
		assert.Equal(t, common.MapStr{"timestamp": expected}, m)
	})

	t.Run("timestamp from encoded MapStr", func(t *testing.T) {
		type testStruct struct {
			Timestamp time.Time
		}

		var v testStruct
		off := int16(-1)
		err := Convert(&v, common.MapStr{
			"timestamp": []uint64{5678 | (uint64(uint16(off)))<<32, 1234},
		})
		require.NoError(t, err)
		expected := time.Unix(1234, 5678).UTC()
		assert.Equal(t, testStruct{expected}, v)
	})

	t.Run("timestamp from string", func(t *testing.T) {
		type testStruct struct {
			Timestamp time.Time
		}

		var v testStruct
		ts := time.Now()
		err := Convert(&v, common.MapStr{
			"timestamp": ts.Format(time.RFC3339Nano),
		})
		require.NoError(t, err)
		assert.Equal(t, v.Timestamp.Format(time.RFC3339Nano), ts.Format(time.RFC3339Nano))
	})
}
