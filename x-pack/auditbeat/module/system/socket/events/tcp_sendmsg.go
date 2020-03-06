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
}

func (e *TCPSendmsgCall) Flow() *common.Flow {
	var local, remote *common.Endpoint
	if e.AF == unix.AF_INET {
		local = common.NewEndpointIPv4(e.LAddr, e.LPort, 0, 0)
		remote = common.NewEndpointIPv4(e.RAddr, e.RPort, 0, 0)
	} else {
		local = common.NewEndpointIPv6(e.LAddr6a, e.LAddr6b, e.LPort, 0, 0)
		remote = common.NewEndpointIPv6(e.RAddr6a, e.RAddr6b, e.RPort, 0, 0)
	}

	return common.NewFlow(
		e.Socket,
		e.Meta.PID,
		e.AF,
		unix.IPPROTO_TCP,
		e.Meta.Timestamp,
		local,
		remote,
	).MarkOutbound()
}

// String returns a representation of the event.
func (e *TCPSendmsgCall) String() string {
	flow := e.Flow()
	return fmt.Sprintf(
		"%s tcp_sendmsg(sock=0x%x, size=%d, af=%s, %s -> %s)",
		header(e.Meta),
		e.Socket,
		e.Size,
		flow.Type,
		flow.Local,
		flow.Remote)
}

// Update the state with the contents of this event.
func (e *TCPSendmsgCall) Update(s common.EventTracker) {
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
}

func (e *TCPSendmsgV4Call) Flow() *common.Flow {
	return common.NewFlow(
		e.Socket,
		e.Meta.PID,
		e.AF,
		unix.IPPROTO_TCP,
		e.Meta.Timestamp,
		common.NewEndpointIPv4(e.LAddr, e.LPort, 0, 0),
		common.NewEndpointIPv4(e.RAddr, e.RPort, 0, 0),
	).MarkOutbound()
}

// String returns a representation of the event.
func (e *TCPSendmsgV4Call) String() string {
	flow := e.Flow()
	return fmt.Sprintf(
		"%s tcp_sendmsg(sock=0x%x, size=%d, af=%s, %s -> %s)",
		header(e.Meta),
		e.Socket,
		e.Size,
		flow.Type,
		flow.Local,
		flow.Remote)
}

// Update the state with the contents of this event.
func (e *TCPSendmsgV4Call) Update(s common.EventTracker) {
	s.UpdateFlow(e.Flow())
}
