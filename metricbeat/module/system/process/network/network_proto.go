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

package network

import (
	"context"
	"fmt"
	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/afpacket"
	"github.com/google/gopacket/layers"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/packetbeat/procs"
	"github.com/elastic/beats/v7/packetbeat/protos/applayer"
)

// runPacketHandle starts the packet capture process
// As of now, this uses AF_PACKET, which is linux-only. However, unlike pcap, we're not
// importing libpcap, which would introduce another runtime dependency into metricbeat.
func runPacketHandle(ctx context.Context, afHandle *afpacket.TPacket, watcher *procs.ProcessesWatcher, procTracker *Tracker) error {
	defer afHandle.Close()
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		packet, ci, perr := afHandle.ZeroCopyReadPacketData()
		if perr != nil {
			return fmt.Errorf("error reading packet data: %w", perr)
		}

		parsed := gopacket.NewPacket(packet, layers.LinkTypeEthernet, gopacket.NoCopy)

		tuple, valid := createTuple(parsed)
		if !valid {
			continue
		}
		layerType := applayer.TransportTCP
		if transLayer := parsed.TransportLayer(); transLayer != nil {
			if transLayer.LayerType() == layers.LayerTypeUDP {
				layerType = applayer.TransportUDP
			}
		}

		procInfo := watcher.FindProcessesTuple(&tuple, layerType)
		procTracker.Update(ci.CaptureLength, layerType, procInfo)

	}

}

func createTuple(parsed gopacket.Packet) (common.IPPortTuple, bool) {
	// all the gopacket.Packet methods love to panic if you've done something wrong,
	// so unpack things carefully. Don't skip nil interface checks.
	networkData := parsed.NetworkLayer()
	dstIP := net.IP{}
	srcIP := net.IP{}
	ipType := 4
	valid := false
	if networkData != nil {
		if ipv4handle, ok := networkData.(*layers.IPv4); ok {
			dstIP = ipv4handle.DstIP
			srcIP = ipv4handle.SrcIP
		}
		if ipv6handle, ok := networkData.(*layers.IPv6); ok {
			dstIP = ipv6handle.DstIP
			srcIP = ipv6handle.SrcIP
		}

		if networkData.LayerType() == layers.LayerTypeIPv6 {
			ipType = 16
		}
	}

	transportData := parsed.TransportLayer()
	var srcPort, dstPort uint16
	if transportData != nil {
		if udpHandle, ok := transportData.(*layers.UDP); ok {
			valid = true
			srcPort = uint16(udpHandle.SrcPort)
			dstPort = uint16(udpHandle.DstPort)
		}
		if tcpHandle, ok := transportData.(*layers.TCP); ok {
			valid = true
			srcPort = uint16(tcpHandle.SrcPort)
			dstPort = uint16(tcpHandle.DstPort)
		}
	}

	return common.NewIPPortTuple(ipType, srcIP, srcPort, dstIP, dstPort), valid
}
