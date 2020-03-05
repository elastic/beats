// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package events

import (
	"fmt"

	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/state"
	"github.com/elastic/beats/v7/x-pack/auditbeat/tracing"
)

type InetReleaseCall struct {
	Meta   tracing.Metadata `kprobe:"metadata"`
	Socket uintptr          `kprobe:"sock"`
}

// String returns a representation of the event.
func (e *InetReleaseCall) String() string {
	return fmt.Sprintf("%s inet_release(sock=0x%x)", header(e.Meta), e.Socket)
}

// Update the state with the contents of this event.
func (e *InetReleaseCall) Update(s *state.State) {
	s.SocketEnd(e.Socket, e.Meta.PID)
}

type DoExitCall struct {
	Meta tracing.Metadata `kprobe:"metadata"`
}

// String returns a representation of the event.
func (e *DoExitCall) String() string {
	whatExited := "process"
	if e.Meta.PID != e.Meta.TID {
		whatExited = "thread"
	}
	return fmt.Sprintf("%s do_exit(%s)", header(e.Meta), whatExited)
}

// Update the state with the contents of this event.
func (e *DoExitCall) Update(s *state.State) {
	// Only report exits of the main thread, a.k.a process exit
	if e.Meta.PID == e.Meta.TID {
		s.ProcessEnd(e.Meta.PID)
	}
	// Cleanup any saved thread state
	s.PopThreadEvent(e.Meta.TID)
}
