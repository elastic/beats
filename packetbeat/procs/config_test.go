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

//go:build !integration

package procs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestProcsConfigValidate(t *testing.T) {
	for _, backend := range []string{"", BackendProcfs, BackendKernelTracing, BackendAuto} {
		cfg := ProcsConfig{Backend: backend}
		assert.NoError(t, cfg.Validate(), "backend %q must be valid", backend)
	}

	cfg := ProcsConfig{Backend: "ptrace"}
	assert.ErrorContains(t, cfg.Validate(), `invalid procs backend "ptrace"`)

	// Validate is invoked by Unpack.
	c := conf.MustNewConfigFrom(map[string]any{"enabled": true, "backend": "nonsense"})
	var unpacked ProcsConfig
	assert.ErrorContains(t, c.Unpack(&unpacked), "invalid procs backend")
}

func TestGraceTimeForFlows(t *testing.T) {
	// Defaults: 30s timeout + 10s period + margin.
	assert.Equal(t, 50*time.Second, GraceTimeForFlows(30*time.Second, 10*time.Second))
	// A negative period disables intermediate reports and must not
	// shorten the grace.
	assert.Equal(t, 40*time.Second, GraceTimeForFlows(30*time.Second, -1))
	// KernelTracingGraceTime is computed, not user-configurable.
	c := conf.MustNewConfigFrom(map[string]any{
		"enabled":                   true,
		"kernel_tracing_grace_time": "5m",
		"kerneltracinggracetime":    "5m",
	})
	var unpacked ProcsConfig
	require.NoError(t, c.Unpack(&unpacked))
	assert.Zero(t, unpacked.KernelTracingGraceTime)
}

func TestInitBackend(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "procs")
	mock := newMockWatcher(nil, nil)

	t.Run("auto never fails", func(t *testing.T) {
		proc := &ProcessesWatcher{}
		err := proc.init(ProcsConfig{Enabled: true, Backend: BackendAuto}, mock, logger)
		require.NoError(t, err)
		proc.Close()
	})

	t.Run("kernel_tracing fails hard when unavailable", func(t *testing.T) {
		proc := &ProcessesWatcher{}
		err := proc.init(ProcsConfig{Enabled: true, Backend: BackendKernelTracing}, mock, logger)
		if err == nil {
			// Quark could start: Linux with eBPF and enough privileges.
			require.NotNil(t, proc.kernelTracing)
			proc.Close()
			return
		}
		assert.ErrorContains(t, err, "kernel_tracing")
		assert.Nil(t, proc.kernelTracing)
	})

	t.Run("kernel_tracing is not started when disabled", func(t *testing.T) {
		proc := &ProcessesWatcher{}
		err := proc.init(ProcsConfig{Enabled: false, Backend: BackendKernelTracing}, mock, logger)
		require.NoError(t, err)
		assert.Nil(t, proc.kernelTracing)
		proc.Close()
	})
}
