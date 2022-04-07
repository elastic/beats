// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build (linux && 386) || (linux && amd64)
// +build linux,386 linux,amd64

package guess

import (
	"encoding/binary"
	"fmt"

	"golang.org/x/sys/unix"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/x-pack/auditbeat/module/system/socket/helper"
	"github.com/elastic/beats/v8/x-pack/auditbeat/tracing"
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
	if err := Registry.AddGuess(func() Guesser { return &guessSockaddrIn6{} }); err != nil {
		panic(err)
	}
}

type guessSockaddrIn6 struct {
	ctx                    Context
	loopback               helper.IPv6Loopback
	clientAddr, serverAddr unix.SockaddrInet6
	client, server         int
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

// Condition allows this probe to run only when IPv6 is enabled.
func (g *guessSockaddrIn6) Condition(ctx Context) (bool, error) {
	return isIPv6Enabled(ctx.Vars)
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
func (g *guessSockaddrIn6) Prepare(ctx Context) (err error) {
	g.ctx = ctx
	g.loopback, err = helper.NewIPv6Loopback()
	if err != nil {
		return fmt.Errorf("detect IPv6 loopback failed: %w", err)
	}
	defer func() {
		if err != nil {
			g.loopback.Cleanup()
		}
	}()
	clientIP, err := g.loopback.AddRandomAddress()
	if err != nil {
		return fmt.Errorf("failed adding first device address: %w", err)
	}
	serverIP, err := g.loopback.AddRandomAddress()
	if err != nil {
		return fmt.Errorf("failed adding second device address: %w", err)
	}
	copy(g.clientAddr.Addr[:], clientIP)
	copy(g.serverAddr.Addr[:], serverIP)

	if g.client, g.clientAddr, err = createSocket6WithProto(unix.SOCK_STREAM, g.clientAddr); err != nil {
		return fmt.Errorf("error creating server: %w", err)
	}
	if g.server, g.serverAddr, err = createSocket6WithProto(unix.SOCK_STREAM, g.serverAddr); err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}
	if err = unix.Listen(g.server, 1); err != nil {
		return fmt.Errorf("error in listen: %w", err)
	}
	return nil
}

// Terminate is a no-op.
func (g *guessSockaddrIn6) Terminate() error {
	unix.Close(g.client)
	unix.Close(g.server)
	if err := g.loopback.Cleanup(); err != nil {
		return err
	}
	return nil
}

// Trigger performs a connection attempt on the random address.
func (g *guessSockaddrIn6) Trigger() error {
	if err := unix.Connect(g.client, &g.serverAddr); err != nil {
		return fmt.Errorf("connect failed: %w", err)
	}
	fd, _, err := unix.Accept(g.server)
	if err != nil {
		return fmt.Errorf("accept failed: %w", err)
	}
	unix.Close(fd)
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

	binary.BigEndian.PutUint16(needle[:], uint16(g.serverAddr.Port))
	offsetOfPort := indexAligned(arr, needle[:], offsetOfFamily+2, 2)
	if offsetOfPort == -1 {
		return nil, false
	}

	offsetOfAddr := indexAligned(arr, g.serverAddr.Addr[:], offsetOfPort+2, 1)
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
