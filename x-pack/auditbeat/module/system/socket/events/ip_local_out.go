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

type IPLocalOutCall struct {
	Meta   tracing.Metadata `kprobe:"metadata"`
	Socket uintptr          `kprobe:"sock"`
	Size   uint32           `kprobe:"size"`
	LAddr  uint32           `kprobe:"laddr"`
	RAddr  uint32           `kprobe:"raddr"`
	LPort  uint16           `kprobe:"lport"`
	RPort  uint16           `kprobe:"rport"`
}

func (e *IPLocalOutCall) Flow() *common.Flow {
	return common.NewFlow(
		e.Socket,
		e.Meta.PID,
		unix.AF_INET,
		0,
		e.Meta.Timestamp,
		common.NewEndpointIPv4(e.LAddr, e.LPort, 1, uint64(e.Size)),
		common.NewEndpointIPv4(e.RAddr, e.RPort, 0, 0),
	).MarkOutbound()
}

func (e *IPLocalOutCall) String() string {
	flow := e.Flow()
	return fmt.Sprintf(
		"%s ip_local_out(sock=0x%x, size=%d, %s -> %s)",
		header(e.Meta),
		e.Socket,
		e.Size,
		flow.Local,
		flow.Remote)
}

// Update the state with the contents of this event.
func (e *IPLocalOutCall) Update(s common.EventTracker) {
	flow := e.Flow()
	if flow.Remote == nil {
		// Unconnected-UDP flows have nil destination in here.
		return
	}
	// Only count non-UDP packets.
	// Those are already counted by udp_sendmsg, but there is no way
	// to discriminate UDP in ip_local_out at kprobe level.
	s.UpdateFlowWithCondition(flow, func(f *common.Flow) bool {
		return f.Proto != common.ProtoUDP
	})
}
