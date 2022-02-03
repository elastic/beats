// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build (linux && 386) || (linux && amd64)
// +build linux,386 linux,amd64

package socket

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"

	"github.com/elastic/beats/v7/x-pack/auditbeat/tracing"
)

const (
	// This compensates the size argument of udp_sendmsg which is only
	// UDP payload. 28 is the size of an IPv4 header (no options) + UDP header.
	minIPv4UdpPacketSize = 28

	// Same for udpv6_sendmsg.
	// 40 is the size of an IPv6 header (no extensions) + UDP header.
	minIPv6UdpPacketSize = 48
)

// event is the interface that all the deserialized events from the ring-buffer
// have to conform to in order to be processed by state.
type event interface {
	fmt.Stringer
	Update(*state) error
}

type tcpIPv4ConnectCall struct {
	Meta  tracing.Metadata `kprobe:"metadata"`
	Sock  uintptr          `kprobe:"sock"`
	LAddr uint32           `kprobe:"laddr"`
	RAddr uint32           `kprobe:"addr"`
	LPort uint16           `kprobe:"lport"`
	RPort uint16           `kprobe:"port"`
}

// String returns a representation of the event.
func (e *tcpIPv4ConnectCall) String() string {
	var buf [4]byte
	tracing.MachineEndian.PutUint32(buf[:], e.LAddr)
	laddr := net.IPv4(buf[0], buf[1], buf[2], buf[3])
	tracing.MachineEndian.PutUint16(buf[:], e.LPort)
	lport := binary.BigEndian.Uint16(buf[:])
	tracing.MachineEndian.PutUint32(buf[:], e.RAddr)
	raddr := net.IPv4(buf[0], buf[1], buf[2], buf[3])
	tracing.MachineEndian.PutUint16(buf[:], e.RPort)
	rport := binary.BigEndian.Uint16(buf[:])
	return fmt.Sprintf(
		"%s connect(sock=0x%x, %s:%d -> %s:%d)",
		header(e.Meta),
		e.Sock,
		laddr.String(),
		lport,
		raddr.String(),
		rport)
}

// Update the state with the contents of this event.
func (e *tcpIPv4ConnectCall) Update(s *state) error {
	return s.ThreadEnter(e.Meta.TID, e)
}

type tcpIPv6ConnectCall struct {
	Meta   tracing.Metadata `kprobe:"metadata"`
	Sock   uintptr          `kprobe:"sock"`
	LAddrA uint64           `kprobe:"laddra"`
	LAddrB uint64           `kprobe:"laddrb"`
	RAddrA uint64           `kprobe:"addra"`
	RAddrB uint64           `kprobe:"addrb"`
	LPort  uint16           `kprobe:"lport"`
	RPort  uint16           `kprobe:"port"`
}

// String returns a representation of the event.
func (e *tcpIPv6ConnectCall) String() string {
	var buf [16]byte
	tracing.MachineEndian.PutUint64(buf[:], e.LAddrA)
	tracing.MachineEndian.PutUint64(buf[8:], e.LAddrB)
	laddr := net.IP(buf[:]).String()
	tracing.MachineEndian.PutUint64(buf[:], e.RAddrA)
	tracing.MachineEndian.PutUint64(buf[8:], e.RAddrB)
	raddr := net.IP(buf[:]).String()
	tracing.MachineEndian.PutUint16(buf[:], e.LPort)
	lport := binary.BigEndian.Uint16(buf[:])
	tracing.MachineEndian.PutUint16(buf[:], e.RPort)
	rport := binary.BigEndian.Uint16(buf[:])
	return fmt.Sprintf(
		"%s connect6(sock=0x%x, %s:%d -> %s:%d)",
		header(e.Meta),
		e.Sock,
		laddr,
		lport,
		raddr,
		rport)
}

// Update the state with the contents of this event.
func (e *tcpIPv6ConnectCall) Update(s *state) error {
	return s.ThreadEnter(e.Meta.TID, e)
}

type tcpConnectResult struct {
	Meta   tracing.Metadata `kprobe:"metadata"`
	Retval int32            `kprobe:"retval"`
}

// String returns a representation of the event.
func (e *tcpConnectResult) String() string {
	return fmt.Sprintf("%s <- connect %s", header(e.Meta), kernErrorDesc(e.Retval))
}

// Update the state with the contents of this event.
func (e *tcpConnectResult) Update(s *state) error {
	ev, found := s.ThreadLeave(e.Meta.TID)
	if !found || e.Retval != 0 {
		return nil
	}
	switch call := ev.(type) {
	case *tcpIPv4ConnectCall:
		return s.UpdateFlow(flow{
			sock:     call.Sock,
			pid:      e.Meta.PID,
			inetType: inetTypeIPv4,
			proto:    protoTCP,
			dir:      directionEgress,
			complete: true,
			lastSeen: kernelTime(call.Meta.Timestamp),
			local:    newEndpointIPv4(call.LAddr, call.LPort, 0, 0),
			remote:   newEndpointIPv4(call.RAddr, call.RPort, 0, 0),
		})
	case *tcpIPv6ConnectCall:
		return s.UpdateFlow(flow{
			sock:     call.Sock,
			pid:      e.Meta.PID,
			inetType: inetTypeIPv6,
			proto:    protoTCP,
			dir:      directionEgress,
			complete: true,
			lastSeen: kernelTime(call.Meta.Timestamp),
			local:    newEndpointIPv6(call.LAddrA, call.LAddrB, call.LPort, 0, 0),
			remote:   newEndpointIPv6(call.RAddrA, call.RAddrB, call.RPort, 0, 0),
		})
	}
	return fmt.Errorf("stored thread event has unexpected type %T", ev)
}

