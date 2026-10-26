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

package cgcommon

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/opt"
)

func TestOmitZeroJSON(t *testing.T) {
	t.Run("CPUUsage/zero struct fields omitted", func(t *testing.T) {
		m := marshalToMap(t, CPUUsage{})
		assert.NotContains(t, m, "pct")
		assert.NotContains(t, m, "norm")
	})

	t.Run("CPUUsage/non-zero struct fields present", func(t *testing.T) {
		m := marshalToMap(t, CPUUsage{
			Pct:  opt.FloatWith(0.5),
			Norm: opt.PctOpt{Pct: opt.FloatWith(0.3)},
		})
		assert.Contains(t, m, "pct")
		assert.Contains(t, m, "norm")
	})

	t.Run("Pressure/zero struct fields omitted", func(t *testing.T) {
		m := marshalToMap(t, Pressure{})
		assert.NotContains(t, m, "10")
		assert.NotContains(t, m, "60")
		assert.NotContains(t, m, "300")
		assert.NotContains(t, m, "total")
	})

	t.Run("Pressure/non-zero struct fields present", func(t *testing.T) {
		m := marshalToMap(t, Pressure{
			Ten:          opt.Pct{Pct: 1.5},
			Sixty:        opt.Pct{Pct: 2.3},
			ThreeHundred: opt.Pct{Pct: 0.75},
			Total:        opt.UintWith(123456),
		})
		assert.Contains(t, m, "10")
		assert.Contains(t, m, "60")
		assert.Contains(t, m, "300")
		assert.Contains(t, m, "total")
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
