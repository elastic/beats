// +build !integration

// Unit tests and benchmarks for the dns package.
//
// The byte array test data was generated from pcap files using the gopacket
// test_creator.py script contained in the gopacket repository. The script was
// modified to drop the Ethernet, IP, and UDP headers from the byte arrays
// (skip the first 54 bytes).

package dns

import (
	"fmt"
	"math/rand"
	"net"
	"testing"

	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/protos/tcp"
	"github.com/elastic/beats/packetbeat/publish"

	"github.com/elastic/beats/libbeat/common"

	"github.com/stretchr/testify/assert"
)

// Verify that the interface TCP has been satisfied.
var _ protos.TcpPlugin = &Dns{}

var (
	messagesTcp = []DnsTestMessage{
		elasticATcp,
		zoneAxfrTcp,
		githubPtrTcp,
		sophosTxtTcp,
	}

	elasticATcp = DnsTestMessage{
		id:          11674,
		opcode:      "QUERY",
		flags:       []string{"rd", "ra"},
		rcode:       "NOERROR",
		q_class:     "IN",
		q_type:      "A",
		q_name:      "elastic.co.",
		q_etld:      "elastic.co.",
		answers:     []string{"54.201.204.244", "54.200.185.88"},
		authorities: []string{"NS-835.AWSDNS-40.NET.", "NS-1183.AWSDNS-19.ORG.", "NS-2007.AWSDNS-58.CO.UK.", "NS-66.AWSDNS-08.COM."},
		request: []byte{
			0x00, 0x1c, 0x2d, 0x9a, 0x01, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x07, 0x65,
			0x6c, 0x61, 0x73, 0x74, 0x69, 0x63, 0x02, 0x63, 0x6f, 0x00, 0x00, 0x01, 0x00, 0x01,
		},
		response: []byte{
			0x00, 0xc7, 0x2d, 0x9a, 0x81, 0x80, 0x00, 0x01, 0x00, 0x02, 0x00, 0x04, 0x00, 0x00, 0x07, 0x65,
			0x6c, 0x61, 0x73, 0x74, 0x69, 0x63, 0x02, 0x63, 0x6f, 0x00, 0x00, 0x01, 0x00, 0x01, 0xc0, 0x0c,
			0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x3c, 0x00, 0x04, 0x36, 0xc8, 0xb9, 0x58, 0xc0, 0x0c,
			0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x3c, 0x00, 0x04, 0x36, 0xc9, 0xcc, 0xf4, 0xc0, 0x0c,
			0x00, 0x02, 0x00, 0x01, 0x00, 0x00, 0x16, 0x82, 0x00, 0x16, 0x06, 0x4e, 0x53, 0x2d, 0x38, 0x33,
			0x35, 0x09, 0x41, 0x57, 0x53, 0x44, 0x4e, 0x53, 0x2d, 0x34, 0x30, 0x03, 0x4e, 0x45, 0x54, 0x00,
			0xc0, 0x0c, 0x00, 0x02, 0x00, 0x01, 0x00, 0x00, 0x16, 0x82, 0x00, 0x17, 0x07, 0x4e, 0x53, 0x2d,
			0x31, 0x31, 0x38, 0x33, 0x09, 0x41, 0x57, 0x53, 0x44, 0x4e, 0x53, 0x2d, 0x31, 0x39, 0x03, 0x4f,
			0x52, 0x47, 0x00, 0xc0, 0x0c, 0x00, 0x02, 0x00, 0x01, 0x00, 0x00, 0x16, 0x82, 0x00, 0x19, 0x07,
			0x4e, 0x53, 0x2d, 0x32, 0x30, 0x30, 0x37, 0x09, 0x41, 0x57, 0x53, 0x44, 0x4e, 0x53, 0x2d, 0x35,
			0x38, 0x02, 0x43, 0x4f, 0x02, 0x55, 0x4b, 0x00, 0xc0, 0x0c, 0x00, 0x02, 0x00, 0x01, 0x00, 0x00,
			0x16, 0x82, 0x00, 0x15, 0x05, 0x4e, 0x53, 0x2d, 0x36, 0x36, 0x09, 0x41, 0x57, 0x53, 0x44, 0x4e,
			0x53, 0x2d, 0x30, 0x38, 0x03, 0x43, 0x4f, 0x4d, 0x00,
		},
	}

	zoneAxfrTcp = DnsTestMessage{
		id:      0,
		opcode:  "QUERY",
		rcode:   "NOERROR",
		q_class: "IN",
		q_type:  "AXFR",
		q_name:  "etas.com.",
		q_etld:  "etas.com.",
		answers: []string{"training2003p.", "training2003p.", "1.1.1.1", "training2003p."},
		request: []byte{
			0x00, 0x1c, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04, 0x65,
			0x74, 0x61, 0x73, 0x03, 0x63, 0x6f, 0x6d, 0x00, 0x00, 0xfc, 0x00, 0x01, 0x4d, 0x53,
		},
		response: []byte{
			0x00, 0xc3, 0x00, 0x00, 0x80, 0x80, 0x00, 0x01, 0x00, 0x04, 0x00, 0x00, 0x00, 0x00, 0x04, 0x65,
			0x74, 0x61, 0x73, 0x03, 0x63, 0x6f, 0x6d, 0x00, 0x00, 0xfc, 0x00, 0x01, 0xc0, 0x0c, 0x00, 0x06,
			0x00, 0x01, 0x00, 0x00, 0x0e, 0x10, 0x00, 0x2f, 0x0d, 0x74, 0x72, 0x61, 0x69, 0x6e, 0x69, 0x6e,
			0x67, 0x32, 0x30, 0x30, 0x33, 0x70, 0x00, 0x0a, 0x68, 0x6f, 0x73, 0x74, 0x6d, 0x61, 0x73, 0x74,
			0x65, 0x72, 0x00, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00, 0x3c, 0x00, 0x00, 0x02, 0x58, 0x00,
			0x01, 0x51, 0x80, 0x00, 0x00, 0x0e, 0x10, 0xc0, 0x0c, 0x00, 0x02, 0x00, 0x01, 0x00, 0x00, 0x0e,
			0x10, 0x00, 0x0f, 0x0d, 0x74, 0x72, 0x61, 0x69, 0x6e, 0x69, 0x6e, 0x67, 0x32, 0x30, 0x30, 0x33,
			0x70, 0x00, 0x07, 0x77, 0x65, 0x6c, 0x63, 0x6f, 0x6d, 0x65, 0xc0, 0x0c, 0x00, 0x01, 0x00, 0x01,
			0x00, 0x00, 0x0e, 0x10, 0x00, 0x04, 0x01, 0x01, 0x01, 0x01, 0xc0, 0x0c, 0x00, 0x06, 0x00, 0x01,
			0x00, 0x00, 0x0e, 0x10, 0x00, 0x2f, 0x0d, 0x74, 0x72, 0x61, 0x69, 0x6e, 0x69, 0x6e, 0x67, 0x32,
			0x30, 0x30, 0x33, 0x70, 0x00, 0x0a, 0x68, 0x6f, 0x73, 0x74, 0x6d, 0x61, 0x73, 0x74, 0x65, 0x72,
			0x00, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00, 0x3c, 0x00, 0x00, 0x02, 0x58, 0x00, 0x01, 0x51,
			0x80, 0x00, 0x00, 0x0e, 0x10,
		},
	}

	githubPtrTcp = DnsTestMessage{
		id:          6766,
		opcode:      "QUERY",
		flags:       []string{"rd", "ra"},
		rcode:       "NOERROR",
		q_class:     "IN",
		q_type:      "PTR",
		q_name:      "131.252.30.192.in-addr.arpa.",
		q_etld:      "192.in-addr.arpa.",
		answers:     []string{"github.com."},
		authorities: []string{"ns1.p16.dynect.net.", "ns3.p16.dynect.net.", "ns4.p16.dynect.net.", "ns2.p16.dynect.net."},
		request: []byte{
			0x00, 0x2d, 0x1a, 0x6e, 0x01, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0x31,
			0x33, 0x31, 0x03, 0x32, 0x35, 0x32, 0x02, 0x33, 0x30, 0x03, 0x31, 0x39, 0x32, 0x07, 0x69, 0x6e,
			0x2d, 0x61, 0x64, 0x64, 0x72, 0x04, 0x61, 0x72, 0x70, 0x61, 0x00, 0x00, 0x0c, 0x00, 0x01,
		},
		response: []byte{
			0x00, 0x9b, 0x1a, 0x6e, 0x81, 0x80, 0x00, 0x01, 0x00, 0x01, 0x00, 0x04, 0x00, 0x00, 0x03, 0x31,
			0x33, 0x31, 0x03, 0x32, 0x35, 0x32, 0x02, 0x33, 0x30, 0x03, 0x31, 0x39, 0x32, 0x07, 0x69, 0x6e,
			0x2d, 0x61, 0x64, 0x64, 0x72, 0x04, 0x61, 0x72, 0x70, 0x61, 0x00, 0x00, 0x0c, 0x00, 0x01, 0xc0,
			0x0c, 0x00, 0x0c, 0x00, 0x01, 0x00, 0x00, 0x0e, 0x07, 0x00, 0x0c, 0x06, 0x67, 0x69, 0x74, 0x68,
			0x75, 0x62, 0x03, 0x63, 0x6f, 0x6d, 0x00, 0xc0, 0x10, 0x00, 0x02, 0x00, 0x01, 0x00, 0x01, 0x51,
			0x77, 0x00, 0x14, 0x03, 0x6e, 0x73, 0x31, 0x03, 0x70, 0x31, 0x36, 0x06, 0x64, 0x79, 0x6e, 0x65,
			0x63, 0x74, 0x03, 0x6e, 0x65, 0x74, 0x00, 0xc0, 0x10, 0x00, 0x02, 0x00, 0x01, 0x00, 0x01, 0x51,
			0x77, 0x00, 0x06, 0x03, 0x6e, 0x73, 0x33, 0xc0, 0x55, 0xc0, 0x10, 0x00, 0x02, 0x00, 0x01, 0x00,
			0x01, 0x51, 0x77, 0x00, 0x06, 0x03, 0x6e, 0x73, 0x34, 0xc0, 0x55, 0xc0, 0x10, 0x00, 0x02, 0x00,
			0x01, 0x00, 0x01, 0x51, 0x77, 0x00, 0x06, 0x03, 0x6e, 0x73, 0x32, 0xc0, 0x55,
		},
	}

	sophosTxtTcp = DnsTestMessage{
		id:      35009,
		opcode:  "QUERY",
		flags:   []string{"rd", "ra"},
		rcode:   "NXDOMAIN",
		q_class: "IN",
		q_type:  "TXT",
		q_name: "3.1o19ss00s2s17s4qp375sp49r830n2n4n923s8839052s7p7768s53365226pp3.659p1r741os37393" +
			"648s2348o762q1066q53rq5p4614r1q4781qpr16n809qp4.879o3o734q9sns005o3pp76q83.2q65qns3spns" +
			"1081s5rn5sr74opqrqnpq6rn3ro5.i.00.mac.sophosxl.net.",
		q_etld: "sophosxl.net.",
		request: []byte{
			0x00, 0xed, 0x88, 0xc1, 0x01, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x33,
			0x3f, 0x31, 0x6f, 0x31, 0x39, 0x73, 0x73, 0x30, 0x30, 0x73, 0x32, 0x73, 0x31, 0x37, 0x73, 0x34,
			0x71, 0x70, 0x33, 0x37, 0x35, 0x73, 0x70, 0x34, 0x39, 0x72, 0x38, 0x33, 0x30, 0x6e, 0x32, 0x6e,
			0x34, 0x6e, 0x39, 0x32, 0x33, 0x73, 0x38, 0x38, 0x33, 0x39, 0x30, 0x35, 0x32, 0x73, 0x37, 0x70,
			0x37, 0x37, 0x36, 0x38, 0x73, 0x35, 0x33, 0x33, 0x36, 0x35, 0x32, 0x32, 0x36, 0x70, 0x70, 0x33,
			0x3f, 0x36, 0x35, 0x39, 0x70, 0x31, 0x72, 0x37, 0x34, 0x31, 0x6f, 0x73, 0x33, 0x37, 0x33, 0x39,
			0x33, 0x36, 0x34, 0x38, 0x73, 0x32, 0x33, 0x34, 0x38, 0x6f, 0x37, 0x36, 0x32, 0x71, 0x31, 0x30,
			0x36, 0x36, 0x71, 0x35, 0x33, 0x72, 0x71, 0x35, 0x70, 0x34, 0x36, 0x31, 0x34, 0x72, 0x31, 0x71,
			0x34, 0x37, 0x38, 0x31, 0x71, 0x70, 0x72, 0x31, 0x36, 0x6e, 0x38, 0x30, 0x39, 0x71, 0x70, 0x34,
			0x1a, 0x38, 0x37, 0x39, 0x6f, 0x33, 0x6f, 0x37, 0x33, 0x34, 0x71, 0x39, 0x73, 0x6e, 0x73, 0x30,
			0x30, 0x35, 0x6f, 0x33, 0x70, 0x70, 0x37, 0x36, 0x71, 0x38, 0x33, 0x28, 0x32, 0x71, 0x36, 0x35,
			0x71, 0x6e, 0x73, 0x33, 0x73, 0x70, 0x6e, 0x73, 0x31, 0x30, 0x38, 0x31, 0x73, 0x35, 0x72, 0x6e,
			0x35, 0x73, 0x72, 0x37, 0x34, 0x6f, 0x70, 0x71, 0x72, 0x71, 0x6e, 0x70, 0x71, 0x36, 0x72, 0x6e,
			0x33, 0x72, 0x6f, 0x35, 0x01, 0x69, 0x02, 0x30, 0x30, 0x03, 0x6d, 0x61, 0x63, 0x08, 0x73, 0x6f,
			0x70, 0x68, 0x6f, 0x73, 0x78, 0x6c, 0x03, 0x6e, 0x65, 0x74, 0x00, 0x00, 0x10, 0x00, 0x01,
		},
		response: []byte{
			0x00, 0xed, 0x88, 0xc1, 0x81, 0x83, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x33,
			0x3f, 0x31, 0x6f, 0x31, 0x39, 0x73, 0x73, 0x30, 0x30, 0x73, 0x32, 0x73, 0x31, 0x37, 0x73, 0x34,
			0x71, 0x70, 0x33, 0x37, 0x35, 0x73, 0x70, 0x34, 0x39, 0x72, 0x38, 0x33, 0x30, 0x6e, 0x32, 0x6e,
			0x34, 0x6e, 0x39, 0x32, 0x33, 0x73, 0x38, 0x38, 0x33, 0x39, 0x30, 0x35, 0x32, 0x73, 0x37, 0x70,
			0x37, 0x37, 0x36, 0x38, 0x73, 0x35, 0x33, 0x33, 0x36, 0x35, 0x32, 0x32, 0x36, 0x70, 0x70, 0x33,
			0x3f, 0x36, 0x35, 0x39, 0x70, 0x31, 0x72, 0x37, 0x34, 0x31, 0x6f, 0x73, 0x33, 0x37, 0x33, 0x39,
			0x33, 0x36, 0x34, 0x38, 0x73, 0x32, 0x33, 0x34, 0x38, 0x6f, 0x37, 0x36, 0x32, 0x71, 0x31, 0x30,
			0x36, 0x36, 0x71, 0x35, 0x33, 0x72, 0x71, 0x35, 0x70, 0x34, 0x36, 0x31, 0x34, 0x72, 0x31, 0x71,
			0x34, 0x37, 0x38, 0x31, 0x71, 0x70, 0x72, 0x31, 0x36, 0x6e, 0x38, 0x30, 0x39, 0x71, 0x70, 0x34,
			0x1a, 0x38, 0x37, 0x39, 0x6f, 0x33, 0x6f, 0x37, 0x33, 0x34, 0x71, 0x39, 0x73, 0x6e, 0x73, 0x30,
			0x30, 0x35, 0x6f, 0x33, 0x70, 0x70, 0x37, 0x36, 0x71, 0x38, 0x33, 0x28, 0x32, 0x71, 0x36, 0x35,
			0x71, 0x6e, 0x73, 0x33, 0x73, 0x70, 0x6e, 0x73, 0x31, 0x30, 0x38, 0x31, 0x73, 0x35, 0x72, 0x6e,
			0x35, 0x73, 0x72, 0x37, 0x34, 0x6f, 0x70, 0x71, 0x72, 0x71, 0x6e, 0x70, 0x71, 0x36, 0x72, 0x6e,
			0x33, 0x72, 0x6f, 0x35, 0x01, 0x69, 0x02, 0x30, 0x30, 0x03, 0x6d, 0x61, 0x63, 0x08, 0x73, 0x6f,
			0x70, 0x68, 0x6f, 0x73, 0x78, 0x6c, 0x03, 0x6e, 0x65, 0x74, 0x00, 0x00, 0x10, 0x00, 0x01,
		},
	}
)

