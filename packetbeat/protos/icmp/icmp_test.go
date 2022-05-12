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
	"encoding/hex"
	"net"
	"testing"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"

	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/elastic/beats/v7/packetbeat/procs"
	"github.com/elastic/beats/v7/packetbeat/protos"

	"github.com/stretchr/testify/assert"
)

func TestIcmpIsLocalIp(t *testing.T) {
	icmp := icmpPlugin{localIps: []net.IP{net.IPv4(192, 168, 0, 1), net.IPv4(192, 168, 0, 2)}}

	assert.True(t, icmp.isLocalIP(net.IPv4(127, 0, 0, 1)), "loopback IP")
	assert.True(t, icmp.isLocalIP(net.IPv4(192, 168, 0, 1)), "local IP")
	assert.False(t, icmp.isLocalIP(net.IPv4(10, 0, 0, 1)), "remote IP")
}

func TestIcmpDirection(t *testing.T) {
	icmp := icmpPlugin{}

	trans1 := &icmpTransaction{tuple: icmpTuple{srcIP: net.IPv4(127, 0, 0, 1), dstIP: net.IPv4(127, 0, 0, 1)}}
	assert.Equal(t, uint8(directionLocalOnly), icmp.direction(trans1), "local communication")

	trans2 := &icmpTransaction{tuple: icmpTuple{srcIP: net.IPv4(10, 0, 0, 1), dstIP: net.IPv4(127, 0, 0, 1)}}
	assert.Equal(t, uint8(directionFromOutside), icmp.direction(trans2), "client to server")

	trans3 := &icmpTransaction{tuple: icmpTuple{srcIP: net.IPv4(127, 0, 0, 1), dstIP: net.IPv4(10, 0, 0, 1)}}
	assert.Equal(t, uint8(directionFromInside), icmp.direction(trans3), "server to client")
}

func BenchmarkIcmpProcessICMPv4(b *testing.B) {
	logp.TestingSetup(logp.WithSelectors("icmp", "icmpdetailed"))

	icmp, err := New(true, func(beat.Event) {}, procs.ProcessesWatcher{}, conf.NewConfig())
	if err != nil {
		b.Error("Failed to create ICMP processor")
		return
	}

	icmpRequestData := createICMPv4Layer(b, "08"+"00"+"0000"+"ffff"+"0001")
	packetRequestData := new(protos.Packet)

	icmpResponseData := createICMPv4Layer(b, "00"+"00"+"0000"+"ffff"+"0001")
	packetResponseData := new(protos.Packet)

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		icmp.ProcessICMPv4(nil, icmpRequestData, packetRequestData)
		icmp.ProcessICMPv4(nil, icmpResponseData, packetResponseData)
	}
}

func createICMPv4Layer(b *testing.B, hexstr string) *layers.ICMPv4 {
	data, err := hex.DecodeString(hexstr)
	if err != nil {
		b.Error("Failed to decode hex string")
		return nil
	}

	var df gopacket.DecodeFeedback
	var icmp4 layers.ICMPv4
	err = icmp4.DecodeFromBytes(data, df)
	if err != nil {
		b.Error("Failed to decode ICMPv4 data")
		return nil
	}

	return &icmp4
}
