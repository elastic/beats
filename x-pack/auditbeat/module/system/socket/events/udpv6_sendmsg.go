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
}

func (e *UDPv6SendmsgCall) Flow() *common.Flow {
	raddra, raddrb, rport := e.RAddrA, e.RAddrB, e.RPort
	if e.SI6Ptr == 0 || e.SI6AF != unix.AF_INET6 {
		raddra, raddrb = e.AltRAddrA, e.AltRAddrB
		rport = e.AltRPort
	}
	return common.NewFlow(
		e.Socket,
		e.Meta.PID,
		unix.AF_INET6,
		unix.IPPROTO_UDP,
		e.Meta.Timestamp,
		// In IPv6, udpv6_sendmsg increments local counters as there is no
		// corresponding ip6_local_out call.
		common.NewEndpointIPv6(e.LAddrA, e.LAddrB, e.LPort, 1, uint64(e.Size)+minIPv6UdpPacketSize),
		common.NewEndpointIPv6(raddra, raddrb, rport, 0, 0),
	).MarkOutbound()
}

// String returns a representation of the event.
func (e *UDPv6SendmsgCall) String() string {
	flow := e.Flow()
	return fmt.Sprintf(
		"%s udpv6_sendmsg(sock=0x%x, size=%d, %s -> %s)",
		header(e.Meta),
		e.Socket,
		e.Size,
		flow.Local,
		flow.Remote)
}

// Update the state with the contents of this event.
func (e *UDPv6SendmsgCall) Update(s common.EventTracker) {
	s.UpdateFlow(e.Flow())
}