func testTcpTuple() *common.TcpTuple {
	t := &common.TcpTuple{
		Ip_length: 4,
		Src_ip:    net.IPv4(192, 168, 0, 1), Dst_ip: net.IPv4(192, 168, 0, 2),
		Src_port: ClientPort, Dst_port: ServerPort,
	}
	t.ComputeHashebles()
	return t
}

func TestDecodeTcp_nonDnsMsgRequest(t *testing.T) {
	rawData := []byte{0, 2, 1, 2}

	_, err := decodeDnsData(TransportTcp, rawData)
	assert.Equal(t, err, NonDnsMsg)
}

// Verify that the split lone request packet is decoded.
func TestDecodeTcp_splitRequest(t *testing.T) {
	stream := &DnsStream{rawData: sophosTxtTcp.request[:10], message: new(DnsMessage)}
	_, err := decodeDnsData(TransportTcp, stream.rawData)

	assert.NotNil(t, err, "Not expecting a complete message yet")

	stream.rawData = append(stream.rawData, sophosTxtTcp.request[10:]...)
	_, err = decodeDnsData(TransportTcp, stream.rawData)

	assert.Nil(t, err, "Message should be complete")
}

func TestParseTcp_errorNonDnsMsgResponse(t *testing.T) {
	var private protos.ProtocolData
	dns := newDns(testing.Verbose())
	tcptuple := testTcpTuple()
	q := elasticATcp
	packet := newPacket(forward, q.request)

	private = dns.Parse(packet, tcptuple, tcp.TcpDirectionOriginal, private)
	assert.Equal(t, 1, dns.transactions.Size(), "There should be one transaction.")

	r := []byte{0, 2, 1, 2}
	packet = newPacket(reverse, r)
	dns.Parse(packet, tcptuple, tcp.TcpDirectionReverse, private)
	assert.Empty(t, dns.transactions.Size(), "There should be no transaction.")

	m := expectResult(t, dns)
	assertRequest(t, m, q)
	assert.Equal(t, "tcp", mapValue(t, m, "transport"))
	assert.Equal(t, len(q.request), mapValue(t, m, "bytes_in"))
	assert.Nil(t, mapValue(t, m, "bytes_out"))
	assert.Equal(t, common.ERROR_STATUS, mapValue(t, m, "status"))
	assert.Equal(t, NonDnsMsg.ResponseError(), mapValue(t, m, "notes"))
}

