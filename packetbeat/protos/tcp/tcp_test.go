package tcp

import (
	"fmt"
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

	init  func(testMode bool, results publisher.Client) error
	parse func(*protos.Packet, *common.TcpTuple, uint8, protos.ProtocolData) protos.ProtocolData
	onFin func(*common.TcpTuple, uint8, protos.ProtocolData) protos.ProtocolData
	gap   func(*common.TcpTuple, uint8, int, protos.ProtocolData) (protos.ProtocolData, bool)
}

var _ protos.ProtocolPlugin = &TestProtocol{
	init: func(m bool, r publisher.Client) error { return nil },
	parse: func(p *protos.Packet, t *common.TcpTuple, d uint8, priv protos.ProtocolData) protos.ProtocolData {
		return priv
	},
	onFin: func(t *common.TcpTuple, d uint8, p protos.ProtocolData) protos.ProtocolData {
		return p
	},
	gap: func(t *common.TcpTuple, d uint8, b int, p protos.ProtocolData) (protos.ProtocolData, bool) {
		return p, true
	},
}

func (proto *TestProtocol) Init(test_mode bool, results publisher.Client) error {
	return proto.init(test_mode, results)
}

func (proto TestProtocol) GetPorts() []int {
	return proto.Ports
}

func (proto TestProtocol) Parse(pkt *protos.Packet, tcptuple *common.TcpTuple,
	dir uint8, private protos.ProtocolData) protos.ProtocolData {
	return proto.parse(pkt, tcptuple, dir, private)
}

func (proto TestProtocol) ReceivedFin(tcptuple *common.TcpTuple, dir uint8,
	private protos.ProtocolData) protos.ProtocolData {
	return proto.onFin(tcptuple, dir, private)
}

func (proto TestProtocol) GapInStream(tcptuple *common.TcpTuple, dir uint8,
	nbytes int, private protos.ProtocolData) (priv protos.ProtocolData, drop bool) {
	return proto.gap(tcptuple, dir, nbytes, private)
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

func TestGapInStreamShouldDropState(t *testing.T) {
	gap := 0
	var state []byte

	data1 := []byte{1, 2, 3, 4}
	data2 := []byte{5, 6, 7, 8}

	tp := &TestProtocol{Ports: []int{ServerPort}}
	tp.gap = func(t *common.TcpTuple, d uint8, n int, p protos.ProtocolData) (protos.ProtocolData, bool) {
		fmt.Printf("lost: %v\n", n)
		gap += n
		return p, true // drop state
	}
	tp.parse = func(p *protos.Packet, t *common.TcpTuple, d uint8, priv protos.ProtocolData) protos.ProtocolData {
		if priv == nil {
			state = nil
		}
		state = append(state, p.Payload...)
		return state
	}

	p := protocols{}
	p.tcp = map[protos.Protocol]protos.TcpProtocolPlugin{
		protos.HttpProtocol: tp,
	}
	tcp, _ := NewTcp(p)

	addr := common.NewIpPortTuple(4,
		net.ParseIP(ServerIp), ServerPort,
		net.ParseIP(ClientIp), uint16(rand.Intn(65535)))

	hdr := &layers.TCP{}
	tcp.Process(nil, hdr, &protos.Packet{Ts: time.Now(), Tuple: addr, Payload: data1})
	hdr.Seq += uint32(len(data1) + 10)
	tcp.Process(nil, hdr, &protos.Packet{Ts: time.Now(), Tuple: addr, Payload: data2})

	// validate
	assert.Equal(t, 10, gap)
	assert.Equal(t, data2, state)
}

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
			tcp.Process(nil, &layers.TCP{}, pkt)
		}
	})
}
