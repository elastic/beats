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

type InetCreateCall struct {
	Meta  tracing.Metadata `kprobe:"metadata"`
	Proto int32            `kprobe:"proto"`
}

// String returns a representation of the event.
func (e *InetCreateCall) String() string {
	return fmt.Sprintf("%s inet_create(proto=%d)", header(e.Meta), e.Proto)
}

// Update the state with the contents of this event.
func (e *InetCreateCall) Update(s common.EventTracker) {
	if e.Proto == 0 || e.Proto == unix.IPPROTO_TCP || e.Proto == unix.IPPROTO_UDP {
		s.PushThreadEvent(e.Meta.TID, e)
	}
}

type SockInitDataCall struct {
	Meta   tracing.Metadata `kprobe:"metadata"`
	Socket uintptr          `kprobe:"sock"`
}

// String returns a representation of the event.
func (e *SockInitDataCall) String() string {
	return fmt.Sprintf("%s sock_init_data(sock=0x%x)", header(e.Meta), e.Socket)
}

// Update the state with the contents of this event.
func (e *SockInitDataCall) Update(s common.EventTracker) {
	if event := s.PopThreadEvent(e.Meta.TID); event != nil {
		// Only track socks created by inet_create / inet6_create
		if call, ok := event.(*InetCreateCall); ok {
			s.UpdateFlow(common.NewFlow(
				e.Socket,
				e.Meta.PID,
				0,
				uint16(call.Proto),
				e.Meta.Timestamp,
				nil,
				nil,
			).SetCreated(e.Meta.Timestamp).MarkComplete())
		}
	}
}