var tcpStates = []string{
	"(zero)",
	"TCP_ESTABLISHED",
	"TCP_SYN_SENT",
	"TCP_SYN_RECV",
	"TCP_FIN_WAIT1",
	"TCP_FIN_WAIT2",
	"TCP_TIME_WAIT",
	"TCP_CLOSE",
	"TCP_CLOSE_WAIT",
	"TCP_LAST_ACK",
	"TCP_LISTEN",
	"TCP_CLOSING",
	"TCP_NEW_SYN_RECV",
}

type tcpAcceptResult struct {
	Meta    tracing.Metadata `kprobe:"metadata"`
	Sock    uintptr          `kprobe:"sock"`
	LAddr   uint32           `kprobe:"laddr"`
	RAddr   uint32           `kprobe:"raddr"`
	LPort   uint16           `kprobe:"lport"`
	RPort   uint16           `kprobe:"rport"`
	LAddr6a uint64           `kprobe:"laddr6a"`
	LAddr6b uint64           `kprobe:"laddr6b"`
	RAddr6a uint64           `kprobe:"raddr6a"`
	RAddr6b uint64           `kprobe:"raddr6b"`
	Af      uint16           `kprobe:"family"`
}

func (e *tcpAcceptResult) asFlow() flow {
	evTime := kernelTime(e.Meta.Timestamp)
	f := flow{
		sock:     e.Sock,
		pid:      e.Meta.PID,
		inetType: inetType(e.Af),
		proto:    protoTCP,
		dir:      directionIngress,
		complete: true,
		lastSeen: evTime,
		created:  evTime,
	}
	if e.Af == unix.AF_INET {
		f.local = newEndpointIPv4(e.LAddr, e.LPort, 0, 0)
		f.remote = newEndpointIPv4(e.RAddr, e.RPort, 0, 0)
	} else {
		f.local = newEndpointIPv6(e.LAddr6a, e.LAddr6b, e.LPort, 0, 0)
		f.remote = newEndpointIPv6(e.RAddr6a, e.RAddr6b, e.RPort, 0, 0)
	}
	return f
}

// String returns a representation of the event.
func (e *tcpAcceptResult) String() string {
	f := e.asFlow()
	return fmt.Sprintf("%s <- accept(sock=0x%x, af=%s, %s <- %s)", header(e.Meta), e.Sock, inetType(e.Af), f.local.String(), f.remote.String())
}

// Update the state with the contents of this event.
func (e *tcpAcceptResult) Update(s *state) error {
	if e.Sock != 0 {
		return s.CreateSocket(e.asFlow())
	}
	return nil
}

type tcpAcceptResult4 struct {
	Meta  tracing.Metadata `kprobe:"metadata"`
	Sock  uintptr          `kprobe:"sock"`
	LAddr uint32           `kprobe:"laddr"`
	RAddr uint32           `kprobe:"raddr"`
	LPort uint16           `kprobe:"lport"`
	RPort uint16           `kprobe:"rport"`
	Af    uint16           `kprobe:"family"`
}

func (e *tcpAcceptResult4) asFlow() flow {
	evTime := kernelTime(e.Meta.Timestamp)
	f := flow{
		sock:     e.Sock,
		pid:      e.Meta.PID,
		inetType: inetType(e.Af),
		proto:    protoTCP,
		dir:      directionIngress,
		complete: true,
		lastSeen: evTime,
		created:  evTime,
	}
	f.local = newEndpointIPv4(e.LAddr, e.LPort, 0, 0)
	f.remote = newEndpointIPv4(e.RAddr, e.RPort, 0, 0)
	return f
}

// String returns a representation of the event.
func (e *tcpAcceptResult4) String() string {
	f := e.asFlow()
	return fmt.Sprintf("%s <- accept(sock=0x%x, af=%s, %s <- %s)", header(e.Meta), e.Sock, inetType(e.Af), f.local.String(), f.remote.String())
}

// Update the state with the contents of this event.
func (e *tcpAcceptResult4) Update(s *state) error {
	if e.Sock != 0 {
		return s.CreateSocket(e.asFlow())
	}
	return nil
}

type tcpSendMsgCall struct {
	Meta    tracing.Metadata `kprobe:"metadata"`
	Sock    uintptr          `kprobe:"sock"`
	Size    uintptr          `kprobe:"size"`
	LAddr   uint32           `kprobe:"laddr"`
	RAddr   uint32           `kprobe:"raddr"`
	LPort   uint16           `kprobe:"lport"`
	RPort   uint16           `kprobe:"rport"`
	LAddr6a uint64           `kprobe:"laddr6a"`
	LAddr6b uint64           `kprobe:"laddr6b"`
	RAddr6a uint64           `kprobe:"raddr6a"`
	RAddr6b uint64           `kprobe:"raddr6b"`
	Af      uint16           `kprobe:"family"`
}

