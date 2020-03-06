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

	flow *common.Flow // for caching
}

func (e *UDPSendmsgCall) Flow() *common.Flow {
	raddr, rport := e.RAddr, e.RPort
	if e.SIPtr == 0 || e.SIAF != unix.AF_INET {
		raddr = e.AltRAddr
		rport = e.AltRPort
	}

	return common.NewFlow(
		e.Socket,
		e.Meta.PID,
		unix.AF_INET,
		unix.IPPROTO_UDP,
		e.Meta.Timestamp,
		common.NewEndpointIPv4(e.LAddr, e.LPort, 1, uint64(e.Size)+minIPv4UdpPacketSize),
		common.NewEndpointIPv4(raddr, rport, 0, 0),
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
func (e *UDPSendmsgCall) Update(s common.EventTracker) {
	s.UpdateFlow(e.Flow())
}
