// Unit tests and benchmarks for the dns package.
//
// The byte array test data was generated from pcap files using the gopacket
// test_creator.py script contained in the gopacket repository. The script was
// modified to drop the Ethernet, IP, and UDP headers from the byte arrays
// (skip the first 42 bytes for UDP packets and the first 54 bytes for TCP packets).
//
// TODO:
//   * Add test validation for responsetime to make sure unit conversion
//     is being done correctly.
//   * Add validation of special fields provided in MX, SOA, NS queries.
//   * Add test case to verify that Include_authorities and Include_additionals
//     are working.
//   * Add test case for Send_request and validate the stringified DNS message.
//   * Add test case for Send_response and validate the stringified DNS message.

package dns

import (
	"fmt"
	"math/rand"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/protos/tcp"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/stretchr/testify/assert"
	"github.com/tsg/gopacket/layers"
)

// Test Constants
const (
	ServerIp   = "192.168.0.1"
	ServerPort = 53
	ClientIp   = "10.0.0.1"
	ClientPort = 34898
)

// DnsTestMessage holds the data that is expected to be returned when parsing
// the raw DNS layer payloads for the request and response packet.
type DnsTestMessage struct {
	id          uint16
	opcode      string
	flags       []string
	rcode       string
	q_class     string
	q_type      string
	q_name      string
	answers     []string
	authorities []string
	additionals []string
	request     []byte
	response    []byte
}