func (e *tcpSendMsgCall) asFlow() flow {
	f := flow{
		sock:     e.Sock,
		pid:      e.Meta.PID,
		inetType: inetType(e.Af),
		proto:    protoTCP,
		lastSeen: kernelTime(e.Meta.Timestamp),
	}
	if e.Af == unix.AF_INET {
		f.local = newEndpointIPv4(e.LAddr, e.LPort, 0, 0)
		f.remote = newEndpointIPv4(e.RAddr, e.RPort, 0, 0)
	} else {
		f.local = newEndpointIPv6(e.LAddr6a, e.LAddr6b, e.LPort, 0, 0)
		f.remote = newEndpointIPv6(e.RAddr6a, e.RAddr6b, e.RPort, 0, 0)
	}
	return f
}

// String returns a representation of the event.
func (e *tcpSendMsgCall) String() string {
	flow := e.asFlow()
	return fmt.Sprintf(
		"%s tcp_sendmsg(sock=0x%x, size=%d, af=%s, %s -> %s)",
		header(e.Meta),
		flow.sock,
		e.Size,
		inetType(e.Af),
		flow.local.String(),
		flow.remote.String())
}

// Update the state with the contents of this event.
func (e *tcpSendMsgCall) Update(s *state) error {
	return s.UpdateFlow(e.asFlow())
}

type tcpSendMsgCall4 struct {
	Meta  tracing.Metadata `kprobe:"metadata"`
	Sock  uintptr          `kprobe:"sock"`
	Size  uintptr          `kprobe:"size"`
	LAddr uint32           `kprobe:"laddr"`
	RAddr uint32           `kprobe:"raddr"`
	LPort uint16           `kprobe:"lport"`
	RPort uint16           `kprobe:"rport"`
	Af    uint16           `kprobe:"family"`
}

func (e *tcpSendMsgCall4) asFlow() flow {
	f := flow{
		sock:     e.Sock,
		pid:      e.Meta.PID,
		inetType: inetType(e.Af),
		proto:    protoTCP,
		lastSeen: kernelTime(e.Meta.Timestamp),
	}
	f.local = newEndpointIPv4(e.LAddr, e.LPort, 0, 0)
	f.remote = newEndpointIPv4(e.RAddr, e.RPort, 0, 0)
	return f
}

// String returns a representation of the event.
func (e *tcpSendMsgCall4) String() string {
	flow := e.asFlow()
	return fmt.Sprintf(
		"%s tcp_sendmsg(sock=0x%x, size=%d, af=%s, %s -> %s)",
		header(e.Meta),
		flow.sock,
		e.Size,
		inetType(e.Af),
		flow.local.String(),
		flow.remote.String())
}

// Update the state with the contents of this event.
func (e *tcpSendMsgCall4) Update(s *state) error {
	return s.UpdateFlow(e.asFlow())
}

type ipLocalOutCall struct {
	Meta  tracing.Metadata `kprobe:"metadata"`
	Sock  uintptr          `kprobe:"sock"`
	Size  uint32           `kprobe:"size"`
	LAddr uint32           `kprobe:"laddr"`
	RAddr uint32           `kprobe:"raddr"`
	LPort uint16           `kprobe:"lport"`
	RPort uint16           `kprobe:"rport"`
}

func (e *ipLocalOutCall) asFlow() flow {
	return flow{
		sock:     e.Sock,
		pid:      e.Meta.PID,
		inetType: inetTypeIPv4,
		lastSeen: kernelTime(e.Meta.Timestamp),
		local:    newEndpointIPv4(e.LAddr, e.LPort, 1, uint64(e.Size)),
		remote:   newEndpointIPv4(e.RAddr, e.RPort, 0, 0),
	}
}

// String returns a representation of the event.
func (e *ipLocalOutCall) String() string {
	f := e.asFlow()
	return fmt.Sprintf(
		"%s ip_local_out(sock=0x%x, size=%d, %s -> %s)",
		header(e.Meta),
		e.Sock,
		e.Size,
		f.local.String(),
		f.remote.String())
}

func isNotUDP(f *flow) bool {
	return f.proto != protoUDP
}

// Update the state with the contents of this event.
func (e *ipLocalOutCall) Update(s *state) error {
	flow := e.asFlow()
	if flow.remote.addr.IP == nil {
		// Unconnected-UDP flows have nil destination in here.
		return nil
	}
	// Only count non-UDP packets.
	// Those are already counted by udp_sendmsg, but there is no way
	// to discriminate UDP in ip_local_out at kprobe level.
	return s.UpdateFlowWithCondition(flow, isNotUDP)
}

type inet6CskXmitCall struct {
	Meta    tracing.Metadata `kprobe:"metadata"`
	Sock    uintptr          `kprobe:"sock"`
	LAddr6a uint64           `kprobe:"laddr6a"`
	LAddr6b uint64           `kprobe:"laddr6b"`
	RAddr6a uint64           `kprobe:"raddr6a"`
	RAddr6b uint64           `kprobe:"raddr6b"`
	LPort   uint16           `kprobe:"lport"`
	RPort   uint16           `kprobe:"rport"`
	Size    uint32           `kprobe:"size"`
}

