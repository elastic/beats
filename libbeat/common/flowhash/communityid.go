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
	"crypto"
	_ "crypto/sha1"
	"encoding/binary"
	"net"
)

type communityIDHasher struct {
	encoder Encoding
	seed    [2]byte
	hash    crypto.Hash
}

// CommunityID is a flow hasher instance using the default values
// in the community ID specification.
var CommunityID = NewCommunityID(0, Base64Encoding, crypto.SHA1)

// NewCommunityID allows to instantiate a flow hasher with custom settings.
func NewCommunityID(seed uint16, encoder Encoding, hash crypto.Hash) Hasher {
	h := &communityIDHasher{
		encoder: encoder,
		hash:    hash,
	}
	binary.BigEndian.PutUint16(h.seed[:], seed)
	return h
}

// Hash returns the hash for the given flow.
func (h *communityIDHasher) Hash(flow Flow) string {
	switch flow.Protocol {
	// For ICMP, populate source and destination port with ICMP data
	case kIPProtoICMPv4, kIPProtoICMPv6:
		var isOneWay bool
		table := icmpV4Equiv
		if flow.Protocol == kIPProtoICMPv6 {
			table = icmpV6Equiv
		}
		flow.SourcePort, flow.DestinationPort, isOneWay = icmpPortEquivalents(flow, table)
		if !isOneWay && !flow.isSorted() {
			flow.reverse()
		}
	// For all other protocols, make (srcip, srcport) < (dstip, dstport)
	default:
		if !flow.isSorted() {
			flow.reverse()
		}
	}

	hasher := h.hash.New()
	hasher.Write(h.seed[:])
	hasher.Write(getRawIP(flow.SourceIP))
	hasher.Write(getRawIP(flow.DestinationIP))
	// protocol + zero padding
	buf := [2]byte{
		flow.Protocol,
		0,
	}
	slice := buf[:]
	hasher.Write(slice)

	switch flow.Protocol {
	case kIPProtoTCP, kIPProtoUDP, kIPProtoSCTP, kIPProtoICMPv4, kIPProtoICMPv6:
		binary.BigEndian.PutUint16(slice, flow.SourcePort)
		hasher.Write(slice)
		binary.BigEndian.PutUint16(slice, flow.DestinationPort)
		hasher.Write(slice)
	}
	return "1:" + h.encoder.EncodeToString(hasher.Sum(nil))
}

func getRawIP(ip net.IP) []byte {
	// This is a workaround to make sure IPv4 addresses are the right
	// length. It's needed because net.ParseIP / net.IPv4 returns IPv6
	// style IPv4 addresses.
	if asV4 := ip.To4(); asV4 != nil {
		return asV4
	}
	return ip
}

var icmpV4Equiv = map[uint8]uint8{
	kICMPv4TypeEchoRequest:         kICMPv4TypeEchoReply,
	kICMPv4TypeEchoReply:           kICMPv4TypeEchoRequest,
	kICMPv4TypeTimestampRequest:    kICMPv4TypeTimestampReply,
	kICMPv4TypeTimestampReply:      kICMPv4TypeTimestampRequest,
	kICMPv4TypeInfoRequest:         kICMPv4TypeInfoReply,
	kICMPv4TypeRouterSolicitation:  kICMPv4TypeRouterAdvertisement,
	kICMPv4TypeRouterAdvertisement: kICMPv4TypeRouterSolicitation,
	kICMPv4TypeAddressMaskRequest:  kICMPv4TypeAddressMaskReply,
	kICMPv4TypeAddressMaskReply:    kICMPv4TypeAddressMaskRequest,
}

var icmpV6Equiv = map[uint8]uint8{
	kICMPv6TypeEchoRequest:                        kICMPv6TypeEchoReply,
	kICMPv6TypeEchoReply:                          kICMPv6TypeEchoRequest,
	kICMPv6TypeRouterSolicitation:                 kICMPv6TypeRouterAdvertisement,
	kICMPv6TypeRouterAdvertisement:                kICMPv6TypeRouterSolicitation,
	kICMPv6TypeNeighborAdvertisement:              kICMPv6TypeNeighborSolicitation,
	kICMPv6TypeNeighborSolicitation:               kICMPv6TypeNeighborAdvertisement,
	kICMPv6TypeMLDv1MulticastListenerQueryMessage: kICMPv6TypeMLDv1MulticastListenerReportMessage,
	kICMPv6TypeWhoAreYouRequest:                   kICMPv6TypeWhoAreYouReply,
	kICMPv6TypeWhoAreYouReply:                     kICMPv6TypeWhoAreYouRequest,
	kICMPv6TypeHomeAddressDiscoveryRequest:        kICMPv6TypeHomeAddressDiscoveryResponse,
	kICMPv6TypeHomeAddressDiscoveryResponse:       kICMPv6TypeHomeAddressDiscoveryRequest,
}

func icmpPortEquivalents(flow Flow, table map[uint8]uint8) (src uint16, dst uint16, isOneWay bool) {
	if equiv, found := table[flow.ICMP.Type]; found {
		return uint16(flow.ICMP.Type), uint16(equiv), false
	}
	return uint16(flow.ICMP.Type), uint16(flow.ICMP.Code), true
}
