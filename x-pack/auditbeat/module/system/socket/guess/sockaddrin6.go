// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package guess

import (
	"encoding/binary"
	"net"

	"golang.org/x/sys/unix"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/x-pack/auditbeat/module/system/socket/helper"
	"github.com/elastic/beats/x-pack/auditbeat/tracing"
)

/*
	Guess the layout of a sockaddr_in6 structure. This struct has a fixed
	layout in intel archs but its safer to check.

	Output:
	"SOCKADDR_IN6_AF":0
	"SOCKADDR_IN6_PORT":2
	"SOCKADDR_IN6_ADDRA":8
	"SOCKADDR_IN6_ADDRB":16
*/

func init() {
	if err := Registry.AddGuess(
		&guessSockaddrIn6{
			address: net.TCPAddr{
				IP:   []byte{0xFD, 0xE5, 0x7C, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xAA, 0xBB, 0xCC, 0xDD},
				Port: 0xFEEF,
			},
		}); err != nil {
		panic(err)
	}
}

type guessSockaddrIn6 struct {
	ctx     Context
	address net.TCPAddr
}

// Name of this guess.
func (g *guessSockaddrIn6) Name() string {
	return "guess_sockaddr_in6"
}

// Provides returns the list of variables discovered.
func (g *guessSockaddrIn6) Provides() []string {
	return []string{
		"SOCKADDR_IN6_AF",
		"SOCKADDR_IN6_PORT",
		"SOCKADDR_IN6_ADDRA",
		"SOCKADDR_IN6_ADDRB",
	}
}

// Requires declares the variables required to run this guess.
func (g *guessSockaddrIn6) Requires() []string {
	return []string{
		"P2",
	}
}

// Probes returns a probe on tcp_v6_connect, dumping its second argument,
// a struct sockaddr* (struct sockaddr_in6* for AF_INET6).
func (g *guessSockaddrIn6) Probes() ([]helper.ProbeDef, error) {
	return []helper.ProbeDef{
		{
			Probe: tracing.Probe{
				Name:      "sockaddr_in6_guess",
				Address:   "tcp_v6_connect",
				Fetchargs: helper.MakeMemoryDump("{{.P2}}", 0, 64),
			},
			Decoder: tracing.NewDumpDecoder,
		},
	}, nil
}

// Prepare is a no-op.
func (g *guessSockaddrIn6) Prepare(ctx Context) error {
	g.ctx = ctx
	return nil
}

// Terminate is a no-op.
func (g *guessSockaddrIn6) Terminate() error {
	return nil
}

// Trigger performs a connection attempt on the random address.
func (g *guessSockaddrIn6) Trigger() error {
	dialer := net.Dialer{
		Timeout: g.ctx.Timeout,
	}
	conn, err := dialer.Dial("tcp", g.address.String())
	if err == nil {
		conn.Close()
	}
	return nil
}

// Extract receives the dumped struct sockaddr_in6 and scans it for the
// expected values.
func (g *guessSockaddrIn6) Extract(ev interface{}) (common.MapStr, bool) {
	arr := ev.([]byte)
	if len(arr) < 8 {
		return nil, false
	}
	var needle [2]byte
	tracing.MachineEndian.PutUint16(needle[:], unix.AF_INET6)
	offsetOfFamily := indexAligned(arr, needle[:], 0, 2)
	if offsetOfFamily == -1 {
		return nil, false
	}

	binary.BigEndian.PutUint16(needle[:], uint16(g.address.Port))
	offsetOfPort := indexAligned(arr, needle[:], offsetOfFamily+2, 2)
	if offsetOfPort == -1 {
		return nil, false
	}

	offsetOfAddr := indexAligned(arr, g.address.IP, offsetOfPort+2, 1)
	if offsetOfAddr == -1 {
		return nil, false
	}
	return common.MapStr{
		"SOCKADDR_IN6_AF":    offsetOfFamily,
		"SOCKADDR_IN6_PORT":  offsetOfPort,
		"SOCKADDR_IN6_ADDRA": offsetOfAddr,
		"SOCKADDR_IN6_ADDRB": offsetOfAddr + 8,
	}, true
}