func (e *inet6CskXmitCall) asFlow() flow {
	return flow{
		sock:     e.Sock,
		pid:      e.Meta.PID,
		inetType: inetTypeIPv6,
		proto:    protoTCP,
		lastSeen: kernelTime(e.Meta.Timestamp),
		local:    newEndpointIPv6(e.LAddr6a, e.LAddr6b, e.LPort, 1, uint64(e.Size)),
		remote:   newEndpointIPv6(e.RAddr6a, e.RAddr6b, e.RPort, 0, 0),
	}
}

// String returns a representation of the event.
func (e *inet6CskXmitCall) String() string {
	f := e.asFlow()
	return fmt.Sprintf(
		"%s inet6_csk_xmit(sock=0x%x, size=%d, %s -> %s)",
		header(e.Meta),
		e.Sock,
		e.Size,
		f.local.String(),
		f.remote.String())
}

// Update the state with the contents of this event.
func (e *inet6CskXmitCall) Update(s *state) error {
	return s.UpdateFlow(e.asFlow())
}

type tcpV4DoRcv struct {
	Meta  tracing.Metadata `kprobe:"metadata"`
	Sock  uintptr          `kprobe:"sock"`
	Size  uint32           `kprobe:"size"`
	LAddr uint32           `kprobe:"laddr"`
	RAddr uint32           `kprobe:"raddr"`
	LPort uint16           `kprobe:"lport"`
	RPort uint16           `kprobe:"rport"`
}

func (e *tcpV4DoRcv) asFlow() flow {
	return flow{
		sock:     e.Sock,
		pid:      e.Meta.PID,
		inetType: inetTypeIPv4,
		proto:    protoTCP,
		lastSeen: kernelTime(e.Meta.Timestamp),
		local:    newEndpointIPv4(e.LAddr, e.LPort, 0, 0),
		remote:   newEndpointIPv4(e.RAddr, e.RPort, 1, uint64(e.Size)),
	}
}

// String returns a representation of the event.
func (e *tcpV4DoRcv) String() string {
	f := e.asFlow()
	return fmt.Sprintf(
		"%s tcp_v4_do_rcv(sock=0x%x, size=%d, %s <- %s)",
		header(e.Meta),
		e.Sock,
		e.Size,
		f.local.String(),
		f.remote.String())
}

// Update the state with the contents of this event.
func (e *tcpV4DoRcv) Update(s *state) error {
	return s.UpdateFlow(e.asFlow())
}

type tcpV6DoRcv struct {
	Meta    tracing.Metadata `kprobe:"metadata"`
	Sock    uintptr          `kprobe:"sock"`
	LAddr6a uint64           `kprobe:"laddr6a"`
	LAddr6b uint64           `kprobe:"laddr6b"`
	RAddr6a uint64           `kprobe:"raddr6a"`
	RAddr6b uint64           `kprobe:"raddr6b"`
	LPort   uint16           `kprobe:"lport"`
	RPort   uint16           `kprobe:"rport"`
	Size    uint32           `kprobe:"size"`
}

func (e *tcpV6DoRcv) asFlow() flow {
	return flow{
		sock:     e.Sock,
		pid:      e.Meta.PID,
		inetType: inetTypeIPv6,
		proto:    protoTCP,
		lastSeen: kernelTime(e.Meta.Timestamp),
		local:    newEndpointIPv6(e.LAddr6a, e.LAddr6b, e.LPort, 0, 0),
		remote:   newEndpointIPv6(e.RAddr6a, e.RAddr6b, e.RPort, 1, uint64(e.Size)),
	}
}

// String returns a representation of the event.
func (e *tcpV6DoRcv) String() string {
	f := e.asFlow()
	return fmt.Sprintf(
		"%s tcp_v6_do_rcv(sock=0x%x, size=%d, %s <- %s)",
		header(e.Meta),
		e.Sock,
		e.Size,
		f.local.String(),
		f.remote.String())
}

// Update the state with the contents of this event.
func (e *tcpV6DoRcv) Update(s *state) error {
	return s.UpdateFlow(e.asFlow())
}

type udpSendMsgCall struct {
	Meta     tracing.Metadata `kprobe:"metadata"`
	Sock     uintptr          `kprobe:"sock"`
	Size     uintptr          `kprobe:"size"`
	LAddr    uint32           `kprobe:"laddr"`
	RAddr    uint32           `kprobe:"raddr"`
	AltRAddr uint32           `kprobe:"altraddr"`
	LPort    uint16           `kprobe:"lport"`
	RPort    uint16           `kprobe:"rport"`
	AltRPort uint16           `kprobe:"altrport"`
	// SIPtr is the struct sockaddr_in pointer.
	SIPtr uintptr `kprobe:"siptr"`
	// SIAF is the address family in (struct sockaddr_in*)->sin_family.
	SIAF uint16 `kprobe:"siaf"`
}