// DNS messages for testing. When adding a new test message, add it to the
// messages array and create a new benchmark test for the message.
var (
	// An array of all test messages.
	messages = []DnsTestMessage{
		elasticA,
		zoneIxfr,
		githubPtr,
		sophosTxt,
	}
	messagesTcp = []DnsTestMessage{
		elasticATcp,
		zoneAxfrTcp,
		githubPtrTcp,
		sophosTxtTcp,
	}

	elasticA = DnsTestMessage{
		id:      8529,
		opcode:  "QUERY",
		flags:   []string{"rd", "ra"},
		rcode:   "NOERROR",
		q_class: "IN",
		q_type:  "A",
		q_name:  "elastic.co",
		answers: []string{"54.148.130.30", "54.69.104.66"},
		request: []byte{
			0x21, 0x51, 0x01, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x07, 0x65, 0x6c, 0x61,
			0x73, 0x74, 0x69, 0x63, 0x02, 0x63, 0x6f, 0x00, 0x00, 0x01, 0x00, 0x01,
		},
		response: []byte{
			0x21, 0x51, 0x81, 0x80, 0x00, 0x01, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x07, 0x65, 0x6c, 0x61,
			0x73, 0x74, 0x69, 0x63, 0x02, 0x63, 0x6f, 0x00, 0x00, 0x01, 0x00, 0x01, 0xc0, 0x0c, 0x00, 0x01,
			0x00, 0x01, 0x00, 0x00, 0x00, 0x39, 0x00, 0x04, 0x36, 0x94, 0x82, 0x1e, 0xc0, 0x0c, 0x00, 0x01,
			0x00, 0x01, 0x00, 0x00, 0x00, 0x39, 0x00, 0x04, 0x36, 0x45, 0x68, 0x42,
		},
	}

	elasticATcp = DnsTestMessage{
		id:          11674,
		opcode:      "QUERY",
		flags:       []string{"rd", "ra"},
		rcode:       "NOERROR",
		q_class:     "IN",
		q_type:      "A",
		q_name:      "elastic.co",
		answers:     []string{"54.201.204.244", "54.200.185.88"},
		authorities: []string{"NS-835.AWSDNS-40.NET", "NS-1183.AWSDNS-19.ORG", "NS-2007.AWSDNS-58.CO.UK", "NS-66.AWSDNS-08.COM"},
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

	zoneIxfr = DnsTestMessage{
		id:      16384,
		opcode:  "QUERY",
		flags:   []string{"ra"},
		rcode:   "NOERROR",
		q_class: "IN",
		q_type:  "IXFR",
		q_name:  "etas.com",
		answers: []string{"training2003p", "training2003p", "training2003p",
			"training2003p", "1.1.1.100"},
		request: []byte{
			0x40, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x04, 0x65, 0x74, 0x61,
			0x73, 0x03, 0x63, 0x6f, 0x6d, 0x00, 0x00, 0xfb, 0x00, 0x01, 0xc0, 0x0c, 0x00, 0x06, 0x00, 0x01,
			0x00, 0x00, 0x0e, 0x10, 0x00, 0x2f, 0x0d, 0x74, 0x72, 0x61, 0x69, 0x6e, 0x69, 0x6e, 0x67, 0x32,
			0x30, 0x30, 0x33, 0x70, 0x00, 0x0a, 0x68, 0x6f, 0x73, 0x74, 0x6d, 0x61, 0x73, 0x74, 0x65, 0x72,
			0x00, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00, 0x3c, 0x00, 0x00, 0x02, 0x58, 0x00, 0x01, 0x51,
			0x80, 0x00, 0x00, 0x0e, 0x10, 0x4d, 0x53,
		},
		response: []byte{
			0x40, 0x00, 0x80, 0x80, 0x00, 0x01, 0x00, 0x05, 0x00, 0x00, 0x00, 0x00, 0x04, 0x65, 0x74, 0x61,
			0x73, 0x03, 0x63, 0x6f, 0x6d, 0x00, 0x00, 0xfb, 0x00, 0x01, 0xc0, 0x0c, 0x00, 0x06, 0x00, 0x01,
			0x00, 0x00, 0x0e, 0x10, 0x00, 0x2f, 0x0d, 0x74, 0x72, 0x61, 0x69, 0x6e, 0x69, 0x6e, 0x67, 0x32,
			0x30, 0x30, 0x33, 0x70, 0x00, 0x0a, 0x68, 0x6f, 0x73, 0x74, 0x6d, 0x61, 0x73, 0x74, 0x65, 0x72,
			0x00, 0x00, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x3c, 0x00, 0x00, 0x02, 0x58, 0x00, 0x01, 0x51,
			0x80, 0x00, 0x00, 0x0e, 0x10, 0xc0, 0x0c, 0x00, 0x06, 0x00, 0x01, 0x00, 0x00, 0x0e, 0x10, 0x00,
			0x18, 0xc0, 0x26, 0xc0, 0x35, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00, 0x3c, 0x00, 0x00, 0x02,
			0x58, 0x00, 0x01, 0x51, 0x80, 0x00, 0x00, 0x0e, 0x10, 0xc0, 0x0c, 0x00, 0x06, 0x00, 0x01, 0x00,
			0x00, 0x0e, 0x10, 0x00, 0x18, 0xc0, 0x26, 0xc0, 0x35, 0x00, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00,
			0x3c, 0x00, 0x00, 0x02, 0x58, 0x00, 0x01, 0x51, 0x80, 0x00, 0x00, 0x0e, 0x10, 0x05, 0x69, 0x6e,
			0x64, 0x65, 0x78, 0xc0, 0x0c, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x0e, 0x10, 0x00, 0x04, 0x01,
			0x01, 0x01, 0x64, 0xc0, 0x0c, 0x00, 0x06, 0x00, 0x01, 0x00, 0x00, 0x0e, 0x10, 0x00, 0x18, 0xc0,
			0x26, 0xc0, 0x35, 0x00, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x3c, 0x00, 0x00, 0x02, 0x58, 0x00,
			0x01, 0x51, 0x80, 0x00, 0x00, 0x0e, 0x10,
		},
	}

	zoneAxfrTcp = DnsTestMessage{
		id:      0,
		opcode:  "QUERY",
		rcode:   "NOERROR",
		q_class: "IN",
		q_type:  "AXFR",
		q_name:  "etas.com",
		answers: []string{"training2003p", "training2003p", "1.1.1.1", "training2003p"},
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

	githubPtr = DnsTestMessage{
		id:      344,
		opcode:  "QUERY",
		flags:   []string{"rd", "ra"},
		rcode:   "NOERROR",
		q_class: "IN",
		q_type:  "PTR",
		q_name:  "131.252.30.192.in-addr.arpa",
		answers: []string{"github.com"},
		authorities: []string{"a.root-servers.net", "b.root-servers.net", "c.root-servers.net",
			"d.root-servers.net", "e.root-servers.net", "f.root-servers.net", "g.root-servers.net",
			"h.root-servers.net", "i.root-servers.net", "j.root-servers.net", "k.root-servers.net",
			"l.root-servers.net", "m.root-servers.net"},
		request: []byte{
			0x01, 0x58, 0x01, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0x31, 0x33, 0x31,
			0x03, 0x32, 0x35, 0x32, 0x02, 0x33, 0x30, 0x03, 0x31, 0x39, 0x32, 0x07, 0x69, 0x6e, 0x2d, 0x61,
			0x64, 0x64, 0x72, 0x04, 0x61, 0x72, 0x70, 0x61, 0x00, 0x00, 0x0c, 0x00, 0x01,
		},
		response: []byte{
			0x01, 0x58, 0x81, 0x80, 0x00, 0x01, 0x00, 0x01, 0x00, 0x0d, 0x00, 0x00, 0x03, 0x31, 0x33, 0x31,
			0x03, 0x32, 0x35, 0x32, 0x02, 0x33, 0x30, 0x03, 0x31, 0x39, 0x32, 0x07, 0x69, 0x6e, 0x2d, 0x61,
			0x64, 0x64, 0x72, 0x04, 0x61, 0x72, 0x70, 0x61, 0x00, 0x00, 0x0c, 0x00, 0x01, 0xc0, 0x0c, 0x00,
			0x0c, 0x00, 0x01, 0x00, 0x00, 0x09, 0xe2, 0x00, 0x0c, 0x06, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62,
			0x03, 0x63, 0x6f, 0x6d, 0x00, 0x00, 0x00, 0x02, 0x00, 0x01, 0x00, 0x00, 0x07, 0xb8, 0x00, 0x14,
			0x01, 0x6c, 0x0c, 0x72, 0x6f, 0x6f, 0x74, 0x2d, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x73, 0x03,
			0x6e, 0x65, 0x74, 0x00, 0x00, 0x00, 0x02, 0x00, 0x01, 0x00, 0x00, 0x07, 0xb8, 0x00, 0x04, 0x01,
			0x65, 0xc0, 0x52, 0x00, 0x00, 0x02, 0x00, 0x01, 0x00, 0x00, 0x07, 0xb8, 0x00, 0x04, 0x01, 0x63,
			0xc0, 0x52, 0x00, 0x00, 0x02, 0x00, 0x01, 0x00, 0x00, 0x07, 0xb8, 0x00, 0x04, 0x01, 0x62, 0xc0,
			0x52, 0x00, 0x00, 0x02, 0x00, 0x01, 0x00, 0x00, 0x07, 0xb8, 0x00, 0x04, 0x01, 0x61, 0xc0, 0x52,
			0x00, 0x00, 0x02, 0x00, 0x01, 0x00, 0x00, 0x07, 0xb8, 0x00, 0x04, 0x01, 0x68, 0xc0, 0x52, 0x00,
			0x00, 0x02, 0x00, 0x01, 0x00, 0x00, 0x07, 0xb8, 0x00, 0x04, 0x01, 0x66, 0xc0, 0x52, 0x00, 0x00,
			0x02, 0x00, 0x01, 0x00, 0x00, 0x07, 0xb8, 0x00, 0x04, 0x01, 0x69, 0xc0, 0x52, 0x00, 0x00, 0x02,
			0x00, 0x01, 0x00, 0x00, 0x07, 0xb8, 0x00, 0x04, 0x01, 0x67, 0xc0, 0x52, 0x00, 0x00, 0x02, 0x00,
			0x01, 0x00, 0x00, 0x07, 0xb8, 0x00, 0x04, 0x01, 0x6d, 0xc0, 0x52, 0x00, 0x00, 0x02, 0x00, 0x01,
			0x00, 0x00, 0x07, 0xb8, 0x00, 0x04, 0x01, 0x64, 0xc0, 0x52, 0x00, 0x00, 0x02, 0x00, 0x01, 0x00,
			0x00, 0x07, 0xb8, 0x00, 0x04, 0x01, 0x6a, 0xc0, 0x52, 0x00, 0x00, 0x02, 0x00, 0x01, 0x00, 0x00,
			0x07, 0xb8, 0x00, 0x04, 0x01, 0x6b, 0xc0, 0x52,
		},
	}

	githubPtrTcp = DnsTestMessage{
		id:          6766,
		opcode:      "QUERY",
		flags:       []string{"rd", "ra"},
		rcode:       "NOERROR",
		q_class:     "IN",
		q_type:      "PTR",
		q_name:      "131.252.30.192.in-addr.arpa",
		answers:     []string{"github.com"},
		authorities: []string{"ns1.p16.dynect.net", "ns3.p16.dynect.net", "ns4.p16.dynect.net", "ns2.p16.dynect.net"},
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

	sophosTxt = DnsTestMessage{
		id:      8238,
		opcode:  "QUERY",
		flags:   []string{"rd", "ra"},
		rcode:   "NXDOMAIN",
		q_class: "IN",
		q_type:  "TXT",
		q_name: "3.1o19ss00s2s17s4qp375sp49r830n2n4n923s8839052s7p7768s53365226pp3.659p1r741os37393" +
			"648s2348o762q1066q53rq5p4614r1q4781qpr16n809qp4.879o3o734q9sns005o3pp76q83.2q65qns3spns" +
			"1081s5rn5sr74opqrqnpq6rn3ro5.i.00.mac.sophosxl.net",
		request: []byte{
			0x20, 0x2e, 0x01, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x33, 0x3f, 0x31,
			0x6f, 0x31, 0x39, 0x73, 0x73, 0x30, 0x30, 0x73, 0x32, 0x73, 0x31, 0x37, 0x73, 0x34, 0x71, 0x70,
			0x33, 0x37, 0x35, 0x73, 0x70, 0x34, 0x39, 0x72, 0x38, 0x33, 0x30, 0x6e, 0x32, 0x6e, 0x34, 0x6e,
			0x39, 0x32, 0x33, 0x73, 0x38, 0x38, 0x33, 0x39, 0x30, 0x35, 0x32, 0x73, 0x37, 0x70, 0x37, 0x37,
			0x36, 0x38, 0x73, 0x35, 0x33, 0x33, 0x36, 0x35, 0x32, 0x32, 0x36, 0x70, 0x70, 0x33, 0x3f, 0x36,
			0x35, 0x39, 0x70, 0x31, 0x72, 0x37, 0x34, 0x31, 0x6f, 0x73, 0x33, 0x37, 0x33, 0x39, 0x33, 0x36,
			0x34, 0x38, 0x73, 0x32, 0x33, 0x34, 0x38, 0x6f, 0x37, 0x36, 0x32, 0x71, 0x31, 0x30, 0x36, 0x36,
			0x71, 0x35, 0x33, 0x72, 0x71, 0x35, 0x70, 0x34, 0x36, 0x31, 0x34, 0x72, 0x31, 0x71, 0x34, 0x37,
			0x38, 0x31, 0x71, 0x70, 0x72, 0x31, 0x36, 0x6e, 0x38, 0x30, 0x39, 0x71, 0x70, 0x34, 0x1a, 0x38,
			0x37, 0x39, 0x6f, 0x33, 0x6f, 0x37, 0x33, 0x34, 0x71, 0x39, 0x73, 0x6e, 0x73, 0x30, 0x30, 0x35,
			0x6f, 0x33, 0x70, 0x70, 0x37, 0x36, 0x71, 0x38, 0x33, 0x28, 0x32, 0x71, 0x36, 0x35, 0x71, 0x6e,
			0x73, 0x33, 0x73, 0x70, 0x6e, 0x73, 0x31, 0x30, 0x38, 0x31, 0x73, 0x35, 0x72, 0x6e, 0x35, 0x73,
			0x72, 0x37, 0x34, 0x6f, 0x70, 0x71, 0x72, 0x71, 0x6e, 0x70, 0x71, 0x36, 0x72, 0x6e, 0x33, 0x72,
			0x6f, 0x35, 0x01, 0x69, 0x02, 0x30, 0x30, 0x03, 0x6d, 0x61, 0x63, 0x08, 0x73, 0x6f, 0x70, 0x68,
			0x6f, 0x73, 0x78, 0x6c, 0x03, 0x6e, 0x65, 0x74, 0x00, 0x00, 0x10, 0x00, 0x01,
		},
		response: []byte{
			0x20, 0x2e, 0x81, 0x83, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x33, 0x3f, 0x31,
			0x6f, 0x31, 0x39, 0x73, 0x73, 0x30, 0x30, 0x73, 0x32, 0x73, 0x31, 0x37, 0x73, 0x34, 0x71, 0x70,
			0x33, 0x37, 0x35, 0x73, 0x70, 0x34, 0x39, 0x72, 0x38, 0x33, 0x30, 0x6e, 0x32, 0x6e, 0x34, 0x6e,
			0x39, 0x32, 0x33, 0x73, 0x38, 0x38, 0x33, 0x39, 0x30, 0x35, 0x32, 0x73, 0x37, 0x70, 0x37, 0x37,
			0x36, 0x38, 0x73, 0x35, 0x33, 0x33, 0x36, 0x35, 0x32, 0x32, 0x36, 0x70, 0x70, 0x33, 0x3f, 0x36,
			0x35, 0x39, 0x70, 0x31, 0x72, 0x37, 0x34, 0x31, 0x6f, 0x73, 0x33, 0x37, 0x33, 0x39, 0x33, 0x36,
			0x34, 0x38, 0x73, 0x32, 0x33, 0x34, 0x38, 0x6f, 0x37, 0x36, 0x32, 0x71, 0x31, 0x30, 0x36, 0x36,
			0x71, 0x35, 0x33, 0x72, 0x71, 0x35, 0x70, 0x34, 0x36, 0x31, 0x34, 0x72, 0x31, 0x71, 0x34, 0x37,
			0x38, 0x31, 0x71, 0x70, 0x72, 0x31, 0x36, 0x6e, 0x38, 0x30, 0x39, 0x71, 0x70, 0x34, 0x1a, 0x38,
			0x37, 0x39, 0x6f, 0x33, 0x6f, 0x37, 0x33, 0x34, 0x71, 0x39, 0x73, 0x6e, 0x73, 0x30, 0x30, 0x35,
			0x6f, 0x33, 0x70, 0x70, 0x37, 0x36, 0x71, 0x38, 0x33, 0x28, 0x32, 0x71, 0x36, 0x35, 0x71, 0x6e,
			0x73, 0x33, 0x73, 0x70, 0x6e, 0x73, 0x31, 0x30, 0x38, 0x31, 0x73, 0x35, 0x72, 0x6e, 0x35, 0x73,
			0x72, 0x37, 0x34, 0x6f, 0x70, 0x71, 0x72, 0x71, 0x6e, 0x70, 0x71, 0x36, 0x72, 0x6e, 0x33, 0x72,
			0x6f, 0x35, 0x01, 0x69, 0x02, 0x30, 0x30, 0x03, 0x6d, 0x61, 0x63, 0x08, 0x73, 0x6f, 0x70, 0x68,
			0x6f, 0x73, 0x78, 0x6c, 0x03, 0x6e, 0x65, 0x74, 0x00, 0x00, 0x10, 0x00, 0x01,
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
			"1081s5rn5sr74opqrqnpq6rn3ro5.i.00.mac.sophosxl.net",
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

// Request and response addresses.
var (
	forward = common.NewIpPortTuple(4,
		net.ParseIP(ServerIp), ServerPort,
		net.ParseIP(ClientIp), ClientPort)
	reverse = common.NewIpPortTuple(4,
		net.ParseIP(ClientIp), ClientPort,
		net.ParseIP(ServerIp), ServerPort)
)

// Verify that the interfaces for UDP and TCP have been satisfied.
var _ protos.UdpProtocolPlugin = &Dns{}
var _ protos.TcpProtocolPlugin = &Dns{}

func newDns(verbose bool) *Dns {
	if verbose {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"dns"})
	} else {
		logp.LogInit(logp.LOG_EMERG, "", false, true, []string{"dns"})
	}

	dns := &Dns{}
	err := dns.Init(true, publisher.ChanClient{make(chan common.MapStr, 100)})
	if err != nil {
		return nil
	}

	dns.Ports = []int{ServerPort}
	dns.Include_authorities = true
	dns.Include_additionals = true
	dns.Send_request = true
	dns.Send_response = true
	return dns
}

func newPacket(t common.IpPortTuple, payload []byte) *protos.Packet {
	return &protos.Packet{
		Ts:      time.Now(),
		Tuple:   t,
		Payload: payload,
	}
}

// Verify that nameToString encodes non-printable characters.
func Test_nameToString_encodesNonPrintable(t *testing.T) {
	name := "\n \r \t \" \\ \u2318.dnstunnel.com"
	escapedName := "\\n \\r \\t \\\" \\\\ \\226\\140\\152.dnstunnel.com"
	assert.Equal(t, escapedName, nameToString([]byte(name)))
}

// Verify that an empty packet is safely handled (no panics).
func TestParseUdp_emptyPacket(t *testing.T) {
	dns := newDns(testing.Verbose())
	packet := newPacket(forward, []byte{})
	dns.ParseUdp(packet)
	assert.Empty(t, dns.transactions.Size(), "There should be no transactions.")
	client := dns.results.(publisher.ChanClient)
	close(client.Channel)
	assert.Nil(t, <-client.Channel, "No result should have been published.")
}

// Verify that a malformed packet is safely handled (no panics).
func TestParseUdp_malformedPacket(t *testing.T) {
	dns := newDns(testing.Verbose())
	garbage := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13}
	packet := newPacket(forward, garbage)
	dns.ParseUdp(packet)
	assert.Empty(t, dns.transactions.Size(), "There should be no transactions.")

	// As a future addition, a malformed message should publish a result.
}

// Verify that the lone request packet is parsed.
func TestParseUdp_requestPacket(t *testing.T) {
	dns := newDns(testing.Verbose())
	packet := newPacket(forward, elasticA.request)
	dns.ParseUdp(packet)
	assert.Equal(t, 1, dns.transactions.Size(), "There should be one transaction.")
	client := dns.results.(publisher.ChanClient)
	close(client.Channel)
	assert.Nil(t, <-client.Channel, "No result should have been published.")
}

// Verify that the lone response packet is parsed and that an error
// result is published.
func TestParseUdp_responseOnly(t *testing.T) {
	dns := newDns(testing.Verbose())
	q := elasticA
	packet := newPacket(reverse, q.response)
	dns.ParseUdp(packet)

	m := expectResult(t, dns)
	assert.Equal(t, "udp", mapValue(t, m, "transport"))
	assert.Nil(t, mapValue(t, m, "bytes_in"))
	assert.Equal(t, len(q.response), mapValue(t, m, "bytes_out"))
	assert.Nil(t, mapValue(t, m, "responsetime"))
	assert.Equal(t, common.ERROR_STATUS, mapValue(t, m, "status"))
	assert.Equal(t, OrphanedResponseMsg, mapValue(t, m, "notes"))
	assertMapStrData(t, m, q)
}

// Verify that the first request is published without a response and that
// the status is error. This second packet will remain in the transaction
// map awaiting a response.
func TestParseUdp_duplicateRequests(t *testing.T) {
	dns := newDns(testing.Verbose())
	q := elasticA
	packet := newPacket(forward, q.request)
	dns.ParseUdp(packet)
	assert.Equal(t, 1, dns.transactions.Size(), "There should be one transaction.")
	packet = newPacket(forward, q.request)
	dns.ParseUdp(packet)
	assert.Equal(t, 1, dns.transactions.Size(), "There should be one transaction.")

	m := expectResult(t, dns)
	assert.Equal(t, "udp", mapValue(t, m, "transport"))
	assert.Equal(t, len(q.request), mapValue(t, m, "bytes_in"))
	assert.Nil(t, mapValue(t, m, "bytes_out"))
	assert.Nil(t, mapValue(t, m, "responsetime"))
	assert.Equal(t, common.ERROR_STATUS, mapValue(t, m, "status"))
	assert.Equal(t, DuplicateQueryMsg, mapValue(t, m, "notes"))
}

// Verify that the request/response pair are parsed and that a result
// is published.
func TestParseUdp_requestResponse(t *testing.T) {
	parseUdpRequestResponse(t, newDns(testing.Verbose()), elasticA)
}

// Verify all DNS test messages are parsed correctly.
func TestParseUdp_allTestMessages(t *testing.T) {
	dns := newDns(testing.Verbose())
	for _, q := range messages {
		t.Logf("Testing with query for %s", q.q_name)
		parseUdpRequestResponse(t, dns, q)
	}
}

// Verify that expireTransaction publishes an event with an error status
// and note.
func TestExpireTransaction(t *testing.T) {
	dns := newDns(testing.Verbose())

	trans := newTransaction(time.Now(), DnsTuple{}, common.CmdlineTuple{})
	trans.Request = &DnsMessage{
		Data: &layers.DNS{
			Questions: []layers.DNSQuestion{{}},
		},
	}
	dns.expireTransaction(trans)

	m := expectResult(t, dns)
	assert.Nil(t, mapValue(t, m, "bytes_out"))
	assert.Nil(t, mapValue(t, m, "responsetime"))
	assert.Equal(t, common.ERROR_STATUS, mapValue(t, m, "status"))
	assert.Equal(t, NoResponse, mapValue(t, m, "notes"))
}

// Verify that an empty DNS request packet can be published.
func TestPublishTransaction_emptyDnsRequest(t *testing.T) {
	dns := newDns(testing.Verbose())

	trans := newTransaction(time.Now(), DnsTuple{}, common.CmdlineTuple{})
	trans.Request = &DnsMessage{
		Data: &layers.DNS{},
	}
	dns.publishTransaction(trans)

	m := expectResult(t, dns)
	assert.Equal(t, common.ERROR_STATUS, mapValue(t, m, "status"))
}

// Verify that an empty DNS response packet can be published.
func TestPublishTransaction_emptyDnsResponse(t *testing.T) {
	dns := newDns(testing.Verbose())

	trans := newTransaction(time.Now(), DnsTuple{}, common.CmdlineTuple{})
	trans.Response = &DnsMessage{
		Data: &layers.DNS{},
	}
	dns.publishTransaction(trans)

	m := expectResult(t, dns)
	assert.Equal(t, common.ERROR_STATUS, mapValue(t, m, "status"))
}

// Benchmarks UDP parsing for the given test message.
func benchmarkUdp(b *testing.B, q DnsTestMessage) {
	dns := newDns(false)
	for i := 0; i < b.N; i++ {
		packet := newPacket(forward, q.request)
		dns.ParseUdp(packet)
		packet = newPacket(reverse, q.response)
		dns.ParseUdp(packet)

		client := dns.results.(publisher.ChanClient)
		<-client.Channel
	}
}

// Benchmark UDP parsing against each test message.
func BenchmarkUdpElasticA(b *testing.B)  { benchmarkUdp(b, elasticA) }
func BenchmarkUdpZoneIxfr(b *testing.B)  { benchmarkUdp(b, zoneIxfr) }
func BenchmarkUdpGithubPtr(b *testing.B) { benchmarkUdp(b, githubPtr) }
func BenchmarkUdpSophosTxt(b *testing.B) { benchmarkUdp(b, sophosTxt) }

// Benchmark that runs with parallelism to help find concurrency related
// issues. To run with parallelism, the 'go test' cpu flag must be set
// greater than 1, otherwise it just runs concurrently but not in parallel.
func BenchmarkParallelUdpParse(b *testing.B) {
	rand.Seed(22)
	numMessages := len(messages)
	dns := newDns(false)
	client := dns.results.(publisher.ChanClient)

	// Drain the results channal while the test is running.
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
			q := messages[rand.Intn(numMessages)]
			var packet *protos.Packet
			if rand.Intn(2) == 0 {
				packet = newPacket(forward, q.request)
			} else {
				packet = newPacket(reverse, q.response)
			}
			dns.ParseUdp(packet)
		}
	})

	defer close(client.Channel)
}

