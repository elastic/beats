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

package network

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/go-sysinfo/types"
)

func TestFilter(t *testing.T) {
	exampleData := &types.NetworkCountersInfo{SNMP: types.SNMP{
		IP: map[string]uint64{"DefaultTTL": 0x40, "ForwDatagrams": 0x3ef68, "Forwarding": 0x1, "FragCreates": 0x0, "FragFails": 0x0, "FragOKs": 0x0, "InAddrErrors": 0x2,
			"InDelivers": 0x132b5d, "InReceives": 0x1904f4, "InUnknownProtos": 0x0, "OutDiscards": 0x0, "OutNoRoutes": 0xe,
			"OutRequests": 0x143a7e},
		ICMP:    map[string]uint64{"InAddrMaskReps": 0x0},
		ICMPMsg: map[string]uint64{"InType3": 0x2, "OutType3": 0x85},
		TCP: map[string]uint64{"ActiveOpens": 0x63d, "AttemptFails": 0x72, "CurrEstab": 0x9, "EstabResets": 0x54, "InCsumErrors": 0x0, "InErrs": 0x0, "InSegs": 0x13270b,
			"MaxConn": 0xffffffffffffffff, "OutRsts": 0x2f6, "OutSegs": 0x111424},
		UDP:     map[string]uint64{"NoPorts": 0x1, "OutDatagrams": 0x4d5},
		UDPLite: map[string]uint64{"SndbufErrors": 0x0},
	},
		Netstat: types.Netstat{
			TCPExt: map[string]uint64{"TCPAbortOnClose": 0x6, "TCPAbortOnData": 0x51, "TCPAbortOnLinger": 0x0, "TCPAbortOnMemory": 0x0,
				"TCPAbortOnTimeout": 0x7f, "TCPAckCompressed": 0x3346, "TCPAutoCorking": 0x1063, "TCPBacklogCoalesce": 0x38a, "TCPBacklogDrop": 0x0, "TCPChallengeACK": 0x0, "TCPDSACKIgnoredDubious": 0x0,
				"TCPHPHits": 0x99fc8, "TCPHystartDelayCwnd": 0x294, "TCPHystartDelayDetect": 0x3, "TCPHystartTrainCwnd": 0xa1e},
			IPExt: map[string]uint64{"InBcastOctets": 0x514d4c, "InBcastPkts": 0x1e999, "InCEPkts": 0x0, "InCsumErrors": 0x0, "InECT0Pkts": 0x0, "InECT1Pkts": 0x0, "InMcastOctets": 0x44a,
				"InMcastPkts": 0x12, "InNoECTPkts": 0x1ec6d9, "InNoRoutes": 0x0, "InOctets": 0x50701313, "OutMcastPkts": 0x71, "OutOctets": 0x47f14f8c, "ReasmOverlaps": 0x0},
		},
	}
	// test with no filter
	testAll := []string{"all"}
	allMap := MapProcNetCountersWithFilter(exampleData, testAll)
	require.Equal(t, len(exampleData.SNMP.ICMP)+len(exampleData.SNMP.ICMPMsg), len(allMap["icmp"].(map[string]interface{})))
	require.Equal(t, len(exampleData.SNMP.TCP)+len(exampleData.Netstat.TCPExt), len(allMap["tcp"].(map[string]interface{})))

	//test With filter
	testTwo := []string{"TCPAbortOnClose", "InBcastOctets"}
	filteredMap := MapProcNetCountersWithFilter(exampleData, testTwo)
	require.Equal(t, 1, len(filteredMap["tcp"].(map[string]interface{})))
	require.Equal(t, uint64(0x6), filteredMap["tcp"].(map[string]interface{})["TCPAbortOnClose"])

	require.Equal(t, uint64(0x514d4c), filteredMap["ip"].(map[string]interface{})["InBcastOctets"])
}
