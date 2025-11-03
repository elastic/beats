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

package useragent

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	v            = "10.10.11"
	commit       = "a408c834fe3674b21546885890da17be05c91a51"
	buildTime    = "2024-11-21 16:41:00 +0000"
	mode         = AgentManagementModeManaged
	unprivileged = AgentUnprivilegedModeUnprivileged
)

func TestUserAgent(t *testing.T) {
	ua := UserAgent("FakeBeat", v, commit, buildTime)
	assert.Regexp(t, regexp.MustCompile(`^Elastic-FakeBeat`), ua)

	ua2 := UserAgent("FakeBeat", v, commit, buildTime, "integration_name/1.2.3")
	assert.Regexp(t, regexp.MustCompile(`; integration_name\/1\.2\.3\)$`), ua2)
}

func TestUserAgentWithBeatTelemetry(t *testing.T) {
	isFIPSDistribution := true
	ua2 := UserAgentWithBeatTelemetry("FakeBeat", v, mode, unprivileged, isFIPSDistribution)
	assert.Regexp(t, regexp.MustCompile(`^Elastic-FakeBeat`), ua2)
	assert.Regexp(t, regexp.MustCompile(`; Managed; Unprivileged; FIPS\)$`), ua2)

	// Require deliberate update in case we want to extend the User Agent later
	assert.LessOrEqual(t, len(ua2), 100, "User agent string should be less than 100 characters")
}
