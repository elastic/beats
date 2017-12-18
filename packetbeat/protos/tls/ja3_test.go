// +build !integration

package tls

import (
	"encoding/hex"
	"testing"

	"github.com/elastic/beats/packetbeat/protos"

	"github.com/stretchr/testify/assert"
)

var ja3test = []struct {
	Packet, Fingerprint, Str string
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
		Str:         "771,49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53-10,65281-0-23-35-13-5-18-16-30032-11-10,29-23-24,0",
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
		Str:         "771,255-49196-49195-49188-49187-49162-49161-49200-49199-49192-49191-49172-49171-157-156-61-60-53-47,0-10-11-13-13172-16-5-18-23,23-24-25,0",
	},
	// Handmade
	{
		Packet: "160301003d010000390301ffffffffffffffffffffffffffffffffffffff" +
			"ffffffffffffffffffffffffff0000080035002f000a00ff0100000800230000" +
			"000f0000",
		Fingerprint: "7a75198d3e18354a6763860d331ff46a",
		Str:         "769,53-47-10-255,35-15,,",
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
		actual, err := event.Fields.GetValue("tls.fingerprints.ja3.str")
		assert.NoError(t, err)
		assert.Equal(t, test.Str, actual)
		actual, err = event.Fields.GetValue("tls.fingerprints.ja3.hash")
		assert.NoError(t, err)
		assert.Equal(t, test.Fingerprint, actual)
	}
}
