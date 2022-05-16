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

// Unit tests and benchmarks for the dns package.
// This file contains tests for queries' RR type
//
// TODO:
//   * Add validation of special fields provided in MX, SOA, NS...
//   * Use struct DnsTestMsg fields question, answers, authorities,... for struct DnsTestMessage

package dns

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/packetbeat/pb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type dnsTestMsg struct {
	rawData     []byte
	question    mapstr.M
	answers     []mapstr.M
	authorities []mapstr.M
	additionals []mapstr.M
	opt         mapstr.M
}

// DNS messages for testing.
var (
	// An array of all test messages.
	dnsTestRRs = []dnsTestMsg{
		unhandledRR,
		unknownRR,
		opt,
	}

	unhandledRR = dnsTestMsg{ // RR specified in a RFC but not implemented in the package dns
		rawData: []byte{
			0x21, 0x51, 0x01, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x07, 0x65, 0x6c, 0x61,
			0x73, 0x74, 0x69, 0x63, 0x02, 0x63, 0x6f, 0x00, 0x00, 0x1e, 0x00, 0x01,
		},
		question: mapstr.M{
			"type": "NXT",
			"name": "elastic.co",
		},
	}

	unknownRR = dnsTestMsg{ // RR unspecified in any known RFC
		rawData: []byte{
			0x21, 0x51, 0x01, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x07, 0x65, 0x6c, 0x61,
			0x73, 0x74, 0x69, 0x63, 0x02, 0x63, 0x6f, 0x00, 0xff, 0x00, 0x00, 0x01,
		},
		question: mapstr.M{
			"type": "65280",
			"name": "elastic.co",
		},
	}

	opt = dnsTestMsg{
		rawData: []byte{
			0x50, 0x12, 0x01, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x03, 0x77, 0x77, 0x77,
			0x04, 0x69, 0x65, 0x74, 0x66, 0x03, 0x6f, 0x72, 0x67, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00,
			0x29, 0x10, 0x00, 0x00, 0x00, 0x80, 0x00, 0x00, 0x00,
		},
		question: mapstr.M{
			"type": "A",
			"name": "www.ietf.org",
		},
		opt: mapstr.M{
			"version": "0",
			"do":      true,
		},
	}
)

// oracleRRs and rrs should be sorted in the same order
func assertRRs(t testing.TB, oracleRRs []mapstr.M, rrs []mapstr.M) {
	assert.Equal(t, len(oracleRRs), len(rrs))
	for i, oracleRR := range oracleRRs {
		rr := rrs[i]
		for k, v := range oracleRR {
			assert.NotNil(t, rr[k])
			assert.Equal(t, v, rr[k])
		}
	}
}

func assertDNSMessage(t testing.TB, q dnsTestMsg) {
	dns, err := decodeDNSData(transportUDP, q.rawData)
	if err != nil {
		t.Error("failed to decode dns data")
	}

	mapStr := mapstr.M{}
	addDNSToMapStr(mapStr, pb.NewFields(), dns, true, true)
	if q.question != nil {
		for k, v := range q.question {
			assert.NotNil(t, mapStr["question"].(mapstr.M)[k])
			assert.Equal(t, v, mapStr["question"].(mapstr.M)[k])
		}
	}
	if len(q.answers) > 0 {
		assertRRs(t, q.answers, mapStr["answer"].([]mapstr.M))
	}
	if len(q.authorities) > 0 {
		assertRRs(t, q.authorities, mapStr["authorities"].([]mapstr.M))
	}
	if len(q.additionals) > 0 {
		assertRRs(t, q.additionals, mapStr["additionals"].([]mapstr.M))
	}
	if q.opt != nil {
		for k, v := range q.opt {
			assert.NotNil(t, mapStr["opt"].(mapstr.M)[k])
			assert.Equal(t, v, mapStr["opt"].(mapstr.M)[k])
		}
	}
}

func TestAllRR(t *testing.T) {
	for _, q := range dnsTestRRs {
		assertDNSMessage(t, q)
	}
}
