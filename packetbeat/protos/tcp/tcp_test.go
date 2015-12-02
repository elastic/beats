package tcp

import (
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/beats/packetbeat/protos"

	"github.com/stretchr/testify/assert"
	"github.com/tsg/gopacket/layers"
)

// Test Constants
const (
	ServerIp   = "192.168.0.1"
	ServerPort = 12345
	ClientIp   = "10.0.0.1"
)

type TestProtocol struct {
	Ports []int
}

var _ protos.ProtocolPlugin = &TestProtocol{}

func (proto *TestProtocol) Init(test_mode bool, results publisher.Client) error {
	return nil
}

func (proto TestProtocol) GetPorts() []int {
	return proto.Ports
}

func (proto TestProtocol) Parse(pkt *protos.Packet, tcptuple *common.TcpTuple,
	dir uint8, private protos.ProtocolData) protos.ProtocolData {
	return private
}

func (proto TestProtocol) ReceivedFin(tcptuple *common.TcpTuple, dir uint8,
	private protos.ProtocolData) protos.ProtocolData {
	return private
}

func (proto TestProtocol) GapInStream(tcptuple *common.TcpTuple, dir uint8,
	nbytes int, private protos.ProtocolData) (priv protos.ProtocolData, drop bool) {
	return private, true
}

func (proto TestProtocol) ConnectionTimeout() time.Duration {
	return 0
}

func Test_configToPortsMap(t *testing.T) {

	type configTest struct {
		Input  map[protos.Protocol]protos.TcpProtocolPlugin
		Output map[uint16]protos.Protocol
	}

	config_tests := []configTest{
		{
			Input: map[protos.Protocol]protos.TcpProtocolPlugin{
				protos.HttpProtocol: &TestProtocol{Ports: []int{80, 8080}},
			},
			Output: map[uint16]protos.Protocol{
				80:   protos.HttpProtocol,
				8080: protos.HttpProtocol,
			},
		},
		{
			Input: map[protos.Protocol]protos.TcpProtocolPlugin{
				protos.HttpProtocol:  &TestProtocol{Ports: []int{80, 8080}},
				protos.MysqlProtocol: &TestProtocol{Ports: []int{3306}},
				protos.RedisProtocol: &TestProtocol{Ports: []int{6379, 6380}},
			},
			Output: map[uint16]protos.Protocol{
				80:   protos.HttpProtocol,
				8080: protos.HttpProtocol,
				3306: protos.MysqlProtocol,
				6379: protos.RedisProtocol,
				6380: protos.RedisProtocol,
			},
		},

		// should ignore duplicate ports in the same protocol
		{
			Input: map[protos.Protocol]protos.TcpProtocolPlugin{
				protos.HttpProtocol:  &TestProtocol{Ports: []int{80, 8080, 8080}},
				protos.MysqlProtocol: &TestProtocol{Ports: []int{3306}},
			},
			Output: map[uint16]protos.Protocol{
				80:   protos.HttpProtocol,
				8080: protos.HttpProtocol,
				3306: protos.MysqlProtocol,
			},
		},
	}

	for _, test := range config_tests {
		output, err := buildPortsMap(test.Input)
		assert.Nil(t, err)
		assert.Equal(t, test.Output, output)
	}
}

func Test_configToPortsMap_negative(t *testing.T) {

	type errTest struct {
		Input map[protos.Protocol]protos.TcpProtocolPlugin
		Err   string
	}

	tests := []errTest{
		{
			// should raise error on duplicate port
			Input: map[protos.Protocol]protos.TcpProtocolPlugin{
				protos.HttpProtocol:  &TestProtocol{Ports: []int{80, 8080}},
				protos.MysqlProtocol: &TestProtocol{Ports: []int{3306}},
				protos.RedisProtocol: &TestProtocol{Ports: []int{6379, 6380, 3306}},
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

// Mock protos.Protocols used for testing the tcp package.
type protocols struct {
	tcp map[protos.Protocol]protos.TcpProtocolPlugin
}

// Verify protocols implements the protos.Protocols interface.
var _ protos.Protocols = &protocols{}

func (p protocols) BpfFilter(with_vlans bool, with_icmp bool) string             { return "" }
func (p protocols) GetTcp(proto protos.Protocol) protos.TcpProtocolPlugin        { return p.tcp[proto] }
func (p protocols) GetUdp(proto protos.Protocol) protos.UdpProtocolPlugin        { return nil }
func (p protocols) GetAll() map[protos.Protocol]protos.ProtocolPlugin            { return nil }
func (p protocols) GetAllTcp() map[protos.Protocol]protos.TcpProtocolPlugin      { return p.tcp }
func (p protocols) GetAllUdp() map[protos.Protocol]protos.UdpProtocolPlugin      { return nil }
func (p protocols) Register(proto protos.Protocol, plugin protos.ProtocolPlugin) { return }

// Benchmark that runs with parallelism to help find concurrency related
// issues. To run with parallelism, the 'go test' cpu flag must be set
// greater than 1, otherwise it just runs concurrently but not in parallel.
func BenchmarkParallelProcess(b *testing.B) {
	rand.Seed(18)
	p := protocols{}
	p.tcp = make(map[protos.Protocol]protos.TcpProtocolPlugin)
	p.tcp[1] = &TestProtocol{Ports: []int{ServerPort}}
	tcp, _ := NewTcp(p)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			pkt := &protos.Packet{
				Ts: time.Now(),
				Tuple: common.NewIpPortTuple(4,
					net.ParseIP(ServerIp), ServerPort,
					net.ParseIP(ClientIp), uint16(rand.Intn(65535))),
				Payload: []byte{1, 2, 3, 4},
			}
			tcp.Process(&layers.TCP{}, pkt)
		}
	})
}