func (e *udpSendMsgCall) asFlow() flow {
	raddr, rport := e.RAddr, e.RPort
	if e.SIPtr == 0 || e.SIAF != unix.AF_INET {
		raddr = e.AltRAddr
		rport = e.AltRPort
	}
	return flow{
		sock:     e.Sock,
		pid:      e.Meta.PID,
		inetType: inetTypeIPv4,
		proto:    protoUDP,
		dir:      directionEgress,
		lastSeen: kernelTime(e.Meta.Timestamp),
		local:    newEndpointIPv4(e.LAddr, e.LPort, 1, uint64(e.Size)+minIPv4UdpPacketSize),
		remote:   newEndpointIPv4(raddr, rport, 0, 0),
	}
}

// String returns a representation of the event.
func (e *udpSendMsgCall) String() string {
	flow := e.asFlow()
	return fmt.Sprintf(
		"%s udp_sendmsg(sock=0x%x, size=%d, %s -> %s)",
		header(e.Meta),
		flow.sock,
		e.Size,
		flow.local.String(),
		flow.remote.String())
}

// Update the state with the contents of this event.
func (e *udpSendMsgCall) Update(s *state) error {
	return s.UpdateFlow(e.asFlow())
}

type udpv6SendMsgCall struct {
	Meta      tracing.Metadata `kprobe:"metadata"`
	Sock      uintptr          `kprobe:"sock"`
	Size      uintptr          `kprobe:"size"`
	LAddrA    uint64           `kprobe:"laddra"`
	LAddrB    uint64           `kprobe:"laddrb"`
	RAddrA    uint64           `kprobe:"raddra"`
	RAddrB    uint64           `kprobe:"raddrb"`
	AltRAddrA uint64           `kprobe:"altraddra"`
	AltRAddrB uint64           `kprobe:"altraddrb"`
	LPort     uint16           `kprobe:"lport"`
	RPort     uint16           `kprobe:"rport"`
	AltRPort  uint16           `kprobe:"altrport"`
	// SI6Ptr is the struct sockaddr_in6 pointer.
	SI6Ptr uintptr `kprobe:"si6ptr"`
	// Si6AF is the address family field ((struct sockaddr_in6*)->sin6_family)
	SI6AF uint16 `kprobe:"si6af"`
}

func (e *udpv6SendMsgCall) asFlow() flow {
	raddra, raddrb, rport := e.RAddrA, e.RAddrB, e.RPort
	if e.SI6Ptr == 0 || e.SI6AF != unix.AF_INET6 {
		raddra, raddrb = e.AltRAddrA, e.AltRAddrB
		rport = e.AltRPort
	}
	return flow{
		sock:     e.Sock,
		pid:      e.Meta.PID,
		inetType: inetTypeIPv6,
		proto:    protoUDP,
		dir:      directionEgress,
		lastSeen: kernelTime(e.Meta.Timestamp),
		// In IPv6, udpv6_sendmsg increments local counters as there is no
		// corresponding ip6_local_out call.
		local:  newEndpointIPv6(e.LAddrA, e.LAddrB, e.LPort, 1, uint64(e.Size)+minIPv6UdpPacketSize),
		remote: newEndpointIPv6(raddra, raddrb, rport, 0, 0),
	}
}

// String returns a representation of the event.
func (e *udpv6SendMsgCall) String() string {
	flow := e.asFlow()
	return fmt.Sprintf(
		"%s udpv6_sendmsg(sock=0x%x, size=%d, %s -> %s)",
		header(e.Meta),
		flow.sock,
		e.Size,
		flow.local.String(),
		flow.remote.String())
}

// Update the state with the contents of this event.
func (e *udpv6SendMsgCall) Update(s *state) error {
	return s.UpdateFlow(e.asFlow())
}

type udpQueueRcvSkb struct {
	Meta   tracing.Metadata          `kprobe:"metadata"`
	Sock   uintptr                   `kprobe:"sock"`
	Size   uint32                    `kprobe:"size"`
	LAddr  uint32                    `kprobe:"laddr"`
	LPort  uint16                    `kprobe:"lport"`
	IPHdr  uint16                    `kprobe:"iphdr"`
	UDPHdr uint16                    `kprobe:"udphdr"`
	Base   uintptr                   `kprobe:"base"`
	Packet [skBuffDataDumpBytes]byte `kprobe:"packet,greedy"`
}

func validIPv4Headers(ipHdr uint16, udpHdr uint16, data []byte) bool {
	return ipHdr != 0 &&
		int(ipHdr)+20 < len(data) &&
		data[ipHdr]&0xF0 == 0x40 &&
		udpHdr != 0 &&
		int(udpHdr)+12 < len(data)
}

func validIPv6Headers(ipHdr uint16, udpHdr uint16, data []byte) bool {
	return ipHdr != 0 &&
		int(ipHdr)+40 < len(data) &&
		data[ipHdr]&0xF0 == 0x60 &&
		udpHdr != 0 &&
		int(udpHdr)+12 < len(data)
}

