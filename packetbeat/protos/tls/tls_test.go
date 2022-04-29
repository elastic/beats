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
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/packetbeat/procs"
	"github.com/elastic/beats/v7/packetbeat/protos"
	"github.com/elastic/beats/v7/packetbeat/publish"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type eventStore struct {
	events []beat.Event
}

const (
	expectedClientHello = `{"client":{"ip":"192.168.0.1","port":6512},"destination":{"domain":"example.org","ip":"192.168.0.2","port":27017},"event":{"category":["network"],"dataset":"tls","kind":"event","type":["connection","protocol"]},"network":{"community_id":"1:jKfewJN/czjTuEpVvsKdYXXiMzs=","direction":"unknown","protocol":"tls","transport":"tcp","type":"ipv4"},"related":{"ip":["192.168.0.1","192.168.0.2"]},"server":{"domain":"example.org","ip":"192.168.0.2","port":27017},"source":{"ip":"192.168.0.1","port":6512},"status":"Error","tls":{"client":{"ja3":"94c485bca29d5392be53f2b8cf7f4304","server_name":"example.org","supported_ciphers":["TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256","TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256","TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384","TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384","TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256","TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256","TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA","TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA","TLS_RSA_WITH_AES_128_GCM_SHA256","TLS_RSA_WITH_AES_256_GCM_SHA384","TLS_RSA_WITH_AES_128_CBC_SHA","TLS_RSA_WITH_AES_256_CBC_SHA","TLS_RSA_WITH_3DES_EDE_CBC_SHA"]},"detailed":{"client_certificate_requested":false,"client_hello":{"extensions":{"_unparsed_":["renegotiation_info","23","18","30032"],"application_layer_protocol_negotiation":["h2","http/1.1"],"ec_points_formats":["uncompressed"],"server_name_indication":["example.org"],"session_ticket":"","signature_algorithms":["ecdsa_secp256r1_sha256","rsa_pss_sha256","rsa_pkcs1_sha256","ecdsa_secp384r1_sha384","rsa_pss_sha384","rsa_pkcs1_sha384","rsa_pss_sha512","rsa_pkcs1_sha512","rsa_pkcs1_sha1"],"status_request":{"request_extensions":0,"responder_id_list_length":0,"type":"ocsp"},"supported_groups":["x25519","secp256r1","secp384r1"]},"random":"3367dfae0d46ec0651e49cca2ae47317e8989df710ee7570a88b9a7d5d56b3af","supported_compression_methods":["NULL"],"version":"3.3"},"version":"TLS 1.2"},"established":false,"resumed":false,"version":"1.2","version_protocol":"tls"},"type":"tls"}`
	expectedServerHello = `{"extensions":{"_unparsed_":["renegotiation_info"],"application_layer_protocol_negotiation":["h2"],"ec_points_formats":["uncompressed","ansiX962_compressed_prime","ansiX962_compressed_char2"],"session_ticket":"","status_request":{"response":true}},"random":"7806e1be0c363bcc1fe14a906d1ff1b11dc5369d91c631ed660d6c0f156f4207","selected_compression_method":"NULL","version":"3.3"}`
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
	publish.MarshalPacketbeatFields(&event, nil, nil)
	e.events = append(e.events, event)
}

// Helper function returning a TLS module that can be used
// in tests. It publishes the transactions in the results structure.
func testInit() (*eventStore, *tlsPlugin) {
	logp.TestingSetup(logp.WithSelectors("tls", "tlsdetailed"))

	results := &eventStore{}
	tls, err := New(true, results.publish, procs.ProcessesWatcher{}, nil)
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
	assert.NoError(t, err)
	tcpTuple := testTCPTuple()
	req := protos.Packet{Payload: reqData}
	var private protos.ProtocolData

	private = tls.Parse(&req, tcpTuple, 0, private)
	tls.ReceivedFin(tcpTuple, 0, private)

	assert.Len(t, results.events, 1)
	event := results.events[0]
	_, ok := event.Fields["tls"]
	assert.True(t, ok)
	alertsIf, err := event.GetValue("tls.detailed.alerts")
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	alerts := alertsIf.([]mapstr.M)
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
	assert.NoError(t, err)
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

	assert.NoError(t, err)
	tcpTuple := testTCPTuple()
	req := protos.Packet{Payload: reqData}
	var private protos.ProtocolData

	private = tls.Parse(&req, tcpTuple, 0, private)
	tls.GapInStream(tcpTuple, 0, 1024, private)

	assert.Len(t, results.events, 1)
	event := results.events[0]

	b, err := json.Marshal(event.Fields)
	assert.NoError(t, err)
	assert.Equal(t, expectedClientHello, string(b))
}

func TestServerHello(t *testing.T) {
	results, tls := testInit()

	reqData, err := hex.DecodeString(rawServerHello)

	assert.NoError(t, err)
	tcpTuple := testTCPTuple()
	req := protos.Packet{Payload: reqData}
	var private protos.ProtocolData

	private = tls.Parse(&req, tcpTuple, 0, private)
	tls.ReceivedFin(tcpTuple, 0, private)

	assert.Len(t, results.events, 1)
	event := results.events[0]

	hello, err := event.GetValue("tls.detailed.server_hello")
	assert.NoError(t, err)
	b, err := json.Marshal(hello)
	assert.NoError(t, err)
	assert.Equal(t, expectedServerHello, string(b))
}

