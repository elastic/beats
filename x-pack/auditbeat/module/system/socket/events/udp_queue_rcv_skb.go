// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package events

import (
	"fmt"

	"golang.org/x/sys/unix"

	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/common"
	"github.com/elastic/beats/v7/x-pack/auditbeat/tracing"
)

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
}

func (e *UDPQueueRcvSkbCall) Flow() *common.Flow {
	var remote *common.Endpoint
	var valid bool
	if valid = validIPv4Headers(e.IPHdr, e.UDPHdr, e.Packet[:]); !valid {
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
	}
	if valid {
		raddr := tracing.MachineEndian.Uint32(e.Packet[e.IPHdr+12:])
		rport := tracing.MachineEndian.Uint16(e.Packet[e.UDPHdr:])
		remote = common.NewEndpointIPv4(raddr, rport, 1, uint64(e.Size)+minIPv4UdpPacketSize)
	}

	return common.NewFlow(
		e.Socket,
		e.Meta.PID,
		unix.AF_INET,
		unix.IPPROTO_UDP,
		e.Meta.Timestamp,
		common.NewEndpointIPv4(e.LAddr, e.LPort, 0, 0),
		remote,
	).MarkInbound()
}

// String returns a representation of the event.
func (e *UDPQueueRcvSkbCall) String() string {
	flow := e.Flow()
	return fmt.Sprintf(
		"%s udp_queue_rcv_skb(sock=0x%x, size=%d, %s <- %s)",
		header(e.Meta),
		e.Socket,
		e.Size,
		flow.Local,
		flow.Remote,
	)
}

// Update the state with the contents of this event.
func (e *UDPQueueRcvSkbCall) Update(s common.EventTracker) {
	s.UpdateFlow(e.Flow())
}
