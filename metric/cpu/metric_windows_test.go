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

//go:build windows

package cpu

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-system-metrics/dev-tools/systemtests"
)

func TestCounterLength(t *testing.T) {
	monitor, err := New(systemtests.DockerTestResolver(logptest.NewTestingLogger(t, "")), WithWindowsPerformanceCounter())
	require.NoError(t, err)
	require.NoError(t, monitor.query.CollectData())

	query := monitor.query
	kernelRawData, err := query.GetRawCounterArray(totalKernelTimeCounter, true)
	require.NoError(t, err)

	idleRawData, err := query.GetRawCounterArray(totalIdleTimeCounter, true)
	require.NoError(t, err)

	userRawData, err := query.GetRawCounterArray(totalUserTimeCounter, true)
	require.NoError(t, err)

	require.Equal(t, len(kernelRawData), len(idleRawData))
	require.Equal(t, len(userRawData), len(idleRawData))

	for i := range userRawData {
		require.Equal(t, userRawData[i].InstanceName, kernelRawData[i].InstanceName, "InstanceName should be equal")
	}
	for i := range kernelRawData {
		require.Equal(t, kernelRawData[i].InstanceName, idleRawData[i].InstanceName, "InstanceName should be equal")
	}
}

func TestCounterDisabled(t *testing.T) {
	monitor, err := New(systemtests.DockerTestResolver(logptest.NewTestingLogger(t, "")))
	require.NoError(t, err)
	require.Nil(t, monitor.query)
}
