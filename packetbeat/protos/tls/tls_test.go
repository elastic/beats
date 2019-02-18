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

package tls

import (
	"encoding/hex"
	"encoding/json"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/publish"
)

type eventStore struct {
	events []beat.Event
}

const (
	expectedClientHello = `{"client":{"ip":"192.168.0.1","port":6512},"destination":{"domain":"example.org","ip":"192.168.0.2","port":27017},"event":{"category":"network_traffic","dataset":"tls","kind":"event"},"network":{"community_id":"1:jKfewJN/czjTuEpVvsKdYXXiMzs=","protocol":"tls","transport":"tcp","type":"ipv4"},"server":{"domain":"example.org","ip":"192.168.0.2","port":27017},"source":{"ip":"192.168.0.1","port":6512},"status":"Error","tls":{"client_certificate_requested":false,"client_hello":{"extensions":{"_unparsed_":["renegotiation_info","23","status_request","18","30032"],"application_layer_protocol_negotiation":["h2","http/1.1"],"ec_points_formats":["uncompressed"],"server_name_indication":["example.org"],"session_ticket":"","signature_algorithms":["ecdsa_secp256r1_sha256","rsa_pss_sha256","rsa_pkcs1_sha256","ecdsa_secp384r1_sha384","rsa_pss_sha384","rsa_pkcs1_sha384","rsa_pss_sha512","rsa_pkcs1_sha512","rsa_pkcs1_sha1"],"supported_groups":["x25519","secp256r1","secp384r1"]},"supported_ciphers":["TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256","TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256","TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384","TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384","TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256","TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256","TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA","TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA","TLS_RSA_WITH_AES_128_GCM_SHA256","TLS_RSA_WITH_AES_256_GCM_SHA384","TLS_RSA_WITH_AES_128_CBC_SHA","TLS_RSA_WITH_AES_256_CBC_SHA","TLS_RSA_WITH_3DES_EDE_CBC_SHA"],"supported_compression_methods":["NULL"],"version":"3.3"},"fingerprints":{"ja3":{"hash":"94c485bca29d5392be53f2b8cf7f4304","str":"771,49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53-10,65281-0-23-35-13-5-18-16-30032-11-10,29-23-24,0"}},"handshake_completed":false,"resumed":false},"type":"tls"}`
	expectedServerHello = `{"extensions":{"_unparsed_":["renegotiation_info","status_request"],"application_layer_protocol_negotiation":["h2"],"ec_points_formats":["uncompressed","ansiX962_compressed_prime","ansiX962_compressed_char2"],"session_ticket":""},"selected_cipher":"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256","selected_compression_method":"NULL","version":"3.3"}`
	rawClientHello      = "16030100c2010000be03033367dfae0d46ec0651e49cca2ae47317e8989df710" +
		"ee7570a88b9a7d5d56b3af00001c3a3ac02bc02fc02cc030cca9cca8c013c014" +
		"009c009d002f0035000a01000079dada0000ff0100010000000010000e00000b" +
		"6578616d706c652e6f72670017000000230000000d0014001204030804040105" +
		"0308050501080606010201000500050100000000001200000010000e000c0268" +
		"3208687474702f312e3175500000000b00020100000a000a00086a6a001d0017" +
		"0018aaaa000100"
	rawServerHello = "160303004a0200004603037806e1be0c363bcc1fe14a906d1ff1b11dc5369d91" +
		"c631ed660d6c0f156f420700c02f00001eff01000100000b0004030001020023" +
		"000000050000001000050003026832"

	rawChangeCipherSpec = "1403030000"
)

func (e *eventStore) publish(event beat.Event) {
	publish.MarshalPacketbeatFields(&event, nil)
	e.events = append(e.events, event)
}

// Helper function returning a TLS module that can be used
// in tests. It publishes the transactions in the results structure.
func testInit() (*eventStore, *tlsPlugin) {
	logp.TestingSetup(logp.WithSelectors("tls", "tlsdetailed"))

	results := &eventStore{}
	tls, err := New(true, results.publish, nil)
	if err != nil {
		return nil, nil
	}
	return results, tls.(*tlsPlugin)
}

// Helper function that returns an example TcpTuple
func testTCPTuple() *common.TCPTuple {
	t := &common.TCPTuple{
		IPLength: 4,
		BaseTuple: common.BaseTuple{
			SrcIP: net.IPv4(192, 168, 0, 1), DstIP: net.IPv4(192, 168, 0, 2),
			SrcPort: 6512, DstPort: 27017,
		},
	}
	t.ComputeHashables()
	return t
}