func (e *udpQueueRcvSkb) asFlow() flow {
	f := flow{
		sock:     e.Sock,
		pid:      e.Meta.PID,
		inetType: inetTypeIPv4,
		proto:    protoUDP,
		dir:      directionIngress,
		lastSeen: kernelTime(e.Meta.Timestamp),
		local:    newEndpointIPv4(e.LAddr, e.LPort, 0, 0),
	}
	if valid := validIPv4Headers(e.IPHdr, e.UDPHdr, e.Packet[:]); !valid {
		// Check if we're dealing with pointers
		// TODO: This should check for SK_BUFF_HAS_POINTERS. Instead is just
		//		 treating IPHdr/UDPHdr as the lower 16bits of a pointer which
		//       is enough as the headers are never more than 64k bytes into the
		//		 packet.
		//       This hacky solution will only work on little-endian archs
		//		 which is fine for now as only 386/amd64 is supported.
		//		 In the future a different set of kprobes must be used
		//		 when SK_BUFF_HAS_POINTERS so that IPHdr and UDPHdr are
		//		 the size of a pointer, not uint16.
		base := uint16(e.Base)
		if e.IPHdr > base &&
			e.UDPHdr > base {
			ipOff := e.IPHdr - base
			udpOff := e.UDPHdr - base
			if valid = validIPv4Headers(ipOff, udpOff, e.Packet[:]); valid {
				e.IPHdr = ipOff
				e.UDPHdr = udpOff
			}
		}
		if !valid {
			return f
		}
	}
	var raddr uint32
	var rport uint16
	// the remote is this packet's source
	raddr = tracing.MachineEndian.Uint32(e.Packet[e.IPHdr+12:])
	rport = tracing.MachineEndian.Uint16(e.Packet[e.UDPHdr:])
	f.remote = newEndpointIPv4(raddr, rport, 1, uint64(e.Size)+minIPv4UdpPacketSize)
	return f
}

// String returns a representation of the event.
func (e *udpQueueRcvSkb) String() string {
	flow := e.asFlow()
	return fmt.Sprintf(
		"%s udp_queue_rcv_skb(sock=0x%x, size=%d, %s <- %s)",
		header(e.Meta),
		flow.sock,
		e.Size,
		flow.local.String(),
		flow.remote.String())
}

// Update the state with the contents of this event.
func (e *udpQueueRcvSkb) Update(s *state) error {
	return s.UpdateFlow(e.asFlow())
}

type udpv6QueueRcvSkb struct {
	Meta   tracing.Metadata          `kprobe:"metadata"`
	Sock   uintptr                   `kprobe:"sock"`
	Size   uint32                    `kprobe:"size"`
	LAddrA uint64                    `kprobe:"laddra"`
	LAddrB uint64                    `kprobe:"laddrb"`
	LPort  uint16                    `kprobe:"lport"`
	IPHdr  uint16                    `kprobe:"iphdr"`
	UDPHdr uint16                    `kprobe:"udphdr"`
	Base   uintptr                   `kprobe:"base"`
	Packet [skBuffDataDumpBytes]byte `kprobe:"packet,greedy"`
}

func (e *udpv6QueueRcvSkb) asFlow() flow {
	f := flow{
		sock:     e.Sock,
		pid:      e.Meta.PID,
		inetType: inetTypeIPv6,
		proto:    protoUDP,
		dir:      directionIngress,
		lastSeen: kernelTime(e.Meta.Timestamp),
		local:    newEndpointIPv6(e.LAddrA, e.LAddrB, e.LPort, 0, 0),
	}
	if valid := validIPv6Headers(e.IPHdr, e.UDPHdr, e.Packet[:]); !valid {
		// Check if we're dealing with pointers
		// TODO: This only works in little-endian, same as in udpQueueRcvSkb
		base := uint16(e.Base)
		if e.IPHdr > base &&
			e.UDPHdr > base {
			ipOff := e.IPHdr - base
			udpOff := e.UDPHdr - base
			if valid = validIPv6Headers(ipOff, udpOff, e.Packet[:]); valid {
				e.IPHdr = ipOff
				e.UDPHdr = udpOff
			}
		}
		if !valid {
			return f
		}
	}
	var raddrA, raddrB uint64
	var rport uint16
	// the remote is this packet's source
	raddrA = tracing.MachineEndian.Uint64(e.Packet[e.IPHdr+8:])
	raddrB = tracing.MachineEndian.Uint64(e.Packet[e.IPHdr+16:])
	rport = tracing.MachineEndian.Uint16(e.Packet[e.UDPHdr:])
	f.remote = newEndpointIPv6(raddrA, raddrB, rport, 1, uint64(e.Size)+minIPv6UdpPacketSize)
	return f
}

// String returns a representation of the event.
func (e *udpv6QueueRcvSkb) String() string {
	flow := e.asFlow()
	return fmt.Sprintf(
		"%s udpv6_queue_rcv_skb(sock=0x%x, size=%d, %s <- %s)",
		header(e.Meta),
		flow.sock,
		e.Size,
		flow.local.String(),
		flow.remote.String())
}

// Update the state with the contents of this event.
func (e *udpv6QueueRcvSkb) Update(s *state) error {
	return s.UpdateFlow(e.asFlow())
}

type sockInitData struct {
	Meta   tracing.Metadata `kprobe:"metadata"`
	Socket uintptr          `kprobe:"socket"`
	Sock   uintptr          `kprobe:"sock"`
}

