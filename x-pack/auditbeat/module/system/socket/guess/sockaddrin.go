// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build (linux && 386) || (linux && amd64)
// +build linux,386 linux,amd64

package guess

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"golang.org/x/sys/unix"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/helper"
	"github.com/elastic/beats/v7/x-pack/auditbeat/tracing"
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
	if err := Registry.AddGuess(func() Guesser { return &guessSockaddrIn{} }); err != nil {
		panic(err)
	}
}

type guessSockaddrIn struct {
	ctx            Context
	local, remote  unix.SockaddrInet4
	server, client int
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
func (g *guessSockaddrIn) Prepare(ctx Context) (err error) {
	g.ctx = ctx
	g.local = unix.SockaddrInet4{
		Port: 0,
		Addr: randomLocalIP(),
	}
	g.remote = unix.SockaddrInet4{
		Port: 0,
		Addr: randomLocalIP(),
	}
	for bytes.Equal(g.local.Addr[:], g.remote.Addr[:]) {
		g.remote.Addr = randomLocalIP()
	}
	if g.server, g.local, err = createSocket(g.local); err != nil {
		return fmt.Errorf("error creating server: %w", err)
	}
	if g.client, g.remote, err = createSocket(g.remote); err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}
	if err = unix.Listen(g.server, 1); err != nil {
		return fmt.Errorf("error in listen: %w", err)
	}
	return nil
}

// Terminate is a no-op.
func (g *guessSockaddrIn) Terminate() error {
	unix.Close(g.client)
	unix.Close(g.server)
	return nil
}

// Trigger connects a socket to a random local address (127.x.x.x).
func (g *guessSockaddrIn) Trigger() error {
	if err := unix.Connect(g.client, &g.local); err != nil {
		return err
	}
	fd, _, err := unix.Accept(g.server)
	if err != nil {
		return err
	}
	unix.Close(fd)
	return nil
}

// Extract takes the dumped sockaddr_in and scans it for the expected values.
func (g *guessSockaddrIn) Extract(ev interface{}) (common.MapStr, bool) {
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

	binary.BigEndian.PutUint16(needle[:], uint16(g.local.Port))
	offsetOfPort := indexAligned(arr, needle[:], offsetOfFamily+2, 2)
	if offsetOfPort == -1 {
		return nil, false
	}

	offsetOfAddr := indexAligned(arr, []byte(g.local.Addr[:]), offsetOfPort+2, 4)
	if offsetOfAddr == -1 {
		return nil, false
	}
	return common.MapStr{
		"SOCKADDR_IN_AF":   offsetOfFamily,
		"SOCKADDR_IN_PORT": offsetOfPort,
		"SOCKADDR_IN_ADDR": offsetOfAddr,
	}, true
}
