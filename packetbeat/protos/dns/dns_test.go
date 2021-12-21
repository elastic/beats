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

// Common variables, functions and tests for the dns package tests

package dns

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	mkdns "github.com/miekg/dns"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/packetbeat/procs"
	"github.com/elastic/beats/v7/packetbeat/protos"
	"github.com/elastic/beats/v7/packetbeat/publish"
)

// Test Constants
const (
	serverIP   = "192.168.0.1"
	serverPort = 53
	clientIP   = "10.0.0.1"
	clientPort = 34898
)

// DnsTestMessage holds the data that is expected to be returned when parsing
// the raw DNS layer payloads for the request and response packet.
type dnsTestMessage struct {
	id          uint16
	opcode      string
	flags       []string
	rcode       string
	qClass      string
	qType       string
	qName       string
	qEtld       string
	qSubdomain  interface{}
	qTLD        interface{}
	answers     []string
	authorities []string
	additionals []string
	request     []byte
	response    []byte
}

// Request and response addresses.
var (
	forward = common.NewIPPortTuple(4,
		net.ParseIP(serverIP), serverPort,
		net.ParseIP(clientIP), clientPort)
	reverse = common.NewIPPortTuple(4,
		net.ParseIP(clientIP), clientPort,
		net.ParseIP(serverIP), serverPort)
)

type eventStore struct {
	events []beat.Event
}

func (e *eventStore) publish(event beat.Event) {
	publish.MarshalPacketbeatFields(&event, nil, nil)
	e.events = append(e.events, event)
}

func (e *eventStore) empty() bool {
	return len(e.events) == 0
}

func newDNS(store *eventStore, verbose bool) *dnsPlugin {
	level := logp.WarnLevel
	if verbose {
		level = logp.DebugLevel
	}
	logp.DevelopmentSetup(
		logp.WithLevel(level),
		logp.WithSelectors("dns"),
	)

	callback := func(beat.Event) {}
	if store != nil {
		callback = store.publish
	}

	cfg, _ := common.NewConfigFrom(map[string]interface{}{
		"ports":               []int{serverPort},
		"include_authorities": true,
		"include_additionals": true,
		"send_request":        true,
		"send_response":       true,
	})
	dns, err := New(false, callback, procs.ProcessesWatcher{}, cfg)
	if err != nil {
		panic(err)
	}

	return dns.(*dnsPlugin)
}

func newPacket(t common.IPPortTuple, payload []byte) *protos.Packet {
	return &protos.Packet{
		Ts:      time.Now(),
		Tuple:   t,
		Payload: payload,
	}
}

// expectResult returns one MapStr result from the Dns results channel. If
// no result is available then the test fails.
func expectResult(t testing.TB, e *eventStore) common.MapStr {
	if len(e.events) == 0 {
		t.Error("No transaction")
		return nil
	}

	event := e.events[0]
	e.events = e.events[1:]
	return event.Fields
}

// Retrieves a map value. The key should be the full dotted path to the element.
func mapValue(t testing.TB, m common.MapStr, key string) interface{} {
	t.Helper()
	return mapValueHelper(t, m, strings.Split(key, "."))
}

// Retrieves nested MapStr values.
func mapValueHelper(t testing.TB, m common.MapStr, keys []string) interface{} {
	t.Helper()

	key := keys[0]
	if len(keys) == 1 {
		return m[key]
	}

	if len(keys) > 1 {
		value, exists := m[key]
		if !exists {
			return nil
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
func assertMapStrData(t testing.TB, m common.MapStr, q dnsTestMessage) {
	t.Helper()

	assertRequest(t, m, q)

	// Answers
	assertFlags(t, m, q.flags)
	assert.Equal(t, q.rcode, mapValue(t, m, "dns.response_code"))

	truncated, ok := mapValue(t, m, "dns.flags.truncated_response").(bool)
	if !ok {
		t.Fatal("dns.flags.truncated_response value is not a bool.")
	}
	if !truncated {
		assert.Equal(t, len(q.answers), mapValue(t, m, "dns.answers_count"),
			"Expected dns.answers_count to be %d", len(q.answers))
	}
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

func assertRequest(t testing.TB, m common.MapStr, q dnsTestMessage) {
	t.Helper()

	assert.Equal(t, "dns", mapValue(t, m, "type"))
	assert.Equal(t, forward.SrcIP.String(), mapValue(t, m, "source.ip"))
	assert.EqualValues(t, forward.SrcPort, mapValue(t, m, "source.port"))
	assert.Equal(t, forward.DstIP.String(), mapValue(t, m, "destination.ip"))
	assert.EqualValues(t, forward.DstPort, mapValue(t, m, "destination.port"))
	assert.Equal(t, fmt.Sprintf("class %s, type %s, %s", q.qClass, q.qType, q.qName), mapValue(t, m, "query"))
	assert.Equal(t, q.qName, mapValue(t, m, "resource"))
	assert.Equal(t, q.opcode, mapValue(t, m, "method"))
	assert.Equal(t, q.id, mapValue(t, m, "dns.id"))
	assert.Equal(t, q.opcode, mapValue(t, m, "dns.op_code"))
	assert.Equal(t, q.qClass, mapValue(t, m, "dns.question.class"))
	assert.Equal(t, q.qType, mapValue(t, m, "dns.question.type"))
	assert.Equal(t, q.qName, mapValue(t, m, "dns.question.name"))
	assert.Equal(t, q.qTLD, mapValue(t, m, "dns.question.top_level_domain"))
	assert.Equal(t, q.qSubdomain, mapValue(t, m, "dns.question.subdomain"))
	assert.Equal(t, q.qEtld, mapValue(t, m, "dns.question.etld_plus_one"))
	assert.Equal(t, q.qEtld, mapValue(t, m, "dns.question.registered_domain"))
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
		case "ad":
			key = "dns.flags.authentic_data"
		case "cd":
			key = "dns.flags.checking_disabled"
		case "ra":
			key = "dns.flags.recursion_available"
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

func TestRRsToMapStrsWithOPTRecord(t *testing.T) {
	o := new(mkdns.OPT)
	o.Hdr.Name = "." // MUST be the root zone, per definition.
	o.Hdr.Rrtype = mkdns.TypeOPT

	r := new(mkdns.MX)
	r.Hdr = mkdns.RR_Header{
		Name: "miek.nl", Rrtype: mkdns.TypeMX,
		Class: mkdns.ClassINET, Ttl: 3600,
	}
	r.Preference = 10
	r.Mx = "mx.miek.nl"

	// The OPT record is a pseudo-record so it doesn't become a real record
	// in our conversion, and there will be 1 entry instead of 2.
	mapStrs, _ := rrsToMapStrs([]mkdns.RR{o, r}, false)
	assert.Len(t, mapStrs, 1)

	mapStr := mapStrs[0]
	assert.Equal(t, "IN", mapStr["class"])
	assert.Equal(t, "MX", mapStr["type"])
	assert.Equal(t, "mx.miek.nl", mapStr["data"])
	assert.Equal(t, "miek.nl", mapStr["name"])
	assert.EqualValues(t, 10, mapStr["preference"])
}
