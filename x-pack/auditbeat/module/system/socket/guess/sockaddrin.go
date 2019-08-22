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
	Guess the layout of a sockaddr_in structure. This struct has a fixed
	layout in intel archs but its safer to check.

	Output:
	"SOCKADDR_IN_AF":0
	"SOCKADDR_IN_PORT":2
	"SOCKADDR_IN_ADDR":4
*/

func init() {
	if err := Registry.AddGuess(
		&guessSockaddrIn{
			address: net.TCPAddr{
				IP:   net.IPv4(127, 0x12, 0x34, 0x56).To4(),
				Port: 0xABCD,
			},
		}); err != nil {
		panic(err)
	}
}

type guessSockaddrIn struct {
	ctx     Context
	address net.TCPAddr
}

// Name of this guess.
func (g *guessSockaddrIn) Name() string {
	return "guess_sockaddr_in"
}

// Provides returns the list of variables discovered.
func (g *guessSockaddrIn) Provides() []string {
	return []string{
		"SOCKADDR_IN_AF",
		"SOCKADDR_IN_PORT",
		"SOCKADDR_IN_ADDR",
	}
}

// Requires declares the variables required to run this guess.
func (g *guessSockaddrIn) Requires() []string {
	return []string{
		"P2",
	}
}

// Probes sets a probe on tcp_v4_connect and dumps its second argument, which
// has a type of struct sockaddr* (struct sockaddr_in* for AF_INET).
func (g *guessSockaddrIn) Probes() ([]helper.ProbeDef, error) {
	return []helper.ProbeDef{
		{
			Probe: tracing.Probe{
				Name:      "sockaddr_in_guess",
				Address:   "tcp_v4_connect",
				Fetchargs: helper.MakeMemoryDump("{{.P2}}", 0, 32),
			},
			Decoder: tracing.NewDumpDecoder,
		},
	}, nil
}

// Prepare is a no-op.
func (g *guessSockaddrIn) Prepare(ctx Context) error {
	g.ctx = ctx
	return nil
}

// Terminate is a no-op.
func (g *guessSockaddrIn) Terminate() error {
	return nil
}

// Trigger connects a socket to a random local address (127.x.x.x).
func (g *guessSockaddrIn) Trigger() error {
	dialer := net.Dialer{
		Timeout: g.ctx.Timeout,
	}
	conn, err := dialer.Dial("tcp", g.address.String())
	if err == nil {
		conn.Close()
	}
	return nil
}

// Validate takes the dumped sockaddr_in and scans it for the expected values.
func (g *guessSockaddrIn) Validate(ev interface{}) (common.MapStr, bool) {
	arr := ev.([]byte)
	if len(arr) < 8 {
		return nil, false
	}
	var needle [2]byte
	tracing.MachineEndian.PutUint16(needle[:], unix.AF_INET)
	offsetOfFamily := indexAligned(arr, needle[:], 0, 2)
	if offsetOfFamily == -1 {
		return nil, false
	}

	binary.BigEndian.PutUint16(needle[:], uint16(g.address.Port))
	offsetOfPort := indexAligned(arr, needle[:], offsetOfFamily+2, 2)
	if offsetOfPort == -1 {
		return nil, false
	}

	offsetOfAddr := indexAligned(arr, []byte(g.address.IP), offsetOfPort+2, 4)
	if offsetOfAddr == -1 {
		return nil, false
	}
	return common.MapStr{
		"SOCKADDR_IN_AF":   offsetOfFamily,
		"SOCKADDR_IN_PORT": offsetOfPort,
		"SOCKADDR_IN_ADDR": offsetOfAddr,
	}, true
}
