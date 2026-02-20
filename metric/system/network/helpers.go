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
	"github.com/elastic/elastic-agent-libs/mapstr"
	sysinfotypes "github.com/elastic/go-sysinfo/types"
)

// MapProcNetCountersWithFilter converts the NetworkCountersInfo to a formatted mapstring,
// and applies a filter to the resulting map. The filter should be an array key values, taken from /proc/PID/net/snmp or /proc/PID/net/netstat
func MapProcNetCountersWithFilter(raw *sysinfotypes.NetworkCountersInfo, filter []string) mapstr.M {
	return createMap(raw, filter)
}

// MapProcNetCounters converts the NetworkCountersInfo struct into a MapStr acceptable for sending upstream
func MapProcNetCounters(raw *sysinfotypes.NetworkCountersInfo) mapstr.M {
	return createMap(raw, []string{"all"})
}

func createMap(raw *sysinfotypes.NetworkCountersInfo, filter []string) mapstr.M {
	eventByProto := mapstr.M{
		"ip":       combineMap(raw.Netstat.IPExt, raw.SNMP.IP, filter),
		"tcp":      combineMap(raw.Netstat.TCPExt, raw.SNMP.TCP, filter),
		"udp":      raw.SNMP.UDP,
		"udp_lite": raw.SNMP.UDPLite,
		"icmp":     combineMap(raw.SNMP.ICMPMsg, raw.SNMP.ICMP, filter),
	}

	return eventByProto
}

// combineMap concatinates two given maps
func combineMap(map1, map2 map[string]uint64, filter []string) map[string]any {
	var compMap = make(map[string]any)

	if len(filter) == 0 || filter[0] == "all" {
		for k, v := range map1 {
			compMap[k] = checkMaxConn(k, v)
		}
		for k, v := range map2 {
			compMap[k] = checkMaxConn(k, v)
		}
	} else {
		for _, key := range filter {
			if value, ok := map1[key]; ok {
				compMap[key] = checkMaxConn(key, value)
			}
			if value, ok := map2[key]; ok {
				compMap[key] = checkMaxConn(key, value)
			}

		}
	}

	return compMap
}

// checkMaxConn deals with the "oddball" MaxConn value, which is defined by RFC2012 as a integer
// while the other values are going to be unsigned counters
func checkMaxConn(inKey string, in uint64) any {

	if inKey == "MaxConn" {
		return int64(in)
	}
	return in
}
