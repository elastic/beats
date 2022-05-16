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

package network_summary

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/metric/system/network"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	_ "github.com/elastic/beats/v7/metricbeat/module/system"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-sysinfo/types"
)

func TestMapping(t *testing.T) {
	example := &types.NetworkCountersInfo{
		SNMP: types.SNMP{
			IP:      map[string]uint64{"DefaultTTL": 64},
			ICMP:    map[string]uint64{"InAddrMaskReps": 5},
			ICMPMsg: map[string]uint64{"InType3": 835},
			TCP:     map[string]uint64{"MaxConn": 0xffffffffffffffff},
			UDP:     map[string]uint64{"IgnoredMulti": 10},
			UDPLite: map[string]uint64{"IgnoredMulti": 0},
		},
		Netstat: types.Netstat{
			TCPExt: map[string]uint64{"DelayedACKLocked": 111, "DelayedACKLost": 1587, "DelayedACKs": 516004},
			IPExt:  map[string]uint64{"InBcastOctets": 431773621, "InBcastPkts": 1686995, "InCEPkts": 0},
		},
	}

	exampleOut := mapstr.M{
		"icmp":     map[string]interface{}{"InAddrMaskReps": uint64(5), "InType3": uint64(835)},
		"ip":       map[string]interface{}{"DefaultTTL": uint64(64), "InBcastOctets": uint64(431773621), "InBcastPkts": uint64(1686995), "InCEPkts": uint64(0)},
		"tcp":      map[string]interface{}{"DelayedACKLocked": uint64(111), "DelayedACKLost": uint64(1587), "DelayedACKs": uint64(516004), "MaxConn": int64(-1)},
		"udp":      map[string]uint64{"IgnoredMulti": 10},
		"udp_lite": map[string]uint64{"IgnoredMulti": 0}}

	out := network.MapProcNetCounters(example)
	assert.Equal(t, exampleOut, out)
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
		"module":     "system",
		"metricsets": []string{"network_summary"},
	}
}
