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

package report

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestSystemMetricsReport(t *testing.T) {
	err := SetupMetrics(logptest.NewTestingLogger(t, ""), "TestSys", "test")
	require.NoError(t, err)

	var gotCPU, gotMem, gotInfo atomic.Bool
	testFunc := func(key string, val interface{}) {
		if key == "info.uptime.ms" {
			gotInfo.Store(true)
		}
		if key == "cpu.total.ticks" {
			gotCPU.Store(true)
		}
		if key == "memstats.rss" {
			gotMem.Store(true)
		}
	}

	//iterate over the processes a few times,
	// with the concurrency (hopefully) emulating what might
	// happen if this was an HTTP endpoint getting multiple GET requests
	iter := 100
	var wait sync.WaitGroup
	wait.Add(iter)
	ch := make(chan struct{})
	for i := 0; i < iter; i++ {
		go func() {
			<-ch
			processMetrics.Do(monitoring.Full, testFunc)
			wait.Done()
		}()
	}
	close(ch)

	wait.Wait()
	assert.True(t, gotCPU.Load(), "Didn't find cpu.total.ticks")
	assert.True(t, gotMem.Load(), "Didn't find memstats.rss")
	assert.True(t, gotInfo.Load(), "Didn't find info.uptime.ms")
}
