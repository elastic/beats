// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package common

import (
	"encoding/binary"
	"net"

	"github.com/elastic/beats/v7/x-pack/auditbeat/tracing"
)

type Endpoint struct {
	addr           *net.TCPAddr
	packets, bytes uint64
}

func (e *Endpoint) updateWith(other *Endpoint) {
	if e.addr == nil {
		e.addr = other.addr
	}
	e.packets += other.packets
	e.bytes += other.bytes
}

// String returns the textual representation of the endpoint address:port.
func (e *Endpoint) String() string {
	if e != nil && e.addr != nil {
		return e.addr.String()
	}
	return "(not bound)"
}

func NewEndpointIPv4(beIP uint32, bePort uint16, pkts uint64, bytes uint64) *Endpoint {
	var buf [4]byte
	e := &Endpoint{
		packets: pkts,
		bytes:   bytes,
	}
	if bePort != 0 && beIP != 0 {
		tracing.MachineEndian.PutUint16(buf[:], bePort)
		port := binary.BigEndian.Uint16(buf[:])
		tracing.MachineEndian.PutUint32(buf[:], beIP)
		e.addr = &net.TCPAddr{
			IP:   net.IPv4(buf[0], buf[1], buf[2], buf[3]),
			Port: int(port),
		}
	}
	return e
}

func NewEndpointIPv6(beIPa uint64, beIPb uint64, bePort uint16, pkts uint64, bytes uint64) *Endpoint {
	e := &Endpoint{
		packets: pkts,
		bytes:   bytes,
	}
	if bePort != 0 && (beIPa != 0 || beIPb != 0) {
		addr := make([]byte, 16)
		tracing.MachineEndian.PutUint16(addr[:], bePort)
		port := binary.BigEndian.Uint16(addr[:])
		tracing.MachineEndian.PutUint64(addr, beIPa)
		tracing.MachineEndian.PutUint64(addr[8:], beIPb)
		e.addr = &net.TCPAddr{
			IP:   addr,
			Port: int(port),
		}
	}
	return e
}
