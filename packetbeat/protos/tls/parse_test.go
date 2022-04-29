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
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common/streambuf"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	certsMsg = "1603030ab80b000ab4000ab10005f6308205f2308204daa00302010202100e64c5" +
		"fbc236ade14b172aeb41c78cb0300d06092a864886f70d01010b05003070310b300906035" +
		"504061302555331153013060355040a130c446967694365727420496e6331193017060355" +
		"040b13107777772e64696769636572742e636f6d312f302d0603550403132644696769436" +
		"5727420534841322048696768204173737572616e636520536572766572204341301e170d" +
		"3135313130333030303030305a170d3138313132383132303030305a3081a5310b3009060" +
		"355040613025553311330110603550408130a43616c69666f726e69613114301206035504" +
		"07130b4c6f7320416e67656c6573313c303a060355040a1333496e7465726e657420436f7" +
		"2706f726174696f6e20666f722041737369676e6564204e616d657320616e64204e756d62" +
		"65727331133011060355040b130a546563686e6f6c6f6779311830160603550403130f777" +
		"7772e6578616d706c652e6f726730820122300d06092a864886f70d01010105000382010f" +
		"003082010a0282010100b340962f61633e25c197ad6545fbef1342b32c9986f4b5800b76d" +
		"c06382c1fa362555a3676deae5dfce2e5b4e6ec5dcaeecadf5016242ceefc9ab68cf6a8b3" +
		"ac7a087b2a1fad5fe7fa965925ab90b0f8c23f13042674680fc6782a958a5f42f20eed52a" +
		"6eb682389e543f86d121b62427ba805f359c45ed6c5cc46c04b19b92d4a7172241e5e5544" +
		"93ab78a1474da5dc075a9c67f41168122fd32871bcad72053c1675d4f87258ba19f1dc09e" +
		"df118c6922f7dbc160b378d8aef1b6f4fb9e07a5498bfb5b6cfbbaa937f0a7f1f56eba9d8" +
		"e1dbd539d8185bd1f26433d0d6c423ff09ab6d71cedacfc1179c23be2caf2f921c3f90088" +
		"958f2b1e1106f832ef79f0203010001a38202503082024c301f0603551d23041830168014" +
		"5168ff90af0207753cccd9656462a212b859723b301d0603551d0e04160414a64f601e1f2" +
		"dd1e7f123a02a9516e4e89aea6e483081810603551d11047a3078820f7777772e6578616d" +
		"706c652e6f7267820b6578616d706c652e636f6d820b6578616d706c652e656475820b657" +
		"8616d706c652e6e6574820b6578616d706c652e6f7267820f7777772e6578616d706c652e" +
		"636f6d820f7777772e6578616d706c652e656475820f7777772e6578616d706c652e6e657" +
		"4300e0603551d0f0101ff0404030205a0301d0603551d250416301406082b060105050703" +
		"0106082b0601050507030230750603551d1f046e306c3034a032a030862e687474703a2f2" +
		"f63726c332e64696769636572742e636f6d2f736861322d68612d7365727665722d67342e" +
		"63726c3034a032a030862e687474703a2f2f63726c342e64696769636572742e636f6d2f7" +
		"36861322d68612d7365727665722d67342e63726c304c0603551d20044530433037060960" +
		"86480186fd6c0101302a302806082b06010505070201161c68747470733a2f2f7777772e6" +
		"4696769636572742e636f6d2f4350533008060667810c01020230818306082b0601050507" +
		"010104773075302406082b060105050730018618687474703a2f2f6f6373702e646967696" +
		"36572742e636f6d304d06082b060105050730028641687474703a2f2f636163657274732e" +
		"64696769636572742e636f6d2f446967694365727453484132486967684173737572616e6" +
		"36553657276657243412e637274300c0603551d130101ff04023000300d06092a864886f7" +
		"0d01010b0500038201010084a89a11a7d8bd0b267e52247bb2559dea30895108876fa9ed1" +
		"0ea5b3e0bc72d47044edd4537c7cabc387fb66a1c65426a73742e5a9785d0cc92e22e3889" +
		"d90d69fa1b9bf0c16232654f3d98dbdad666da2a5656e31133ece0a5154cea7549f45def1" +
		"5f5121ce6f8fc9b04214bcf63e77cfcaadcfa43d0c0bbf289ea916dcb858e6a9fc8f994bf" +
		"553d4282384d08a4a70ed3654d3361900d3f80bf823e11cb8f3fce7994691bf2da4bc897b" +
		"811436d6a2532b9b2ea2262860da3727d4fea573c653b2f2773fc7c16fb0d03a40aed01ab" +
		"a423c68d5f8a21154292c034a220858858988919b11e20ed13205c045564ce9db365fdf68" +
		"f5e99392115e271aa6a88820004b5308204b130820399a003020102021004e1e7a4dc5cf2" +
		"f36dc02b42b85d159f300d06092a864886f70d01010b0500306c310b30090603550406130" +
		"2555331153013060355040a130c446967694365727420496e6331193017060355040b1310" +
		"7777772e64696769636572742e636f6d312b3029060355040313224469676943657274204" +
		"8696768204173737572616e636520455620526f6f74204341301e170d3133313032323132" +
		"303030305a170d3238313032323132303030305a3070310b3009060355040613025553311" +
		"53013060355040a130c446967694365727420496e6331193017060355040b13107777772e" +
		"64696769636572742e636f6d312f302d06035504031326446967694365727420534841322" +
		"048696768204173737572616e63652053657276657220434130820122300d06092a864886" +
		"f70d01010105000382010f003082010a0282010100b6e02fc22406c86d045fd7ef0a6406b" +
		"27d22266516ae42409bcedc9f9f76073ec330558719b94f940e5a941f5556b4c2022aafd0" +
		"98ee0b40d7c4d03b72c8149eef90b111a9aed2c8b8433ad90b0bd5d595f540afc81ded4d9" +
		"c5f57b786506899f58adad2c7051fa897c9dca4b182842dc6ada59cc71982a6850f5e4458" +
		"2a378ffd35f10b0827325af5bb8b9ea4bd51d027e2dd3b4233a30528c4bb28cc9aac2b230" +
		"d78c67be65e71b74a3e08fb81b71616a19d23124de5d79208ac75a49cbacd17b21e443565" +
		"7f532539d11c0a9a631b199274680a37c2c25248cb395aa2b6e15dc1dda020b821a293266" +
		"f144a2141c7ed6d9bf2482ff303f5a26892532f5ee30203010001a3820149308201453012" +
		"0603551d130101ff040830060101ff020100300e0603551d0f0101ff040403020186301d0" +
		"603551d250416301406082b0601050507030106082b06010505070302303406082b060105" +
		"0507010104283026302406082b060105050730018618687474703a2f2f6f6373702e64696" +
		"769636572742e636f6d304b0603551d1f044430423040a03ea03c863a687474703a2f2f63" +
		"726c342e64696769636572742e636f6d2f4469676943657274486967684173737572616e6" +
		"3654556526f6f7443412e63726c303d0603551d200436303430320604551d2000302a3028" +
		"06082b06010505070201161c68747470733a2f2f7777772e64696769636572742e636f6d2" +
		"f435053301d0603551d0e041604145168ff90af0207753cccd9656462a212b859723b301f" +
		"0603551d23041830168014b13ec36903f8bf4701d498261a0802ef63642bc3300d06092a8" +
		"64886f70d01010b05000382010100188a958903e66ddf5cfc1d68ea4a8f83d6512f8d6b44" +
		"169eac63f5d26e6c84998baa8171845bed344eb0b7799229cc2d806af08e20e179a4fe034" +
		"713eaf586ca59717df404966bd359583dfed331255c183884a3e69f82fd8c5b98314ecd78" +
		"9e1afd85cb49aaf2278b9972fc3eaad5410bdad536a1bf1c6e47497f5ed9487c03d9fd8b4" +
		"9a098264240ebd69211a4640a5754c4f51dd6025e6baceec4809a1272fa5693d7ffbf3085" +
		"0630bf0b7f4eff57059d24ed85c32bfba675a8ac2d16ef7d7927b2ebc29d0b07eaaa85d30" +
		"1a3202841594328d281e3aaf6ec7b3b77b640628005414501ef17063edec0339b67d3612e" +
		"7287e469fc120057401e70f51ec9b4"
)

