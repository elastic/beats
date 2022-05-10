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

package icmp

import (
	"encoding/binary"
	"time"

	"github.com/google/gopacket/layers"

	"github.com/elastic/elastic-agent-libs/logp"
)

// TODO: more types (that are not provided as constants in gopacket)

// ICMPv4 types that represent a response (all other types represent a request)
var icmp4ResponseTypes = map[uint8]bool{
	layers.ICMPv4TypeEchoReply:        true,
	layers.ICMPv4TypeTimestampReply:   true,
	layers.ICMPv4TypeInfoReply:        true,
	layers.ICMPv4TypeAddressMaskReply: true,
}

// ICMPv6 types that represent a response (all other types represent a request)
var icmp6ResponseTypes = map[uint8]bool{
	layers.ICMPv6TypeEchoReply: true,
}

// ICMPv4 types that represent an error
var icmp4ErrorTypes = map[uint8]bool{
	layers.ICMPv4TypeDestinationUnreachable: true,
	layers.ICMPv4TypeSourceQuench:           true,
	layers.ICMPv4TypeTimeExceeded:           true,
	layers.ICMPv4TypeParameterProblem:       true,
}

// ICMPv6 types that represent an error
var icmp6ErrorTypes = map[uint8]bool{
	layers.ICMPv6TypeDestinationUnreachable: true,
	layers.ICMPv6TypePacketTooBig:           true,
	layers.ICMPv6TypeTimeExceeded:           true,
	layers.ICMPv6TypeParameterProblem:       true,
}

// ICMPv4 types that require a request & a response
var icmp4PairTypes = map[uint8]bool{
	layers.ICMPv4TypeEchoRequest:        true,
	layers.ICMPv4TypeEchoReply:          true,
	layers.ICMPv4TypeTimestampRequest:   true,
	layers.ICMPv4TypeTimestampReply:     true,
	layers.ICMPv4TypeInfoRequest:        true,
	layers.ICMPv4TypeInfoReply:          true,
	layers.ICMPv4TypeAddressMaskRequest: true,
	layers.ICMPv4TypeAddressMaskReply:   true,
}

// ICMPv6 types that require a request & a response
var icmp6PairTypes = map[uint8]bool{
	layers.ICMPv6TypeEchoRequest: true,
	layers.ICMPv6TypeEchoReply:   true,
}

// Contains all used information from the ICMP message on the wire.
type icmpMessage struct {
	ts     time.Time
	Type   uint8
	code   uint8
	length int
}

func isRequest(tuple *icmpTuple, msg *icmpMessage) bool {
	if tuple.icmpVersion == 4 {
		return !icmp4ResponseTypes[msg.Type]
	}
	if tuple.icmpVersion == 6 {
		return !icmp6ResponseTypes[msg.Type]
	}
	logp.NewLogger("icmp").DPanic("Invalid ICMP version[%d]", tuple.icmpVersion)
	return true
}

func isError(tuple *icmpTuple, msg *icmpMessage) bool {
	if tuple.icmpVersion == 4 {
		return icmp4ErrorTypes[msg.Type]
	}
	if tuple.icmpVersion == 6 {
		return icmp6ErrorTypes[msg.Type]
	}
	logp.NewLogger("icmp").DPanic("Invalid ICMP version[%d]", tuple.icmpVersion)
	return true
}

func requiresCounterpart(tuple *icmpTuple, msg *icmpMessage) bool {
	if tuple.icmpVersion == 4 {
		return icmp4PairTypes[msg.Type]
	}
	if tuple.icmpVersion == 6 {
		return icmp6PairTypes[msg.Type]
	}
	logp.NewLogger("icmp").DPanic("Invalid ICMP version[%d]", tuple.icmpVersion)
	return false
}

func extractTrackingData(icmpVersion uint8, msgType uint8, baseLayer *layers.BaseLayer) (uint16, uint16) {
	if icmpVersion == 4 {
		if icmp4PairTypes[msgType] {
			id := binary.BigEndian.Uint16(baseLayer.Contents[4:6])
			seq := binary.BigEndian.Uint16(baseLayer.Contents[6:8])
			return id, seq
		}
		return 0, 0
	}
	if icmpVersion == 6 {
		if icmp6PairTypes[msgType] {
			id := binary.BigEndian.Uint16(baseLayer.Contents[4:6])
			seq := binary.BigEndian.Uint16(baseLayer.Contents[6:8])
			return id, seq
		}
		return 0, 0
	}
	logp.NewLogger("icmp").DPanic("Invalid ICMP version[%d]", icmpVersion)
	return 0, 0
}

func humanReadable(tuple *icmpTuple, msg *icmpMessage) string {
	if tuple.icmpVersion == 4 {
		return layers.ICMPv4TypeCode(binary.BigEndian.Uint16([]byte{msg.Type, msg.code})).String()
	}
	if tuple.icmpVersion == 6 {
		return layers.ICMPv6TypeCode(binary.BigEndian.Uint16([]byte{msg.Type, msg.code})).String()
	}
	logp.NewLogger("icmp").DPanic("Invalid ICMP version[%d]", tuple.icmpVersion)
	return ""
}
