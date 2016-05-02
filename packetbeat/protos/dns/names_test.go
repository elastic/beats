// +build !integration

// Unit tests and benchmarks for the dns package.
// This file contains tests for queries' RR type
//
// TODO:
//   * Add validation of special fields provided in MX, SOA, NS...
//   * Use struct DnsTestMsg fields question, answers, authorities,... for struct DnsTestMessage

package dns

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"

	"github.com/stretchr/testify/assert"
)

type DnsTestMsg struct {
	rawData     []byte
	question    common.MapStr
	answers     []common.MapStr
	authorities []common.MapStr
	additionals []common.MapStr
	opt         common.MapStr
}

// DNS messages for testing.
var (
	// An array of all test messages.
	dnsTestRRs = []DnsTestMsg{
		unhandledRR,
		unknownRR,
		opt,
	}

	unhandledRR = DnsTestMsg{ // RR specified in a RFC but not implemented in the package dns
		rawData: []byte{
			0x21, 0x51, 0x01, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x07, 0x65, 0x6c, 0x61,
			0x73, 0x74, 0x69, 0x63, 0x02, 0x63, 0x6f, 0x00, 0x00, 0x1e, 0x00, 0x01,
		},
		question: common.MapStr{
			"type": "NXT",
			"name": "elastic.co.",
		},
	}

	unknownRR = DnsTestMsg{ // RR unspecified in any known RFC
		rawData: []byte{
			0x21, 0x51, 0x01, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x07, 0x65, 0x6c, 0x61,
			0x73, 0x74, 0x69, 0x63, 0x02, 0x63, 0x6f, 0x00, 0xff, 0x00, 0x00, 0x01,
		},
		question: common.MapStr{
			"type": "65280",
			"name": "elastic.co.",
		},
	}

	opt = DnsTestMsg{
		rawData: []byte{
			0x50, 0x12, 0x01, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x03, 0x77, 0x77, 0x77,
			0x04, 0x69, 0x65, 0x74, 0x66, 0x03, 0x6f, 0x72, 0x67, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00,
			0x29, 0x10, 0x00, 0x00, 0x00, 0x80, 0x00, 0x00, 0x00,
		},
		question: common.MapStr{
			"type": "A",
			"name": "www.ietf.org.",
		},
		opt: common.MapStr{
			"version": "0",
			"do":      true,
		},
	}
)

// oracleRRs and rrs should be sorted in the same order
func assertRRs(t testing.TB, oracleRRs []common.MapStr, rrs []common.MapStr) {
	assert.Equal(t, len(oracleRRs), len(rrs))
	for i, oracleRR := range oracleRRs {
		rr := rrs[i]
		for k, v := range oracleRR {
			assert.NotNil(t, rr[k])
			assert.Equal(t, v, rr[k])
		}
	}
}

func assertDnsMessage(t testing.TB, q DnsTestMsg) {
	dns, err := decodeDnsData(TransportUdp, q.rawData)
	if err != nil {
		t.Error("failed to decode dns data")
	}

	mapStr := common.MapStr{}
	addDnsToMapStr(mapStr, dns, true, true)
	if q.question != nil {
		for k, v := range q.question {
			assert.NotNil(t, mapStr["question"].(common.MapStr)[k])
			assert.Equal(t, v, mapStr["question"].(common.MapStr)[k])
		}
	}
	if len(q.answers) > 0 {
		assertRRs(t, q.answers, mapStr["answer"].([]common.MapStr))
	}
	if len(q.authorities) > 0 {
		assertRRs(t, q.authorities, mapStr["authorities"].([]common.MapStr))
	}
	if len(q.additionals) > 0 {
		assertRRs(t, q.additionals, mapStr["additionals"].([]common.MapStr))
	}
	if q.opt != nil {
		for k, v := range q.opt {
			assert.NotNil(t, mapStr["opt"].(common.MapStr)[k])
			assert.Equal(t, v, mapStr["opt"].(common.MapStr)[k])
		}
	}
}

func TestAllRR(t *testing.T) {
	for _, q := range dnsTestRRs {
		assertDnsMessage(t, q)
	}
}