func TestPlugin(t *testing.T) {
	_, plugin := testInit()
	assert.NotNil(t, plugin)
	assert.Empty(t, plugin.GetPorts())
	assert.Equal(t, protos.DefaultTransactionExpiration, plugin.ConnectionTimeout())
	assert.Len(t, plugin.fingerprints, 1)
	assert.Equal(t, "sha1", plugin.fingerprints[0].name)
}

func TestNotTLS(t *testing.T) {
	results, tls := testInit()

	reqData := []byte(
		"GET / HTTP/1.1\r\n" +
			"Host: example.net\r\n" +
			"\r\n")
	tcpTuple := testTCPTuple()
	req := protos.Packet{Payload: reqData}
	var private protos.ProtocolData

	private = tls.Parse(&req, tcpTuple, 0, private)
	tls.ReceivedFin(tcpTuple, 0, private)
	assert.Empty(t, results.events)
}

func TestAlert(t *testing.T) {
	results, tls := testInit()

	reqData, err := hex.DecodeString(
		"1503010002022d")
	assert.Nil(t, err)
	tcpTuple := testTCPTuple()
	req := protos.Packet{Payload: reqData}
	var private protos.ProtocolData

	private = tls.Parse(&req, tcpTuple, 0, private)
	tls.ReceivedFin(tcpTuple, 0, private)

	assert.Len(t, results.events, 1)
	event := results.events[0]
	_, ok := event.Fields["tls"]
	assert.True(t, ok)
	tlsMap, ok := event.Fields["tls"].(common.MapStr)
	assert.True(t, ok)

	alerts, ok := tlsMap["alerts"].([]common.MapStr)
	assert.True(t, ok)
	assert.Len(t, alerts, 1)
	severity, ok := alerts[0]["severity"]
	assert.True(t, ok)
	assert.Equal(t, "fatal", severity)
	code, ok := alerts[0]["code"]
	assert.True(t, ok)
	assert.Equal(t, 0x2d, code)
	str, ok := alerts[0]["type"]
	assert.True(t, ok)
	assert.Equal(t, "certificate_expired", str)
}

func TestInvalidAlert(t *testing.T) {
	results, tls := testInit()

	reqData, err := hex.DecodeString(
		"1503010003010203")
	assert.Nil(t, err)
	tcpTuple := testTCPTuple()
	req := protos.Packet{Payload: reqData}
	var private protos.ProtocolData

	private = tls.Parse(&req, tcpTuple, 0, private)
	tls.ReceivedFin(tcpTuple, 0, private)

	assert.Empty(t, results.events)
}

func TestClientHello(t *testing.T) {
	results, tls := testInit()

	reqData, err := hex.DecodeString(rawClientHello)

	assert.Nil(t, err)
	tcpTuple := testTCPTuple()
	req := protos.Packet{Payload: reqData}
	var private protos.ProtocolData

	private = tls.Parse(&req, tcpTuple, 0, private)
	tls.GapInStream(tcpTuple, 0, 1024, private)

	assert.Len(t, results.events, 1)
	event := results.events[0]

	b, err := json.Marshal(event.Fields)
	assert.Nil(t, err)
	assert.Equal(t, expectedClientHello, string(b))
}

func TestServerHello(t *testing.T) {
	results, tls := testInit()

	reqData, err := hex.DecodeString(rawServerHello)

	assert.Nil(t, err)
	tcpTuple := testTCPTuple()
	req := protos.Packet{Payload: reqData}
	var private protos.ProtocolData

	private = tls.Parse(&req, tcpTuple, 0, private)
	tls.ReceivedFin(tcpTuple, 0, private)

	assert.Len(t, results.events, 1)
	event := results.events[0]

	hello, err := event.GetValue("tls.server_hello")
	assert.Nil(t, err)
	b, err := json.Marshal(hello)
	assert.Nil(t, err)
	assert.Equal(t, expectedServerHello, string(b))
}

