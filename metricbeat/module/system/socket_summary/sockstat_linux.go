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
// +build linux

package socket_summary

import (
	"bufio"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/v3/net"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/metric/system/resolve"
)

// SockStat contains data from /proc/net/sockstat
type SockStat struct {
	//The count of all sockets in use on the system, in the most liberal definition. See `ss -s` and `ss -a` for more.
	SocketsUsed int
	//Total in use TCP sockets
	TCPInUse int
	//Total 'orphaned' TCP sockets
	TCPOrphan int
	//Sockets in TIME_WAIT
	TCPTW int
	//Total allocated sockets
	TCPAlloc int
	//Socket memory use, in pages
	TCPMem int
	//In use UDP sockets
	UDPInUse int
	//Socket memory use, in pages
	UDPMem int
	//UDP-Lite in use sockets
	UDPLiteInUse int
	//In Use raw sockets
	RawInUse int
	//FRAG sockets in use
	FragInUse int
	//Frag memory, in bytes
	FragMemory int
}

// applyEnhancements gets a list of platform-specific enhancements and apply them to our mapStr object.
func applyEnhancements(data common.MapStr, sys resolve.Resolver) (common.MapStr, error) {
	dir := sys.ResolveHostFS("/proc/net/sockstat")
	pageSize := os.Getpagesize()

	stat, err := parseSockstat(dir)
	if err != nil {
		return nil, errors.Wrap(err, "error getting sockstat data")
	}
	data.Put("tcp.all.orphan", stat.TCPOrphan)
	data.Put("tcp.memory", pageSize*stat.TCPMem)
	data.Put("udp.memory", pageSize*stat.UDPMem)

	return data, nil

}

// parseSockstat parses the ipv4 sockstat file
//see net/ipv4/proc.c
func parseSockstat(path string) (SockStat, error) {
	fd, err := os.Open(path)
	if err != nil {
		return SockStat{}, err
	}

	var ss SockStat
	scanfLines := []string{
		"sockets: used %d",
		"TCP: inuse %d orphan %d tw %d alloc %d mem %d",
		"UDP: inuse %d mem %d",
		"UDPLITE: inuse %d",
		"RAW: inuse %d",
		"FRAG: inuse %d memory %d",
	}
	scanfOut := [][]interface{}{
		{&ss.SocketsUsed},
		{&ss.TCPInUse, &ss.TCPOrphan, &ss.TCPTW, &ss.TCPAlloc, &ss.TCPMem},
		{&ss.UDPInUse, &ss.UDPMem},
		{&ss.UDPLiteInUse},
		{&ss.RawInUse},
		{&ss.FragInUse, &ss.FragMemory},
	}

	scanner := bufio.NewScanner(fd)

	iter := 0
	for scanner.Scan() {
		//bail if we've iterated more times than expected
		if iter >= len(scanfLines) {
			return ss, nil
		}
		txt := scanner.Text()
		count, err := fmt.Sscanf(txt, scanfLines[iter], scanfOut[iter]...)
		if err != nil {
			return ss, errors.Wrap(err, "error reading sockstat")
		}
		if count != len(scanfOut[iter]) {
			return ss, fmt.Errorf("did not match fields in line %s", scanfLines[iter])
		}

		iter++
	}

	if err = scanner.Err(); err != nil {
		return ss, errors.Wrap(err, "error in scan")
	}

	return ss, nil
}

// connections gets connection information
// The linux function doesn't query UIDs for performance
func connections(kind string) ([]net.ConnectionStat, error) {
	return net.ConnectionsWithoutUids(kind)
}