func sBuf(t *testing.T, hexString string) *streambuf.Buffer {
	bytes, err := hex.DecodeString(hexString)
	assert.NoError(t, err)
	return streambuf.New(bytes)
}

func mapGet(t *testing.T, m mapstr.M, key string) interface{} {
	value, err := m.GetValue(key)
	assert.NoError(t, err)
	return value
}

func TestParseRecordHeader(t *testing.T) {
	if testing.Verbose() {
		isDebug = true
		logp.TestingSetup(logp.WithSelectors("tls", "tlsdetailed"))
	}

	_, err := readRecordHeader(sBuf(t, ""))
	assert.Error(t, err)
	_, err = readRecordHeader(sBuf(t, "11"))
	assert.Error(t, err)
	_, err = readRecordHeader(sBuf(t, "1122"))
	assert.Error(t, err)
	_, err = readRecordHeader(sBuf(t, "112233"))
	assert.Error(t, err)
	_, err = readRecordHeader(sBuf(t, "11223344"))
	assert.Error(t, err)
	header, err := readRecordHeader(sBuf(t, "1103024455"))
	assert.NoError(t, err)
	assert.Equal(t, recordType(0x11), header.recordType)
	assert.Equal(t, "TLS 1.1", header.version.String())
	assert.Equal(t, uint16(0x4455), header.length)
	assert.Equal(t, "recordHeader type[17] version[TLS 1.1] length[17493]", header.String())
	assert.True(t, header.isValid())
	header.version.major = 2
	assert.False(t, header.isValid())
}

