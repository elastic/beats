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

//go:build !integration
// +build !integration

package icmp

import (
	"testing"

	"github.com/google/gopacket/layers"

	"github.com/stretchr/testify/assert"
)

func TestIcmpTransactionHasErrorICMPv4(t *testing.T) {
	tuple := icmpTuple{icmpVersion: 4}

	trans1 := icmpTransaction{tuple: tuple, request: nil}
	assert.True(t, trans1.HasError(), "request missing")

	trans2 := icmpTransaction{tuple: tuple, request: &icmpMessage{}, response: &icmpMessage{Type: layers.ICMPv4TypeDestinationUnreachable}}
	assert.True(t, trans2.HasError(), "response with error type")

	trans3 := icmpTransaction{tuple: tuple, request: &icmpMessage{}, response: &icmpMessage{Type: layers.ICMPv4TypeEchoReply}}
	assert.False(t, trans3.HasError(), "response with non-error type")

	trans4 := icmpTransaction{tuple: tuple, request: &icmpMessage{Type: layers.ICMPv4TypeEchoRequest}, response: nil}
	assert.True(t, trans4.HasError(), "transactional request without response")

	trans5 := icmpTransaction{tuple: tuple, request: &icmpMessage{Type: layers.ICMPv4TypeRedirect}, response: nil}
	assert.False(t, trans5.HasError(), "non-transactional request without response")
}

func TestIcmpTransactionHasErrorICMPv6(t *testing.T) {
	tuple := icmpTuple{icmpVersion: 6}

	trans1 := icmpTransaction{tuple: tuple, request: nil}
	assert.True(t, trans1.HasError(), "request missing")

	trans2 := icmpTransaction{tuple: tuple, request: &icmpMessage{}, response: &icmpMessage{Type: layers.ICMPv6TypeDestinationUnreachable}}
	assert.True(t, trans2.HasError(), "response with error type")

	trans3 := icmpTransaction{tuple: tuple, request: &icmpMessage{}, response: &icmpMessage{Type: layers.ICMPv6TypeEchoReply}}
	assert.False(t, trans3.HasError(), "response with non-error type")

	trans4 := icmpTransaction{tuple: tuple, request: &icmpMessage{Type: layers.ICMPv6TypeEchoRequest}, response: nil}
	assert.True(t, trans4.HasError(), "transactional request without response")

	trans5 := icmpTransaction{tuple: tuple, request: &icmpMessage{Type: layers.ICMPv6TypeRedirect}, response: nil}
	assert.False(t, trans5.HasError(), "non-transactional request without response")
}
