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

type TCPv4DoRcvCall struct {
	Meta   tracing.Metadata `kprobe:"metadata"`
	Socket uintptr          `kprobe:"sock"`
	Size   uint32           `kprobe:"size"`
	LAddr  uint32           `kprobe:"laddr"`
	RAddr  uint32           `kprobe:"raddr"`
	LPort  uint16           `kprobe:"lport"`
	RPort  uint16           `kprobe:"rport"`
}

func (e *TCPv4DoRcvCall) Flow() *common.Flow {
	return common.NewFlow(
		e.Socket,
		e.Meta.PID,
		unix.AF_INET,
		unix.IPPROTO_TCP,
		e.Meta.Timestamp,
		common.NewEndpointIPv4(e.LAddr, e.LPort, 0, 0),
		common.NewEndpointIPv4(e.RAddr, e.RPort, 1, uint64(e.Size)),
	)
}

// String returns a representation of the event.
func (e *TCPv4DoRcvCall) String() string {
	flow := e.Flow()
	return fmt.Sprintf(
		"%s tcp_v4_do_rcv(sock=0x%x, size=%d, %s <- %s)",
		header(e.Meta),
		e.Socket,
		e.Size,
		flow.Local,
		flow.Remote,
	)
}

// Update the state with the contents of this event.
func (e *TCPv4DoRcvCall) Update(s common.EventTracker) {
	s.UpdateFlow(e.Flow())
}
