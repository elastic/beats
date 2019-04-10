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

package osd_tree

import (
	"fmt"
	"os"
	"testing"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestData(t *testing.T) {
	f := mbtest.NewReportingMetricSetV2(t, getConfig())

	if err := mbtest.WriteEventsReporterV2(f, t, ""); err != nil {
		t.Fatal("write", err)
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "ceph",
		"metricsets": []string{"osd_tree"},
		"hosts":      getTestCephHost(),
	}
}

const (
	cephDefaultHost = "127.0.0.1"
	cephDefaultPort = "5000"
)

func getTestCephHost() string {
	return fmt.Sprintf("%v:%v",
		getenv("CEPH_HOST", cephDefaultHost),
		getenv("CEPH_PORT", cephDefaultPort),
	)
}

func getenv(name, defaultValue string) string {
	return strDefault(os.Getenv(name), defaultValue)
}

func strDefault(a, defaults string) string {
	if len(a) == 0 {
		return defaults
	}
	return a
}
