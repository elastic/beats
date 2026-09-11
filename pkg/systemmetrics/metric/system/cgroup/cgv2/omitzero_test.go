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
	"maps"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/opt"
)

func TestOmitZeroJSON(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected []string
	}{
		{
			name:     "CPUStats/zero struct fields omitted",
			value:    CPUStats{},
			expected: []string{"system", "usage", "user"},
		},
		{
			name: "CPUStats/non-zero struct fields present",
			value: CPUStats{
				Periods:   opt.UintWith(1),
				Throttled: ThrottledField{Us: opt.UintWith(10)},
			},
			expected: []string{"periods", "system", "throttled", "usage", "user"},
		},
		{
			name:     "ThrottledField/zero struct fields omitted",
			value:    ThrottledField{},
			expected: nil,
		},
		{
			name: "ThrottledField/non-zero struct fields present",
			value: ThrottledField{
				Us:      opt.UintWith(10),
				Periods: opt.UintWith(4),
			},
			expected: []string{"periods", "us"},
		},
		{
			name:     "MemoryData/zero BytesOpt omitted",
			value:    MemoryData{},
			expected: []string{"events", "low", "usage"},
		},
		{
			name: "MemoryData/non-zero BytesOpt present",
			value: MemoryData{
				High: opt.BytesOpt{Bytes: opt.UintWith(1024)},
				Max:  opt.BytesOpt{Bytes: opt.UintWith(4096)},
			},
			expected: []string{"events", "high", "low", "max", "usage"},
		},
		{
			name:     "Events/zero opt.Uint omitted",
			value:    Events{},
			expected: []string{"high", "max"},
		},
		{
			name: "Events/non-zero opt.Uint present",
			value: Events{
				Low:     opt.UintWith(1),
				OOM:     opt.UintWith(2),
				OOMKill: opt.UintWith(3),
				Fail:    opt.UintWith(4),
			},
			expected: []string{"fail", "high", "low", "max", "oom", "oom_kill"},
		},
		{
			name:     "CPUSubsystem/zero CFS omitted",
			value:    CPUSubsystem{},
			expected: []string{"stats"},
		},
		{
			name:     "CPUSubsystem/non-zero CFS present",
			value:    CPUSubsystem{CFS: CFS{Weight: opt.UintWith(100)}},
			expected: []string{"cfs", "stats"},
		},
		{
			name:     "CFS/zero UsOpt fields omitted",
			value:    CFS{},
			expected: nil,
		},
		{
			name: "CFS/non-zero UsOpt fields present",
			value: CFS{
				Period: UsOpt{Us: opt.UintWith(100000)},
				Quota:  UsOpt{Us: opt.UintWith(50000)},
				Weight: opt.UintWith(200),
			},
			expected: []string{"period", "quota", "weight"},
		},
		{
			name: "CFS/unlimited quota 0 (max) still present",
			value: CFS{
				Period: UsOpt{Us: opt.UintWith(100000)},
				Quota:  UsOpt{Us: opt.UintWith(0)},
				Weight: opt.UintWith(200),
			},
			expected: []string{"period", "quota", "weight"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, marshalKeys(t, test.value))
		})
	}
}

func marshalKeys(t *testing.T, v any) []string {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	var m map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &m))
	return slices.Sorted(maps.Keys(m))
}