func TestOCSPStatus(t *testing.T) {
	results, tls := testInit()
	tcpTuple := testTCPTuple()
	var private protos.ProtocolData

	for i, test := range []struct {
		msg  string
		want interface{}
	}{
		// Packets from https://github.com/elastic/beats/issues/29962#issue-1112502582
		//
		// 6	0.017079	TLSv1.2	1516	Server Hello
		{ // TLSv1.2 Record Layer: Handshake Protocol: Server Hello
			msg: "160303005b0200005703032c468bdbb2af2e7bd8e09c90e7992c61a8f468a03bbe74ec311fa33a14a35bfd20fc2b8e95f18fa299253278dd98178a61f75cdddd69f8f1d1e0592e0ce8275af1c02b00000fff0100010000050000000b00020100",
		},
		// 12	0.017170	TLSv1.2	2145	Certificate, Certificate Status, Server Key Exchange, Server Hello Done
		{ // TLSv1.2 Record Layer: Handshake Protocol: Certificate
			msg: "16030311400b00113c0011390005433082053f30820327a0030201020214213e825a875eb349390d11117c6c14f894135fe3300d06092a864886f70d01010b05003060310b3009060355040613024652310f300d060355040a0c064f72616e676531193017060355040b0c10464f52204c414220555345204f4e4c593125302306035504030c1c4f72616e6765204465766963657320504b49205456204c4142204341301e170d3231303630333133333831365a170d3232303630333133333831365a3040310b3009060355040613024652310f300d060355040b0c064f72616e67653120301e06035504030c1773657276657232207465737420504b49205456204c41423059301306072a8648ce3d020106082a8648ce3d03010703420004fc7a2bae61a536e74d4d3138b83e09ef618de444fff8dc8874863e3e1f557f9008e2f777693ba6d7b8fe59f360006d55561e24edfc0608436b0bbf329df1463ca38201da308201d6301d0603551d0e04160414dfd5c4bdbea6b4c89a57fd4a835573d53f51cbff301f0603551d230418301680143ae35027fb6f7337d4eaa8c82139f9627d116bf4300e0603551d0f0101ff0404030205a030130603551d25040c300a06082b06010505070301305c0603551d1f045530533051a04fa04d864b687474703a2f2f706b692d63726c2d6c622e73656375726974792e696e7472616f72616e67652f63726c2f6f72616e67655f646576696365735f706b695f74765f6c61625f63612e63726c3082010f0603551d11048201063082010282102a2e656e61312e6f72616e67652e6672820f2a2e6974762e6f72616e67652e667282192a2e6a6575787476697074762d70702e6f72616e67652e667282162a2e6a6575787476697074762e6f72616e67652e667282102a2e6e7476312e6f72616e67652e667282102a2e6e7476332e6f72616e67652e667282102a2e6e7476342e6f72616e67652e667282102a2e6e7476352e6f72616e67652e6672820f2a2e7067772e6f72616e67652e667282132a2e70702d656e61312e6f72616e67652e667282122a2e70702d6974762e6f72616e67652e667282132a2e70702d6e7476312e6f72616e67652e667282132a2e70702d6e7476322e6f72616e67652e6672300d06092a864886f70d01010b05000382020100b8d7a4819a31204d58fd9ca0364c544f845b4c9e48dde8aadecad7149256e209df85b8b76f86ec62382932f9b0370ef9d257d5112ae3b72bf8d0809fd437a20419a46f82851701f32e0897a659cf5e19079615cb62205f2abf37a24d51ca9ca954d5b5c4485bbb9441109f71a3e6b5fccccfe1e19d97d8403a60615bf1e4ef20c5bb069a2d26e78c5ad18a431934276b9556153a0b7f3c992684a202ea8403c1f59b362389111e5fcb0fa89e40b86300be3949f0e5690abc3314ee7b53f0adb9019ea8e804e48a197f9d77b3c2fb37dbe732a8aed100fb8624fecece4119e1ea045cdc7156765a89aa3d1228f062eb5f109ac25afc63fa8948be9087eeb6c8ebe807b944230cba453aec493054c4df4d5c34fc21d8739a6d40ddc36c95861f207b4dfcdd97224473a5784120237831b86f398f62205945a7befca9f60c2c63578b2e9870723b4c5933c70317edb1071d38ccdd2f91c3c50dc90906ccf14ecbb9b394cd471f33a92d6f210b994b7c085abdc1dd789a46eac33503725f25376a5b438cf8d6dbb2b07ee128f3be21d50bdfb052271d079f4ccae174509a31ac1dfe2a483dce8eb624c181c616a497414f6616c21b8fd24e08aadd2c9c43944df5088e2bdbf121649ca1e405e1e95695d52afa1c265b123344a9f5594b661e7d3406b0f6d60c7f776a9723bcec995f4b4da3e6d42dc446b6a33904b7a56f74ba5301000626308206223082040aa00302010202121121e97d5d37348c572c555a3a59b7b65d2b300d06092a864886f70d01010b0500305e310b3009060355040613024652310f300d060355040a0c064f72616e676531193017060355040b0c10464f52204c414220555345204f4e4c593123302106035504030c1a4f72616e6765204465766963657320526f6f74204c4142204341301e170d3230303330343039303030305a170d3335303330343039303030305a3060310b3009060355040613024652310f300d060355040a0c064f72616e676531193017060355040b0c10464f52204c414220555345204f4e4c593125302306035504030c1c4f72616e6765204465766963657320504b49205456204c414220434130820222300d06092a864886f70d01010105000382020f003082020a0282020100e68a8c1fa89076aa12abae68b498f64d8eda23ce72669459578dacc564c20f34980b638bce470e70d9d93eadeea76242cbe56aacd6119f979dbedcd6226848770cf434d454ff04e122c9cdb3c15d973b79d24a368c241b9708bd16494ade0277ad8bcedc28ed54948a5c0a002f21e19e79a81597816a89acc47d7e3d77a81022ca212ca714febe385154d198121dfcad5ba0c1e52629193453f94bdf8e1e014558b73541044ff4693c1aca2164c56f0903fca333bfb226ae26bbb31ad36aacaedb2c7f47516d4e9a254c1fd383f4f2f9183851b97cc2d234a753cc96f9ebbb444d0c5a861150c19a8d065e6d6969973dcda3ab75ccfa3dacfdfe8a7052a50c5b44cb934af9ecc1ee7040af662bc4706ecee22de1af540ea7afd1c10dd75fa0bfbcfc92713402d6feebec629d55b3fa798adf01f5784d649d3f33aff0d70d7280f6ec46953cc5e7b54187959eb01d5a3e93650816d5a838282196cc5b11c866528dd7e4292b7cd19552e0d8840edf2006757ba84e6f8e77128ba4e176600ffbbd8aa27296b9e19854da4280e94a33c5e8ae5e26e60e4e2f078ba8bdacf785245db0b5874536069fae15edf0cd64d6979cd0610631dfad56f942aa08d4ea68f836417468ecb1b3a51483263c99c06111c26e4a6fcddfaa2115632034a8b38ba3d21a32bf7a297e589447a9566ae47bcaf2c94f6298dd11c64e016d492d9b6a3a630203010001a381d73081d4300f0603551d130101ff040530030101ff300e0603551d0f0101ff04040302010630590603551d1f04523050304ea04ca04a8648687474703a2f2f706b692d63726c2d6c622e73656375726974792e696e7472616f72616e67652f63726c2f6f72616e6765646576696365735f726f6f745f6c61625f63612e63726c30160603551d20040f300d300b06092a817a01100c070501301d0603551d0e041604143ae35027fb6f7337d4eaa8c82139f9627d116bf4301f0603551d23041830168014e3caae493099e865158a46bfd0da511e834e8d9d300d06092a864886f70d01010b0500038202010087ea4587cba724f8463933bca424e98ac5a3bf70180b4c938b02c22bbce9050d6fdd563afc1841029f296d5f1df1ebd3f2dfa9a4014b3226aa38d4939a3d48c7a6a28740bac23ade3b35b8709cf6d42404431c96d36d30646aa54d251515fd602692621e974bbfcac96c50d7dc31a9466a28383916ddcef16eb1686ca0885c5ff3c0984312e6f5cfb406cbda14219040334b69c77a153505889a12bf79102b9c858947b8e7f7d7812d437c5127fc14ea34f83e45957148d38da502c1c40a22d10412eba6bd2003cd5939c996dad45713c55694388a081dcc69a3204ff6a093c6116c233227faa9912a9887d54ec5955cc2a1c96490a75995dd9c38cc2d931b231254244455a1a65d8d991c3181021f96ebd5fd8cf6f0ea5da3f87564316e65a31ae563af5e309040f944e2e84582766f63bd3271ac8171ba7c9a78a809f99f0b39f74e1adc199ff4d69e275968073564ed5eaed451581d2f8ba0726eb6a9d542c677b0dce9d92e436eb7cdc9de546f7caf144d780a11ecddd5ccbe620c5755b1fc96d8b9143038030e1b73479b8e4a103811bf3df222f9d31355cdc34cf597ca0c7dcbc09d289b8d71e01ee60fb6884a521beca0de9223a2a794b5d2b0759d5ee91564bb5c1533d9c154fc0524cac37a7d7a6e76943f8a8a341594db5a17d193a954cbcc7a062a9a62fc12198d58fb82ad37bebf9d557ecf3853faa3c5eab8330005c7308205c3308203aba0030201020212112151567790fb40c755010ca9169cf4b498300d06092a864886f70d01010b0500305e310b3009060355040613024652310f300d060355040a0c064f72616e676531193017060355040b0c10464f52204c414220555345204f4e4c593123302106035504030c1a4f72616e6765204465766963657320526f6f74204c4142204341301e170d3230303330323137303030305a170d3430303330323137303030305a305e310b3009060355040613024652310f300d060355040a0c064f72616e676531193017060355040b0c10464f52204c414220555345204f4e4c593123302106035504030c1a4f72616e6765204465766963657320526f6f74204c414220434130820222300d06092a864886f70d01010105000382020f003082020a0282020100be7808368bf6af22d8ccd84a6d45044b0d4401c803eb569610cbaa34f0b5ff65889a9e84bf9b7c820445cb31c09a39304dd00fbfb0b93bf4fb85ec5642e5356198d2841774527ef9d9e87a22767b33972f05daef46524b316aafba8df1fd29889cb5e97b136588cee7de4cebc960c576713953dbeb0a373fd4dcd438b650281c0df79705433898905ddae751a07c867da602189d111b734df3cffa2300c4fcc6419579d3b412959e0730664f5c5f8d143e9c8a049ec9c596e7373fc7128edf179db220bad9622c34cfa8358b95435b9a365a35a8c0906b4f7145b4c9999f9ed6ebabe244f145b7864742ed641c8225781a1b8e55629e21c71bdf7f3700e462499cf604d1ab5532da3c4892f3fbd3339993afba93900cffac22aa824c1ba5c2228a67f33a915450a9c93eff40aaf27aa41a0b71158d11363045087a1f44dc971baae25119527f04fef4aae620b43795959735298ceee53c3a269f0fec3dfeed4170a3876af85bcb493d783b55767df7613b30e6e1186fc9e7ba6892feb99d9eeef29f0f15900b233cd734f0768dfbf7820f465ed969f81777c5e475f060f27169d9ef77b977f4de3e37ad9f707e19477b38554e60bbbbea33a87f655758bfddaeb05c73c4908218ff59d4d941087f7dbbd980de52b56d83aa7e966a5e0f810512e2f546d451166bc0e22722323443711037d6e97d4ecb24b934b14faa4a898b8f0203010001a37b3079300f0603551d130101ff040530030101ff300e0603551d0f0101ff04040302010630160603551d20040f300d300b06092a817a01100c070101301d0603551d0e04160414e3caae493099e865158a46bfd0da511e834e8d9d301f0603551d23041830168014e3caae493099e865158a46bfd0da511e834e8d9d300d06092a864886f70d01010b0500038202010085bb2664c92a2d258710c94a529bc2c351971556e01b02a75af91c59de0925c549b52ed7cfaeffd6a6dc96e98f957618c6558f3a7a92b6e7cdf4e8198e1df544021ab9b94b942917fc50e403c4ae8a0b7b75ac150f1e453a3124e4d6e4e6a7fe92d8cb2f6fa248f7d2d64eaf9b2a5dd18179d21d603a06bdbb0e590f51f336a4d80070c6aee65a9fbd74980933f5c6122dbe9a98203e750c4664527dd83404fd5a50ab068c6ff7b9071bee24d5213f73d26cc3709a3c7e34c250e3323ab62707dc7141a7f248322221bee9c393fb97c566b712967c410b2280dd83dd71d048b5a80eee07baa224a004de032791df639eda4d37a3cd9a004df83c0c32af57ffcb1cd5d4ab44b1c2166945668d75a53eacd3913356e857895047a85b998ec71578fd6c01b50863e40b094f47ae72418ad7f3215993b7c081e1912cf61c22698cc110d19d86fba4e075ef831c0f027c576718481558cd2dfe519e40fd8e4751e1b670c727dbfca048dc8de8d3f6351c4467f459440c442e3b99d6153bfa89bc720108f4e10679954bd15d50ef66b44bac370cedc7c6a72a0cb566bf42d96e4933db6e83982a9df9dcf7b1e0951469a5c41d17fe05d5ce908219abe275dafe6aadd421b45b555511e8af78764811afe834f4067cdb2495eba06a2b597ce6bc642afc3a288fd5e7f7d68666afefaae74245d4e435b8e9b15fbcd51dbc129130bb7dbe",
		},
		{ // TLSv1.2 Record Layer: Handshake Protocol: Certificate Status
			msg: "16030306fa160006f6010006f2308206ee0a0100a08206e7308206e306092b0601050507300101048206d4308206d03081efa15930573120301e06035504030c174c4142204465766963657320504b49205456204f43535031153013060355040b0c0c464f52204c4142204f4e4c59310f300d060355040a0c064f72616e6765310b3009060355040613024652180f32303232303131393132343134365a308180307e3069300d060960864801650304020105000420ea0ffda3ea4e2090c00df6b507f9d626c5fa196dbf0bc34f005728e35ba8dcef0420150104f4cf727e5515390fccf0a79b7de0fa5c91da131914776cf9007ba848d80214213e825a875eb349390d11117c6c14f894135fe38000180f32303232303131393132343134365a300d06092a864886f70d01010b0500038201010055d4e44634ea7d88f76d89d212b54c70c4c816f0229708e3b57668b0feea67dd368383c285a26d27b092f737b8168f52a3fe839842ca7d6a98491ad072b554440a01797f89b36f77241d7cd24691e4ba8c8e43db6c0851e8fa8a7aebc1c9b5e0892c072c93d3bae0fc793510dc0fc9522742c9bcb845679c88d64e8f98f54c90ce24078739d36c57ccd5cb3d1f6b3af53b8c5b8fab338e57615a0f3a2d744cc10d1a607c3364291c68a02c5b498699af56add7d4ef6ff647d8933a494688099d2ef1c88fc99e1c3e614336f8ec93aa4b6c3e249be1d241e3fbdf0655ffc6454358aebdd85375a2e582e47bd0d398775ebc0553f8f84d64827efbfbb0be479ca8a08204c6308204c2308204be308202a6a003020102021500982eedb5d4d6a889df086bde709c259700dc5d0b300d06092a864886f70d01010b05003060310b3009060355040613024652310f300d060355040a0c064f72616e676531193017060355040b0c10464f52204c414220555345204f4e4c593125302306035504030c1c4f72616e6765204465766963657320504b49205456204c4142204341301e170d3231303331313135303631355a170d3233303331313135303631355a30573120301e06035504030c174c4142204465766963657320504b49205456204f43535031153013060355040b0c0c464f52204c4142204f4e4c59310f300d060355040a0c064f72616e6765310b300906035504061302465230820122300d06092a864886f70d01010105000382010f003082010a0282010100db897620e1931205f84d3dccb18596575261eeb3e08a2751bf11bb2f8d50bd45daba8de5e409834a3f7db86f3bbdd466db3f26c8a23349f7068f8c4e5c1359ab658627f2cf13698a27a164789dcef0564ce8e38b118b377b21accc56c6f2c436c937162eb6517ee7769a544395efed849e879bbeee4db037afb638efdb982bafb17226ded7551a832bc32d6890fc6405897ff4ba594663c72882b3ee6d4529effd421d172c2cf472899bc2ef411bb99338a2b7d0848b17651d64a45356c7afaf710108a80daf9ffa1875c0ade37fc2610ada1f4ba22b8baa3ed7cb49f15a330ca3af87e45a7d3a6bbca0cf89c953f889ea2d457c18e8a9c0621c05422668ec370203010001a3783076301d0603551d0e04160414d6d0ee3577e004c093c5b86f2606e711391aa02f301f0603551d230418301680143ae35027fb6f7337d4eaa8c82139f9627d116bf4300e0603551d0f0101ff04040302078030130603551d25040c300a06082b06010505070309300f06092b060105050730010504020500300d06092a864886f70d01010b050003820201008c2d84f26d2868aff7af411065c2fb2c1fce32bb4e8dfc8cb1aa421e85c807619146345c63d5786f50fbe1a3a40cf28c444c591fd73b0f29dee46a5e3d2aae64c425b86bdbba47ea561e25f3040357d589ae21081a0bcff524855b4c221740c67980f3fb9527a5488d4c5a94384e3423833c63c414fd4a927ad9e0ad94a561239b0c5cf173ba823a7b23bc5985343309c55613e11a2e0a7a206fbaf81269a05ce5558534a86d54eecf0fa2596bc12e0b75cc1e385d192a69b900ea42c75deee78cc77121c552dd222e4f0a544f70c42ee9068e036d07a5b43b722479e2be4ac8e3b5248db27c0407b9dc07fee2d035cfdfada1b62c4c89ccfb46144d9a1ff598d7f966cee970e7e2b25c93529bb8f9e3af35a5ae3a873bb25bc4853ac026105f9a9544b1eb797c26574e0789c45af53778685ec83261afd82a227a3eac31acea592822268f6909523b2745eb317703b0acd496caba478300dd855b9e399cf2c24ea71bc69f52009273004e8b486ea6033f5bd843a3c0e0daf7887ee39907453b31cc4c2c62b1ef39263518bafe3319d886dfe9ed86c99508f139786e21209181cd4b205a5676cb261bd7ae3132e07d0ff9a7e18d1507555f2d34393f0992985cd9491f1db501e0f176f55e05ad06690844a0323b586c9c85da4f23b644a032d90a0b279d2ce250e056dfc545070e81c624f1c286d3bf539f60c206999199b583",
		},
		{ // TLSv1.2 Record Layer: Handshake Protocol: Server Key Exchange
			msg: "16030300930c00008f030017410496569a8aa5e214e7451b9bb7ba10c4af011cf3de8daf19f4e3a09f81d697696bfa9ebf4e1bb441f5c2adffeda15a92b9f9b588d4731af7cc5d15d7035fe5203304030046304402203e1e19b17b31b843f6be75643108c7e684a449b437432944a80739c2ab3035e302206a680f72e531b5eef08d3d0a078d65d4dbfce153e83f4f62f261bacc36eb7ae6",
		},
		{ // TLSv1.2 Record Layer: Handshake Protocol: Server Hello Done
			msg: "16030300040e000000",
		},
	} {
		reqData, err := hex.DecodeString(test.msg)
		if err != nil {
			t.Fatalf("unexected error decoding input %d: %v", i, err)
		}

		private = tls.Parse(&protos.Packet{Payload: reqData}, tcpTuple, 0, private)
	}
	tls.ReceivedFin(tcpTuple, 0, private)

	if len(results.events) != 1 {
		t.Fatalf("unexected number of results: got:%d want:1", len(results.events))
	}

	want := mapstr.M{
		"client": mapstr.M{
			"ip":   "192.168.0.1",
			"port": int64(6512),
		},
		"event": mapstr.M{
			"dataset": "tls",
			"kind":    "event",
			"category": []string{
				"network",
			},
			"type": []string{
				"connection",
				"protocol",
			},
		},
		"destination": mapstr.M{
			"ip":   "192.168.0.2",
			"port": int64(27017),
		},
		"network": mapstr.M{
			"type":         "ipv4",
			"transport":    "tcp",
			"protocol":     "tls",
			"direction":    "unknown",
			"community_id": "1:jKfewJN/czjTuEpVvsKdYXXiMzs=",
		},
		"related": mapstr.M{
			"ip": []string{
				"192.168.0.1",
				"192.168.0.2",
			},
		},
		"server": mapstr.M{
			"ip":   "192.168.0.2",
			"port": int64(27017),
		},
		"source": mapstr.M{
			"port": int64(6512),
			"ip":   "192.168.0.1",
		},
		"status": "Error",
		"tls": mapstr.M{
			"cipher": "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
			"detailed": mapstr.M{
				"client_certificate_requested": false,
				"ocsp_response":                "successful",
				"server_certificate_chain": []mapstr.M{
					{
						"issuer": mapstr.M{
							"common_name":         "Orange Devices Root LAB CA",
							"country":             "FR",
							"distinguished_name":  "CN=Orange Devices Root LAB CA,OU=FOR LAB USE ONLY,O=Orange,C=FR",
							"organization":        "Orange",
							"organizational_unit": "FOR LAB USE ONLY",
						},
						"not_before":           time.Date(2020, 3, 4, 9, 0, 0, 0, time.UTC),
						"not_after":            time.Date(2035, 3, 4, 9, 0, 0, 0, time.UTC),
						"public_key_algorithm": "RSA",
						"public_key_size":      4096,
						"serial_number":        "1492448539999078269498416841973088004758827",
						"signature_algorithm":  "SHA256-RSA",
						"subject": mapstr.M{
							"common_name":         "Orange Devices PKI TV LAB CA",
							"country":             "FR",
							"distinguished_name":  "CN=Orange Devices PKI TV LAB CA,OU=FOR LAB USE ONLY,O=Orange,C=FR",
							"organization":        "Orange",
							"organizational_unit": "FOR LAB USE ONLY",
						},
						"version_number": 3,
					},
					{
						"issuer": mapstr.M{
							"common_name":         "Orange Devices Root LAB CA",
							"country":             "FR",
							"distinguished_name":  "CN=Orange Devices Root LAB CA,OU=FOR LAB USE ONLY,O=Orange,C=FR",
							"organization":        "Orange",
							"organizational_unit": "FOR LAB USE ONLY",
						},
						"not_after":            time.Date(2040, 3, 2, 17, 0, 0, 0, time.UTC),
						"not_before":           time.Date(2020, 3, 2, 17, 0, 0, 0, time.UTC),
						"public_key_algorithm": "RSA",
						"public_key_size":      4096,
						"serial_number":        "1492246295378596931754418352553114016724120",
						"signature_algorithm":  "SHA256-RSA",
						"subject": mapstr.M{
							"common_name":         "Orange Devices Root LAB CA",
							"country":             "FR",
							"distinguished_name":  "CN=Orange Devices Root LAB CA,OU=FOR LAB USE ONLY,O=Orange,C=FR",
							"organization":        "Orange",
							"organizational_unit": "FOR LAB USE ONLY",
						},
						"version_number": 3,
					},
				},
				"server_hello": mapstr.M{
					"extensions": mapstr.M{
						"_unparsed_": []string{
							"renegotiation_info",
						},
						"ec_points_formats": []string{
							"uncompressed",
						},
						"status_request": mapstr.M{
							"response": true,
						},
					},
					"random":                      "2c468bdbb2af2e7bd8e09c90e7992c61a8f468a03bbe74ec311fa33a14a35bfd",
					"selected_compression_method": "NULL",
					"session_id":                  "fc2b8e95f18fa299253278dd98178a61f75cdddd69f8f1d1e0592e0ce8275af1",
					"version":                     "3.3",
				},
				"version": "TLS 1.2",
			},
			"established": false,
			"resumed":     false,
			"server": mapstr.M{
				"issuer": "CN=Orange Devices PKI TV LAB CA,OU=FOR LAB USE ONLY,O=Orange,C=FR",
				"hash": mapstr.M{
					"sha1": "D8A11028DAD7E34F5D7F6D41DE01743D8B3CE553",
				},
				"not_after":  time.Date(2022, 6, 3, 13, 38, 16, 0, time.UTC),
				"not_before": time.Date(2021, 6, 3, 13, 38, 16, 0, time.UTC),
				"x509": mapstr.M{
					"alternative_names": []string{
						"*.ena1.orange.fr",
						"*.itv.orange.fr",
						"*.jeuxtviptv-pp.orange.fr",
						"*.jeuxtviptv.orange.fr",
						"*.ntv1.orange.fr",
						"*.ntv3.orange.fr",
						"*.ntv4.orange.fr",
						"*.ntv5.orange.fr",
						"*.pgw.orange.fr",
						"*.pp-ena1.orange.fr",
						"*.pp-itv.orange.fr",
						"*.pp-ntv1.orange.fr",
						"*.pp-ntv2.orange.fr",
					},
					"issuer": mapstr.M{
						"common_name":         "Orange Devices PKI TV LAB CA",
						"country":             "FR",
						"distinguished_name":  "CN=Orange Devices PKI TV LAB CA,OU=FOR LAB USE ONLY,O=Orange,C=FR",
						"organization":        "Orange",
						"organizational_unit": "FOR LAB USE ONLY",
					},
					"not_after":            time.Date(2022, 6, 3, 13, 38, 16, 0, time.UTC),
					"not_before":           time.Date(2021, 6, 3, 13, 38, 16, 0, time.UTC),
					"public_key_algorithm": "ECDSA",
					"public_key_size":      256,
					"serial_number":        "189790697042017246339292011338547986350262673379",
					"signature_algorithm":  "SHA256-RSA",
					"subject": mapstr.M{
						"common_name":         "server2 test PKI TV LAB",
						"country":             "FR",
						"distinguished_name":  "CN=server2 test PKI TV LAB,OU=Orange,C=FR",
						"organizational_unit": "Orange",
					},
					"version_number": 3,
				},
				"subject": "CN=server2 test PKI TV LAB,OU=Orange,C=FR",
			},
			"version":          "1.2",
			"version_protocol": "tls",
		},
		"type": "tls",
	}

	got := results.events[0].Fields
	if !cmp.Equal(got, want) {
		t.Errorf("unexpected result: %s", cmp.Diff(got, want))
	}
}