func TestFragmentedHandshake(t *testing.T) {
	results, tls := testInit()

	// First, a full record containing only half of a handshake message
	reqData, err := hex.DecodeString(
		"160301003f010000be03033367dfae0d46ec0651e49cca2ae47317e8989df710" +
			"ee7570a88b9a7d5d56b3af00001c3a3ac02bc02fc02cc030cca9cca8c013c014" +
			"009c009d")

	assert.Nil(t, err)
	tcpTuple := testTCPTuple()
	req := protos.Packet{Payload: reqData}
	var private protos.ProtocolData

	private = tls.Parse(&req, tcpTuple, 0, private)

	// Second, half a record containing the middle part of a handshake
	reqData, err = hex.DecodeString(
		"1603010083002f0035000a01000079dada0000ff0100010000000010000e00000b" +
			"6578616d706c652e6f72670017000000230000000d0014001204030804040105" +
			"0308050501080606010201000500050100000000001200000010000e000c0268")
	assert.Nil(t, err)
	req = protos.Packet{Payload: reqData}
	private = tls.Parse(&req, tcpTuple, 0, private)

	// Third, the final part of the second record, completing the handshake
	reqData, err = hex.DecodeString(
		"3208687474702f312e3175500000000b00020100000a000a00086a6a001d0017" +
			"0018aaaa000100")
	assert.Nil(t, err)
	req = protos.Packet{Payload: reqData}
	private = tls.Parse(&req, tcpTuple, 0, private)

	tls.ReceivedFin(tcpTuple, 0, private)

	assert.Len(t, results.events, 1)
	event := results.events[0]

	b, err := json.Marshal(event.Fields)
	assert.Nil(t, err)
	assert.Equal(t, expectedClientHello, string(b))
}

func TestInterleavedRecords(t *testing.T) {
	results, tls := testInit()

	// First, a full record containing only half of a handshake message
	reqData, err := hex.DecodeString(
		"160301003f010000be03033367dfae0d46ec0651e49cca2ae47317e8989df710" +
			"ee7570a88b9a7d5d56b3af00001c3a3ac02bc02fc02cc030cca9cca8c013c014" +
			"009c009d")

	assert.Nil(t, err)
	tcpTuple := testTCPTuple()
	req := protos.Packet{Payload: reqData}
	var private protos.ProtocolData

	private = tls.Parse(&req, tcpTuple, 0, private)

	// Then two records containing one alert each, merged in a single packet
	reqData, err = hex.DecodeString(
		"1503010002FFFF15030100020101")
	assert.Nil(t, err)
	req = protos.Packet{Payload: reqData}
	private = tls.Parse(&req, tcpTuple, 0, private)

	// And an application data record
	reqData, err = hex.DecodeString(
		"17030100080123456789abcdef")
	assert.Nil(t, err)
	req = protos.Packet{Payload: reqData}
	private = tls.Parse(&req, tcpTuple, 0, private)

	// Then what's missing from the handshake
	reqData, err = hex.DecodeString(
		"1603010083002f0035000a01000079dada0000ff0100010000000010000e00000b" +
			"6578616d706c652e6f72670017000000230000000d0014001204030804040105" +
			"0308050501080606010201000500050100000000001200000010000e000c0268" +
			"3208687474702f312e3175500000000b00020100000a000a00086a6a001d0017" +
			"0018aaaa000100")
	assert.Nil(t, err)
	req = protos.Packet{Payload: reqData}
	private = tls.Parse(&req, tcpTuple, 0, private)

	tls.ReceivedFin(tcpTuple, 0, private)

	assert.Len(t, results.events, 1)
	event := results.events[0]

	// Event contains the client hello
	_, err = event.GetValue("tls.client_hello")
	assert.Nil(t, err)

	// and the alert
	alerts, err := event.GetValue("tls.alerts")
	assert.Nil(t, err)

	assert.Len(t, alerts.([]common.MapStr), 2)
}

func TestCompletedHandshake(t *testing.T) {
	results, tls := testInit()

	// First, a certificates record
	reqData, err := hex.DecodeString(certsMsg)

	assert.Nil(t, err)
	tcpTuple := testTCPTuple()
	req := protos.Packet{Payload: reqData}
	var private protos.ProtocolData

	private = tls.Parse(&req, tcpTuple, 0, private)
	assert.NotNil(t, private)
	assert.Empty(t, results.events)

	// Then a change cypher spec message
	reqData, err = hex.DecodeString(rawChangeCipherSpec)
	req = protos.Packet{Payload: reqData}
	private = tls.Parse(&req, tcpTuple, 0, private)
	assert.NotNil(t, private)
	assert.Empty(t, results.events)

	// And the corresponding one on the other direction
	private = tls.Parse(&req, tcpTuple, 1, private)
	assert.NotNil(t, private)
	assert.NotEmpty(t, results.events)
}

