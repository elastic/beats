// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package events

import (
	"fmt"

	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/common"
	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/state"
	"github.com/elastic/beats/v7/x-pack/auditbeat/tracing"
	"golang.org/x/sys/unix"
)

type TCPv4DoRcvCall struct {
	Meta   tracing.Metadata `kprobe:"metadata"`
	Socket uintptr          `kprobe:"sock"`
	Size   uint32           `kprobe:"size"`
	LAddr  uint32           `kprobe:"laddr"`
	RAddr  uint32           `kprobe:"raddr"`
	LPort  uint16           `kprobe:"lport"`
	RPort  uint16           `kprobe:"rport"`

	flow *state.Flow // for caching
}

func (e *TCPv4DoRcvCall) Flow() *state.Flow {
	if e.flow != nil {
		return e.flow
	}

	e.flow = state.NewFlow(
		e.Socket,
		e.Meta.PID,
		unix.AF_INET,
		unix.IPPROTO_TCP,
		e.Meta.Timestamp,
		state.NewEndpointIPv4(e.LAddr, e.LPort, 0, 0),
		state.NewEndpointIPv4(e.RAddr, e.RPort, 1, uint64(e.Size)),
	)

	return e.flow
}

// String returns a representation of the event.
func (e *TCPv4DoRcvCall) String() string {
	flow := e.Flow()
	return fmt.Sprintf(
		"%s tcp_v4_do_rcv(sock=0x%x, size=%d, %s <- %s)",
		header(e.Meta),
		e.Socket,
		e.Size,
		flow.Local().String(),
		flow.Remote().String(),
	)
}

// Update the state with the contents of this event.
func (e *TCPv4DoRcvCall) Update(s *state.State) {
	s.UpdateFlow(e.Flow())
}