func TestFragmentedHandshake(t *testing.T) {
	results, tls := testInit()

	// First, a full record containing only half of a handshake message
	reqData, err := hex.DecodeString(
		"160301003f010000be03033367dfae0d46ec0651e49cca2ae47317e8989df710" +
			"ee7570a88b9a7d5d56b3af00001c3a3ac02bc02fc02cc030cca9cca8c013c014" +
			"009c009d")

	assert.NoError(t, err)
	tcpTuple := testTCPTuple()
	req := protos.Packet{Payload: reqData}
	var private protos.ProtocolData

	private = tls.Parse(&req, tcpTuple, 0, private)

	// Second, half a record containing the middle part of a handshake
	reqData, err = hex.DecodeString(
		"1603010083002f0035000a01000079dada0000ff0100010000000010000e00000b" +
			"6578616d706c652e6f72670017000000230000000d0014001204030804040105" +
			"0308050501080606010201000500050100000000001200000010000e000c0268")
	assert.NoError(t, err)
	req = protos.Packet{Payload: reqData}
	private = tls.Parse(&req, tcpTuple, 0, private)

	// Third, the final part of the second record, completing the handshake
	reqData, err = hex.DecodeString(
		"3208687474702f312e3175500000000b00020100000a000a00086a6a001d0017" +
			"0018aaaa000100")
	assert.NoError(t, err)
	req = protos.Packet{Payload: reqData}
	private = tls.Parse(&req, tcpTuple, 0, private)

	tls.ReceivedFin(tcpTuple, 0, private)

	assert.Len(t, results.events, 1)
	event := results.events[0]

	b, err := json.Marshal(event.Fields)
	assert.NoError(t, err)
	assert.Equal(t, expectedClientHello, string(b))
}