func TestParseHandshakeHeader(t *testing.T) {
	if testing.Verbose() {
		isDebug = true
		logp.TestingSetup(logp.WithSelectors("tls", "tlsdetailed"))
	}

	_, err := readHandshakeHeader(sBuf(t, ""))
	assert.Error(t, err)
	_, err = readHandshakeHeader(sBuf(t, "11"))
	assert.Error(t, err)
	_, err = readHandshakeHeader(sBuf(t, "112233"))
	assert.Error(t, err)
	_, err = readHandshakeHeader(sBuf(t, "112233"))
	assert.Error(t, err)
	header, err := readHandshakeHeader(sBuf(t, "11223344"))
	assert.NoError(t, err)
	assert.Equal(t, handshakeType(0x11), header.handshakeType)
	assert.Equal(t, 0x223344, header.length)
}

func TestParserParse(t *testing.T) {
	if testing.Verbose() {
		isDebug = true
		logp.TestingSetup(logp.WithSelectors("tls", "tlsdetailed"))
	}

	parser := &parser{}
	// An incomplete record header is ok but not complete
	assert.Equal(t, resultMore, parser.parse(sBuf(t, "14")))

	// A complete record header with missing payload is ok but not complete
	assert.Equal(t, resultMore, parser.parse(sBuf(t, "1403030001")))

	// Full record of type changeCypherSpec
	assert.Equal(t, resultEncrypted, parser.parse(sBuf(t, "1403030001FF")))

	// Unknown record type is ignored
	assert.Equal(t, resultOK, parser.parse(sBuf(t, "FF0303000155")))

	// Full record of helloRequest
	assert.Equal(t, resultOK, parser.parse(sBuf(t, "160303000400000000")))

	// Certificate request
	assert.Equal(t, resultOK, parser.parse(sBuf(t, "16030300040d000000")))
	assert.True(t, parser.certRequested)
}

