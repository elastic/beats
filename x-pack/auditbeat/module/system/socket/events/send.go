// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package events

import (
	"fmt"

	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/state"
	"github.com/elastic/beats/v7/x-pack/auditbeat/tracing"
	"golang.org/x/sys/unix"
)

type TCPSendmsgCall struct {
	Meta    tracing.Metadata `kprobe:"metadata"`
	Socket  uintptr          `kprobe:"sock"`
	Size    uintptr          `kprobe:"size"`
	LAddr   uint32           `kprobe:"laddr"`
	RAddr   uint32           `kprobe:"raddr"`
	LPort   uint16           `kprobe:"lport"`
	RPort   uint16           `kprobe:"rport"`
	LAddr6a uint64           `kprobe:"laddr6a"`
	LAddr6b uint64           `kprobe:"laddr6b"`
	RAddr6a uint64           `kprobe:"raddr6a"`
	RAddr6b uint64           `kprobe:"raddr6b"`
	AF      uint16           `kprobe:"family"`

	flow *state.Flow // for caching
}

func (e *TCPSendmsgCall) Flow() *state.Flow {
	if e.flow != nil {
		return e.flow
	}

	var local, remote *state.Endpoint
	if e.AF == unix.AF_INET {
		local = state.NewEndpointIPv4(e.LAddr, e.LPort, 0, 0)
		remote = state.NewEndpointIPv4(e.RAddr, e.RPort, 0, 0)
	} else {
		local = state.NewEndpointIPv6(e.LAddr6a, e.LAddr6b, e.LPort, 0, 0)
		remote = state.NewEndpointIPv6(e.RAddr6a, e.RAddr6b, e.RPort, 0, 0)
	}

	e.flow = state.NewFlow(
		e.Socket,
		e.Meta.PID,
		e.AF,
		unix.IPPROTO_TCP,
		e.Meta.Timestamp,
		local,
		remote,
	).MarkOutbound()

	return e.flow
}

// String returns a representation of the event.
func (e *TCPSendmsgCall) String() string {
	flow := e.Flow()
	return fmt.Sprintf(
		"%s tcp_sendmsg(sock=0x%x, size=%d, af=%s, %s -> %s)",
		header(e.Meta),
		e.Socket,
		e.Size,
		flow.Type(),
		flow.Local().String(),
		flow.Remote().String())
}

// Update the state with the contents of this event.
func (e *TCPSendmsgCall) Update(s *state.State) {
	s.UpdateFlow(e.Flow())
}

type TCPSendmsgV4Call struct {
	Meta   tracing.Metadata `kprobe:"metadata"`
	Socket uintptr          `kprobe:"sock"`
	Size   uintptr          `kprobe:"size"`
	LAddr  uint32           `kprobe:"laddr"`
	RAddr  uint32           `kprobe:"raddr"`
	LPort  uint16           `kprobe:"lport"`
	RPort  uint16           `kprobe:"rport"`
	AF     uint16           `kprobe:"family"`

	flow *state.Flow // for caching
}

func (e *TCPSendmsgV4Call) Flow() *state.Flow {
	if e.flow != nil {
		return e.flow
	}

	e.flow = state.NewFlow(
		e.Socket,
		e.Meta.PID,
		e.AF,
		unix.IPPROTO_TCP,
		e.Meta.Timestamp,
		state.NewEndpointIPv4(e.LAddr, e.LPort, 0, 0),
		state.NewEndpointIPv4(e.RAddr, e.RPort, 0, 0),
	).MarkOutbound()

	return e.flow
}

// String returns a representation of the event.
func (e *TCPSendmsgV4Call) String() string {
	flow := e.Flow()
	return fmt.Sprintf(
		"%s tcp_sendmsg(sock=0x%x, size=%d, af=%s, %s -> %s)",
		header(e.Meta),
		e.Socket,
		e.Size,
		flow.Type(),
		flow.Local().String(),
		flow.Remote().String())
}

// Update the state with the contents of this event.
func (e *TCPSendmsgV4Call) Update(s *state.State) {
	s.UpdateFlow(e.Flow())
}

type UDPSendmsgCall struct {
	Meta     tracing.Metadata `kprobe:"metadata"`
	Socket   uintptr          `kprobe:"sock"`
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

	flow *state.Flow // for caching
}

func (e *UDPSendmsgCall) Flow() *state.Flow {
	raddr, rport := e.RAddr, e.RPort
	if e.SIPtr == 0 || e.SIAF != unix.AF_INET {
		raddr = e.AltRAddr
		rport = e.AltRPort
	}

	return state.NewFlow(
		e.Socket,
		e.Meta.PID,
		unix.AF_INET,
		unix.IPPROTO_UDP,
		e.Meta.Timestamp,
		state.NewEndpointIPv4(e.LAddr, e.LPort, 1, uint64(e.Size)+minIPv4UdpPacketSize),
		state.NewEndpointIPv4(raddr, rport, 0, 0),
	).MarkOutbound()
}

// String returns a representation of the event.
func (e *UDPSendmsgCall) String() string {
	flow := e.Flow()
	return fmt.Sprintf(
		"%s udp_sendmsg(sock=0x%x, size=%d, %s -> %s)",
		header(e.Meta),
		e.Socket,
		e.Size,
		flow.Local().String(),
		flow.Remote().String())
}

