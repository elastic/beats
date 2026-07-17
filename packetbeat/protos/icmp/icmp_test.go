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

package icmp

import (
	"encoding/hex"
	"testing"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"

	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/elastic/beats/v7/packetbeat/procs"
	"github.com/elastic/beats/v7/packetbeat/protos"
)

func BenchmarkIcmpProcessICMPv4(b *testing.B) {
	logp.TestingSetup(logp.WithSelectors("icmp", "icmpdetailed"))

	icmp, err := New(true, func(beat.Event) {}, &procs.ProcessesWatcher{}, conf.NewConfig())
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