// parseUdpRequestResponse parses a request then a response packet and validates
// the published result.
func parseUdpRequestResponse(t testing.TB, dns *Dns, q DnsTestMessage) {
	packet := newPacket(forward, q.request)
	dns.ParseUdp(packet)
	packet = newPacket(reverse, q.response)
	dns.ParseUdp(packet)
	assert.Empty(t, dns.transactions.Size(), "There should be no transactions.")

	m := expectResult(t, dns)
	assert.Equal(t, "udp", mapValue(t, m, "transport"))
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

// expectResult returns one MapStr result from the Dns results channel. If
// no result is available then the test fails.
func expectResult(t testing.TB, dns *Dns) common.MapStr {
	client := dns.results.(publisher.ChanClient)
	select {
	case result := <-client.Channel:
		return result
	default:
		t.Error("Expected a result to be published.")
	}
	return nil
}

// Retrieves a map value. The key should be the full dotted path to the element.
func mapValue(t testing.TB, m common.MapStr, key string) interface{} {
	return mapValueHelper(t, m, strings.Split(key, "."))
}

// Retrieves nested MapStr values.
func mapValueHelper(t testing.TB, m common.MapStr, keys []string) interface{} {
	key := keys[0]
	if len(keys) == 1 {
		return m[key]
	}

	if len(keys) > 1 {
		value, exists := m[key]
		if !exists {
			t.Fatalf("%s is missing from MapStr %v.", key, m)
		}

		switch typ := value.(type) {
		default:
			t.Fatalf("Expected %s to return a MapStr but got %v.", key, value)
		case common.MapStr:
			return mapValueHelper(t, typ, keys[1:])
		case []common.MapStr:
			var values []interface{}
			for _, m := range typ {
				values = append(values, mapValueHelper(t, m, keys[1:]))
			}
			return values
		}
	}

	panic("mapValueHelper cannot be called with an empty array of keys")
}

// Assert that the published MapStr data matches the data in the DnsTestMessage.
// The validation provided my this method should only be used on results
// published where the response packet was "sent".
// The following fields are validated by this method:
//     type (must be dns)
//     src (ip and port)
//     dst (ip and port)
//     query
//     resource
//     method
//     dns.id
//     dns.op_code
//     dns.flags
//     dns.response_code
//     dns.question.class
//     dns.question.type
//     dns.question.name
//     dns.answers_count
//     dns.answers.data
//     dns.authorities_count
//     dns.authorities
//     dns.additionals_count
//     dns.additionals
func assertMapStrData(t testing.TB, m common.MapStr, q DnsTestMessage) {
	assert.Equal(t, "dns", mapValue(t, m, "type"))
	assertAddress(t, forward, mapValue(t, m, "src"))
	assertAddress(t, reverse, mapValue(t, m, "dst"))
	assert.Equal(t, fmt.Sprintf("class %s, type %s, %s", q.q_class, q.q_type, q.q_name),
		mapValue(t, m, "query"))
	assert.Equal(t, q.q_name, mapValue(t, m, "resource"))
	assert.Equal(t, q.opcode, mapValue(t, m, "method"))
	assert.Equal(t, q.id, mapValue(t, m, "dns.id"))
	assert.Equal(t, q.opcode, mapValue(t, m, "dns.op_code"))
	assertFlags(t, m, q.flags)
	assert.Equal(t, q.rcode, mapValue(t, m, "dns.response_code"))
	assert.Equal(t, q.q_class, mapValue(t, m, "dns.question.class"))
	assert.Equal(t, q.q_type, mapValue(t, m, "dns.question.type"))
	assert.Equal(t, q.q_name, mapValue(t, m, "dns.question.name"))

	// Answers
	assert.Equal(t, len(q.answers), mapValue(t, m, "dns.answers_count"),
		"Expected dns.answers_count to be %d", len(q.answers))
	if len(q.answers) > 0 {
		assert.Len(t, mapValue(t, m, "dns.answers"), len(q.answers),
			"Expected dns.answers to be length %d", len(q.answers))
		for _, ans := range q.answers {
			assert.Contains(t, mapValue(t, m, "dns.answers.data"), ans)
		}
	} else {
		assert.Nil(t, mapValue(t, m, "dns.answers"))
	}

	// Authorities
	assert.Equal(t, len(q.authorities), mapValue(t, m, "dns.authorities_count"),
		"Expected dns.authorities_count to be %d", len(q.authorities))
	if len(q.authorities) > 0 {
		assert.Len(t, mapValue(t, m, "dns.authorities"), len(q.authorities),
			"Expected dns.authorities to be length %d", len(q.authorities))
		for _, ans := range q.authorities {
			assert.Contains(t, mapValue(t, m, "dns.authorities.data"), ans)
		}
	} else {
		assert.Nil(t, mapValue(t, m, "dns.authorities"))
	}

	// Additionals
	assert.Equal(t, len(q.additionals), mapValue(t, m, "dns.additionals_count"),
		"Expected dns.additionals_count to be length %d", len(q.additionals))
	if len(q.additionals) > 0 {
		assert.Len(t, mapValue(t, m, "dns.additionals"), len(q.additionals),
			"Expected dns.additionals to be length %d", len(q.additionals))
		for _, ans := range q.additionals {
			assert.Contains(t, mapValue(t, m, "dns.additionals.data"), ans)
		}
	} else {
		assert.Nil(t, mapValue(t, m, "dns.additionals"))
	}
}

// Assert that the specified flags are set.
func assertFlags(t testing.TB, m common.MapStr, flags []string) {
	for _, expected := range flags {
		var key string
		switch expected {
		default:
			t.Fatalf("Unknown flag '%s' specified in test.", expected)
		case "aa":
			key = "dns.flags.authoritative"
		case "ra":
			key = "dns.flags.recursion_allowed"
		case "rd":
			key = "dns.flags.recursion_desired"
		case "tc":
			key = "dns.flags.truncated_response"
		}

		f := mapValue(t, m, key)
		flag, ok := f.(bool)
		if !ok {
			t.Fatalf("%s value is not a bool.", key)
		}

		assert.True(t, flag, "Flag %s should be true.", key)
	}
}

// Assert that the given Endpoint matches the IP and port in the given
// IpPortTuple.
func assertAddress(t testing.TB, expected common.IpPortTuple, endpoint interface{}) {
	e, ok := endpoint.(*common.Endpoint)
	if !ok {
		t.Errorf("Expected a common.Endpoint but got %v", endpoint)
	}

	assert.Equal(t, expected.Src_ip.String(), e.Ip)
	assert.Equal(t, expected.Src_port, e.Port)
}

// TCP tests

func testTcpTuple() *common.TcpTuple {
	t := &common.TcpTuple{
		Ip_length: 4,
		Src_ip:    net.IPv4(192, 168, 0, 1), Dst_ip: net.IPv4(192, 168, 0, 2),
		Src_port: ClientPort, Dst_port: ServerPort,
	}
	t.ComputeHashebles()
	return t
}

// Verify that an empty packet is safely handled (no panics).
func TestParseTcp_emptyPacket(t *testing.T) {
	dns := newDns(testing.Verbose())
	packet := newPacket(forward, []byte{})
	tcptuple := testTcpTuple()
	private := protos.ProtocolData(new(dnsPrivateData))

	dns.Parse(packet, tcptuple, tcp.TcpDirectionOriginal, private)
	assert.Empty(t, dns.transactions.Size(), "There should be no transactions.")
	client := dns.results.(publisher.ChanClient)
	close(client.Channel)
	assert.Nil(t, <-client.Channel, "No result should have been published.")
}

// Verify that a malformed packet is safely handled (no panics).
func TestParseTcp_malformedPacket(t *testing.T) {
	dns := newDns(testing.Verbose())
	garbage := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13}
	tcptuple := testTcpTuple()
	packet := newPacket(forward, garbage)
	private := protos.ProtocolData(new(dnsPrivateData))

	dns.Parse(packet, tcptuple, tcp.TcpDirectionOriginal, private)
	assert.Empty(t, dns.transactions.Size(), "There should be no transactions.")
}

