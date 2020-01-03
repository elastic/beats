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

	"github.com/elastic/beats/libbeat/common"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/go-sysinfo/types"
)

func TestMapping(t *testing.T) {
	example := &types.NetworkCountersInfo{
		SNMP: types.SNMP{
			IP:      map[string]int64{"DefaultTTL": 64},
			ICMP:    map[string]int64{"InAddrMaskReps": 5},
			ICMPMsg: map[string]int64{"InType3": 835},
			UDP:     map[string]int64{"IgnoredMulti": 10},
			UDPLite: map[string]int64{"IgnoredMulti": 0},
		},
		Netstat: types.Netstat{
			TCPExt: map[string]int64{"DelayedACKLocked": 111, "DelayedACKLost": 1587, "DelayedACKs": 516004},
			IPExt:  map[string]int64{"InBcastOctets": 431773621, "InBcastPkts": 1686995, "InCEPkts": 0},
		},
	}

	exampleOut := common.MapStr{
		"icmp":    map[string]int64{"InAddrMaskReps": 5, "InType3": 835},
		"ip":      map[string]int64{"DefaultTTL": 64, "InBcastOctets": 431773621, "InBcastPkts": 1686995, "InCEPkts": 0},
		"tcp":     map[string]int64{"DelayedACKLocked": 111, "DelayedACKLost": 1587, "DelayedACKs": 516004},
		"udp":     map[string]int64{"IgnoredMulti": 10},
		"udpLite": map[string]int64{"IgnoredMulti": 0}}

	out := eventMapping(example)
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
