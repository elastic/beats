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

// +build !integration

package icmp

import (
	"testing"

	"github.com/tsg/gopacket/layers"

	"github.com/stretchr/testify/assert"
)

func TestIcmpMessageIsRequestICMPv4(t *testing.T) {
	tuple := &icmpTuple{icmpVersion: 4}

	assert.True(t, isRequest(tuple, &icmpMessage{Type: layers.ICMPv4TypeEchoRequest}))
	assert.False(t, isRequest(tuple, &icmpMessage{Type: layers.ICMPv4TypeEchoReply}))
}

func TestIcmpMessageIsRequestICMPv6(t *testing.T) {
	tuple := &icmpTuple{icmpVersion: 6}

	assert.True(t, isRequest(tuple, &icmpMessage{Type: layers.ICMPv6TypeEchoRequest}))
	assert.False(t, isRequest(tuple, &icmpMessage{Type: layers.ICMPv6TypeEchoReply}))
}

func TestIcmpMessageIsErrorICMPv4(t *testing.T) {
	tuple := &icmpTuple{icmpVersion: 4}

	assert.True(t, isError(tuple, &icmpMessage{Type: layers.ICMPv4TypeDestinationUnreachable}))
	assert.False(t, isError(tuple, &icmpMessage{Type: layers.ICMPv4TypeEchoReply}))
}

func TestIcmpMessageIsErrorICMPv6(t *testing.T) {
	tuple := &icmpTuple{icmpVersion: 6}

	assert.True(t, isError(tuple, &icmpMessage{Type: layers.ICMPv6TypeDestinationUnreachable}))
	assert.False(t, isError(tuple, &icmpMessage{Type: layers.ICMPv6TypeEchoReply}))
}

func TestIcmpMessageRequiresCounterpartICMPv4(t *testing.T) {
	tuple := &icmpTuple{icmpVersion: 4}

	assert.True(t, requiresCounterpart(tuple, &icmpMessage{Type: layers.ICMPv4TypeEchoRequest}))
	assert.False(t, requiresCounterpart(tuple, &icmpMessage{Type: layers.ICMPv4TypeDestinationUnreachable}))
}

func TestIcmpMessageRequiresCounterpartICMPv6(t *testing.T) {
	tuple := &icmpTuple{icmpVersion: 6}

	assert.True(t, requiresCounterpart(tuple, &icmpMessage{Type: layers.ICMPv6TypeEchoRequest}))
	assert.False(t, requiresCounterpart(tuple, &icmpMessage{Type: layers.ICMPv6TypeDestinationUnreachable}))
}

func TestIcmpMessageExtractTrackingDataICMPv4(t *testing.T) {
	baseLayer := &layers.BaseLayer{Contents: []byte{0x0, 0x0, 0x0, 0x0, 0xff, 0x1, 0x0, 0x2}}

	// pair type
	actualID, actualSeq := extractTrackingData(4, layers.ICMPv4TypeEchoRequest, baseLayer)

	assert.Equal(t, uint16(65281), actualID)
	assert.Equal(t, uint16(2), actualSeq)

	// non-pair type
	actualID, actualSeq = extractTrackingData(4, layers.ICMPv4TypeDestinationUnreachable, baseLayer)

	assert.Equal(t, uint16(0), actualID)
	assert.Equal(t, uint16(0), actualSeq)
}

func TestIcmpMessageExtractTrackingDataICMPv6(t *testing.T) {
	baseLayer := &layers.BaseLayer{Contents: []byte{0x0, 0x0, 0x0, 0x0, 0xff, 0x1, 0x0, 0x2}}

	// pair type
	actualID, actualSeq := extractTrackingData(6, layers.ICMPv6TypeEchoRequest, baseLayer)

	assert.Equal(t, uint16(65281), actualID)
	assert.Equal(t, uint16(2), actualSeq)

	// non-pair type
	actualID, actualSeq = extractTrackingData(6, layers.ICMPv6TypeDestinationUnreachable, baseLayer)

	assert.Equal(t, uint16(0), actualID)
	assert.Equal(t, uint16(0), actualSeq)
}

func TestIcmpMessageHumanReadableICMPv4(t *testing.T) {
	tuple := &icmpTuple{icmpVersion: 4}
	msg := &icmpMessage{Type: layers.ICMPv4TypeDestinationUnreachable, code: 3}

	assert.Equal(t, "DestinationUnreachable(Port)", humanReadable(tuple, msg))
}

func TestIcmpMessageHumanReadableICMPv6(t *testing.T) {
	tuple := &icmpTuple{icmpVersion: 6}
	msg := &icmpMessage{Type: layers.ICMPv6TypeDestinationUnreachable, code: 3}

	assert.Equal(t, "DestinationUnreachable(Address)", humanReadable(tuple, msg))
}
