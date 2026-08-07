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

//go:build integration

// Tests for the stdin input, ported from test_stdin.py.

package integration

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestStdinIsExclusive verifies that filebeat exits with code 1 when stdin is
// configured alongside another input type.
func TestStdinIsExclusive(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	// The UDP input is never started — filebeat rejects the config before
	// binding any socket — but we still use AllocatePort to avoid a hardcoded
	// port that could collide if the validation order changes in the future.
	port := AllocatePort(t)
	config := fmt.Sprintf(`filebeat.inputs:
  - type: stdin
    enabled: true
  - type: udp
    host: "127.0.0.1:%d"
    enabled: true

output.console:
  enabled: true
`, port)
	test := NewTest(t, TestOptions{Config: config})
	test.
		ExpectOutput("stdin requires to be run in exclusive mode").
		ExpectStop(1).
		WithReportOptions(networkReportOptions).
		Start(ctx).
		Wait()
}

// TestStdin verifies that filebeat continues reading from stdin after the first
// read and correctly ingests events written in two batches.
func TestStdin(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	config := StdinInputConfig()
	test := NewTest(t, TestOptions{Config: config, StdinEnabled: true})

	var readyCount atomic.Int64
	test.CountOutput(&readyCount, "Harvester started")

	var eventCount atomic.Int64
	test.CountOutput(&eventCount, `"message":"Hello World"`)

	test.ExpectStop(exitSignalKilled)
	test.WithReportOptions(networkReportOptions).ExpectStart().Start(ctx)

	require.Eventually(t, func() bool { return readyCount.Load() >= 1 }, 30*time.Second, 50*time.Millisecond,
		"stdin harvester did not start")

	// First batch: 5 lines.
	for range 5 {
		_, err := test.Stdin().Write([]byte("Hello World\n"))
		require.NoError(t, err)
	}
	require.Eventually(t, func() bool { return eventCount.Load() >= 5 }, 30*time.Second, 50*time.Millisecond,
		"did not receive 5 events from first batch")

	// Second batch: 10 lines.
	for range 10 {
		_, err := test.Stdin().Write([]byte("Hello World\n"))
		require.NoError(t, err)
	}
	require.Eventually(t, func() bool { return eventCount.Load() >= 15 }, 30*time.Second, 50*time.Millisecond,
		"did not receive 15 total events after second batch")

	cancel()
	test.Wait()
}

// TestStdinEOF verifies that filebeat reads data written both before and after
// reaching EOF when close_eof is enabled.
func TestStdinEOF(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	config := `filebeat.inputs:
  - type: stdin
    enabled: true
    close_eof: true

output.console:
  enabled: true
`
	test := NewTest(t, TestOptions{Config: config, StdinEnabled: true})

	var helloCount atomic.Int64
	test.CountOutput(&helloCount, `"message":"Hello World"`)

	var hello2Count atomic.Int64
	test.CountOutput(&hello2Count, `"message":"Hello World2"`)

	test.ExpectStop(exitSignalKilled)
	test.WithReportOptions(networkReportOptions).ExpectStart().Start(ctx)

	_, err := test.Stdin().Write([]byte("Hello World\n"))
	require.NoError(t, err)
	require.Eventually(t, func() bool { return helloCount.Load() >= 1 }, 30*time.Second, 50*time.Millisecond,
		"did not receive first event")

	_, err = test.Stdin().Write([]byte("Hello World2\n"))
	require.NoError(t, err)
	require.NoError(t, test.Stdin().Close())

	require.Eventually(t, func() bool { return hello2Count.Load() >= 1 }, 30*time.Second, 50*time.Millisecond,
		"did not receive second event")

	cancel()
	test.Wait()
}
