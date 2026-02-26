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

package cgv2

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/opt"
)

func TestOmitZeroJSON(t *testing.T) {
	t.Run("CPUStats/zero struct fields omitted", func(t *testing.T) {
		m := marshalToMap(t, CPUStats{})
		assert.NotContains(t, m, "throttled")
		assert.NotContains(t, m, "periods")
	})

	t.Run("CPUStats/non-zero struct fields present", func(t *testing.T) {
		m := marshalToMap(t, CPUStats{
			Periods:   opt.UintWith(1),
			Throttled: ThrottledField{Us: opt.UintWith(10)},
		})
		assert.Contains(t, m, "throttled")
		assert.Contains(t, m, "periods")
	})

	t.Run("ThrottledField/zero struct fields omitted", func(t *testing.T) {
		m := marshalToMap(t, ThrottledField{})
		assert.NotContains(t, m, "us")
		assert.NotContains(t, m, "periods")
	})

	t.Run("ThrottledField/non-zero struct fields present", func(t *testing.T) {
		m := marshalToMap(t, ThrottledField{
			Us:      opt.UintWith(10),
			Periods: opt.UintWith(4),
		})
		assert.Contains(t, m, "us")
		assert.Contains(t, m, "periods")
	})

	t.Run("MemoryData/zero BytesOpt omitted", func(t *testing.T) {
		m := marshalToMap(t, MemoryData{})
		assert.NotContains(t, m, "high")
		assert.NotContains(t, m, "max")
	})

	t.Run("MemoryData/non-zero BytesOpt present", func(t *testing.T) {
		m := marshalToMap(t, MemoryData{
			High: opt.BytesOpt{Bytes: opt.UintWith(1024)},
			Max:  opt.BytesOpt{Bytes: opt.UintWith(4096)},
		})
		assert.Contains(t, m, "high")
		assert.Contains(t, m, "max")
	})

	t.Run("Events/zero opt.Uint omitted", func(t *testing.T) {
		m := marshalToMap(t, Events{})
		assert.NotContains(t, m, "low")
		assert.NotContains(t, m, "oom")
		assert.NotContains(t, m, "oom_kill")
		assert.NotContains(t, m, "fail")
	})

	t.Run("Events/non-zero opt.Uint present", func(t *testing.T) {
		m := marshalToMap(t, Events{
			Low:     opt.UintWith(1),
			OOM:     opt.UintWith(2),
			OOMKill: opt.UintWith(3),
			Fail:    opt.UintWith(4),
		})
		assert.Contains(t, m, "low")
		assert.Contains(t, m, "oom")
		assert.Contains(t, m, "oom_kill")
		assert.Contains(t, m, "fail")
	})
}

func marshalToMap(t *testing.T, v any) map[string]json.RawMessage {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	var m map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &m))
	return m
}
