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

package services

import (
	"testing"
	"time"

	"github.com/coreos/go-systemd/dbus"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestFormProps(t *testing.T) {
	testUnit := dbus.UnitStatus{
		Name:        "test.service",
		LoadState:   "loaded",
		ActiveState: "active",
		SubState:    "running",
	}
	testprops := map[string]interface{}{
		"ExecMainPID":          uint32(0),
		"ExecMainStatus":       int32(0),
		"ExecMainCode":         int32(1),
		"ActiveEnterTimestamp": uint64(1571850129000000),
	}
	event, err := formProperties(testUnit, testprops)
	assert.NoError(t, err)
	//t.Logf("Event: %s", event)

	testOut := common.MapStr{
		"active_state": "active",
		"exec_code":    "exited",
		"exec_rc":      int32(0),
		"load_state":   "loaded",
		"name":         "test.service",
		"resources":    common.MapStr{},
		"state_since":  time.Unix(0, 1571850129000000*1000),
		"sub_state":    "running",
	}

	assert.Equal(t, testOut, event)
}

func TestData(t *testing.T) {
	f := mbtest.NewReportingMetricSetV2Error(t, getConfig())
	err := mbtest.WriteEventsReporterV2Error(f, t, ".")
	if err != nil {
		t.Fatal("write", err)
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":                "system",
		"metricsets":            []string{"services"},
		"services.state_filter": []string{"active"},
	}
}
