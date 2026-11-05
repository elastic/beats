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

//go:build (darwin && cgo) || (freebsd && cgo) || linux || windows

package report

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestSetupInfoMetricsOnlyRegistersInfo(t *testing.T) {
	process := monitoring.NewRegistry()

	SetupInfoMetrics(MetricOptions{
		Name:           "a-name-longer-than-fifteen-characters",
		Version:        "1.2.3",
		EphemeralID:    "eid",
		ProcessMetrics: process,
	})

	snap := monitoring.CollectFlatSnapshot(process, monitoring.Full, false)
	assert.Equal(t, "1.2.3", snap.Strings["info.version"], "info.version")
	assert.Equal(t, "eid", snap.Strings["info.ephemeral_id"], "info.ephemeral_id")
	assert.Contains(t, snap.Ints, "info.uptime.ms", "info.uptime.ms")
	assert.Equal(t, processName("a-name-longer-than-fifteen-characters"), snap.Strings["info.name"], "info.name is truncated like SetupMetricsOptions does")
	for key := range snap.Ints {
		assert.Equal(t, "info.uptime.ms", key, "SetupInfoMetrics must not register system or process metrics")
	}
	for key := range snap.Floats {
		assert.Fail(t, "unexpected float metric", key)
	}
}
