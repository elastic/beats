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

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestConversionWithMapStr(t *testing.T) {
	t.Run("from mapstr.M", func(t *testing.T) {
		type testStruct struct {
			A int
			B int
		}

		var v testStruct
		Convert(&v, &mapstr.M{"a": 1})
		assert.Equal(t, testStruct{1, 0}, v)
	})

	t.Run("to mapstr.M", func(t *testing.T) {
		var m mapstr.M
		err := Convert(&m, struct{ A string }{"test"})
		require.NoError(t, err)
		assert.Equal(t, mapstr.M{"a": "test"}, m)
	})
}

func TestConversionBetweenGoTypes(t *testing.T) {
	t.Run("int to uint", func(t *testing.T) {
		var i = 42
		var u uint
		err := Convert(&u, i)
		require.NoError(t, err)
		assert.Equal(t, uint(42), u)
	})

	t.Run("between structs", func(t *testing.T) {
		type To struct {
			A uint
			F float64
			B string
		}

		input := struct {
			A      int
			F      float64
			B      string
			Ignore uint
		}{100, 3.14, "test", 42}

		var actual To
		err := Convert(&actual, input)
		require.NoError(t, err)

		want := To{100, 3.14, "test"}
		assert.Equal(t, want, actual)
	})

	t.Run("string is parsed to int", func(t *testing.T) {
		var to int
		require.NoError(t, Convert(&to, 1))
		assert.Equal(t, 1, to)
	})
}

func TestTimestamps(t *testing.T) {
	t.Run("timestamp to mapstr.M", func(t *testing.T) {
		var m mapstr.M
		ts := time.Unix(1234, 5678).UTC()

		off := int16(-1)
		expected := []uint64{uint64(5678) | uint64(uint16(off))<<32, 1234}

		err := Convert(&m, struct{ Timestamp time.Time }{ts})
		require.NoError(t, err)
		assert.Equal(t, mapstr.M{"timestamp": expected}, m)
	})

	t.Run("timestamp from encoded mapstr.M", func(t *testing.T) {
		type testStruct struct {
			Timestamp time.Time
		}

		var v testStruct
		off := int16(-1)
		err := Convert(&v, mapstr.M{
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
		err := Convert(&v, mapstr.M{
			"timestamp": ts.Format(time.RFC3339Nano),
		})
		require.NoError(t, err)
		assert.Equal(t, v.Timestamp.Format(time.RFC3339Nano), ts.Format(time.RFC3339Nano))
	})
}

func TestComplexExampleWithIntermediateConversion(t *testing.T) {
	type checkpoint struct {
		Version            int
		Position           string
		RealtimeTimestamp  uint64
		MonotonicTimestamp uint64
	}

	type (
		stateInternal struct {
			TTL     time.Duration
			Updated time.Time
		}

		state struct {
			Internal stateInternal
			Cursor   interface{}
		}
	)

	input := mapstr.M{
		"_key": "test",
		"internal": mapstr.M{
			"ttl":     float64(1800000000000),
			"updated": []interface{}{float64(515579904576), float64(1588432943)},
		},
		"cursor": mapstr.M{
			"monotonictimestamp": float64(24881645756),
			"position":           "s=86a99d3589f54f01804e844bebd787d5;i=4d19f;b=9c5d2b320b7946b4be53c0940a5b1289;m=5cb0fc8bc;t=5a488aeaa1130;x=ccbe23f507e8d0a4",
			"realtimetimestamp":  float64(1588281836441904),
			"version":            float64(1),
		},
	}

	var st state
	if err := Convert(&st, input); err != nil {
		t.Fatalf("failed to unpack checkpoint: %+v", err)
	}
	assert.Equal(t, time.Duration(1800000000000), st.Internal.TTL)
	assert.Equal(t, testDecodeTimestamp(t, 515579904576, 1588432943), st.Internal.Updated)
	require.True(t, st.Cursor != nil)

	var cp checkpoint
	if err := Convert(&cp, st.Cursor); err != nil {
		t.Fatalf("failed to unpack cursor: %+v", err)
	}

	assert.Equal(t, 1, cp.Version)
	assert.Equal(t, "s=86a99d3589f54f01804e844bebd787d5;i=4d19f;b=9c5d2b320b7946b4be53c0940a5b1289;m=5cb0fc8bc;t=5a488aeaa1130;x=ccbe23f507e8d0a4", cp.Position)
	assert.Equal(t, uint64(1588281836441904), cp.RealtimeTimestamp)
	assert.Equal(t, uint64(24881645756), cp.MonotonicTimestamp)
}

func TestFailOnIncompatibleTypes(t *testing.T) {
	t.Run("primitive value to struct fails", func(t *testing.T) {
		var to struct{ A int }
		require.Error(t, Convert(&to, 1))
	})

}

func testDecodeTimestamp(t *testing.T, a, b uint64) time.Time {
	ts, err := bitsToTimestamp(a, b)
	if err != nil {
		t.Fatal(err)
	}
	return ts
}