// Verify that a request message with length (first two bytes value) of zero is not published
func TestParseTcp_zeroLengthMsgRequest(t *testing.T) {
	var private protos.ProtocolData
	dns := newDns(testing.Verbose())
	tcptuple := testTcpTuple()
	packet := newPacket(forward, []byte{0, 0, 1, 2})

	dns.Parse(packet, tcptuple, tcp.TcpDirectionOriginal, private)
	assert.Empty(t, dns.transactions.Size(), "There should be no transactions.")
	client := dns.results.(*publish.ChanTransactions)
	close(client.Channel)
	assert.Nil(t, <-client.Channel, "No result should have been published.")
}

// Verify that a response message with length (first two bytes value) of zero is published with the corresponding Notes
func TestParseTcp_errorZeroLengthMsgResponse(t *testing.T) {
	var private protos.ProtocolData
	dns := newDns(testing.Verbose())
	tcptuple := testTcpTuple()
	q := elasticATcp
	packet := newPacket(forward, q.request)

	private = dns.Parse(packet, tcptuple, tcp.TcpDirectionOriginal, private)
	assert.Equal(t, 1, dns.transactions.Size(), "There should be one transaction.")

	r := []byte{0, 0, 1, 2}
	packet = newPacket(reverse, r)
	dns.Parse(packet, tcptuple, tcp.TcpDirectionReverse, private)
	assert.Empty(t, dns.transactions.Size(), "There should be no transaction.")

	m := expectResult(t, dns)
	assertRequest(t, m, q)
	assert.Equal(t, "tcp", mapValue(t, m, "transport"))
	assert.Equal(t, len(q.request), mapValue(t, m, "bytes_in"))
	assert.Nil(t, mapValue(t, m, "bytes_out"))
	assert.Equal(t, common.ERROR_STATUS, mapValue(t, m, "status"))
	assert.Equal(t, ZeroLengthMsg.ResponseError(), mapValue(t, m, "notes"))
}

