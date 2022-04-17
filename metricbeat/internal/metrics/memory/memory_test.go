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

//go:build !integration && (darwin || freebsd || linux || openbsd || windows)
// +build !integration
// +build darwin freebsd linux openbsd windows

package memory

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/libbeat/metric/system/resolve"
	"github.com/menderesk/beats/v7/libbeat/opt"
)

func TestGetMemory(t *testing.T) {
	mem, err := Get(resolve.NewTestResolver(""))

	assert.NotNil(t, mem)
	assert.NoError(t, err)

	assert.True(t, mem.Total.Exists())
	assert.True(t, (mem.Total.ValueOr(0) > 0))

	assert.True(t, mem.Used.Bytes.Exists())
	assert.True(t, (mem.Used.Bytes.ValueOr(0) > 0))

	assert.True(t, mem.Free.Exists())
	assert.True(t, (mem.Free.ValueOr(0) >= 0))

	assert.True(t, mem.Actual.Free.Exists())
	assert.True(t, (mem.Actual.Free.ValueOr(0) >= 0))

	assert.True(t, mem.Actual.Used.Bytes.Exists())
	assert.True(t, (mem.Actual.Used.Bytes.ValueOr(0) > 0))
}

func TestGetSwap(t *testing.T) {
	if runtime.GOOS == "freebsd" {
		return //no load data on freebsd
	}

	mem, err := Get(resolve.NewTestResolver(""))

	assert.NotNil(t, mem)
	assert.NoError(t, err)

	assert.True(t, mem.Swap.Total.Exists())
	assert.True(t, (mem.Swap.Total.ValueOr(0) >= 0))

	assert.True(t, mem.Swap.Used.Bytes.Exists())
	assert.True(t, (mem.Swap.Used.Bytes.ValueOr(0) >= 0))

	assert.True(t, mem.Swap.Free.Exists())
	assert.True(t, (mem.Swap.Free.ValueOr(0) >= 0))
}

func TestMemPercentage(t *testing.T) {
	m := Memory{
		Total: opt.UintWith(7),
		Used:  UsedMemStats{Bytes: opt.UintWith(5)},
		Free:  opt.UintWith(2),
	}
	m.fillPercentages()
	assert.Equal(t, m.Used.Pct.ValueOr(0), 0.7143)

	m = Memory{
		Total: opt.UintWith(0),
	}
	m.fillPercentages()
	assert.Equal(t, m.Used.Pct.ValueOr(0), 0.0)
}

func TestActualMemPercentage(t *testing.T) {
	m := Memory{
		Total: opt.UintWith(7),
		Actual: ActualMemoryMetrics{
			Used: UsedMemStats{Bytes: opt.UintWith(5)},
			Free: opt.UintWith(2),
		},
	}

	m.fillPercentages()
	assert.Equal(t, m.Actual.Used.Pct.ValueOr(0), 0.7143)

}

func TestMeminfoParse(t *testing.T) {
	// Make sure we're manually calculating Actual correctly on linux
	if runtime.GOOS == "linux" {
		mem, err := Get(resolve.NewTestResolver("./oldkern"))
		assert.NoError(t, err)

		assert.Equal(t, uint64(27307106304), mem.Cached.ValueOr(0))
		assert.Equal(t, uint64(52983070720), mem.Actual.Free.ValueOr(0))
		assert.Equal(t, uint64(10137726976), mem.Actual.Used.Bytes.ValueOr(0))
	}
}

func TestMeminfoPct(t *testing.T) {
	if runtime.GOOS == "linux" {
		memRaw, err := Get(resolve.NewTestResolver("./oldkern"))
		assert.NoError(t, err)
		assert.Equal(t, float64(0.1606), memRaw.Actual.Used.Pct.ValueOr(0))
		assert.Equal(t, float64(0.5933), memRaw.Used.Pct.ValueOr(0))
	}
}