func TestTLS13VersionNegotiation(t *testing.T) {
	results, tls := testInit()

	// First, a client hello
	reqData, err := hex.DecodeString(
		"16030102310100022d03039b9e3d533312e698bdc35c8d86902204c0f2505682" +
			"2e0ae66b5f7bff999a7c6220944f9b7806d887e27500dc6a05cfed8becf3d65a" +
			"9a75ab618828f1b9e418d16800222a2a130113021303c02bc02fc02cc030cca9" +
			"cca8c013c014009c009d002f0035000a010001c2baba0000ff01000100000000" +
			"1d001b000018746c7331332e63727970746f2e6d6f7a696c6c612e6f72670017" +
			"000000230000000d001400120403080404010503080505010806060102010005" +
			"00050100000000001200000010000e000c02683208687474702f312e31755000" +
			"00000b000201000033002b00292a2a000100001d00208c80626064298b32ef53" +
			"5d9305355e992b98baaa5db28e22a718741eab108d48002d00020101002b000b" +
			"0a9a9a0304030303020301000a000a00082a2a001d00170018001b0003020002" +
			"6a6a000100002900ed00c800c21f81d2ec6041f6cecd60949000000000784b0a" +
			"740ce3334a066d552e3d94af270080b67e1a29ea0e6dbccdbe6ea8699cda3e28" +
			"94f98dbea2fa3b1040acdf8dd3f7edefed8f768a6076a034b63c9464e9a22301" +
			"1d6ef9ff0f8ce74e7a5701da7f957116b5a3c0600541f86fb00ca54dc9f4eaec" +
			"6a657331881c1fcd23c59cca16d27af51a71301c38870de721382175d3de8423" +
			"d809edfcd417861a3ca83e40cf631616e0791efbcc79a0fdfe0d57c6ede4dd4f" +
			"8dc54cdb7904a8924f10c55f97e5fcc1f813e6002120720c822a09c99a10b09e" +
			"de25dded2e4c62eff486bf7827f89613f3038d5a200a")
	assert.Nil(t, err)
	tcpTuple := testTCPTuple()
	req := protos.Packet{Payload: reqData}
	var private protos.ProtocolData

	private = tls.Parse(&req, tcpTuple, 0, private)
	assert.NotNil(t, private)
	assert.Empty(t, results.events)

	// Then a server hello + change cypher spec
	reqData, err = hex.DecodeString(
		"160303007a020000760303225084578024a693566bc71ba223826eeffc875b20" +
			"27eec7337bf5fdf0eb1de720944f9b7806d887e27500dc6a05cfed8becf3d65a" +
			"9a75ab618828f1b9e418d168130100002e00330024001d002070b27700b360aa" +
			"3941a22da86901c00e174dc3d83e13cf4159b34b3de6809372002b0002030414" +
			"0303000101")
	req = protos.Packet{Payload: reqData}
	private = tls.Parse(&req, tcpTuple, 1, private)
	assert.NotNil(t, private)
	assert.Empty(t, results.events)

	// Then a change cypher spec from the client
	reqData, err = hex.DecodeString(rawChangeCipherSpec)
	req = protos.Packet{Payload: reqData}
	private = tls.Parse(&req, tcpTuple, 0, private)
	assert.NotNil(t, private)
	assert.NotEmpty(t, results.events)

	iVersion, err := results.events[0].Fields.GetValue("tls.version")
	assert.Nil(t, err)

	version, ok := iVersion.(string)
	assert.True(t, ok)
	assert.Equal(t, "TLS 1.3", version)
}

func TestLegacyVersionNegotiation(t *testing.T) {
	results, tls := testInit()

	// First, a client hello
	reqData, err := hex.DecodeString(rawClientHello)
	assert.Nil(t, err)
	tcpTuple := testTCPTuple()
	req := protos.Packet{Payload: reqData}
	var private protos.ProtocolData

	private = tls.Parse(&req, tcpTuple, 0, private)
	assert.NotNil(t, private)
	assert.Empty(t, results.events)

	// Then a server hello + change cypher spec
	reqData, err = hex.DecodeString(rawServerHello + rawChangeCipherSpec)
	req = protos.Packet{Payload: reqData}
	private = tls.Parse(&req, tcpTuple, 1, private)
	assert.NotNil(t, private)
	assert.Empty(t, results.events)

	// Then a change cypher spec from the client
	reqData, err = hex.DecodeString(rawChangeCipherSpec)
	req = protos.Packet{Payload: reqData}
	private = tls.Parse(&req, tcpTuple, 0, private)
	assert.NotNil(t, private)
	assert.NotEmpty(t, results.events)

	iVersion, err := results.events[0].Fields.GetValue("tls.version")
	assert.Nil(t, err)

	version, ok := iVersion.(string)
	assert.True(t, ok)
	assert.Equal(t, "TLS 1.2", version)
}