// Verify that an empty packet is safely handled (no panics).
func TestParseTcp_emptyPacket(t *testing.T) {
	var private protos.ProtocolData
	dns := newDns(testing.Verbose())
	packet := newPacket(forward, []byte{})
	tcptuple := testTcpTuple()

	dns.Parse(packet, tcptuple, tcp.TcpDirectionOriginal, private)
	assert.Empty(t, dns.transactions.Size(), "There should be no transactions.")
	client := dns.results.(*publish.ChanTransactions)
	close(client.Channel)
	assert.Nil(t, <-client.Channel, "No result should have been published.")
}

// Verify that a malformed packet is safely handled (no panics).
func TestParseTcp_malformedPacket(t *testing.T) {
	var private protos.ProtocolData
	dns := newDns(testing.Verbose())
	garbage := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13}
	tcptuple := testTcpTuple()
	packet := newPacket(forward, garbage)

	dns.Parse(packet, tcptuple, tcp.TcpDirectionOriginal, private)
	assert.Empty(t, dns.transactions.Size(), "There should be no transactions.")
}

// Verify that the lone request packet is parsed.
func TestParseTcp_requestPacket(t *testing.T) {
	var private protos.ProtocolData
	dns := newDns(testing.Verbose())
	packet := newPacket(forward, elasticATcp.request)
	tcptuple := testTcpTuple()

	dns.Parse(packet, tcptuple, tcp.TcpDirectionOriginal, private)
	assert.Equal(t, 1, dns.transactions.Size(), "There should be one transaction.")
	client := dns.results.(*publish.ChanTransactions)
	close(client.Channel)
	assert.Nil(t, <-client.Channel, "No result should have been published.")
}

