package protos

import (
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/packetbeat/publish"

	"github.com/stretchr/testify/assert"
)

type TestProtocol struct {
	Ports []int
}

type TcpProtocol TestProtocol

func (proto *TcpProtocol) Init(test_mode bool, results publish.Transactions) error {
	return nil
}

func (proto *TcpProtocol) Flush() {
	return
}

func (proto *TcpProtocol) GetPorts() []int {
	return proto.Ports
}

func (proto *TcpProtocol) Parse(pkt *Packet, tcptuple *common.TcpTuple,
	dir uint8, private ProtocolData) ProtocolData {
	return private
}

func (proto *TcpProtocol) ReceivedFin(tcptuple *common.TcpTuple, dir uint8,
	private ProtocolData) ProtocolData {
	return private
}

func (proto *TcpProtocol) GapInStream(tcptuple *common.TcpTuple, dir uint8,
	nbytes int, private ProtocolData) (priv ProtocolData, drop bool) {
	return private, true
}

func (proto *TcpProtocol) ConnectionTimeout() time.Duration { return 0 }

type UdpProtocol TestProtocol

func (proto *UdpProtocol) Init(test_mode bool, results publish.Transactions) error {
	return nil
}

func (proto *UdpProtocol) Flush() {
	return
}

func (proto *UdpProtocol) GetPorts() []int {
	return proto.Ports
}

func (proto *UdpProtocol) ParseUdp(pkt *Packet) {
	return
}

type TcpUdpProtocol TestProtocol

func (proto *TcpUdpProtocol) Init(test_mode bool, results publish.Transactions) error {
	return nil
}

func (proto *TcpUdpProtocol) Flush() {
	return
}

func (proto *TcpUdpProtocol) GetPorts() []int {
	return proto.Ports
}

func (proto *TcpUdpProtocol) Parse(pkt *Packet, tcptuple *common.TcpTuple,
	dir uint8, private ProtocolData) ProtocolData {
	return private
}

func (proto *TcpUdpProtocol) ReceivedFin(tcptuple *common.TcpTuple, dir uint8,
	private ProtocolData) ProtocolData {
	return private
}

func (proto *TcpUdpProtocol) GapInStream(tcptuple *common.TcpTuple, dir uint8,
	nbytes int, private ProtocolData) (priv ProtocolData, drop bool) {
	return private, true
}

func (proto *TcpUdpProtocol) ParseUdp(pkt *Packet) {
	return
}

func (proto *TcpUdpProtocol) ConnectionTimeout() time.Duration { return 0 }

func TestProtocolNames(t *testing.T) {
	assert.Equal(t, "unknown", UnknownProtocol.String())
	assert.Equal(t, "http", HttpProtocol.String())
	assert.Equal(t, "mysql", MysqlProtocol.String())
	assert.Equal(t, "redis", RedisProtocol.String())
	assert.Equal(t, "pgsql", PgsqlProtocol.String())
	assert.Equal(t, "thrift", ThriftProtocol.String())
	assert.Equal(t, "mongodb", MongodbProtocol.String())

	assert.Equal(t, "impossible", Protocol(100).String())
}

func newProtocols() Protocols {
	p := ProtocolsStruct{}
	p.all = make(map[Protocol]ProtocolPlugin)
	p.tcp = make(map[Protocol]TcpProtocolPlugin)
	p.udp = make(map[Protocol]UdpProtocolPlugin)

	tcp := &TcpProtocol{Ports: []int{80}}
	udp := &UdpProtocol{Ports: []int{5060}}
	tcpUdp := &TcpUdpProtocol{Ports: []int{53}}

	p.Register(1, tcp)
	p.Register(2, udp)
	p.Register(3, tcpUdp)
	return p
}

func TestBpfFilterWithoutVlanOnlyIcmp(t *testing.T) {
	p := ProtocolsStruct{}
	p.all = make(map[Protocol]ProtocolPlugin)
	p.tcp = make(map[Protocol]TcpProtocolPlugin)
	p.udp = make(map[Protocol]UdpProtocolPlugin)

	filter := p.BpfFilter(false, true)
	assert.Equal(t, "icmp or icmp6", filter)
}

func TestBpfFilterWithoutVlanWithoutIcmp(t *testing.T) {
	p := newProtocols()
	filter := p.BpfFilter(false, false)
	assert.Equal(t, "tcp port 80 or udp port 5060 or port 53", filter)
}

func TestBpfFilterWithVlanWithoutIcmp(t *testing.T) {
	p := newProtocols()
	filter := p.BpfFilter(true, false)
	assert.Equal(t, "tcp port 80 or udp port 5060 or port 53 or "+
		"(vlan and (tcp port 80 or udp port 5060 or port 53))", filter)
}

func TestBpfFilterWithoutVlanWithIcmp(t *testing.T) {
	p := newProtocols()
	filter := p.BpfFilter(false, true)
	assert.Equal(t, "tcp port 80 or udp port 5060 or port 53 or icmp or icmp6", filter)
}

func TestBpfFilterWithVlanWithIcmp(t *testing.T) {
	p := newProtocols()
	filter := p.BpfFilter(true, true)
	assert.Equal(t, "tcp port 80 or udp port 5060 or port 53 or icmp or icmp6 or "+
		"(vlan and (tcp port 80 or udp port 5060 or port 53 or icmp or icmp6))", filter)
}

func TestGetAll(t *testing.T) {
	p := newProtocols()
	all := p.GetAll()
	assert.NotNil(t, all[1])
	assert.NotNil(t, all[2])
	assert.NotNil(t, all[3])
}

func TestGetAllTcp(t *testing.T) {
	p := newProtocols()
	tcp := p.GetAllTcp()
	assert.NotNil(t, tcp[1])
	assert.Nil(t, tcp[2])
	assert.NotNil(t, tcp[3])
}

func TestGetAllUdp(t *testing.T) {
	p := newProtocols()
	udp := p.GetAllUdp()
	assert.Nil(t, udp[1])
	assert.NotNil(t, udp[2])
	assert.NotNil(t, udp[3])
}

func TestGetTcp(t *testing.T) {
	p := newProtocols()
	tcp := p.GetTcp(1)
	assert.NotNil(t, tcp)
	assert.Contains(t, tcp.GetPorts(), 80)

	tcp = p.GetTcp(2)
	assert.Nil(t, tcp)

	tcp = p.GetTcp(3)
	assert.NotNil(t, tcp)
	assert.Contains(t, tcp.GetPorts(), 53)
}

func TestGetUdp(t *testing.T) {
	p := newProtocols()
	udp := p.GetUdp(1)
	assert.Nil(t, udp)

	udp = p.GetUdp(2)
	assert.NotNil(t, udp)
	assert.Contains(t, udp.GetPorts(), 5060)

	udp = p.GetUdp(3)
	assert.NotNil(t, udp)
	assert.Contains(t, udp.GetPorts(), 53)
}
