// +build !integration

package udp

import (
	"net"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/packetbeat/protos"

	// import plugins for testing
	_ "github.com/elastic/beats/packetbeat/protos/http"
	_ "github.com/elastic/beats/packetbeat/protos/mysql"
	_ "github.com/elastic/beats/packetbeat/protos/redis"

	"github.com/stretchr/testify/assert"
)

// Protocol ID and port number used by TestProtocol in various tests.
const (
	PROTO = protos.Protocol(1)
	PORT  = 1234
)

var (
	httpProtocol  = protos.Lookup("http")
	mysqlProtocol = protos.Lookup("mysql")
	redisProtocol = protos.Lookup("redis")
)

type TestProtocols struct {
	udp map[protos.Protocol]protos.UDPPlugin
}

func (p TestProtocols) BpfFilter(withVlans bool, withICMP bool) string {
	return "mock bpf filter"
}

func (p TestProtocols) GetTCP(proto protos.Protocol) protos.TCPPlugin {
	return nil
}

func (p TestProtocols) GetUDP(proto protos.Protocol) protos.UDPPlugin {
	return p.udp[proto]
}

func (p TestProtocols) GetAll() map[protos.Protocol]protos.Plugin {
	return nil
}

func (p TestProtocols) GetAllTCP() map[protos.Protocol]protos.TCPPlugin {
	return nil
}

func (p TestProtocols) GetAllUDP() map[protos.Protocol]protos.UDPPlugin {
	return p.udp
}

func (p TestProtocols) Register(proto protos.Protocol, plugin protos.Plugin) {
	return
}

type TestProtocol struct {
	Ports []int          // Ports that the protocol operates on.
	pkt   *protos.Packet // UDP packet that the plugin was called to process.
}

func (proto *TestProtocol) Init(testMode bool, results protos.Reporter) error {
	return nil
}

func (proto *TestProtocol) GetPorts() []int {
	return proto.Ports
}

func (proto *TestProtocol) ParseUDP(pkt *protos.Packet) {
	proto.pkt = pkt
}

type TestStruct struct {
	protocols *TestProtocols
	udp       *UDP
	plugin    *TestProtocol
}

// Helper method for creating mocks and the Udp instance under test.
func testSetup(t *testing.T) *TestStruct {
	logp.TestingSetup(logp.WithSelectors("udp"))

	protocols := &TestProtocols{}
	protocols.udp = make(map[protos.Protocol]protos.UDPPlugin)
	plugin := &TestProtocol{Ports: []int{PORT}}
	protocols.udp[PROTO] = plugin

	udp, err := NewUDP(protocols)
	if err != nil {
		t.Error("Error creating UDP handler: ", err)
	}

	return &TestStruct{protocols: protocols, udp: udp, plugin: plugin}
}

func Test_buildPortsMap(t *testing.T) {
	type configTest struct {
		Input  map[protos.Protocol]protos.UDPPlugin
		Output map[uint16]protos.Protocol
	}

	// The protocols named here are not necessarily UDP. They are just used
	// for testing purposes.
	configTests := []configTest{
		{
			Input: map[protos.Protocol]protos.UDPPlugin{
				httpProtocol: &TestProtocol{Ports: []int{80, 8080}},
			},
			Output: map[uint16]protos.Protocol{
				80:   httpProtocol,
				8080: httpProtocol,
			},
		},
		{
			Input: map[protos.Protocol]protos.UDPPlugin{
				httpProtocol:  &TestProtocol{Ports: []int{80, 8080}},
				mysqlProtocol: &TestProtocol{Ports: []int{3306}},
				redisProtocol: &TestProtocol{Ports: []int{6379, 6380}},
			},
			Output: map[uint16]protos.Protocol{
				80:   httpProtocol,
				8080: httpProtocol,
				3306: mysqlProtocol,
				6379: redisProtocol,
				6380: redisProtocol,
			},
		},

		// should ignore duplicate ports in the same protocol
		{
			Input: map[protos.Protocol]protos.UDPPlugin{
				httpProtocol:  &TestProtocol{Ports: []int{80, 8080, 8080}},
				mysqlProtocol: &TestProtocol{Ports: []int{3306}},
			},
			Output: map[uint16]protos.Protocol{
				80:   httpProtocol,
				8080: httpProtocol,
				3306: mysqlProtocol,
			},
		},
	}

	for _, test := range configTests {
		output, err := buildPortsMap(test.Input)
		assert.Nil(t, err)
		assert.Equal(t, test.Output, output)
	}
}