// Verify that the lone response packet is parsed and that an error
// result is published.
func TestParseTcp_errorResponseOnly(t *testing.T) {
	var private protos.ProtocolData
	dns := newDns(testing.Verbose())
	q := elasticATcp
	packet := newPacket(reverse, q.response)
	tcptuple := testTcpTuple()

	dns.Parse(packet, tcptuple, tcp.TcpDirectionOriginal, private)
	m := expectResult(t, dns)
	assert.Equal(t, "tcp", mapValue(t, m, "transport"))
	assert.Nil(t, mapValue(t, m, "bytes_in"))
	assert.Equal(t, len(q.response), mapValue(t, m, "bytes_out"))
	assert.Nil(t, mapValue(t, m, "responsetime"))
	assert.Equal(t, common.ERROR_STATUS, mapValue(t, m, "status"))
	assert.Equal(t, OrphanedResponse.Error(), mapValue(t, m, "notes"))
	assertMapStrData(t, m, q)
}

// Verify that the first request is published without a response and that
// the status is error. This second packet will remain in the transaction
// map awaiting a response.
func TestParseTcp_errorDuplicateRequests(t *testing.T) {
	var private protos.ProtocolData
	dns := newDns(testing.Verbose())
	q := elasticATcp
	packet := newPacket(forward, q.request)
	tcptuple := testTcpTuple()

	dns.Parse(packet, tcptuple, tcp.TcpDirectionOriginal, private)
	assert.Equal(t, 1, dns.transactions.Size(), "There should be one transaction.")

	dns.Parse(packet, tcptuple, tcp.TcpDirectionOriginal, private)
	// The first request is published and this one becomes a transaction
	assert.Equal(t, 1, dns.transactions.Size(), "There should be one transaction.")

	m := expectResult(t, dns)
	assertRequest(t, m, q)
	assert.Equal(t, "tcp", mapValue(t, m, "transport"))
	assert.Equal(t, len(q.request), mapValue(t, m, "bytes_in"))
	assert.Nil(t, mapValue(t, m, "bytes_out"))
	assert.Nil(t, mapValue(t, m, "responsetime"))
	assert.Equal(t, common.ERROR_STATUS, mapValue(t, m, "status"))
	assert.Equal(t, DuplicateQueryMsg.Error(), mapValue(t, m, "notes"))
}

