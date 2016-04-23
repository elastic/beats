// +build !integration

// Common variables, functions and tests for the dns package tests

package dns

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/publish"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/stretchr/testify/assert"
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
	q_etld      string
	answers     []string
	authorities []string
	additionals []string
	request     []byte
	response    []byte
}

// Request and response addresses.
var (
	forward = common.NewIpPortTuple(4,
		net.ParseIP(ServerIp), ServerPort,
		net.ParseIP(ClientIp), ClientPort)
	reverse = common.NewIpPortTuple(4,
		net.ParseIP(ClientIp), ClientPort,
		net.ParseIP(ServerIp), ServerPort)
)

func newDns(verbose bool) *Dns {
	if verbose {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"dns"})
	} else {
		logp.LogInit(logp.LOG_EMERG, "", false, true, []string{"dns"})
	}

	results := &publish.ChanTransactions{make(chan common.MapStr, 100)}
	cfg, _ := common.NewConfigFrom(map[string]interface{}{
		"ports":               []int{ServerPort},
		"include_authorities": true,
		"include_additionals": true,
		"send_request":        true,
		"send_response":       true,
	})
	dns, err := New(false, results, cfg)
	if err != nil {
		panic(err)
	}

	return dns.(*Dns)
}

func newPacket(t common.IpPortTuple, payload []byte) *protos.Packet {
	return &protos.Packet{
		Ts:      time.Now(),
		Tuple:   t,
		Payload: payload,
	}
}

// expectResult returns one MapStr result from the Dns results channel. If
// no result is available then the test fails.
func expectResult(t testing.TB, dns *Dns) common.MapStr {
	client := dns.results.(*publish.ChanTransactions)
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
	assertRequest(t, m, q)

	// Answers
	assertFlags(t, m, q.flags)
	assert.Equal(t, q.rcode, mapValue(t, m, "dns.response_code"))

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

func assertRequest(t testing.TB, m common.MapStr, q DnsTestMessage) {
	assert.Equal(t, "dns", mapValue(t, m, "type"))
	assertAddress(t, forward, mapValue(t, m, "src"))
	assertAddress(t, reverse, mapValue(t, m, "dst"))
	assert.Equal(t, fmt.Sprintf("class %s, type %s, %s", q.q_class, q.q_type, q.q_name),
		mapValue(t, m, "query"))
	assert.Equal(t, q.q_name, mapValue(t, m, "resource"))
	assert.Equal(t, q.opcode, mapValue(t, m, "method"))
	assert.Equal(t, q.id, mapValue(t, m, "dns.id"))
	assert.Equal(t, q.opcode, mapValue(t, m, "dns.op_code"))
	assert.Equal(t, q.q_class, mapValue(t, m, "dns.question.class"))
	assert.Equal(t, q.q_type, mapValue(t, m, "dns.question.type"))
	assert.Equal(t, q.q_name, mapValue(t, m, "dns.question.name"))
	assert.Equal(t, q.q_etld, mapValue(t, m, "dns.question.etld_plus_one"))
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
