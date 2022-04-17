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
	"github.com/menderesk/beats/v7/libbeat/common"
	sysinfotypes "github.com/menderesk/go-sysinfo/types"
)

// MapProcNetCounters converts the NetworkCountersInfo struct into a MapStr acceptable for sending upstream
func MapProcNetCounters(raw *sysinfotypes.NetworkCountersInfo) common.MapStr {

	eventByProto := common.MapStr{
		"ip":       combineMap(raw.Netstat.IPExt, raw.SNMP.IP),
		"tcp":      combineMap(raw.Netstat.TCPExt, raw.SNMP.TCP),
		"udp":      raw.SNMP.UDP,
		"udp_lite": raw.SNMP.UDPLite,
		"icmp":     combineMap(raw.SNMP.ICMPMsg, raw.SNMP.ICMP),
	}

	return eventByProto
}

// combineMap concatinates two given maps
func combineMap(map1, map2 map[string]uint64) map[string]interface{} {
	var compMap = make(map[string]interface{})

	for k, v := range map1 {
		compMap[k] = checkMaxConn(k, v)
	}
	for k, v := range map2 {
		compMap[k] = checkMaxConn(k, v)
	}
	return compMap
}

// checkMaxConn deals with the "oddball" MaxConn value, which is defined by RFC2012 as a integer
// while the other values are going to be unsigned counters
func checkMaxConn(inKey string, in uint64) interface{} {

	if inKey == "MaxConn" {
		return int64(in)
	}
	return in
}
