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

package memory

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/opt"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
)

func TestGetMemory(t *testing.T) {
	mem, err := Get(resolve.NewTestResolver(""))

	assert.NotNil(t, mem)
	assert.NoError(t, err)

	assert.Greater(t, mem.Total.ValueOr(0), uint64(0))
	assert.Greater(t, mem.Used.Bytes.ValueOr(0), uint64(0))
	assert.True(t, mem.Total.Exists())
	assert.True(t, mem.Actual.Free.Exists())
	assert.Greater(t, mem.Actual.Used.Bytes.ValueOr(0), uint64(0))
}

func TestGetSwap(t *testing.T) {
	if runtime.GOOS == "freebsd" {
		t.Skip("Skip freebsd")
	}

	mem, err := Get(resolve.NewTestResolver(""))

	assert.NotNil(t, mem)
	assert.NoError(t, err)

	assert.True(t, mem.Swap.Total.Exists())
	assert.True(t, mem.Swap.Used.Bytes.Exists())
	assert.True(t, mem.Swap.Free.Exists())
}

func TestMemPercentage(t *testing.T) {
	m := Memory{
		Total: opt.UintWith(7),
		Used:  UsedMemStats{Bytes: opt.UintWith(5)},
		Free:  opt.UintWith(2),
	}
	m.fillPercentages()
	assert.InDelta(t, 0.7143, m.Used.Pct.ValueOr(0), 0.0001)

	m = Memory{
		Total: opt.UintWith(0),
	}
	m.fillPercentages()
	assert.Equal(t, 0.0, m.Used.Pct.ValueOr(0))
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
	assert.InDelta(t, 0.7143, m.Actual.Used.Pct.ValueOr(0), 0.0001)
}

func TestMeminfoParse(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux specific test")
	}

	mem, err := Get(resolve.NewTestResolver("./testdata/oldkern"))
	assert.NoError(t, err)

	expected := Memory{
		Total:  opt.UintWith(63120797696),
		Free:   opt.UintWith(25673170944),
		Cached: opt.UintWith(27307106304),
		Used: UsedMemStats{
			Bytes: opt.UintWith(37447626752),
			Pct:   opt.FloatWith(0.5933),
		},
		Actual: ActualMemoryMetrics{
			Free: opt.UintWith(52983070720),
			Used: UsedMemStats{
				Bytes: opt.UintWith(10137726976),
				Pct:   opt.FloatWith(0.1606),
			},
		},
		Swap: SwapMetrics{
			Total: opt.UintWith(8589930496),
			Free:  opt.UintWith(8588095488),
			Used: UsedMemStats{
				Bytes: opt.UintWith(1835008),
				Pct:   opt.FloatWith(0.0002),
			},
		},
		Zswap: ZswapMetrics{
			Compressed:   opt.UintWith(3023044608),
			Uncompressed: opt.UintWith(4439736320),
			Debug: ZswapDebugMetrics{
				PoolLimitHit:        opt.NewUintNone(),
				PoolTotalSize:       opt.NewUintNone(),
				RejectAllocFail:     opt.NewUintNone(),
				RejectCompressFail:  opt.NewUintNone(),
				RejectCompressPoor:  opt.NewUintNone(),
				RejectKmemcacheFail: opt.NewUintNone(),
				RejectReclaimFail:   opt.NewUintNone(),
				StoredPages:         opt.NewUintNone(),
				WrittenBackPages:    opt.NewUintNone(),
			},
		},
	}
	assert.Equal(t, expected, mem)
}

func TestZswapMetricsIsZero(t *testing.T) {
	z := ZswapMetrics{}
	assert.True(t, z.IsZero())

	z.Compressed = opt.UintWith(0)
	assert.False(t, z.IsZero())

	z = ZswapMetrics{Uncompressed: opt.UintWith(200)}
	assert.False(t, z.IsZero())

	// Test with nested debug metrics
	z = ZswapMetrics{}
	z.Debug.StoredPages = opt.UintWith(50)
	assert.False(t, z.IsZero())
}
