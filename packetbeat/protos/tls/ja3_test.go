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

package tls

import (
	"encoding/hex"
	"testing"

	"github.com/elastic/beats/v7/packetbeat/protos"

	"github.com/stretchr/testify/assert"
)

var ja3test = []struct {
	Packet, Fingerprint string
}{
	// Chrome on OSX
	{
		Packet: "16030100c2010000be03033367dfae0d46ec0651e49cca2ae47317e8989d" +
			"f710ee7570a88b9a7d5d56b3af00001c3a3ac02bc02fc02cc030cca9cca8c013" +
			"c014009c009d002f0035000a01000079dada0000ff0100010000000010000e00" +
			"000b6578616d706c652e6f72670017000000230000000d001400120403080404" +
			"01050308050501080606010201000500050100000000001200000010000e000c" +
			"02683208687474702f312e3175500000000b00020100000a000a00086a6a001d" +
			"00170018aaaa000100",
		Fingerprint: "94c485bca29d5392be53f2b8cf7f4304",
	},
	// Safari
	{
		Packet: "16030100db010000d703035a1de562860f7943063eaeafa6bffdcfb30b9b" +
			"54e275391ee70a720f05c5a80a00002600ffc02cc02bc024c023c00ac009c030" +
			"c02fc028c027c014c013009d009c003d003c0035002f01000088000000130011" +
			"00000e7777772e656c61737469632e636f000a00080006001700180019000b00" +
			"020100000d001200100401020105010601040302030503060333740000001000" +
			"30002e0268320568322d31360568322d31350568322d313408737064792f332e" +
			"3106737064792f3308687474702f312e31000500050100000000001200000017" +
			"0000",
		Fingerprint: "c07cb55f88702033a8f52c046d23e0b2",
	},
	// Handmade
	{
		Packet: "160301003d010000390301ffffffffffffffffffffffffffffffffffffff" +
			"ffffffffffffffffffffffffff0000080035002f000a00ff0100000800230000" +
			"000f0000",
		Fingerprint: "7a75198d3e18354a6763860d331ff46a",
	},
}

func TestJa3(t *testing.T) {
	for _, test := range ja3test {
		results, tls := testInit()
		reqData, err := hex.DecodeString(test.Packet)
		assert.NoError(t, err)

		tcpTuple := testTCPTuple()
		req := protos.Packet{Payload: reqData}
		var private protos.ProtocolData

		private = tls.Parse(&req, tcpTuple, 0, private)
		tls.ReceivedFin(tcpTuple, 0, private)
		assert.Len(t, results.events, 1)
		event := results.events[0]
		actual, err := event.Fields.GetValue("tls.client.ja3")
		assert.NoError(t, err)
		assert.Equal(t, test.Fingerprint, actual)
	}
}
