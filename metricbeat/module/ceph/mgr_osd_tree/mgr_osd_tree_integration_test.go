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

// +build integration,linux

package mgr_osd_tree

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/ceph/mgrtest"
)

const user = "demo"

func TestData(t *testing.T) {
	service := compose.EnsureUp(t, "ceph",
		compose.UpWithTimeout(120*time.Second),
		compose.UpWithEnvironmentVariable("CEPH_CODENAME=nautilus"),
		compose.UpWithEnvironmentVariable("CEPH_VERSION=master-97985eb-nautilus-centos-7-x86_64"),
	)

	f := mbtest.NewReportingMetricSetV2Error(t,
		getConfig(service.HostForPort(8003), mgrtest.GetPassword(t, service.HostForPort(5000), user)))
	err := mbtest.WriteEventsReporterV2Error(f, t, "")
	require.NoError(t, err)
}

func getConfig(host, password string) map[string]interface{} {
	return map[string]interface{}{
		"module":                "ceph",
		"metricsets":            []string{"mgr_osd_tree"},
		"hosts":                 []string{host},
		"username":              user,
		"password":              password,
		"ssl.verification_mode": "none",
	}
}
