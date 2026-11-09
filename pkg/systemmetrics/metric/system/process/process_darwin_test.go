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

//go:build darwin && cgo

package process

import (
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetInfoForPid_cpuTicks(t *testing.T) {
	burnCPU(300 * time.Millisecond)

	state, err := GetInfoForPid(nil, os.Getpid())
	require.NoError(t, err, "GetInfoForPid failed for the test process")

	var rusage syscall.Rusage
	require.NoError(t, syscall.Getrusage(syscall.RUSAGE_SELF, &rusage), "getrusage failed")
	wantMs := float64(time.Duration(rusage.Utime.Nano()+rusage.Stime.Nano()) / time.Millisecond)

	gotMs := float64(state.CPU.Total.Ticks.ValueOr(0))
	assert.InDeltaf(t, wantMs, gotMs, max(50, wantMs*0.2),
		"cpu.total.ticks should match the CPU time reported by getrusage; ticks=%v getrusage=%vms", gotMs, wantMs)
}

func burnCPU(d time.Duration) {
	deadline := time.Now().Add(d)
	for x := 0; time.Now().Before(deadline); x++ {
		_ = x * x
	}
}