// String returns a representation of the event.
func (e *sockInitData) String() string {
	return fmt.Sprintf("%s sock_init_data(sock=0x%x)", header(e.Meta), e.Sock)
}

// Update the state with the contents of this event.
func (e *sockInitData) Update(s *state) error {
	if ev, found := s.ThreadLeave(e.Meta.TID); found {
		// Only track socks created by inet_create / inet6_create
		if iCreate, ok := ev.(*inetCreate); ok {
			return s.CreateSocket(flow{
				sock:     e.Sock,
				pid:      e.Meta.PID,
				proto:    flowProto(iCreate.Proto),
				created:  kernelTime(e.Meta.Timestamp),
				lastSeen: kernelTime(e.Meta.Timestamp),
				complete: true,
			})
		}
	}
	return nil
}

type inetCreate struct {
	Meta  tracing.Metadata `kprobe:"metadata"`
	Proto int32            `kprobe:"proto"`
}

// String returns a representation of the event.
func (e *inetCreate) String() string {
	return fmt.Sprintf("%s inet_create(proto=%d)", header(e.Meta), e.Proto)
}

// Update the state with the contents of this event.
func (e *inetCreate) Update(s *state) error {
	if proto := flowProto(e.Proto); proto == protoUnknown || proto == protoTCP || proto == protoUDP {
		return s.ThreadEnter(e.Meta.TID, e)
	}
	return nil
}

type inetReleaseCall struct {
	Meta tracing.Metadata `kprobe:"metadata"`
	Sock uintptr          `kprobe:"sock"`
}

// String returns a representation of the event.
func (e *inetReleaseCall) String() string {
	return fmt.Sprintf("%s inet_release(sock=0x%x)", header(e.Meta), e.Sock)
}

// Update the state with the contents of this event.
func (e *inetReleaseCall) Update(s *state) error {
	return s.OnSockDestroyed(e.Sock, e.Meta.PID)
}

// Fetching data from execve is complicated as support for strings or arrays
// in Kprobes appeared in recent kernels (~2018). To be compatible with older
// kernels it needs to dump fixed-size arrays in 8-byte chunks. As the total
// number of fetchargs available is limited, we have to dump only the first
// 128 bytes of every argument.
const (
	maxProgArgLen = 128
	maxProgArgs   = 5
)

type execveCall struct {
	Meta tracing.Metadata    `kprobe:"metadata"`
	Path [maxProgArgLen]byte `kprobe:"path,greedy"`
	// extra ptr is to detect if there are more than maxProcArgs arguments
	Ptrs   [maxProgArgs + 1]uintptr `kprobe:"argptrs,greedy"`
	Param0 [maxProgArgLen]byte      `kprobe:"param0,greedy"`
	Param1 [maxProgArgLen]byte      `kprobe:"param1,greedy"`
	Param2 [maxProgArgLen]byte      `kprobe:"param2,greedy"`
	Param3 [maxProgArgLen]byte      `kprobe:"param3,greedy"`
	Param4 [maxProgArgLen]byte      `kprobe:"param4,greedy"`

	// Extra user information for enrichment.
	creds *commitCreds
}

func (e *execveCall) getProcess() *process {
	p := &process{
		pid:     e.Meta.PID,
		created: kernelTime(e.Meta.Timestamp),
	}

	if idx := bytes.IndexByte(e.Path[:], 0); idx >= 0 {
		// Fast path if we already have the path.
		p.path = string(e.Path[:idx])
	} else {
		// Attempt to get the path from the /prox/<pid>/exe symlink.
		var err error
		p.path, err = filepath.EvalSymlinks(fmt.Sprintf("/proc/%d/exe", e.Meta.PID))
		if err != nil {
			if pe, ok := err.(*os.PathError); ok && strings.Contains(pe.Path, "(deleted)") {
				// Keep the deleted path from the PathError.
				p.path = pe.Path
			} else {
				// Fallback to the truncated path.
				p.path = string(e.Path[:]) + " ..."
			}
		}
	}

	// Check for truncation of arg list or arguments.
	params := [...][]byte{
		e.Param0[:],
		e.Param1[:],
		e.Param2[:],
		e.Param3[:],
		e.Param4[:],
	}
	var (
		argc         int
		truncatedArg bool
	)
	for argc = 0; argc < len(e.Ptrs); argc++ {
		if e.Ptrs[argc] == 0 {
			break
		}
		if argc < len(params) && bytes.IndexByte(params[argc], 0) < 0 {
			truncatedArg = true
		}
	}
	if argc > maxProgArgs || truncatedArg {
		// Attempt to get complete args list from /proc/<pid>/cmdline.
		cmdline, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", e.Meta.PID))
		if err == nil {
			p.args = strings.Split(strings.TrimRight(string(cmdline), "\x00"), "\x00")
		}
	}

	if p.args == nil {
		// Fallback to arg list if unsuccessful or no truncation.
		p.args = make([]string, argc)
		if argc > maxProgArgs {
			argc = maxProgArgs
			p.args[argc] = "..."
		}
		for i, par := range params[:argc] {
			p.args[i] = readCString(par)
		}
	}

	// Get name from first argument.
	p.name = filepath.Base(p.args[0])

	if e.creds != nil {
		p.hasCreds = true
		p.uid = e.creds.UID
		p.gid = e.creds.GID
		p.euid = e.creds.EUID
		p.egid = e.creds.EGID
	}

	return p
}

