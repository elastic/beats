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

package flowhash

import (
	"bytes"
	"net"
)

// Flow is the representation of a flow.
type Flow struct {
	// SourceIP is the source IP address (required).
	SourceIP net.IP
	// DestinationIP is the destination IP address (required).
	DestinationIP net.IP
	// Protocol is the IP protocol (required).
	Protocol uint8
	// SourcePort is the source transport port.
	// This field is ignored in ICMP.
	SourcePort uint16
	// DestinationPort is the source transport port.
	// This field is ignored in ICMP.
	DestinationPort uint16
	// ICMP contains the ICMP flow information.
	ICMP struct {
		// Type is the ICMP type.
		Type uint8
		// Code is the ICMP code.
		Code uint8
	}
}

const (
	kIPProtoICMPv4 = 1
	kIPProtoTCP    = 6
	kIPProtoUDP    = 17
	kIPProtoICMPv6 = 58
	kIPProtoSCTP   = 132
)

// From github.com/google/gopacket/layers/icmp4.go
const (
	kICMPv4TypeEchoReply           = 0
	kICMPv4TypeEchoRequest         = 8
	kICMPv4TypeRouterAdvertisement = 9
	kICMPv4TypeRouterSolicitation  = 10
	kICMPv4TypeTimestampRequest    = 13
	kICMPv4TypeTimestampReply      = 14
	kICMPv4TypeInfoRequest         = 15
	kICMPv4TypeInfoReply           = 16
	kICMPv4TypeAddressMaskRequest  = 17
	kICMPv4TypeAddressMaskReply    = 18
)

const (
	kICMPv6TypeEchoRequest           = 128
	kICMPv6TypeEchoReply             = 129
	kICMPv6TypeRouterSolicitation    = 133
	kICMPv6TypeRouterAdvertisement   = 134
	kICMPv6TypeNeighborSolicitation  = 135
	kICMPv6TypeNeighborAdvertisement = 136

	// The following are from RFC 2710
	kICMPv6TypeMLDv1MulticastListenerQueryMessage  = 130
	kICMPv6TypeMLDv1MulticastListenerReportMessage = 131

	kICMPv6TypeWhoAreYouRequest             = 139
	kICMPv6TypeWhoAreYouReply               = 140
	kICMPv6TypeHomeAddressDiscoveryRequest  = 144
	kICMPv6TypeHomeAddressDiscoveryResponse = 145
)

func (f Flow) isSorted() bool {
	cmp := bytes.Compare(f.SourceIP, f.DestinationIP)
	return cmp < 0 || (cmp == 0 && f.SourcePort < f.DestinationPort)
}

func (f *Flow) reverse() {
	f.SourceIP, f.DestinationIP = f.DestinationIP, f.SourceIP
	f.SourcePort, f.DestinationPort = f.DestinationPort, f.SourcePort
}