// Verify that the lone request packet is parsed.
func TestParseTcp_requestPacket(t *testing.T) {
	dns := newDns(testing.Verbose())
	packet := newPacket(forward, elasticATcp.request)
	tcptuple := testTcpTuple()
	private := protos.ProtocolData(new(dnsPrivateData))

	dns.Parse(packet, tcptuple, tcp.TcpDirectionOriginal, private)
	assert.Equal(t, 1, dns.transactions.Size(), "There should be one transaction.")
	client := dns.results.(publisher.ChanClient)
	close(client.Channel)
	assert.Nil(t, <-client.Channel, "No result should have been published.")
}

// Verify that the lone response packet is parsed and that an error
// result is published.
func TestParseTcp_responseOnly(t *testing.T) {
	dns := newDns(testing.Verbose())
	q := elasticATcp
	packet := newPacket(reverse, q.response)
	tcptuple := testTcpTuple()
	private := protos.ProtocolData(new(dnsPrivateData))

	dns.Parse(packet, tcptuple, tcp.TcpDirectionOriginal, private)
	m := expectResult(t, dns)
	assert.Equal(t, "tcp", mapValue(t, m, "transport"))
	assert.Nil(t, mapValue(t, m, "bytes_in"))
	assert.Equal(t, len(q.response), mapValue(t, m, "bytes_out"))
	assert.Nil(t, mapValue(t, m, "responsetime"))
	assert.Equal(t, common.ERROR_STATUS, mapValue(t, m, "status"))
	assert.Equal(t, OrphanedResponseMsg, mapValue(t, m, "notes"))
	assertMapStrData(t, m, q)
}

