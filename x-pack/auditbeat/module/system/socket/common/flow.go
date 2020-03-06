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

	inetType          InetType
	proto             FlowProto
	direction         FlowDirection
	created, lastSeen uint64 // kernel time
	pid               uint32
	local, remote     *Endpoint
	complete          bool

	// these are automatically calculated by state from kernelTimes above
	createdTime, lastSeenTime time.Time
}

func NewFlow(sock uintptr, pid uint32, inetType, proto uint16, lastSeen uint64, local, remote *Endpoint) *Flow {
	return &Flow{
		socket:   sock,
		pid:      pid,
		inetType: InetType(inetType),
		proto:    FlowProto(proto),
		lastSeen: lastSeen,
		local:    local,
		remote:   remote,
	}
}

func (f *Flow) MarkOutbound() *Flow {
	f.direction = DirectionOutbound
	return f
}

func (f *Flow) MarkInbound() *Flow {
	f.direction = DirectionInbound
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

func (f *Flow) Terminate() {
	if f.Socket != nil && f.HasKey() {
		f.Socket.RemoveFlow(f.Key())
	}
}

func (f *Flow) SocketPtr() uintptr {
	return f.socket
}

func (f *Flow) FormatLastSeen(formatter func(uint64) time.Time) *Flow {
	f.lastSeenTime = formatter(f.lastSeen)
	return f
}

func (f *Flow) FormatCreated(formatter func(uint64) time.Time) *Flow {
	f.createdTime = formatter(f.created)
	return f
}

func (f *Flow) FirstSeen() *Flow {
	f.createdTime = f.lastSeenTime
	return f
}

func (f *Flow) PID() uint32 {
	return f.pid
}

func (f *Flow) Proto() FlowProto {
	return f.proto
}

func (f *Flow) Local() *Endpoint {
	return f.local
}

func (f *Flow) Remote() *Endpoint {
	return f.remote
}

func (f *Flow) Type() InetType {
	return f.inetType
}

func (f *Flow) LocalIP() net.IP {
	if f.local != nil {
		return f.local.addr.IP
	}
	return nil
}

func (f *Flow) LocalPort() int {
	if f.local != nil {
		return f.local.addr.Port
	}
	return 0
}

func (f *Flow) IsUDP() bool {
	return f.proto == ProtoUDP
}

func (f *Flow) IsCurrentProcess() bool {
	return int(f.pid) == currentProcesPID
}

func (f *Flow) IsDNS() bool {
	return f.remote != nil && f.remote.addr.Port == 53 && f.proto == ProtoUDP && f.pid != 0 && f.Process != nil
}

func (f *Flow) HasRemote() bool {
	return f.remote != nil
}

func (f *Flow) HasKey() bool {
	return f.remote != nil && f.local != nil
}

func (f *Flow) IsValid() bool {
	return f.inetType != InetTypeUnknown && f.proto != ProtoUnknown && f.local.addr.IP != nil && f.remote.addr.IP != nil
}

func (f *Flow) Key() string {
	return f.remote.addr.String() + "|" + f.local.addr.String()
}

func (f *Flow) Merge(ref *Flow) {
	f.lastSeenTime = ref.lastSeenTime
	if f.inetType == InetTypeUnknown {
		f.inetType = ref.inetType
	}

	if f.proto == ProtoUnknown {
		f.proto = ref.proto
	}

	if f.pid == 0 {
		f.pid = ref.pid
	}

	if f.direction == DirectionUnknown {
		f.direction = ref.direction
	}

	if ref.complete {
		f.complete = ref.complete
	}

	f.local.updateWith(ref.local)
	f.remote.updateWith(ref.remote)
}
