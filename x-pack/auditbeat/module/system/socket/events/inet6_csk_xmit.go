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
}

func (e *Inet6CskXmitCall) Flow() *common.Flow {
	return common.NewFlow(
		e.Socket,
		e.Meta.PID,
		unix.AF_INET6,
		unix.IPPROTO_TCP,
		e.Meta.Timestamp,
		common.NewEndpointIPv6(e.LAddr6a, e.LAddr6b, e.LPort, 1, uint64(e.Size)),
		common.NewEndpointIPv6(e.RAddr6a, e.RAddr6b, e.RPort, 0, 0),
	).MarkOutbound()
}

func (e *Inet6CskXmitCall) String() string {
	flow := e.Flow()
	return fmt.Sprintf(
		"%s inet6_csk_xmit(sock=0x%x, size=%d, %s -> %s)",
		header(e.Meta),
		e.Socket,
		e.Size,
		flow.Local,
		flow.Remote)
}

func (e *Inet6CskXmitCall) Update(s common.EventTracker) {
	s.UpdateFlow(e.Flow())
}