func TestInterleavedRecords(t *testing.T) {
	results, tls := testInit()

	// First, a full record containing only half of a handshake message
	reqData, err := hex.DecodeString(
		"160301003f010000be03033367dfae0d46ec0651e49cca2ae47317e8989df710" +
			"ee7570a88b9a7d5d56b3af00001c3a3ac02bc02fc02cc030cca9cca8c013c014" +
			"009c009d")

	assert.NoError(t, err)
	tcpTuple := testTCPTuple()
	req := protos.Packet{Payload: reqData}
	var private protos.ProtocolData

	private = tls.Parse(&req, tcpTuple, 0, private)

	// Then two records containing one alert each, merged in a single packet
	reqData, err = hex.DecodeString(
		"1503010002FFFF15030100020101")
	assert.NoError(t, err)
	req = protos.Packet{Payload: reqData}
	private = tls.Parse(&req, tcpTuple, 0, private)

	// And an application data record
	reqData, err = hex.DecodeString(
		"17030100080123456789abcdef")
	assert.NoError(t, err)
	req = protos.Packet{Payload: reqData}
	private = tls.Parse(&req, tcpTuple, 0, private)

	// Then what's missing from the handshake
	reqData, err = hex.DecodeString(
		"1603010083002f0035000a01000079dada0000ff0100010000000010000e00000b" +
			"6578616d706c652e6f72670017000000230000000d0014001204030804040105" +
			"0308050501080606010201000500050100000000001200000010000e000c0268" +
			"3208687474702f312e3175500000000b00020100000a000a00086a6a001d0017" +
			"0018aaaa000100")
	assert.NoError(t, err)
	req = protos.Packet{Payload: reqData}
	private = tls.Parse(&req, tcpTuple, 0, private)

	tls.ReceivedFin(tcpTuple, 0, private)

	assert.Len(t, results.events, 1)
	event := results.events[0]

	// Event contains the client hello
	_, err = event.GetValue("tls.detailed.client_hello")
	assert.NoError(t, err)

	// and the alert
	alerts, err := event.GetValue("tls.detailed.alerts")
	assert.NoError(t, err)

	assert.Len(t, alerts.([]mapstr.M), 2)
}

