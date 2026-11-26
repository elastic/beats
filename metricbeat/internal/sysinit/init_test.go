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

package sysinit_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/management"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	_ "github.com/elastic/beats/v7/metricbeat/module/system"
	_ "github.com/elastic/beats/v7/metricbeat/module/system/uptime"
)

func TestHostFSDoesNotChangeEnvironment(t *testing.T) {
	for _, underAgent := range []bool{false, true} {
		management.SetUnderAgent(underAgent)
		t.Cleanup(func() { management.SetUnderAgent(false) })

		envKeys := []string{"HOST_PROC", "HOST_SYS", "HOST_ETC"}
		for _, key := range envKeys {
			t.Setenv(key, "/original")
		}

		mbtest.NewReportingMetricSetV2Error(t, map[string]any{
			"module":     "system",
			"metricsets": []string{"uptime"},
			"hostfs":     "/hostfs",
		})

		for _, key := range envKeys {
			assert.Equal(t, "/original", os.Getenv(key), "%s must not be changed by hostfs (under agent: %v)", key, underAgent)
		}
	}
}