func TestParserHello(t *testing.T) {
	if testing.Verbose() {
		isDebug = true
		logp.TestingSetup(logp.WithSelectors("tls", "tlsdetailed"))
	}

	parser := &parser{}
	// An incomplete handshake header is ok and complete
	assert.Equal(t, resultOK, parser.parse(sBuf(t, "160301000502000002FF")))
	assert.Equal(t, 5, parser.handshakeBuf.Len())

	// Completing the bogus handshake with another record
	assert.Equal(t, resultFailed, parser.parse(sBuf(t, "1603010001AA")))
	assert.Equal(t, 0, parser.handshakeBuf.Len())

	// Hanshake message length limit
	assert.Equal(t, resultFailed, parser.parse(sBuf(t, "160301000502040000FF")))
	assert.Equal(t, 0, parser.handshakeBuf.Len())

	// Correct server hello, with missing extensions
	parser.hello = nil
	result := parser.parse(sBuf(t,
		"160301002d02000029"+
			"030312345678"+ // 3.3 + timestamp
			"FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"+ // random
			"03abcdef"+ // Session ID
			"C00A01")) // cipher + compression
	assert.Equal(t, resultOK, result)
	assert.Equal(t, 0, parser.handshakeBuf.Len())
	assert.NotNil(t, parser.hello)

	helloMap := parser.hello.toMap()
	assert.Equal(t, "3.3", mapGet(t, helloMap, "version").(string))
	assert.Equal(t, "TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA", parser.hello.selected.cipherSuite.String())
	assert.Equal(t, "DEFLATE", mapGet(t, helloMap, "selected_compression_method"))
	assert.Equal(t, "abcdef", parser.hello.sessionID)
	hasExts := parser.hello.extensions.Parsed != nil
	assert.False(t, hasExts)

	// Correct server hello, with empty extensions
	parser.hello = nil
	result = parser.parse(sBuf(t,
		"160301002f0200002b"+
			"030312345678"+ // 3.3 + timestamp
			"FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"+ // random
			"03abcdef"+ // Session ID
			"C00A010000")) // cipher + compression
	assert.Equal(t, resultOK, result)
	assert.Equal(t, 0, parser.handshakeBuf.Len())
	assert.NotNil(t, parser.hello)

	helloMap = parser.hello.toMap()
	assert.Equal(t, "3.3", mapGet(t, helloMap, "version").(string))
	assert.Equal(t, "TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA", parser.hello.selected.cipherSuite.String())
	assert.Equal(t, "DEFLATE", mapGet(t, helloMap, "selected_compression_method"))
	assert.Equal(t, "abcdef", parser.hello.sessionID)
	hasExts = parser.hello.extensions.Parsed != nil
	assert.False(t, hasExts)

	// Server hello with bad version
	parser.hello = nil
	result = parser.parse(sBuf(t,
		"160301002f0200002b"+
			"F30312345678"+ // 3.3 + timestamp
			"FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"+ // random
			"03abcdef"+ // Session ID
			"C00A010000")) // cipher + compression
	assert.Equal(t, resultFailed, result)
	assert.Equal(t, 0, parser.handshakeBuf.Len())
	assert.Nil(t, parser.hello)

	// Server hello with session ID out of bounds
	parser.hello = nil
	result = parser.parse(sBuf(t,
		"160301004d02000049"+
			"030312345678"+ // 3.3 + timestamp
			"FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"+ // random
			"21eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"+ // Session ID (33 byte)
			"C00A010000")) // cipher + compression
	assert.Equal(t, resultFailed, result)
	assert.Equal(t, 0, parser.handshakeBuf.Len())
	assert.Nil(t, parser.hello)
}

func TestCertificates(t *testing.T) {
	parser := &parser{}

	// A certificates message with two certificates
	assert.Equal(t, resultOK, parser.parse(sBuf(t, certsMsg)))
	assert.NotNil(t, parser.certificates)
	assert.Len(t, parser.certificates, 2)

	c := parser.certificates
	assert.Equal(t, "www.example.org", c[0].Subject.CommonName)
	assert.Equal(t, "DigiCert SHA2 High Assurance Server CA", c[1].Subject.CommonName)
	assert.Equal(t, c[0].Issuer.CommonName, c[1].Subject.CommonName)
	assert.Nil(t, c[0].CheckSignatureFrom(c[1]))

	expected := map[string]string{
		"not_after":                   "2018-11-28 12:00:00 +0000 UTC",
		"not_before":                  "2015-11-03 00:00:00 +0000 UTC",
		"public_key_algorithm":        "RSA",
		"public_key_size":             "2048",
		"serial_number":               "19132437207909210467858529073412672688",
		"signature_algorithm":         "SHA256-RSA",
		"issuer.common_name":          "DigiCert SHA2 High Assurance Server CA",
		"issuer.country":              "US",
		"issuer.organization":         "DigiCert Inc",
		"issuer.organizational_unit":  "www.digicert.com",
		"subject.common_name":         "www.example.org",
		"subject.country":             "US",
		"subject.locality":            "Los Angeles",
		"subject.organization":        "Internet Corporation for Assigned Names and Numbers",
		"subject.organizational_unit": "Technology",
	}

	certMap := certToMap(c[0])

	for key, expectedValue := range expected {
		value, err := certMap.GetValue(key)
		assert.Nil(t, err, key)
		if t, ok := value.(time.Time); ok {
			value = t.String()
		} else if n, ok := value.(int); ok {
			value = strconv.Itoa(n)
		}
		assert.Equal(t, expectedValue, value, key)
	}
	san, err := certMap.GetValue("alternative_names")
	assert.NoError(t, err)
	assert.Equal(t, []string{
		"www.example.org",
		"example.com",
		"example.edu",
		"example.net",
		"example.org",
		"www.example.com",
		"www.example.edu",
		"www.example.net",
	}, san)

	type fpTest struct {
		expected, actual string
	}
	fingerPrints := map[string]*fpTest{
		"md5":    {expected: "68423D55EA27D0B4FDA1878FCAB7A1EB"},
		"sha1":   {expected: "2509FB22F7671AEA2D0A28AE80516F390DE0CA21"},
		"sha256": {expected: "642DE54D84C30494157F53F657BF9F89B4EA6C8B16351FD7EC258D556F821040"},
	}
	req := make(map[string]*string)
	var algos []*FingerprintAlgorithm
	for algo, testCase := range fingerPrints {
		ptr, err := GetFingerprintAlgorithm(algo)
		if err != nil {
			t.Fatal(err)
		}
		algos = append(algos, ptr)
		req[algo] = &testCase.actual
	}
	hashCert(c[0], algos, req)
	for k, v := range fingerPrints {
		assert.Equal(t, v.expected, v.actual, k)
	}
}

