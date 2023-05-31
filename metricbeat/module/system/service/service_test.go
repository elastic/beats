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

//go:build linux

package service

import (
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/elastic/beats/v7/metricbeat/module/system"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var exampleUnits = []dbus.UnitStatus{
	dbus.UnitStatus{
		Name:      "sshd.service",
		LoadState: "active",
	},
	dbus.UnitStatus{
		Name:      "metricbeat.service",
		LoadState: "active",
	},
	dbus.UnitStatus{
		Name: "filebeat.service",
	},
}

func TestDbusEnvConnection(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Test is linux-only")
	}

	// Set specific env var
	// This format is for the newer versions of the godbus/dbus library
	// Older versions use a format with out the `path` prefix.
	err := os.Setenv("DBUS_SYSTEM_BUS_ADDRESS", "unix:path=/var/run/dbus/system_bus_socket")
	require.NoError(t, err)

	// call internal dbus functions
	// This calls a lower-level bus library
	conn, err := instrospectForUnitMethods()
	require.NoError(t, err)
	require.NotNil(t, conn)

	// test the higher-level systemd library
	_, err = dbus.New()
	require.NoError(t, err)

}

func TestFormProps(t *testing.T) {
	testUnit := dbus.UnitStatus{
		Name:        "test.service",
		LoadState:   "loaded",
		ActiveState: "active",
		SubState:    "running",
	}
	testprops := Properties{
		ExecMainPID:          0,
		ExecMainStatus:       0,
		ExecMainCode:         1,
		ActiveEnterTimestamp: 1571850129000000,
		IPAccounting:         true,
		IPEgressBytes:        100,
		IPIngressBytes:       50,
		IPEgressPackets:      100,
		IPIngressPackets:     50,
	}
	event, err := formProperties(testUnit, testprops)
	assert.NoError(t, err)

	testEvent := mapstr.M{
		"state":       "active",
		"exec_code":   "exited",
		"load_state":  "loaded",
		"name":        "test.service",
		"state_since": time.Unix(0, 1571850129000000*1000),
		"sub_state":   "running",
		"resources": mapstr.M{"network": mapstr.M{
			"in": mapstr.M{
				"bytes":   50,
				"packets": 50},
			"out": mapstr.M{
				"bytes":   100,
				"packets": 100},
		},
		},
	}

	assert.NotEmpty(t, event.MetricSetFields["resources"])
	assert.Equal(t, event.MetricSetFields["state_since"], testEvent["state_since"])
	assert.NotEmpty(t, event.RootFields)
}

func TestFilterEmpty(t *testing.T) {

	filtersBad := []string{
		"asdf",
	}
	shouldNotMatch, err := matchUnitPatterns(filtersBad, exampleUnits)
	assert.NoError(t, err)
	assert.Empty(t, shouldNotMatch)
}

func TestFilterMatches(t *testing.T) {
	filtersMatch := []string{
		"ssh*",
	}

	shouldMatch, err := matchUnitPatterns(filtersMatch, exampleUnits)
	assert.NoError(t, err)
	assert.Len(t, shouldMatch, 1)
}

func TestNoFilter(t *testing.T) {
	shouldReturnResults, err := matchUnitPatterns([]string{}, exampleUnits)
	assert.NoError(t, err)
	assert.Len(t, shouldReturnResults, 3)
}

func TestUnitStateFilter(t *testing.T) {
	stateFilter := []string{
		"active",
	}
	shouldReturnResults := matchUnitState(stateFilter, exampleUnits)
	assert.Len(t, shouldReturnResults, 2)

}

func TestUnitStateNoFilter(t *testing.T) {
	shouldReturnResults := matchUnitState([]string{}, exampleUnits)
	assert.Len(t, shouldReturnResults, 3)
}
