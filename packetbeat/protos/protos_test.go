// +build !integration

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

func (proto *TcpProtocol) GetPorts() []int {
	return proto.Ports
}

func (proto *TcpProtocol) Parse(pkt *Packet, tcptuple *common.TCPTuple,
	dir uint8, private ProtocolData) ProtocolData {
	return private
}

func (proto *TcpProtocol) ReceivedFin(tcptuple *common.TCPTuple, dir uint8,
	private ProtocolData) ProtocolData {
	return private
}

func (proto *TcpProtocol) GapInStream(tcptuple *common.TCPTuple, dir uint8,
	nbytes int, private ProtocolData) (priv ProtocolData, drop bool) {
	return private, true
}

func (proto *TcpProtocol) ConnectionTimeout() time.Duration { return 0 }

type UdpProtocol TestProtocol

func (proto *UdpProtocol) Init(test_mode bool, results publish.Transactions) error {
	return nil
}

func (proto *UdpProtocol) GetPorts() []int {
	return proto.Ports
}

func (proto *UdpProtocol) ParseUDP(pkt *Packet) {
	return
}

type TcpUdpProtocol TestProtocol

func (proto *TcpUdpProtocol) Init(test_mode bool, results publish.Transactions) error {
	return nil
}

func (proto *TcpUdpProtocol) GetPorts() []int {
	return proto.Ports
}

func (proto *TcpUdpProtocol) Parse(pkt *Packet, tcptuple *common.TCPTuple,
	dir uint8, private ProtocolData) ProtocolData {
	return private
}

func (proto *TcpUdpProtocol) ReceivedFin(tcptuple *common.TCPTuple, dir uint8,
	private ProtocolData) ProtocolData {
	return private
}

func (proto *TcpUdpProtocol) GapInStream(tcptuple *common.TCPTuple, dir uint8,
	nbytes int, private ProtocolData) (priv ProtocolData, drop bool) {
	return private, true
}

func (proto *TcpUdpProtocol) ParseUDP(pkt *Packet) {
	return
}

func (proto *TcpUdpProtocol) ConnectionTimeout() time.Duration { return 0 }

func TestProtocolNames(t *testing.T) {
	assert.Equal(t, "unknown", UnknownProtocol.String())
	assert.Equal(t, "impossible", Protocol(100).String())
}

func newProtocols() Protocols {
	p := ProtocolsStruct{}
	p.all = make(map[Protocol]Plugin)
	p.tcp = make(map[Protocol]TCPPlugin)
	p.udp = make(map[Protocol]UDPPlugin)

	tcp := &TcpProtocol{Ports: []int{80}}
	udp := &UdpProtocol{Ports: []int{5060}}
	tcpUdp := &TcpUdpProtocol{Ports: []int{53}}

	p.register(1, tcp)
	p.register(2, udp)
	p.register(3, tcpUdp)
	return p
}

func TestBpfFilterWithoutVlanOnlyIcmp(t *testing.T) {
	p := ProtocolsStruct{}
	p.all = make(map[Protocol]Plugin)
	p.tcp = make(map[Protocol]TCPPlugin)
	p.udp = make(map[Protocol]UDPPlugin)

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

func TestGetAllTCP(t *testing.T) {
	p := newProtocols()
	tcp := p.GetAllTCP()
	assert.NotNil(t, tcp[1])
	assert.Nil(t, tcp[2])
	assert.NotNil(t, tcp[3])
}

func TestGetAllUDP(t *testing.T) {
	p := newProtocols()
	udp := p.GetAllUDP()
	assert.Nil(t, udp[1])
	assert.NotNil(t, udp[2])
	assert.NotNil(t, udp[3])
}

func TestGetTCP(t *testing.T) {
	p := newProtocols()
	tcp := p.GetTCP(1)
	assert.NotNil(t, tcp)
	assert.Contains(t, tcp.GetPorts(), 80)

	tcp = p.GetTCP(2)
	assert.Nil(t, tcp)

	tcp = p.GetTCP(3)
	assert.NotNil(t, tcp)
	assert.Contains(t, tcp.GetPorts(), 53)
}

func TestGetUDP(t *testing.T) {
	p := newProtocols()
	udp := p.GetUDP(1)
	assert.Nil(t, udp)

	udp = p.GetUDP(2)
	assert.NotNil(t, udp)
	assert.Contains(t, udp.GetPorts(), 5060)

	udp = p.GetUDP(3)
	assert.NotNil(t, udp)
	assert.Contains(t, udp.GetPorts(), 53)
}