// String returns a representation of the event.
func (e *execveCall) String() string {
	p := e.getProcess()
	list := make([]string, len(p.args))
	for idx, val := range p.args {
		list[idx] = fmt.Sprintf("arg%d='%s'", idx, val)
	}
	return fmt.Sprintf("%s execve(name='%s', path='%s', %s)", header(e.Meta), p.name, p.path, strings.Join(list, " "))
}

// Update the state with the contents of this event.
func (e *execveCall) Update(s *state) error {
	return s.ThreadEnter(e.Meta.TID, e)
}

type execveRet struct {
	Meta   tracing.Metadata `kprobe:"metadata"`
	Retval int32            `kprobe:"retval"`
}

// String returns a representation of the event.
func (e *execveRet) String() string {
	return fmt.Sprintf("%s <- execve %s", header(e.Meta), kernErrorDesc(e.Retval))
}

// Update the state with the contents of this event.
func (e *execveRet) Update(s *state) error {
	if prev, found := s.ThreadLeave(e.Meta.TID); found {
		if call, ok := prev.(*execveCall); ok {
			if e.Retval >= 0 {
				return s.CreateProcess(call.getProcess())
			}
		}
	}
	return nil
}

type forkRet struct {
	Meta   tracing.Metadata `kprobe:"metadata"`
	Retval int              `kprobe:"retval"`
}

// String returns a representation of the event.
func (e *forkRet) String() string {
	return fmt.Sprintf("%s <- fork %d", header(e.Meta), e.Retval)
}

// Update the state with the contents of this event.
func (e *forkRet) Update(s *state) error {
	if e.Retval <= 0 {
		return nil
	}
	return s.ForkProcess(e.Meta.PID, uint32(e.Retval), kernelTime(e.Meta.Timestamp))
}

type doExit struct {
	Meta tracing.Metadata `kprobe:"metadata"`
}

// String returns a representation of the event.
func (e *doExit) String() string {
	whatExited := "process"
	if e.Meta.PID != e.Meta.TID {
		whatExited = "thread"
	}
	return fmt.Sprintf("%s do_exit(%s)", header(e.Meta), whatExited)
}

// Update the state with the contents of this event.
func (e *doExit) Update(s *state) (err error) {
	// Only report exits of the main thread, a.k.a process exit
	if e.Meta.PID == e.Meta.TID {
		err = s.TerminateProcess(e.Meta.PID)
	}
	// Cleanup any saved thread state
	s.ThreadLeave(e.Meta.TID)
	return err
}

type commitCreds struct {
	Meta tracing.Metadata `kprobe:"metadata"`
	UID  uint32           `kprobe:"uid"`
	GID  uint32           `kprobe:"gid"`
	EUID uint32           `kprobe:"euid"`
	EGID uint32           `kprobe:"egid"`
}

// String returns a representation of the event.
func (e *commitCreds) String() string {
	return fmt.Sprintf("%s commit_creds(uid=%d, gid=%d, euid=%d, egid=%d)",
		header(e.Meta),
		e.UID, e.GID, e.EUID, e.EGID)
}

// Update the state with the contents of this event.
func (e *commitCreds) Update(s *state) error {
	if prev, found := s.ThreadLeave(e.Meta.TID); found {
		if call, ok := prev.(*execveCall); ok {
			// Only inspect commit_creds() calls that happen in the context
			// of an execve call. Enrich the process with user information.
			call.creds = e
			// Re-install the information after enrichment so that execveRet
			// can access it.
			return s.ThreadEnter(e.Meta.TID, call)
		}
	}
	return nil
}

type clockSyncCall struct {
	Meta tracing.Metadata `kprobe:"metadata"`
	Ts   uint64           `kprobe:"timestamp"`
}

// String returns a representation of the event.
func (e *clockSyncCall) String() string {
	return fmt.Sprintf("%s sys_uname[clock-sync](ts=0x%x)", header(e.Meta), e.Ts)
}

// Update the state with the contents of this event.
func (e *clockSyncCall) Update(s *state) error {
	if int(e.Meta.PID) == os.Getpid() {
		return s.SyncClocks(e.Meta.Timestamp, e.Ts)
	}
	return nil
}

func header(meta tracing.Metadata) string {
	return fmt.Sprintf("%d probe=%d pid=%d tid=%d",
		meta.Timestamp,
		meta.EventID,
		meta.PID,
		meta.TID)
}

func kernErrorDesc(retval int32) string {
	switch {
	case retval < 0:
		errno := syscall.Errno(uintptr(0 - retval))
		return fmt.Sprintf("failed errno=%d (%s)", errno, errno.Error())
	case retval == 0:
		return "ok"
	default:
		return fmt.Sprintf("ok (value=%d)", retval)
	}
}

func readCString(buf []byte) string {
	if pos := bytes.IndexByte(buf, 0); pos != -1 {
		return string(buf[:pos])
	}
	return string(buf) + " ..."
}