func TestRandom(t *testing.T) {
	parser := &parser{}

	for i, test := range []struct {
		msg  string
		want mapstr.M
	}{
		{
			msg: "16030100ba010000b603032338f219562c78ca216984f33434bfe952354edf50" +
				"7588bddb96b35bd1a7639b000026c030c02cc028c024c014c00ac02fc02bc027" +
				"c023c013c009c012c008c011c007c010c00600ff010000670000001700150000" +
				"12746573742e6974762e6f72616e67652e6672000b000403000102000a000a00" +
				"08001700190018001600230000000d0020001e06010602060305010502050304" +
				"0104020403030103020303020102020203000500050100000000000f000101",
			want: mapstr.M{
				"extensions": mapstr.M{
					"_unparsed_": []string{
						"15",
					},
					"ec_points_formats": []string{
						"uncompressed",
						"ansiX962_compressed_prime",
						"ansiX962_compressed_char2",
					},
					"server_name_indication": []string{
						"test.itv.orange.fr",
					},
					"session_ticket": "",
					"signature_algorithms": []string{
						"rsa_pkcs1_sha512",
						"(unknown:0x0602)",
						"ecdsa_secp521r1_sha512",
						"rsa_pkcs1_sha384",
						"(unknown:0x0502)",
						"ecdsa_secp384r1_sha384",
						"rsa_pkcs1_sha256",
						"(unknown:0x0402)",
						"ecdsa_secp256r1_sha256",
						"(unknown:0x0301)",
						"(unknown:0x0302)",
						"(unknown:0x0303)",
						"rsa_pkcs1_sha1",
						"(unknown:0x0202)",
						"ecdsa_sha1",
					},
					"status_request": mapstr.M{
						"type":                     "ocsp",
						"responder_id_list_length": uint16(0),
						"request_extensions":       uint16(0),
					},
					"supported_groups": []string{
						"secp256r1",
						"secp521r1",
						"secp384r1",
						"secp256k1",
					},
				},
				"random": "2338f219562c78ca216984f33434bfe952354edf507588bddb96b35bd1a7639b",
				"supported_compression_methods": []string{
					"NULL",
				},
				"version": "3.3",
			},
		},
		{
			msg: "160303005b0200005703032c468bdbb2af2e7bd8e09c90e7992c61a8f468a03b" +
				"be74ec311fa33a14a35bfd20fc2b8e95f18fa299253278dd98178a61f75cdddd" +
				"69f8f1d1e0592e0ce8275af1c02b00000fff0100010000050000000b00020100" +
				"16030311400b00113c0011390005433082053f30820327a0030201020214213e" +
				"825a875eb349390d11117c6c14f894135fe3300d06092a864886f70d01010b05" +
				"003060310b3009060355040613024652310f300d060355040a0c064f72616e67" +
				"6531193017060355040b0c10464f52204c414220555345204f4e4c5931253023" +
				"06035504030c1c4f72616e6765204465766963657320504b49205456204c4142" +
				"204341301e170d3231303630333133333831365a170d32323036303331333338" +
				"31365a3040310b3009060355040613024652310f300d060355040b0c064f7261" +
				"6e67653120301e06035504030c1773657276657232207465737420504b492054" +
				"56204c41423059301306072a8648ce3d020106082a8648ce3d03010703420004" +
				"fc7a2bae61a536e74d4d3138b83e09ef618de444fff8dc8874863e3e1f557f90" +
				"08e2f777693ba6d7b8fe59f360006d55561e24edfc0608436b0bbf329df1463c" +
				"a38201da308201d6301d0603551d0e04160414dfd5c4bdbea6b4c89a57fd4a83" +
				"5573d53f51cbff301f0603551d230418301680143ae35027fb6f7337d4eaa8c8" +
				"2139f9627d116bf4300e0603551d0f0101ff0404030205a030130603551d2504" +
				"0c300a06082b06010505070301305c0603551d1f045530533051a04fa04d864b" +
				"687474703a2f2f706b692d63726c2d6c622e73656375726974792e696e747261" +
				"6f72616e67652f63726c2f6f72616e67655f646576696365735f706b695f7476" +
				"5f6c61625f63612e63726c3082010f0603551d11048201063082010282102a2e" +
				"656e61312e6f72616e67652e6672820f2a2e6974762e6f72616e67652e667282" +
				"192a2e6a6575787476697074762d70702e6f72616e67652e667282162a2e6a65" +
				"75787476697074762e6f72616e67652e667282102a2e6e7476312e6f72616e67" +
				"652e667282102a2e6e7476332e6f72616e67652e667282102a2e6e7476342e6f" +
				"72616e67652e667282102a2e6e7476352e6f72616e67652e6672820f2a2e7067" +
				"772e6f72616e67652e667282132a2e70702d656e61312e6f72616e67652e6672" +
				"82122a2e70702d6974762e6f72616e67652e667282132a2e70702d6e7476312e" +
				"6f72616e67652e667282132a2e70702d6e7476322e6f72616e67652e6672300d" +
				"06092a864886f70d01010b05000382020100b8d7a4819a31204d58fd9ca0364c" +
				"544f845b4c9e48dde8aadecad7149256e209df85b8b76f86ec62382932f9b037" +
				"0ef9d257d5112ae3b72bf8d0809fd437a20419a46f82851701f32e0897a659cf" +
				"5e19079615cb62205f2abf37a24d51ca9ca954d5b5c4485bbb9441109f71a3e6" +
				"b5fccccfe1e19d97d8403a60615bf1e4ef20c5bb069a2d26e78c5ad18a431934" +
				"276b9556153a0b7f3c992684a202ea8403c1f59b362389111e5fcb0fa89e40b8" +
				"6300be3949f0e5690abc3314ee7b53f0adb9019ea8e804e48a197f9d77b3c2fb" +
				"37dbe732a8aed100fb8624fecece4119e1ea045cdc7156765a89aa3d1228f062" +
				"eb5f109ac25afc63fa8948be9087eeb6c8ebe807b944230cba453aec493054c4" +
				"df4d5c34fc21d8739a6d40ddc36c95861f207b4dfcdd97224473a57841202378" +
				"31b86f398f62205945a7befca9f60c2c63578b2e9870723b4c5933c70317edb1" +
				"071d38ccdd2f91c3c50dc90906ccf14ecbb9b394cd471f33a92d6f210b994b7c" +
				"085abdc1dd789a46eac33503725f25376a5b438cf8d6dbb2b07ee128f3be21d5" +
				"0bdfb052271d079f4ccae174509a31ac1dfe2a483dce8eb624c181c616a49741" +
				"4f6616c21b8fd24e08aadd2c9c43944df5088e2bdbf121649ca1e405e1e95695" +
				"d52afa1c265b123344a9f5594b661e7d3406b0f6d60c7f776a9723bcec995f4b" +
				"4da3e6d42dc446b6a33904b7a56f74ba53010006",
			want: mapstr.M{
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
		},
	} {
		parser.parse(sBuf(t, test.msg))
		got := parser.hello.toMap()
		if !cmp.Equal(got, test.want) {
			t.Errorf("unexpected result for msg %d: %s", i, cmp.Diff(got, test.want))
		}
	}
}

func TestBadCertMessage(t *testing.T) {
	parser := &parser{}

	msgs := []string{
		// empty message
		"16030300040b000000",
		// no certificates
		"16030300070b000003000000",
		// certificates length out of bounds
		"16030300070b000003000fff",
		// certificate of size zero
		"160303000a0b000006000003000000",
		// certificate size out of bounds
		"160303000b0b000007000004000fff33",
		// bad certificate
		"160303000b0b00000700000400000133",
	}
	for idx, msg := range msgs {

		log := fmt.Sprintf("Message %d : '%s'", idx, msg)
		assert.Equal(t, resultOK, parser.parse(sBuf(t, msg)), log)
		assert.Nil(t, parser.certificates, log)
	}
}
