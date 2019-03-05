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

// +build !integration
// +build darwin freebsd linux openbsd windows

package memory

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/gosigar"
)

func TestGetMemory(t *testing.T) {
	mem, err := Get()

	assert.NotNil(t, mem)
	assert.Nil(t, err)

	assert.True(t, (mem.Total > 0))
	assert.True(t, (mem.Used > 0))
	assert.True(t, (mem.Free >= 0))
	assert.True(t, (mem.ActualFree >= 0))
	assert.True(t, (mem.ActualUsed > 0))
}

func TestGetSwap(t *testing.T) {
	if runtime.GOOS == "windows" {
		return //no load data on windows
	}

	swap, err := GetSwap()

	assert.NotNil(t, swap)
	assert.Nil(t, err)

	assert.True(t, (swap.Total >= 0))
	assert.True(t, (swap.Used >= 0))
	assert.True(t, (swap.Free >= 0))
}

func TestMemPercentage(t *testing.T) {
	m := MemStat{
		Mem: gosigar.Mem{
			Total: 7,
			Used:  5,
			Free:  2,
		},
	}
	AddMemPercentage(&m)
	assert.Equal(t, m.UsedPercent, 0.7143)

	m = MemStat{
		Mem: gosigar.Mem{Total: 0},
	}
	AddMemPercentage(&m)
	assert.Equal(t, m.UsedPercent, 0.0)
}

func TestActualMemPercentage(t *testing.T) {
	m := MemStat{
		Mem: gosigar.Mem{
			Total:      7,
			ActualUsed: 5,
			ActualFree: 2,
		},
	}
	AddMemPercentage(&m)
	assert.Equal(t, m.ActualUsedPercent, 0.7143)

	m = MemStat{
		Mem: gosigar.Mem{
			Total: 0,
		},
	}
	AddMemPercentage(&m)
	assert.Equal(t, m.ActualUsedPercent, 0.0)
}
