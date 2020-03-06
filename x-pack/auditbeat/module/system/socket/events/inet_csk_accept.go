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

type InetCskAcceptReturn struct {
	Meta    tracing.Metadata `kprobe:"metadata"`
	Socket  uintptr          `kprobe:"sock"`
	LAddr   uint32           `kprobe:"laddr"`
	RAddr   uint32           `kprobe:"raddr"`
	LPort   uint16           `kprobe:"lport"`
	RPort   uint16           `kprobe:"rport"`
	LAddr6a uint64           `kprobe:"laddr6a"`
	LAddr6b uint64           `kprobe:"laddr6b"`
	RAddr6a uint64           `kprobe:"raddr6a"`
	RAddr6b uint64           `kprobe:"raddr6b"`
	AF      uint16           `kprobe:"family"`

	flow *common.Flow // for caching
}

func (e *InetCskAcceptReturn) Flow() *common.Flow {
	if e.flow != nil {
		return e.flow
	}

	var local, remote *common.Endpoint
	if e.AF == unix.AF_INET {
		local = common.NewEndpointIPv4(e.LAddr, e.LPort, 0, 0)
		remote = common.NewEndpointIPv4(e.RAddr, e.RPort, 0, 0)
	} else {
		local = common.NewEndpointIPv6(e.LAddr6a, e.LAddr6b, e.LPort, 0, 0)
		remote = common.NewEndpointIPv6(e.RAddr6a, e.RAddr6b, e.RPort, 0, 0)
	}

	e.flow = common.NewFlow(
		e.Socket,
		e.Meta.PID,
		e.AF,
		unix.IPPROTO_TCP,
		e.Meta.Timestamp,
		local,
		remote,
	).MarkInbound().MarkComplete()

	return e.flow
}

// String returns a representation of the event.
func (e *InetCskAcceptReturn) String() string {
	f := e.Flow()
	return fmt.Sprintf("%s <- accept(sock=0x%x, af=%s, %s <- %s)", header(e.Meta), e.Socket, f.Type(), f.Local(), f.Remote())
}

// Update the state with the contents of this event.
func (e *InetCskAcceptReturn) Update(s common.EventTracker) {
	if e.Socket != 0 {
		s.UpdateFlow(e.Flow())
	}
}

type InetCskAcceptV4Return struct {
	Meta   tracing.Metadata `kprobe:"metadata"`
	Socket uintptr          `kprobe:"sock"`
	LAddr  uint32           `kprobe:"laddr"`
	RAddr  uint32           `kprobe:"raddr"`
	LPort  uint16           `kprobe:"lport"`
	RPort  uint16           `kprobe:"rport"`
	AF     uint16           `kprobe:"family"`

	flow *common.Flow // for caching
}

func (e *InetCskAcceptV4Return) Flow() *common.Flow {
	if e.flow != nil {
		return e.flow
	}

	e.flow = common.NewFlow(
		e.Socket,
		e.Meta.PID,
		e.AF,
		unix.IPPROTO_TCP,
		e.Meta.Timestamp,
		common.NewEndpointIPv4(e.LAddr, e.LPort, 0, 0),
		common.NewEndpointIPv4(e.RAddr, e.RPort, 0, 0),
	).MarkInbound().MarkComplete()

	return e.flow
}

// String returns a representation of the event.
func (e *InetCskAcceptV4Return) String() string {
	f := e.Flow()
	return fmt.Sprintf("%s <- accept(sock=0x%x, af=%s, %s <- %s)", header(e.Meta), e.Socket, f.Type(), f.Local().String(), f.Remote().String())
}

// Update the state with the contents of this event.
func (e *InetCskAcceptV4Return) Update(s common.EventTracker) {
	if e.Socket != 0 {
		s.UpdateFlow(e.Flow())
	}
}
