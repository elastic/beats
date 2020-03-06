// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package events

import (
	"fmt"

	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/common"
	"github.com/elastic/beats/v7/x-pack/auditbeat/tracing"
	"golang.org/x/sys/unix"
)

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

	flow *common.Flow // for caching
}

func (e *UDPv6QueueRcvSkbCall) Flow() *common.Flow {
	if e.flow != nil {
		return e.flow
	}

	var remote *common.Endpoint
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
			remote = common.NewEndpointIPv6(raddrA, raddrB, rport, 1, uint64(e.Size)+minIPv6UdpPacketSize)
		}
	}

	e.flow = common.NewFlow(
		e.Socket,
		e.Meta.PID,
		unix.AF_INET6,
		unix.IPPROTO_UDP,
		e.Meta.Timestamp,
		common.NewEndpointIPv6(e.LAddrA, e.LAddrB, e.LPort, 0, 0),
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
func (e *UDPv6QueueRcvSkbCall) Update(s common.EventTracker) {
	s.UpdateFlow(e.Flow())
}
