// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package common

import (
	"net"
	"os"
	"time"
)

var currentProcesPID int

func init() {
	currentProcesPID = os.Getpid()
}

type Flow struct {
	*Socket

	socket uintptr

	PID           uint32
	Type          InetType
	Proto         FlowProto
	Local, Remote *Endpoint
	Direction     FlowDirection

	created, lastSeen uint64 // kernel time
	complete          bool

	// these are automatically calculated by state from kernelTimes above
	createdTime, lastSeenTime time.Time
}

func NewFlow(sock uintptr, pid uint32, inetType, proto uint16, lastSeen uint64, local, remote *Endpoint) *Flow {
	return &Flow{
		socket:   sock,
		PID:      pid,
		Type:     InetType(inetType),
		Proto:    FlowProto(proto),
		Local:    local,
		Remote:   remote,
		lastSeen: lastSeen,
	}
}

func (f *Flow) MarkOutbound() *Flow {
	f.Direction = DirectionOutbound
	return f
}

func (f *Flow) MarkInbound() *Flow {
	f.Direction = DirectionInbound
	return f
}

func (f *Flow) MarkComplete() *Flow {
	f.complete = true
	return f
}

func (f *Flow) SetCreated(ts uint64) *Flow {
	f.created = ts
	return f
}

func (f *Flow) MarkNew() *Flow {
	f.createdTime = f.lastSeenTime
	return f
}

func (f *Flow) FormatLastSeen(formatter func(uint64) time.Time) *Flow {
	f.lastSeenTime = formatter(f.lastSeen)
	return f
}

func (f *Flow) FormatCreated(formatter func(uint64) time.Time) *Flow {
	f.createdTime = formatter(f.created)
	return f
}

func (f *Flow) Ptr() uintptr {
	return f.socket
}

func (f *Flow) DNSAddress() *net.UDPAddr {
	if f.Remote != nil &&
		f.Local != nil &&
		f.Remote.addr != nil &&
		f.Local.addr != nil &&
		f.Remote.addr.Port == 53 &&
		f.Proto == ProtoUDP &&
		f.PID != 0 &&
		f.Process != nil {
		return &net.UDPAddr{
			IP:   f.Local.addr.IP,
			Port: f.Local.addr.Port,
		}
	}
	return nil
}

func (f *Flow) IsCurrentProcess() bool {
	return int(f.PID) == currentProcesPID
}

func (f *Flow) IsValid() bool {
	return f.Type != InetTypeUnknown && f.Proto != ProtoUnknown && f.Local != nil && f.Remote != nil
}

func (f *Flow) HasKey() bool {
	return f.Remote != nil && f.Local != nil && f.Remote.addr != nil && f.Local.addr != nil
}

func (f *Flow) Key() string {
	return f.Remote.addr.String() + "|" + f.Local.addr.String()
}

func (f *Flow) Terminate() {
	if f.Socket != nil && f.HasKey() {
		f.Socket.removeFlow(f.Key())
	}
}

func (f *Flow) Merge(ref *Flow) {
	f.lastSeenTime = ref.lastSeenTime
	if f.Type == InetTypeUnknown {
		f.Type = ref.Type
	}

	if f.Proto == ProtoUnknown {
		f.Proto = ref.Proto
	}

	if f.PID == 0 {
		f.PID = ref.PID
	}

	if f.Direction == DirectionUnknown {
		f.Direction = ref.Direction
	}

	if ref.complete {
		f.complete = ref.complete
	}

	f.Local.updateWith(ref.Local)
	f.Remote.updateWith(ref.Remote)
}