func TestCompletedHandshake(t *testing.T) {
	results, tls := testInit()

	// First, a certificates record
	reqData, err := hex.DecodeString(certsMsg)

	assert.NoError(t, err)
	tcpTuple := testTCPTuple()
	req := protos.Packet{Payload: reqData}
	var private protos.ProtocolData

	private = tls.Parse(&req, tcpTuple, 0, private)
	assert.NotNil(t, private)
	assert.Empty(t, results.events)

	// Then a change cypher spec message
	reqData, err = hex.DecodeString(rawChangeCipherSpec)
	assert.NoError(t, err)
	req = protos.Packet{Payload: reqData}
	private = tls.Parse(&req, tcpTuple, 0, private)
	assert.NotNil(t, private)
	assert.Empty(t, results.events)

	// And the corresponding one on the other direction
	private = tls.Parse(&req, tcpTuple, 1, private)
	assert.NotNil(t, private)
	assert.NotEmpty(t, results.events)

	// #19039
	// If tls.detailed.client_certificate or
	// tls.detailed.server_certificate
	// are present they are removed in favor of
	// tls.client.x509 and tls.server.x509
	// check if the resultant event indeed has no
	// client_certificate and has the corresponding
	// x509 entries
	assert.Equal(t, true, tls.includeDetailedFields,
		"Turn on includeDetailedFields or the following tests will fail")
	event := results.events[0]
	flatEvent := event.Fields.Flatten()
	_, err = flatEvent.GetValue("tls.detailed.client_certificate.subject.common_name")
	assert.NotNil(t, err, "Expected tls.detailed.client_certificate to be removed")
	// check the existence of a key
	_, err = flatEvent.GetValue("tls.client.x509.version_number")
	assert.Nil(t, err)
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
	assert.NoError(t, err)
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
	assert.NoError(t, err)
	req = protos.Packet{Payload: reqData}
	private = tls.Parse(&req, tcpTuple, 1, private)
	assert.NotNil(t, private)
	assert.Empty(t, results.events)

	// Then a change cypher spec from the client
	reqData, err = hex.DecodeString(rawChangeCipherSpec)
	assert.NoError(t, err)
	req = protos.Packet{Payload: reqData}
	private = tls.Parse(&req, tcpTuple, 0, private)
	assert.NotNil(t, private)
	assert.NotEmpty(t, results.events)

	for key, expected := range map[string]string{
		"tls.version":          "1.3",
		"tls.version_protocol": "tls",
		"tls.detailed.version": "TLS 1.3",
	} {
		version, err := results.events[0].Fields.GetValue(key)
		assert.NoError(t, err)
		assert.Equal(t, expected, version)
	}
}