// Same than the previous one but on the same stream
// Checks that PrepareNewMessage and Parse can manage two messages on the same stream, in different packets
func TestParseTcp_errorDuplicateRequestsOneStream(t *testing.T) {
	var private protos.ProtocolData
	dns := newDns(testing.Verbose())
	q := elasticATcp
	packet := newPacket(forward, q.request)
	tcptuple := testTcpTuple()

	private = dns.Parse(packet, tcptuple, tcp.TcpDirectionOriginal, private)
	assert.Equal(t, 1, dns.transactions.Size(), "There should be one transaction.")

	dns.Parse(packet, tcptuple, tcp.TcpDirectionOriginal, private)
	// The first query is published and this one becomes a transaction
	assert.Equal(t, 1, dns.transactions.Size(), "There should be one transaction.")

	m := expectResult(t, dns)
	assertRequest(t, m, q)
	assert.Equal(t, "tcp", mapValue(t, m, "transport"))
	assert.Equal(t, len(q.request), mapValue(t, m, "bytes_in"))
	assert.Nil(t, mapValue(t, m, "bytes_out"))
	assert.Nil(t, mapValue(t, m, "responsetime"))
	assert.Equal(t, common.ERROR_STATUS, mapValue(t, m, "status"))
	assert.Equal(t, DuplicateQueryMsg.Error(), mapValue(t, m, "notes"))
}

// Checks that PrepareNewMessage and Parse can manage two messages sharing one packet on the same stream
// It typically happens when a SOA is followed by AXFR
func TestParseTcp_errorDuplicateRequestsOnePacket(t *testing.T) {
	var private protos.ProtocolData
	dns := newDns(testing.Verbose())
	q := elasticATcp
	offset := 4

	concatRequest := append(q.request, q.request[:offset]...)
	packet := newPacket(forward, concatRequest)
	tcptuple := testTcpTuple()

	private = dns.Parse(packet, tcptuple, tcp.TcpDirectionOriginal, private)
	assert.Equal(t, 1, dns.transactions.Size(), "There should be one transaction.")

	packet = newPacket(forward, q.request[offset:])
	private = dns.Parse(packet, tcptuple, tcp.TcpDirectionOriginal, private)
	assert.Equal(t, 1, dns.transactions.Size(), "There should be one transaction.")

	m := expectResult(t, dns)
	assertRequest(t, m, q)
	assert.Equal(t, "tcp", mapValue(t, m, "transport"))
	assert.Equal(t, len(q.request), mapValue(t, m, "bytes_in"))
	assert.Nil(t, mapValue(t, m, "bytes_out"))
	assert.Nil(t, mapValue(t, m, "responsetime"))
	assert.Equal(t, common.ERROR_STATUS, mapValue(t, m, "status"))
	assert.Equal(t, DuplicateQueryMsg.Error(), mapValue(t, m, "notes"))
}

