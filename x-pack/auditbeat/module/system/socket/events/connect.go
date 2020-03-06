// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package events

import (
	"encoding/binary"
	"fmt"
	"net"

	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/common"
	"github.com/elastic/beats/v7/x-pack/auditbeat/tracing"
	"golang.org/x/sys/unix"
)

type TCPv4ConnectCall struct {
	Meta   tracing.Metadata `kprobe:"metadata"`
	Socket uintptr          `kprobe:"sock"`
	LAddr  uint32           `kprobe:"laddr"`
	RAddr  uint32           `kprobe:"addr"`
	LPort  uint16           `kprobe:"lport"`
	RPort  uint16           `kprobe:"port"`
}

// String returns a representation of the event.
func (e *TCPv4ConnectCall) String() string {
	var buf [4]byte
	tracing.MachineEndian.PutUint32(buf[:], e.LAddr)
	laddr := net.IPv4(buf[0], buf[1], buf[2], buf[3])
	tracing.MachineEndian.PutUint16(buf[:], e.LPort)
	lport := binary.BigEndian.Uint16(buf[:])
	tracing.MachineEndian.PutUint32(buf[:], e.RAddr)
	raddr := net.IPv4(buf[0], buf[1], buf[2], buf[3])
	tracing.MachineEndian.PutUint16(buf[:], e.RPort)
	rport := binary.BigEndian.Uint16(buf[:])
	return fmt.Sprintf(
		"%s connect(sock=0x%x, %s:%d -> %s:%d)",
		header(e.Meta),
		e.Socket,
		laddr.String(),
		lport,
		raddr.String(),
		rport)
}

func (e *TCPv4ConnectCall) Flow() *common.Flow {
	return common.NewFlow(
		e.Socket,
		e.Meta.PID,
		unix.AF_INET,
		unix.IPPROTO_TCP,
		e.Meta.Timestamp,
		common.NewEndpointIPv4(e.LAddr, e.LPort, 0, 0),
		common.NewEndpointIPv4(e.RAddr, e.RPort, 0, 0),
	).MarkOutbound()
}

// Update the state with the contents of this event.
func (e *TCPv4ConnectCall) Update(s common.EventTracker) {
	s.PushThreadEvent(e.Meta.TID, e)
}

type TCPv6ConnectCall struct {
	Meta   tracing.Metadata `kprobe:"metadata"`
	Socket uintptr          `kprobe:"sock"`
	LAddrA uint64           `kprobe:"laddra"`
	LAddrB uint64           `kprobe:"laddrb"`
	RAddrA uint64           `kprobe:"addra"`
	RAddrB uint64           `kprobe:"addrb"`
	LPort  uint16           `kprobe:"lport"`
	RPort  uint16           `kprobe:"port"`
}

// String returns a representation of the event.
func (e *TCPv6ConnectCall) String() string {
	var buf [16]byte
	tracing.MachineEndian.PutUint64(buf[:], e.LAddrA)
	tracing.MachineEndian.PutUint64(buf[8:], e.LAddrB)
	laddr := net.IP(buf[:]).String()
	tracing.MachineEndian.PutUint64(buf[:], e.RAddrA)
	tracing.MachineEndian.PutUint64(buf[8:], e.RAddrB)
	raddr := net.IP(buf[:]).String()
	tracing.MachineEndian.PutUint16(buf[:], e.LPort)
	lport := binary.BigEndian.Uint16(buf[:])
	tracing.MachineEndian.PutUint16(buf[:], e.RPort)
	rport := binary.BigEndian.Uint16(buf[:])
	return fmt.Sprintf(
		"%s connect6(sock=0x%x, %s:%d -> %s:%d)",
		header(e.Meta),
		e.Socket,
		laddr,
		lport,
		raddr,
		rport)
}

func (e *TCPv6ConnectCall) Flow() *common.Flow {
	return common.NewFlow(
		e.Socket,
		e.Meta.PID,
		unix.AF_INET6,
		unix.IPPROTO_TCP,
		e.Meta.Timestamp,
		common.NewEndpointIPv6(e.LAddrA, e.LAddrB, e.LPort, 0, 0),
		common.NewEndpointIPv6(e.RAddrA, e.RAddrB, e.RPort, 0, 0),
	).MarkOutbound()
}

// Update the state with the contents of this event.
func (e *TCPv6ConnectCall) Update(s common.EventTracker) {
	s.PushThreadEvent(e.Meta.TID, e)
}

type TCPConnectReturn struct {
	Meta   tracing.Metadata `kprobe:"metadata"`
	Retval int32            `kprobe:"retval"`
}

// String returns a representation of the event.
func (e *TCPConnectReturn) String() string {
	return fmt.Sprintf("%s <- connect %s", header(e.Meta), kernErrorDesc(e.Retval))
}

// Update the state with the contents of this event.
func (e *TCPConnectReturn) Update(s common.EventTracker) {
	if e.Retval == 0 {
		if event := s.PopThreadEvent(e.Meta.TID); event != nil {
			switch call := event.(type) {
			case *TCPv4ConnectCall:
				s.UpdateFlow(call.Flow().MarkComplete())
			case *TCPv6ConnectCall:
				s.UpdateFlow(call.Flow().MarkComplete())
			}
		}
	}
}
