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
	iPProtoICMPv4 = 1
	iPProtoTCP    = 6
	iPProtoUDP    = 17
	iPProtoICMPv6 = 58
	iPProtoSCTP   = 132
)

// From github.com/google/gopacket/layers/icmp4.go
const (
	iCMPv4TypeEchoReply           = 0
	iCMPv4TypeEchoRequest         = 8
	iCMPv4TypeRouterAdvertisement = 9
	iCMPv4TypeRouterSolicitation  = 10
	iCMPv4TypeTimestampRequest    = 13
	iCMPv4TypeTimestampReply      = 14
	iCMPv4TypeInfoRequest         = 15
	iCMPv4TypeInfoReply           = 16
	iCMPv4TypeAddressMaskRequest  = 17
	iCMPv4TypeAddressMaskReply    = 18
)

const (
	iCMPv6TypeEchoRequest           = 128
	iCMPv6TypeEchoReply             = 129
	iCMPv6TypeRouterSolicitation    = 133
	iCMPv6TypeRouterAdvertisement   = 134
	iCMPv6TypeNeighborSolicitation  = 135
	iCMPv6TypeNeighborAdvertisement = 136

	// The following are from RFC 2710
	iCMPv6TypeMLDv1MulticastListenerQueryMessage  = 130
	iCMPv6TypeMLDv1MulticastListenerReportMessage = 131

	iCMPv6TypeWhoAreYouRequest             = 139
	iCMPv6TypeWhoAreYouReply               = 140
	iCMPv6TypeHomeAddressDiscoveryRequest  = 144
	iCMPv6TypeHomeAddressDiscoveryResponse = 145
)

func (f Flow) isSorted() bool {
	cmp := bytes.Compare(f.SourceIP, f.DestinationIP)
	return cmp < 0 || (cmp == 0 && f.SourcePort < f.DestinationPort)
}

func (f *Flow) reverse() {
	f.SourceIP, f.DestinationIP = f.DestinationIP, f.SourceIP
	f.SourcePort, f.DestinationPort = f.DestinationPort, f.SourcePort
}