// Verify that a split response packet is parsed and published
func TestParseTcp_splitResponse(t *testing.T) {
	var private protos.ProtocolData
	dns := newDns(testing.Verbose())
	tcpQuery := elasticATcp
	q := tcpQuery.request
	r0 := tcpQuery.response[:1]
	r1 := tcpQuery.response[1:10]
	r2 := tcpQuery.response[10:]
	tcptuple := testTcpTuple()

	packet := newPacket(forward, q)
	private = dns.Parse(packet, tcptuple, tcp.TcpDirectionOriginal, private)
	assert.Equal(t, 1, dns.transactions.Size(), "There should be one transaction.")

	packet = newPacket(reverse, r0)
	private = dns.Parse(packet, tcptuple, tcp.TcpDirectionReverse, private)
	assert.Equal(t, 1, dns.transactions.Size(), "There should be one transaction.")

	packet = newPacket(reverse, r1)
	dns.Parse(packet, tcptuple, tcp.TcpDirectionReverse, private)
	assert.Equal(t, 1, dns.transactions.Size(), "There should be one transaction.")

	packet = newPacket(reverse, r2)
	dns.Parse(packet, tcptuple, tcp.TcpDirectionReverse, private)
	assert.Empty(t, dns.transactions.Size(), "There should be no transaction.")

	m := expectResult(t, dns)
	assert.Equal(t, "tcp", mapValue(t, m, "transport"))
	assert.Equal(t, len(tcpQuery.request), mapValue(t, m, "bytes_in"))
	assert.Equal(t, len(tcpQuery.response), mapValue(t, m, "bytes_out"))
	assert.NotNil(t, mapValue(t, m, "responsetime"))
	assert.Equal(t, common.OK_STATUS, mapValue(t, m, "status"))
	assert.Nil(t, mapValue(t, m, "notes"))
	assertMapStrData(t, m, tcpQuery)
}

func TestGap_requestDrop(t *testing.T) {
	var private protos.ProtocolData
	dns := newDns(testing.Verbose())
	q := sophosTxtTcp.request[:10]
	packet := newPacket(forward, q)
	tcptuple := testTcpTuple()

	private = dns.Parse(packet, tcptuple, tcp.TcpDirectionOriginal, private)

	private, drop := dns.GapInStream(tcptuple, tcp.TcpDirectionOriginal, 10, private)

	assert.Equal(t, true, drop)

	dns.ReceivedFin(tcptuple, tcp.TcpDirectionOriginal, private)

	client := dns.results.(*publish.ChanTransactions)
	close(client.Channel)
	mapStr := <-client.Channel
	assert.Nil(t, mapStr, "No result should have been published.")
}

// Verify that a gap during the response publish the request with Notes
func TestGap_errorResponse(t *testing.T) {
	var private protos.ProtocolData
	dns := newDns(testing.Verbose())
	q := sophosTxtTcp.request
	r := sophosTxtTcp.response[:10]
	tcptuple := testTcpTuple()

	packet := newPacket(forward, q)
	private = dns.Parse(packet, tcptuple, tcp.TcpDirectionOriginal, private)
	assert.Equal(t, 1, dns.transactions.Size(), "There should be one transaction.")

	packet = newPacket(reverse, r)
	private = dns.Parse(packet, tcptuple, tcp.TcpDirectionReverse, private)
	assert.Equal(t, 1, dns.transactions.Size(), "There should be one transaction.")

	private, drop := dns.GapInStream(tcptuple, tcp.TcpDirectionReverse, 10, private)
	assert.Equal(t, true, drop)

	dns.ReceivedFin(tcptuple, tcp.TcpDirectionReverse, private)

	m := expectResult(t, dns)
	assertRequest(t, m, sophosTxtTcp)
	assert.Equal(t, IncompleteMsg.ResponseError(), mapValue(t, m, "notes"))
	assert.Nil(t, mapValue(t, m, "answers"))
}

// Verify that a gap/fin happening after a valid query create only one tansaction
func TestGapFin_validMessage(t *testing.T) {
	var private protos.ProtocolData
	dns := newDns(testing.Verbose())
	q := sophosTxtTcp.request
	tcptuple := testTcpTuple()

	packet := newPacket(forward, q)
	private = dns.Parse(packet, tcptuple, tcp.TcpDirectionOriginal, private)
	assert.Equal(t, 1, dns.transactions.Size(), "There should be one transaction.")

	private, drop := dns.GapInStream(tcptuple, tcp.TcpDirectionOriginal, 10, private)
	assert.Equal(t, false, drop)

	dns.ReceivedFin(tcptuple, tcp.TcpDirectionReverse, private)
	assert.Equal(t, 1, dns.transactions.Size(), "There should be one transaction.")

	client := dns.results.(*publish.ChanTransactions)
	close(client.Channel)
	mapStr := <-client.Channel
	assert.Nil(t, mapStr, "No result should have been published.")
	assert.Empty(t, mapStr["notes"], "There should be no notes")
}

// Verify that a Fin during the response publish the request with Notes
func TestFin_errorResponse(t *testing.T) {
	var private protos.ProtocolData
	dns := newDns(testing.Verbose())
	q := zoneAxfrTcp.request
	r := zoneAxfrTcp.response[:10]
	tcptuple := testTcpTuple()

	packet := newPacket(forward, q)
	private = dns.Parse(packet, tcptuple, tcp.TcpDirectionOriginal, private)
	assert.Equal(t, 1, dns.transactions.Size(), "There should be one transaction.")

	packet = newPacket(reverse, r)
	private = dns.Parse(packet, tcptuple, tcp.TcpDirectionReverse, private)
	assert.Equal(t, 1, dns.transactions.Size(), "There should be one transaction.")

	dns.ReceivedFin(tcptuple, tcp.TcpDirectionReverse, private)

	m := expectResult(t, dns)
	assertRequest(t, m, zoneAxfrTcp)
	assert.Equal(t, IncompleteMsg.ResponseError(), mapValue(t, m, "notes"))
	assert.Nil(t, mapValue(t, m, "answers"))
}

