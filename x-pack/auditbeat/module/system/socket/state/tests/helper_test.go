// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package tests

import (
	"encoding/binary"
	"net"
	"testing"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/common"
	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/events"
	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/state"
	"github.com/elastic/beats/v7/x-pack/auditbeat/tracing"
	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func doExitCall(meta tracing.Metadata) *events.DoExitCall {
	event := &events.DoExitCall{}
	event.Meta = meta
	return event
}

func inetReleaseCall(meta tracing.Metadata, socket uintptr) *events.InetReleaseCall {
	event := &events.InetReleaseCall{}
	event.Meta = meta
	event.Socket = socket
	return event
}

func udpQueueRcvSkbCall(meta tracing.Metadata, socket, size uintptr, laddr uint32, lport uint16, ipHdr, udpHdr uint16, packet [256]byte) *events.UDPQueueRcvSkbCall {
	event := &events.UDPQueueRcvSkbCall{}
	event.Meta = meta
	event.Socket = socket
	event.Size = size
	event.LAddr = laddr
	event.LPort = lport
	event.IPHdr = ipHdr
	event.UDPHdr = udpHdr
	event.Packet = packet
	return event
}

func udpv6SendmsgCall(meta tracing.Metadata, si6ptr uintptr, lport, rport, altrport, af uint16) *events.UDPv6SendmsgCall {
	event := &events.UDPv6SendmsgCall{}
	event.Meta = meta
	event.LPort = lport
	event.RPort = rport
	event.AltRPort = altrport
	event.SI6Ptr = si6ptr
	event.SI6AF = af
	return event
}

func udpSendmsgCall(meta tracing.Metadata, socket, size, siptr uintptr, laddr, raddr, altraddr uint32, lport, rport, altrport, af uint16) *events.UDPSendmsgCall {
	event := &events.UDPSendmsgCall{}
	event.Meta = meta
	event.Socket = socket
	event.Size = size
	event.LAddr = laddr
	event.RAddr = raddr
	event.AltRAddr = altraddr
	event.LPort = lport
	event.RPort = rport
	event.AltRPort = altrport
	event.SIPtr = siptr
	event.SIAF = af
	return event
}

func tcpv4DoRcvCall(meta tracing.Metadata, socket, size uintptr, laddr, raddr uint32, lport, rport uint16) *events.TCPv4DoRcvCall {
	event := &events.TCPv4DoRcvCall{}
	event.Meta = meta
	event.Socket = socket
	event.Size = size
	event.LAddr = laddr
	event.RAddr = raddr
	event.LPort = lport
	event.RPort = rport
	return event
}

func ipLocalOutCall(meta tracing.Metadata, socket, size uintptr, laddr, raddr uint32, lport, rport uint16) *events.IPLocalOutCall {
	event := &events.IPLocalOutCall{}
	event.Meta = meta
	event.Socket = socket
	event.Size = size
	event.LAddr = laddr
	event.RAddr = raddr
	event.LPort = lport
	event.RPort = rport
	return event
}

func tcpConnectReturn(meta tracing.Metadata, retval int32) *events.TCPConnectReturn {
	event := &events.TCPConnectReturn{}
	event.Meta = meta
	event.Retval = retval
	return event
}

func tcpv4ConnectCall(meta tracing.Metadata, socket uintptr, raddr uint32, rport uint16) *events.TCPv4ConnectCall {
	event := &events.TCPv4ConnectCall{}
	event.Meta = meta
	event.Socket = socket
	event.RAddr = raddr
	event.RPort = rport
	return event
}

func sockInitDataCall(meta tracing.Metadata, socket uintptr) *events.SockInitDataCall {
	event := &events.SockInitDataCall{}
	event.Meta = meta
	event.Socket = socket
	return event
}

func inetCreateCall(meta tracing.Metadata, proto int32) *events.InetCreateCall {
	event := &events.InetCreateCall{}
	event.Meta = meta
	event.Proto = proto
	return event
}

func commitCredsCall(meta tracing.Metadata, uid, gid, euid, egid uint32) *events.CommitCredsCall {
	event := &events.CommitCredsCall{}
	event.Meta = meta
	event.UID = uid
	event.GID = gid
	event.EUID = euid
	event.EGID = egid
	return event
}

func execveReturn(meta tracing.Metadata, retval int32) *events.ExecveReturn {
	event := &events.ExecveReturn{}
	event.Meta = meta
	event.Retval = retval
	return event
}

func execveCall(meta tracing.Metadata, args []string) *events.ExecveCall {
	event := &events.ExecveCall{
		Meta: meta,
	}
	lim := len(args)
	if lim > common.MaxProgArgs {
		lim = common.MaxProgArgs
	}
	for i := 0; i < lim; i++ {
		event.Ptrs[i] = 1
	}
	if lim < len(args) {
		event.Ptrs[lim] = 1
	}
	switch lim {
	case 5:
		copyCString(event.Param4[:], []byte(args[4]))
		fallthrough
	case 4:
		copyCString(event.Param3[:], []byte(args[3]))
		fallthrough
	case 3:
		copyCString(event.Param2[:], []byte(args[2]))
		fallthrough
	case 2:
		copyCString(event.Param1[:], []byte(args[1]))
		fallthrough
	case 1:
		copyCString(event.Param0[:], []byte(args[0]))
	case 0:
		return nil
	}
	event.Path = event.Param0
	return event
}

func meta(pid uint32, tid uint32, timestamp uint64) tracing.Metadata {
	return tracing.Metadata{
		Timestamp: timestamp,
		TID:       tid,
		PID:       pid,
	}
}

func copyCString(dst []byte, src []byte) {
	copy(dst, src)
	if len(src) < len(dst) {
		dst[len(src)] = 0
	} else {
		dst[len(dst)-1] = 0
	}
}

func ipv4(ip string) uint32 {
	netIP := net.ParseIP(ip).To4()
	if netIP == nil {
		panic("bad ip")
	}
	return tracing.MachineEndian.Uint32(netIP)
}

func ipv6(ip string) (hi uint64, lo uint64) {
	netIP := net.ParseIP(ip).To16()
	if netIP == nil {
		panic("bad ip")
	}
	return tracing.MachineEndian.Uint64(netIP[:]), tracing.MachineEndian.Uint64(netIP[8:])
}

func feedEvents(evs []events.Event, st *state.State, t *testing.T) {
	for idx, ev := range evs {
		t.Logf("Delivering event %d: %s", idx, ev.String())
		ev.Update(st)
	}
}

func getFlows(flows []*state.Flow, filter func(*state.Flow) bool) (evs []beat.Event, err error) {
	var errs multierror.Errors
	for _, flow := range flows {
		if !flow.IsValid() {
			errs = append(errs, errors.New("invalid flow"))
			continue
		}
		if !filter(flow) {
			continue
		}
		ev := flow.ToEvent(true)
		evs = append(evs, ev.BeatEvent("system", "socket"))
	}
	return evs, errs.Err()
}

func assertValue(t *testing.T, ev beat.Event, expected interface{}, field string) bool {
	value, err := ev.GetValue(field)
	return assert.Nil(t, err, field) && assert.Equal(t, expected, value, field)
}

func be16(val uint16) uint16 {
	var buf [2]byte
	binary.BigEndian.PutUint16(buf[:], val)
	return tracing.MachineEndian.Uint16(buf[:])
}

func be32(val uint32) uint32 {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], val)
	return tracing.MachineEndian.Uint32(buf[:])
}

func be64(val uint64) uint64 {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], val)
	return tracing.MachineEndian.Uint64(buf[:])
}

func all(*state.Flow) bool {
	return true
}