type TCPv6DoRcvCall struct {
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

func (e *TCPv6DoRcvCall) Flow() *state.Flow {
	if e.flow != nil {
		return e.flow
	}

	e.flow = state.NewFlow(
		e.Socket,
		e.Meta.PID,
		unix.AF_INET6,
		unix.IPPROTO_TCP,
		e.Meta.Timestamp,
		state.NewEndpointIPv6(e.LAddr6a, e.LAddr6b, e.LPort, 0, 0),
		state.NewEndpointIPv6(e.RAddr6a, e.RAddr6b, e.RPort, 1, uint64(e.Size)),
	)

	return e.flow
}

// String returns a representation of the event.
func (e *TCPv6DoRcvCall) String() string {
	flow := e.Flow()
	return fmt.Sprintf(
		"%s tcp_v6_do_rcv(sock=0x%x, size=%d, %s <- %s)",
		header(e.Meta),
		e.Socket,
		e.Size,
		flow.Local().String(),
		flow.Remote().String(),
	)
}

// Update the state with the contents of this event.
func (e *TCPv6DoRcvCall) Update(s *state.State) {
	s.UpdateFlow(e.Flow())
}

func validIPv4Headers(ipHdr uint16, udpHdr uint16, data []byte) bool {
	return ipHdr != 0 &&
		int(ipHdr)+20 < len(data) &&
		data[ipHdr]&0xF0 == 0x40 &&
		udpHdr != 0 &&
		int(udpHdr)+12 < len(data)
}

type UDPQueueRcvSkbCall struct {
	Meta   tracing.Metadata                 `kprobe:"metadata"`
	Socket uintptr                          `kprobe:"sock"`
	Size   uint32                           `kprobe:"size"`
	LAddr  uint32                           `kprobe:"laddr"`
	LPort  uint16                           `kprobe:"lport"`
	IPHdr  uint16                           `kprobe:"iphdr"`
	UDPHdr uint16                           `kprobe:"udphdr"`
	Base   uintptr                          `kprobe:"base"`
	Packet [common.SkBuffDataDumpBytes]byte `kprobe:"packet,greedy"`

	flow *state.Flow // for caching
}

func (e *UDPQueueRcvSkbCall) Flow() *state.Flow {
	if e.flow != nil {
		return e.flow
	}

	var remote *state.Endpoint
	if valid := validIPv4Headers(e.IPHdr, e.UDPHdr, e.Packet[:]); valid {
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
		if e.IPHdr > base && e.UDPHdr > base {
			ipOff := e.IPHdr - base
			udpOff := e.UDPHdr - base
			if valid = validIPv4Headers(ipOff, udpOff, e.Packet[:]); valid {
				e.IPHdr = ipOff
				e.UDPHdr = udpOff
			}
		}
		if valid {
			raddr := tracing.MachineEndian.Uint32(e.Packet[e.IPHdr+12:])
			rport := tracing.MachineEndian.Uint16(e.Packet[e.UDPHdr:])
			remote = state.NewEndpointIPv4(raddr, rport, 1, uint64(e.Size)+minIPv4UdpPacketSize)
		}
	}

	e.flow = state.NewFlow(
		e.Socket,
		e.Meta.PID,
		unix.AF_INET,
		unix.IPPROTO_UDP,
		e.Meta.Timestamp,
		state.NewEndpointIPv4(e.LAddr, e.LPort, 0, 0),
		remote,
	).MarkInbound()

	return e.flow
}

// String returns a representation of the event.
func (e *UDPQueueRcvSkbCall) String() string {
	flow := e.Flow()
	return fmt.Sprintf(
		"%s udp_queue_rcv_skb(sock=0x%x, size=%d, %s <- %s)",
		header(e.Meta),
		e.Socket,
		e.Size,
		flow.Local().String(),
		flow.Remote().String(),
	)
}

// Update the state with the contents of this event.
func (e *UDPQueueRcvSkbCall) Update(s *state.State) {
	s.UpdateFlow(e.Flow())
}

func validIPv6Headers(ipHdr uint16, udpHdr uint16, data []byte) bool {
	return ipHdr != 0 &&
		int(ipHdr)+40 < len(data) &&
		data[ipHdr]&0xF0 == 0x60 &&
		udpHdr != 0 &&
		int(udpHdr)+12 < len(data)
}

type UDPv6QueueRcvSkbCall struct {
	Meta   tracing.Metadata                 `kprobe:"metadata"`
	Socket uintptr                          `kprobe:"sock"`
	Size   uint32                           `kprobe:"size"`
	LAddrA uint64                           `kprobe:"laddra"`
	LAddrB uint64                           `kprobe:"laddrb"`
	LPort  uint16                           `kprobe:"lport"`
	IPHdr  uint16                           `kprobe:"iphdr"`
	UDPHdr uint16                           `kprobe:"udphdr"`
	Base   uintptr                          `kprobe:"base"`
	Packet [common.SkBuffDataDumpBytes]byte `kprobe:"packet,greedy"`

	flow *state.Flow // for caching
}

func (e *UDPv6QueueRcvSkbCall) Flow() *state.Flow {
	if e.flow != nil {
		return e.flow
	}

	var remote *state.Endpoint
	if valid := validIPv6Headers(e.IPHdr, e.UDPHdr, e.Packet[:]); valid {
		// Check if we're dealing with pointers
		// TODO: This only works in little-endian, same as in udpQueueRcvSkb
		base := uint16(e.Base)
		if e.IPHdr > base && e.UDPHdr > base {
			ipOff := e.IPHdr - base
			udpOff := e.UDPHdr - base
			if valid = validIPv6Headers(ipOff, udpOff, e.Packet[:]); valid {
				e.IPHdr = ipOff
				e.UDPHdr = udpOff
			}
		}
		if valid {
			raddrA := tracing.MachineEndian.Uint64(e.Packet[e.IPHdr+8:])
			raddrB := tracing.MachineEndian.Uint64(e.Packet[e.IPHdr+16:])
			rport := tracing.MachineEndian.Uint16(e.Packet[e.UDPHdr:])
			remote = state.NewEndpointIPv6(raddrA, raddrB, rport, 1, uint64(e.Size)+minIPv6UdpPacketSize)
		}
	}

	e.flow = state.NewFlow(
		e.Socket,
		e.Meta.PID,
		unix.AF_INET6,
		unix.IPPROTO_UDP,
		e.Meta.Timestamp,
		state.NewEndpointIPv6(e.LAddrA, e.LAddrB, e.LPort, 0, 0),
		remote,
	).MarkInbound()

	return e.flow
}

// String returns a representation of the event.
func (e *UDPv6QueueRcvSkbCall) String() string {
	flow := e.Flow()
	return fmt.Sprintf(
		"%s udpv6_queue_rcv_skb(sock=0x%x, size=%d, %s <- %s)",
		header(e.Meta),
		e.Socket,
		e.Size,
		flow.Local().String(),
		flow.Remote().String(),
	)
}

// Update the state with the contents of this event.
func (e *UDPv6QueueRcvSkbCall) Update(s *state.State) {
	s.UpdateFlow(e.Flow())
}