// Verify that the first request is published without a response and that
// the status is error. This second packet will remain in the transaction
// map awaiting a response.
func TestParseTcp_duplicateRequests(t *testing.T) {
	dns := newDns(testing.Verbose())
	q := elasticATcp
	packet := newPacket(forward, q.request)
	tcptuple := testTcpTuple()
	private := protos.ProtocolData(new(dnsPrivateData))

	dns.Parse(packet, tcptuple, tcp.TcpDirectionOriginal, private)
	assert.Equal(t, 1, dns.transactions.Size(), "There should be one transaction.")
	packet = newPacket(forward, q.request)
	dns.Parse(packet, tcptuple, tcp.TcpDirectionOriginal, private)
	assert.Equal(t, 1, dns.transactions.Size(), "There should be one transaction.")

	m := expectResult(t, dns)
	assert.Equal(t, "tcp", mapValue(t, m, "transport"))
	assert.Equal(t, len(q.request), mapValue(t, m, "bytes_in"))
	assert.Nil(t, mapValue(t, m, "bytes_out"))
	assert.Nil(t, mapValue(t, m, "responsetime"))
	assert.Equal(t, common.ERROR_STATUS, mapValue(t, m, "status"))
	assert.Equal(t, DuplicateQueryMsg, mapValue(t, m, "notes"))
}