// Update the state with the contents of this event.
func (e *UDPSendmsgCall) Update(s *state.State) {
	s.UpdateFlow(e.Flow())
}

type UDPv6SendmsgCall struct {
	Meta      tracing.Metadata `kprobe:"metadata"`
	Socket    uintptr          `kprobe:"sock"`
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

	flow *state.Flow // for caching

}

func (e *UDPv6SendmsgCall) Flow() *state.Flow {
	if e.flow != nil {
		return e.flow
	}

	raddra, raddrb, rport := e.RAddrA, e.RAddrB, e.RPort
	if e.SI6Ptr == 0 || e.SI6AF != unix.AF_INET6 {
		raddra, raddrb = e.AltRAddrA, e.AltRAddrB
		rport = e.AltRPort
	}
	e.flow = state.NewFlow(
		e.Socket,
		e.Meta.PID,
		unix.AF_INET6,
		unix.IPPROTO_UDP,
		e.Meta.Timestamp,
		// In IPv6, udpv6_sendmsg increments local counters as there is no
		// corresponding ip6_local_out call.
		state.NewEndpointIPv6(e.LAddrA, e.LAddrB, e.LPort, 1, uint64(e.Size)+minIPv6UdpPacketSize),
		state.NewEndpointIPv6(raddra, raddrb, rport, 0, 0),
	).MarkOutbound()

	return e.flow
}

// String returns a representation of the event.
func (e *UDPv6SendmsgCall) String() string {
	flow := e.Flow()
	return fmt.Sprintf(
		"%s udpv6_sendmsg(sock=0x%x, size=%d, %s -> %s)",
		header(e.Meta),
		e.Socket,
		e.Size,
		flow.Local().String(),
		flow.Remote().String())
}

// Update the state with the contents of this event.
func (e *UDPv6SendmsgCall) Update(s *state.State) {
	s.UpdateFlow(e.Flow())
}

type IPLocalOutCall struct {
	Meta   tracing.Metadata `kprobe:"metadata"`
	Socket uintptr          `kprobe:"sock"`
	Size   uint32           `kprobe:"size"`
	LAddr  uint32           `kprobe:"laddr"`
	RAddr  uint32           `kprobe:"raddr"`
	LPort  uint16           `kprobe:"lport"`
	RPort  uint16           `kprobe:"rport"`

	flow *state.Flow // for caching
}

func (e *IPLocalOutCall) Flow() *state.Flow {
	if e.flow != nil {
		return e.flow
	}

	e.flow = state.NewFlow(
		e.Socket,
		e.Meta.PID,
		unix.AF_INET,
		0,
		e.Meta.Timestamp,
		state.NewEndpointIPv4(e.LAddr, e.LPort, 1, uint64(e.Size)),
		state.NewEndpointIPv4(e.RAddr, e.RPort, 0, 0),
	).MarkOutbound()

	return e.flow
}

func (e *IPLocalOutCall) String() string {
	flow := e.Flow()
	return fmt.Sprintf(
		"%s ip_local_out(sock=0x%x, size=%d, %s -> %s)",
		header(e.Meta),
		e.Socket,
		e.Size,
		flow.Local().String(),
		flow.Remote().String())
}

// Update the state with the contents of this event.
func (e *IPLocalOutCall) Update(s *state.State) {
	flow := e.Flow()
	if flow.RemoteIP() == nil {
		// Unconnected-UDP flows have nil destination in here.
		return
	}
	// Only count non-UDP packets.
	// Those are already counted by udp_sendmsg, but there is no way
	// to discriminate UDP in ip_local_out at kprobe level.
	s.UpdateFlowWithCondition(flow, func(f *state.Flow) bool {
		return !f.IsUDP()
	})
}

type Inet6CskXmitCall struct {
	Meta    tracing.Metadata `kprobe:"metadata"`
	Socket  uintptr          `kprobe:"sock"`
	LAddr6a uint64           `kprobe:"laddr6a"`
	LAddr6b uint64           `kprobe:"laddr6b"`
	RAddr6a uint64           `kprobe:"raddr6a"`
	RAddr6b uint64           `kprobe:"raddr6b"`
	LPort   uint16           `kprobe:"lport"`
	RPort   uint16           `kprobe:"rport"`
	Size    uint32           `kprobe:"size"`

	flow *state.Flow // for caching
}

func (e *Inet6CskXmitCall) Flow() *state.Flow {
	if e.flow != nil {
		return e.flow
	}

	e.flow = state.NewFlow(
		e.Socket,
		e.Meta.PID,
		unix.AF_INET6,
		unix.IPPROTO_TCP,
		e.Meta.Timestamp,
		state.NewEndpointIPv6(e.LAddr6a, e.LAddr6b, e.LPort, 1, uint64(e.Size)),
		state.NewEndpointIPv6(e.RAddr6a, e.RAddr6b, e.RPort, 0, 0),
	).MarkOutbound()

	return e.flow
}

func (e *Inet6CskXmitCall) String() string {
	flow := e.Flow()
	return fmt.Sprintf(
		"%s inet6_csk_xmit(sock=0x%x, size=%d, %s -> %s)",
		header(e.Meta),
		e.Socket,
		e.Size,
		flow.Local().String(),
		flow.Remote().String())
}

func (e *Inet6CskXmitCall) Update(s *state.State) {
	s.UpdateFlow(e.Flow())
}
