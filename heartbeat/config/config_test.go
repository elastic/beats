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

package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestDefaults(t *testing.T) {
	cases := []struct {
		Name      string
		EnvKey    string
		EnvVal    string
		LimitType string
		LimitVal  int64
	}{
		{
			"Browser monitor override",
			"SYNTHETICS_LIMIT_BROWSER",
			"123",
			"browser",
			123,
		},
		{
			"Browser default is 2 when other monitor is overridden",
			"SYNTHETICS_LIMIT_HTTP",
			"123",
			"browser",
			2,
		},
		{
			"Browser default is 2 when nothing is overridden",
			"FOO",
			"bar",
			"browser",
			2,
		},
		{
			"Browser default is 2 when bad value passed",
			"SYNTHETICS_LIMIT_BROWSER",
			"bar",
			"browser",
			2,
		},
		{
			"HTTP monitor override",
			"SYNTHETICS_LIMIT_HTTP",
			"456",
			"http",
			456,
		},
		{
			"TCP monitor override",
			"SYNTHETICS_LIMIT_TCP",
			"789",
			"tcp",
			789,
		},
		{
			"ICMP monitor override",
			"SYNTHETICS_LIMIT_ICMP",
			"911",
			"icmp",
			911,
		},
		{
			"API default is 4 when nothing is overridden",
			"FOO",
			"bar",
			"api",
			4,
		},
		{
			"API monitor override",
			"SYNTHETICS_LIMIT_API",
			"321",
			"api",
			321,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			os.Setenv(c.EnvKey, c.EnvVal)
			defer os.Unsetenv(c.EnvKey)

			dc := DefaultConfig(logptest.NewTestingLogger(t, ""))
			require.NotNil(t, dc.Jobs[c.LimitType])
			assert.Equal(t, dc.Jobs[c.LimitType].Limit, c.LimitVal)
		})
	}
}

func TestSupportedJobTypes(t *testing.T) {
	assert.Equal(t, []string{"http", "tcp", "icmp", "browser", "api"}, SupportedJobTypes(),
		"the supported monitor types are the single source of truth for limits and telemetry")

	SupportedJobTypes()[0] = "mutated"
	assert.Equal(t, []string{"http", "tcp", "icmp", "browser", "api"}, SupportedJobTypes(),
		"callers must not be able to mutate the supported monitor types")
}

func TestIsSupportedJobType(t *testing.T) {
	for _, jobType := range SupportedJobTypes() {
		assert.True(t, IsSupportedJobType(jobType), "%q is a supported monitor type", jobType)
	}

	for _, jobType := range []string{"", "HTTP", "operator-key", "synthetics/browser"} {
		assert.False(t, IsSupportedJobType(jobType), "%q is not a supported monitor type", jobType)
	}
}
