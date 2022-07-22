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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestSystemMetricsReport(t *testing.T) {
	_ = logp.DevelopmentSetup()
	logger := logp.L()
	err := SetupMetrics(logger, "TestSys", "test")
	require.NoError(t, err)

	var gotCPU, gotMem, gotInfo bool
	testFunc := func(key string, val interface{}) {
		if key == "info.uptime.ms" {
			gotInfo = true
		}
		if key == "cpu.total.ticks" {
			gotCPU = true
		}
		if key == "memstats.rss" {
			gotMem = true
		}
	}

	//iterate over the processes a few times,
	// with the concurrency (hopefully) emulating what might
	// happen if this was an HTTP endpoint getting multiple GET requests
	iter := 5
	var wait sync.WaitGroup
	wait.Add(iter)
	for i := 0; i < iter; i++ {
		go func() {
			processMetrics.Do(monitoring.Full, testFunc)
			wait.Done()
		}()
	}

	wait.Wait()
	assert.True(t, gotCPU, "Didn't find cpu.total.ticks")
	assert.True(t, gotMem, "Didn't find memstats.rss")
	assert.True(t, gotInfo, "Didn't find info.uptime.ms")
}
