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

package cgv1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/opt"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/cgroup/cgcommon"
)

func TestOmitZeroJSON(t *testing.T) {
	t.Run("CPUSubsystem/zero struct fields omitted", func(t *testing.T) {
		m := marshalToMap(t, CPUSubsystem{})
		assert.NotContains(t, m, "cfs")
		assert.NotContains(t, m, "rt")
		assert.NotContains(t, m, "stats")
	})

	t.Run("CPUSubsystem/non-zero struct fields present", func(t *testing.T) {
		m := marshalToMap(t, CPUSubsystem{
			CFS:   CFS{Shares: 1024},
			RT:    RT{Period: opt.Us{Us: 1000}},
			Stats: CPUStats{Periods: 10},
		})
		assert.Contains(t, m, "cfs")
		assert.Contains(t, m, "rt")
		assert.Contains(t, m, "stats")
	})

	t.Run("BlockIOSubsystem/zero struct fields omitted", func(t *testing.T) {
		m := marshalToMap(t, BlockIOSubsystem{})
		assert.NotContains(t, m, "total")
		assert.NotContains(t, m, "reads")
		assert.NotContains(t, m, "writes")
	})

	t.Run("BlockIOSubsystem/non-zero struct fields present", func(t *testing.T) {
		m := marshalToMap(t, BlockIOSubsystem{
			Total:  TotalIOs{Bytes: 100},
			Reads:  TotalIOs{Ios: 5},
			Writes: TotalIOs{Bytes: 50},
		})
		assert.Contains(t, m, "total")
		assert.Contains(t, m, "reads")
		assert.Contains(t, m, "writes")
	})

	t.Run("CPUAccountingSubsystem/zero Stats omitted", func(t *testing.T) {
		m := marshalToMap(t, CPUAccountingSubsystem{})
		assert.NotContains(t, m, "stats")
	})

	t.Run("CPUAccountingSubsystem/non-zero Stats present", func(t *testing.T) {
		m := marshalToMap(t, CPUAccountingSubsystem{
			Stats: CPUAccountingStats{
				User: cgcommon.CPUUsage{NS: 100},
			},
		})
		assert.Contains(t, m, "stats")
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