func TestLegacyVersionNegotiation(t *testing.T) {
	results, tls := testInit()

	// First, a client hello
	reqData, err := hex.DecodeString(rawClientHello)
	assert.NoError(t, err)
	tcpTuple := testTCPTuple()
	req := protos.Packet{Payload: reqData}
	var private protos.ProtocolData

	private = tls.Parse(&req, tcpTuple, 0, private)
	assert.NotNil(t, private)
	assert.Empty(t, results.events)

	// Then a server hello + change cypher spec
	reqData, err = hex.DecodeString(rawServerHello + rawChangeCipherSpec)
	assert.NoError(t, err)
	req = protos.Packet{Payload: reqData}
	private = tls.Parse(&req, tcpTuple, 1, private)
	assert.NotNil(t, private)
	assert.Empty(t, results.events)

	// Then a change cypher spec from the client
	reqData, err = hex.DecodeString(rawChangeCipherSpec)
	assert.NoError(t, err)
	req = protos.Packet{Payload: reqData}
	private = tls.Parse(&req, tcpTuple, 0, private)
	assert.NotNil(t, private)
	assert.NotEmpty(t, results.events)

	for key, expected := range map[string]string{
		"tls.version":          "1.2",
		"tls.version_protocol": "tls",
		"tls.detailed.version": "TLS 1.2",
	} {
		version, err := results.events[0].Fields.GetValue(key)
		assert.NoError(t, err)
		assert.Equal(t, expected, version)
	}
}
