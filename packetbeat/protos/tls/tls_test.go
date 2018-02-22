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
)

type eventStore struct {
	events []beat.Event
}

const (
	expectedClientHello = `{"dst":{"IP":"192.168.0.2","Port":27017,"Name":"","Cmdline":"","Proc":""},"server":"example.org","src":{"IP":"192.168.0.1","Port":6512,"Name":"","Cmdline":"","Proc":""},"status":"Error","tls":{"client_certificate_requested":false,"client_hello":{"extensions":{"_unparsed_":["renegotiation_info","23","status_request","18","30032"],"application_layer_protocol_negotiation":["h2","http/1.1"],"ec_points_formats":["uncompressed"],"server_name_indication":["example.org"],"session_ticket":"","signature_algorithms":["ecdsa_secp256r1_sha256","rsa_pss_sha256","rsa_pkcs1_sha256","ecdsa_secp384r1_sha384","rsa_pss_sha384","rsa_pkcs1_sha384","rsa_pss_sha512","rsa_pkcs1_sha512","rsa_pkcs1_sha1"],"supported_groups":["x25519","secp256r1","secp384r1"]},"supported_ciphers":["TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256","TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256","TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384","TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384","TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256","TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256","TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA","TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA","TLS_RSA_WITH_AES_128_GCM_SHA256","TLS_RSA_WITH_AES_256_GCM_SHA384","TLS_RSA_WITH_AES_128_CBC_SHA","TLS_RSA_WITH_AES_256_CBC_SHA","TLS_RSA_WITH_3DES_EDE_CBC_SHA"],"supported_compression_methods":["NULL"],"version":"3.3"},"fingerprints":{"ja3":{"hash":"94c485bca29d5392be53f2b8cf7f4304","str":"771,49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53-10,65281-0-23-35-13-5-18-16-30032-11-10,29-23-24,0"}},"handshake_completed":false,"resumed":false},"type":"tls"}`
	expectedServerHello = `{"extensions":{"_unparsed_":["renegotiation_info","status_request"],"application_layer_protocol_negotiation":["h2"],"ec_points_formats":["uncompressed","ansiX962_compressed_prime","ansiX962_compressed_char2"],"session_ticket":""},"selected_cipher":"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256","selected_compression_method":"NULL","version":"3.3"}`
)

func (e *eventStore) publish(event beat.Event) {
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
		SrcIP:    net.IPv4(192, 168, 0, 1), DstIP: net.IPv4(192, 168, 0, 2),
		SrcPort: 6512, DstPort: 27017,
	}
	t.ComputeHashebles()
	return t
}

func TestPlugin(t *testing.T) {
	_, plugin := testInit()
	assert.NotNil(t, plugin)
	assert.Empty(t, plugin.GetPorts())
	assert.Equal(t, protos.DefaultTransactionExpiration, plugin.ConnectionTimeout())
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

	reqData, err := hex.DecodeString(
		"16030100c2010000be03033367dfae0d46ec0651e49cca2ae47317e8989df710" +
			"ee7570a88b9a7d5d56b3af00001c3a3ac02bc02fc02cc030cca9cca8c013c014" +
			"009c009d002f0035000a01000079dada0000ff0100010000000010000e00000b" +
			"6578616d706c652e6f72670017000000230000000d0014001204030804040105" +
			"0308050501080606010201000500050100000000001200000010000e000c0268" +
			"3208687474702f312e3175500000000b00020100000a000a00086a6a001d0017" +
			"0018aaaa000100")

	assert.Nil(t, err)
	tcpTuple := testTCPTuple()
	req := protos.Packet{Payload: reqData}
	var private protos.ProtocolData

	private = tls.Parse(&req, tcpTuple, 0, private)
	tls.GapInStream(tcpTuple, 0, 1024, private)

	assert.Len(t, results.events, 1)
	event := results.events[0]
	// Remove responsetime (but fail if not present) so that the test
	// does not depend on execution speed
	assert.NoError(t, event.Fields.Delete("responsetime"))
	b, err := json.Marshal(event.Fields)
	assert.Nil(t, err)
	assert.Equal(t, expectedClientHello, string(b))
}

func TestServerHello(t *testing.T) {
	results, tls := testInit()

	reqData, err := hex.DecodeString(
		"160303004a0200004603037806e1be0c363bcc1fe14a906d1ff1b11dc5369d91" +
			"c631ed660d6c0f156f420700c02f00001eff01000100000b0004030001020023" +
			"000000050000001000050003026832")

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
	// Remove responsetime (but fail if not present) so that the test
	// does not depend on execution speed
	assert.NoError(t, event.Delete("responsetime"))
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
	reqData, err = hex.DecodeString("1403030000")
	req = protos.Packet{Payload: reqData}
	private = tls.Parse(&req, tcpTuple, 0, private)
	assert.NotNil(t, private)
	assert.Empty(t, results.events)

	// And the corresponding one on the other direction
	private = tls.Parse(&req, tcpTuple, 1, private)
	assert.NotNil(t, private)
	assert.NotEmpty(t, results.events)

}
