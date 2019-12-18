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
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	sysinfotypes "github.com/elastic/go-sysinfo/types"
)

// eventMapping maps the network counters to a MapStr that wil be sent to report.Event
func eventMapping(raw *sysinfotypes.NetworkCountersInfo) common.MapStr {
	fmt.Printf("%#v\n", raw)
	eventByProto := common.MapStr{
		"ip":      combineMap(raw.Netstat.IPExt, raw.SNMP.IP),
		"tcp":     combineMap(raw.Netstat.TCPExt, raw.SNMP.TCP),
		"udp":     raw.SNMP.UDP,
		"udpLite": raw.SNMP.UDPLite,
		"icmp":    combineMap(raw.SNMP.ICMPMsg, raw.SNMP.ICMP),
	}

	return eventByProto
}

// combineMap concatinates two given maps
func combineMap(map1, map2 map[string]int64) map[string]int64 {
	var compMap = make(map[string]int64)

	for k, v := range map1 {
		compMap[k] = v
	}
	for k, v := range map2 {
		compMap[k] = v
	}
	return compMap
}
