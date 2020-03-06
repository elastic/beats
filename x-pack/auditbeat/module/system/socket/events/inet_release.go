// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package events

import (
	"fmt"

	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/common"
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
func (e *InetReleaseCall) Update(s common.EventTracker) {
	s.SocketEnd(e.Socket, e.Meta.PID)
}