// parseTcpRequestResponse parses a request then a response packet and validates
// the published result.
func parseTcpRequestResponse(t testing.TB, dns *Dns, q DnsTestMessage) {
	packet := newPacket(forward, q.request)
	tcptuple := testTcpTuple()
	private := protos.ProtocolData(new(dnsPrivateData))
	dns.Parse(packet, tcptuple, tcp.TcpDirectionOriginal, private)

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

// Verify that the split lone request packet is decoded.
func TestDecodeTcpSplitRequest(t *testing.T) {
	stream := &DnsStream{data: sophosTxtTcp.request[:10], message: new(DnsMessage)}
	_, err := decodeDnsData(TransportTcp, stream.data)

	assert.NotNil(t, err, "Not expecting a complete message yet")

	stream.data = append(stream.data, sophosTxtTcp.request[10:]...)
	_, err = decodeDnsData(TransportTcp, stream.data)

	assert.Nil(t, err, "Message should be complete")
}

// Verify that the split lone request packet is parsed.
func TestParseTcpSplitResponse(t *testing.T) {
	dns := newDns(testing.Verbose())
	tcpQuery := elasticATcp

	q := tcpQuery.request
	r0 := tcpQuery.response[:10]
	r1 := tcpQuery.response[10:]

	tcptuple := testTcpTuple()
	private := protos.ProtocolData(new(dnsPrivateData))

	packet := newPacket(forward, q)
	private = dns.Parse(packet, tcptuple, tcp.TcpDirectionOriginal, private)
	assert.Equal(t, 1, dns.transactions.Size(), "There should be one transaction.")

	packet = newPacket(reverse, r0)
	private = dns.Parse(packet, tcptuple, tcp.TcpDirectionReverse, private)
	assert.Equal(t, 1, dns.transactions.Size(), "There should be one transaction.")

	packet = newPacket(reverse, r1)
	private = dns.Parse(packet, tcptuple, tcp.TcpDirectionReverse, private)
	assert.Empty(t, dns.transactions.Size(), "There should be no transaction.")

	m := expectResult(t, dns)
	assert.Equal(t, "tcp", mapValue(t, m, "transport"))
	assert.Equal(t, len(tcpQuery.request), mapValue(t, m, "bytes_in"))
	assert.Equal(t, len(tcpQuery.response), mapValue(t, m, "bytes_out"))
	assert.NotNil(t, mapValue(t, m, "responsetime"))

	if assert.ObjectsAreEqual("NOERROR", mapValue(t, m, "dns.response_code")) {
		assert.Equal(t, common.OK_STATUS, mapValue(t, m, "status"))
	} else {
		assert.Equal(t, common.ERROR_STATUS, mapValue(t, m, "status"))
	}

	assert.Nil(t, mapValue(t, m, "notes"))
	assertMapStrData(t, m, tcpQuery)
}

func TestGapRequestDrop(t *testing.T) {
	dns := newDns(testing.Verbose())
	q := sophosTxtTcp.request[:10]

	packet := newPacket(forward, q)
	tcptuple := testTcpTuple()
	private := protos.ProtocolData(new(dnsPrivateData))

	private = dns.Parse(packet, tcptuple, tcp.TcpDirectionOriginal, private)

	private, drop := dns.GapInStream(tcptuple, tcp.TcpDirectionOriginal, 10, private)

	assert.Equal(t, true, drop)

	private = dns.ReceivedFin(tcptuple, tcp.TcpDirectionOriginal, private)

	client := dns.results.(publisher.ChanClient)
	close(client.Channel)
	mapStr := <-client.Channel
	assert.Nil(t, mapStr, "No result should have been published.")
}

// Verify that a gap during the response publish the request with Notes
func TestGapResponse(t *testing.T) {
	dns := newDns(testing.Verbose())
	q := sophosTxtTcp.request
	r := sophosTxtTcp.response[:10]

	tcptuple := testTcpTuple()
	private := protos.ProtocolData(new(dnsPrivateData))

	packet := newPacket(forward, q)
	private = dns.Parse(packet, tcptuple, tcp.TcpDirectionOriginal, private)
	assert.Equal(t, 1, dns.transactions.Size(), "There should be one transaction.")

	packet = newPacket(reverse, r)
	private = dns.Parse(packet, tcptuple, tcp.TcpDirectionReverse, private)
	assert.Equal(t, 1, dns.transactions.Size(), "There should be one transaction.")

	private, drop := dns.GapInStream(tcptuple, tcp.TcpDirectionReverse, 10, private)
	assert.Equal(t, true, drop)

	private = dns.ReceivedFin(tcptuple, tcp.TcpDirectionReverse, private)

	client := dns.results.(publisher.ChanClient)
	close(client.Channel)
	mapStr := <-client.Channel
	assert.NotNil(t, mapStr, "One result should have been published.")
	assert.Equal(t, mapStr["notes"], "Response packet's data could not be decoded as DNS.")
	assert.Nil(t, mapStr["answers"])
}

// Verify that a gap/fin happening after a valid query create only one tansaction
func TestGapFinValidMessage(t *testing.T) {
	dns := newDns(testing.Verbose())
	q := sophosTxtTcp.request

	tcptuple := testTcpTuple()
	private := protos.ProtocolData(new(dnsPrivateData))

	packet := newPacket(forward, q)
	private = dns.Parse(packet, tcptuple, tcp.TcpDirectionOriginal, private)
	assert.Equal(t, 1, dns.transactions.Size(), "There should be one transaction.")

	private, drop := dns.GapInStream(tcptuple, tcp.TcpDirectionOriginal, 10, private)
	assert.Equal(t, false, drop)

	private = dns.ReceivedFin(tcptuple, tcp.TcpDirectionReverse, private)
	assert.Equal(t, 1, dns.transactions.Size(), "There should be one transaction.")

	client := dns.results.(publisher.ChanClient)
	close(client.Channel)
	mapStr := <-client.Channel
	assert.Nil(t, mapStr, "No result should have been published.")
	assert.Empty(t, mapStr["notes"], "There should be no notes")
}

// Verify that a Fin during the response publish the request with Notes
func TestFinResponse(t *testing.T) {
	dns := newDns(testing.Verbose())
	q := zoneAxfrTcp.request
	r := zoneAxfrTcp.response[:10]

	tcptuple := testTcpTuple()
	private := protos.ProtocolData(new(dnsPrivateData))

	packet := newPacket(forward, q)
	private = dns.Parse(packet, tcptuple, tcp.TcpDirectionOriginal, private)
	assert.Equal(t, 1, dns.transactions.Size(), "There should be one transaction.")

	packet = newPacket(reverse, r)
	private = dns.Parse(packet, tcptuple, tcp.TcpDirectionReverse, private)
	assert.Equal(t, 1, dns.transactions.Size(), "There should be one transaction.")

	private = dns.ReceivedFin(tcptuple, tcp.TcpDirectionReverse, private)

	client := dns.results.(publisher.ChanClient)
	close(client.Channel)
	mapStr := <-client.Channel
	assert.NotNil(t, mapStr, "One result should have been published.")
	assert.Equal(t, mapStr["notes"], "Response packet's data could not be decoded as DNS.")
	assert.Nil(t, mapStr["answers"])
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
		packet := newPacket(forward, q.request)
		tcptuple := testTcpTuple()
		private := protos.ProtocolData(new(dnsPrivateData))
		dns.Parse(packet, tcptuple, tcp.TcpDirectionOriginal, private)

		packet = newPacket(reverse, q.response)
		tcptuple = testTcpTuple()
		private = protos.ProtocolData(new(dnsPrivateData))
		dns.Parse(packet, tcptuple, tcp.TcpDirectionOriginal, private)

		client := dns.results.(publisher.ChanClient)
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
	client := dns.results.(publisher.ChanClient)

	// Drain the results channal while the test is running.
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
				private = protos.ProtocolData(new(dnsPrivateData))
			} else {
				packet = newPacket(reverse, q.response)
				tcptuple = testTcpTuple()
				private = protos.ProtocolData(new(dnsPrivateData))
			}
			dns.Parse(packet, tcptuple, tcp.TcpDirectionOriginal, private)
		}
	})

	defer close(client.Channel)
}