// parseTcpRequestResponse parses a request then a response packet and validates
// the published result.
func parseTcpRequestResponse(t testing.TB, dns *Dns, q DnsTestMessage) {
	var private protos.ProtocolData
	packet := newPacket(forward, q.request)
	tcptuple := testTcpTuple()
	private = dns.Parse(packet, tcptuple, tcp.TcpDirectionOriginal, private)

	packet = newPacket(reverse, q.response)
	dns.Parse(packet, tcptuple, tcp.TcpDirectionReverse, private)

	assert.Empty(t, dns.transactions.Size(), "There should be no transactions.")

	m := expectResult(t, dns)
	assert.Equal(t, "tcp", mapValue(t, m, "transport"))
	assert.Equal(t, len(q.request), mapValue(t, m, "bytes_in"))
	assert.Equal(t, len(q.response), mapValue(t, m, "bytes_out"))
	assert.NotNil(t, mapValue(t, m, "responsetime"))

	if assert.ObjectsAreEqual("NOERROR", mapValue(t, m, "dns.response_code")) {
		assert.Equal(t, common.OK_STATUS, mapValue(t, m, "status"))
	} else {
		assert.Equal(t, common.ERROR_STATUS, mapValue(t, m, "status"))
	}

	assert.Nil(t, mapValue(t, m, "notes"))
	assertMapStrData(t, m, q)
}

// Verify that the request/response pair are parsed and that a result
// is published.
func TestParseTcp_requestResponse(t *testing.T) {
	parseTcpRequestResponse(t, newDns(testing.Verbose()), elasticATcp)
}

// Verify all DNS TCP test messages are parsed correctly.
func TestParseTcp_allTestMessages(t *testing.T) {
	dns := newDns(testing.Verbose())
	for _, q := range messagesTcp {
		t.Logf("Testing with query for %s", q.q_name)
		parseTcpRequestResponse(t, dns, q)
	}
}

// Benchmarks TCP parsing for the given test message.
func benchmarkTcp(b *testing.B, q DnsTestMessage) {
	dns := newDns(false)
	for i := 0; i < b.N; i++ {
		var private protos.ProtocolData
		packet := newPacket(forward, q.request)
		tcptuple := testTcpTuple()
		private = dns.Parse(packet, tcptuple, tcp.TcpDirectionOriginal, private)

		packet = newPacket(reverse, q.response)
		dns.Parse(packet, tcptuple, tcp.TcpDirectionReverse, private)

		client := dns.results.(*publish.ChanTransactions)
		<-client.Channel
	}
}

// Benchmark Tcp parsing against each test message.
func BenchmarkTcpElasticA(b *testing.B)  { benchmarkTcp(b, elasticATcp) }
func BenchmarkTcpZoneIxfr(b *testing.B)  { benchmarkTcp(b, zoneAxfrTcp) }
func BenchmarkTcpGithubPtr(b *testing.B) { benchmarkTcp(b, githubPtrTcp) }
func BenchmarkTcpSophosTxt(b *testing.B) { benchmarkTcp(b, sophosTxtTcp) }

// Benchmark that runs with parallelism to help find concurrency related
// issues. To run with parallelism, the 'go test' cpu flag must be set
// greater than 1, otherwise it just runs concurrently but not in parallel.
func BenchmarkParallelTcpParse(b *testing.B) {
	rand.Seed(22)
	numMessages := len(messagesTcp)
	dns := newDns(false)
	client := dns.results.(*publish.ChanTransactions)

	// Drain the results channel while the test is running.
	go func() {
		totalMessages := 0
		for r := range client.Channel {
			_ = r
			totalMessages++
		}
		fmt.Printf("Parsed %d messages.\n", totalMessages)
	}()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		// Each iteration parses one message, either a request or a response.
		// The request and response could be parsed on different goroutines.
		for pb.Next() {
			q := messagesTcp[rand.Intn(numMessages)]
			var packet *protos.Packet
			var tcptuple *common.TcpTuple
			var private protos.ProtocolData

			if rand.Intn(2) == 0 {
				packet = newPacket(forward, q.request)
				tcptuple = testTcpTuple()
			} else {
				packet = newPacket(reverse, q.response)
				tcptuple = testTcpTuple()
			}
			dns.Parse(packet, tcptuple, tcp.TcpDirectionOriginal, private)
		}
	})

	defer close(client.Channel)
}