// Verify that buildPortsMap returns an error when two plugins are registered
// for the same port number.
func Test_buildPortsMap_portOverlapError(t *testing.T) {
	type errTest struct {
		Input map[protos.Protocol]protos.UDPPlugin
		Err   string
	}

	// The protocols named here are not necessarily UDP. They are just used
	// for testing purposes.
	tests := []errTest{
		{
			// Should raise error on duplicate port
			Input: map[protos.Protocol]protos.UDPPlugin{
				httpProtocol:  &TestProtocol{Ports: []int{80, 8080}},
				mysqlProtocol: &TestProtocol{Ports: []int{3306}},
				redisProtocol: &TestProtocol{Ports: []int{6379, 6380, 3306}},
			},
			Err: "Duplicate port (3306) exists",
		},
	}

	for _, test := range tests {
		_, err := buildPortsMap(test.Input)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), test.Err)
	}
}

// Verify that decideProtocol returns the protocol assocated with the
// packet's source port.
func Test_decideProtocol_bySrcPort(t *testing.T) {
	test := testSetup(t)
	tuple := common.NewIPPortTuple(4,
		net.ParseIP("192.168.0.1"), PORT,
		net.ParseIP("10.0.0.1"), 34898)
	assert.Equal(t, PROTO, test.udp.decideProtocol(&tuple))
}

// Verify that decideProtocol returns the protocol assocated with the
// packet's destination port.
func Test_decideProtocol_byDstPort(t *testing.T) {
	test := testSetup(t)
	tuple := common.NewIPPortTuple(4,
		net.ParseIP("10.0.0.1"), 34898,
		net.ParseIP("192.168.0.1"), PORT)
	assert.Equal(t, PROTO, test.udp.decideProtocol(&tuple))
}

// Verify that decideProtocol returns UnknownProtocol when given packet for
// which it does not have a plugin.
func TestProcess_unknownProtocol(t *testing.T) {
	test := testSetup(t)
	tuple := common.NewIPPortTuple(4,
		net.ParseIP("10.0.0.1"), 34898,
		net.ParseIP("192.168.0.1"), PORT+1)
	assert.Equal(t, protos.UnknownProtocol, test.udp.decideProtocol(&tuple))
}

// Verify that Process ignores empty packets.
func TestProcess_emptyPayload(t *testing.T) {
	test := testSetup(t)
	tuple := common.NewIPPortTuple(4,
		net.ParseIP("192.168.0.1"), PORT,
		net.ParseIP("10.0.0.1"), 34898)
	emptyPkt := &protos.Packet{Ts: time.Now(), Tuple: tuple, Payload: []byte{}}
	test.udp.Process(nil, emptyPkt)
	assert.Nil(t, test.plugin.pkt)
}

// Verify that Process finds the plugin associated with the packet and invokes
// ProcessUdp on it.
func TestProcess_nonEmptyPayload(t *testing.T) {
	test := testSetup(t)
	tuple := common.NewIPPortTuple(4,
		net.ParseIP("192.168.0.1"), PORT,
		net.ParseIP("10.0.0.1"), 34898)
	payload := []byte{1}
	pkt := &protos.Packet{Ts: time.Now(), Tuple: tuple, Payload: payload}
	test.udp.Process(nil, pkt)
	assert.Equal(t, pkt, test.plugin.pkt)
}
