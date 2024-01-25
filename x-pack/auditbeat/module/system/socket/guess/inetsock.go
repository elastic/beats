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

	"github.com/elastic/beats/v7/auditbeat/tracing"
	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/helper"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// Guess the offsets within a struct inet_sock where the local and remote
// addresses and ports are found.
//
// This is run multiple times to avoid birthdays.
//
// Most values appear multiple times within the struct, this is normal.
//
// Output:
// INET_SOCK_LADDR : 572
// INET_SOCK_LPORT : 582
// INET_SOCK_RADDR : 576
// INET_SOCK_RPORT : 580
// INET_SOCK_RADDR_LIST : [...array of offsets...]
// INET_SOCK_RADDR_LIST is a list of all the offsets within the structure that
// matched the remote address. This is used by guess_inet_sock6.

func init() {
	if err := Registry.AddGuess(func() Guesser { return &guessInetSockIPv4{} }); err != nil {
		panic(err)
	}
}

type guessInetSockIPv4 struct {
	ctx            Context
	local, remote  unix.SockaddrInet4
	server, client int
}

// Name of this guess.
func (g *guessInetSockIPv4) Name() string {
	return "guess_inet_sock"
}

// Provides returns the list of variables discovered.
func (g *guessInetSockIPv4) Provides() []string {
	return []string{
		"INET_SOCK_LADDR",
		"INET_SOCK_LPORT",
		"INET_SOCK_RADDR",
		"INET_SOCK_RPORT",
		"INET_SOCK_LADDR_LIST",
		"INET_SOCK_LPORT_LIST",
		"INET_SOCK_RADDR_LIST",
		"INET_SOCK_RPORT_LIST",
	}
}

// Requires declares the variables required to run this guess.
func (g *guessInetSockIPv4) Requires() []string {
	return []string{
		"RET",
	}
}

// Probes returns a kretprobe on inet_sock_accept that dumps the return
// value (an inet_sock*).
func (g *guessInetSockIPv4) Probes() ([]helper.ProbeDef, error) {
	return []helper.ProbeDef{
		{
			Probe: tracing.Probe{
				Type:      tracing.TypeKRetProbe,
				Name:      "inet_sock_guess",
				Address:   "inet_csk_accept",
				Fetchargs: helper.MakeMemoryDump("{{.RET}}", 0, 2048),
			},
			Decoder: tracing.NewDumpDecoder,
		},
	}, nil
}

// Prepare creates a TCP/IP client and server bound to random loopback addresses
// (127.x.x.x).
func (g *guessInetSockIPv4) Prepare(ctx Context) (err error) {
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

// Terminate cleans up the client and server.
func (g *guessInetSockIPv4) Terminate() error {
	unix.Close(g.client)
	unix.Close(g.server)
	return nil
}

// Trigger makes the client connect to the server, causing a inet_csk_accept
// event.
func (g *guessInetSockIPv4) Trigger() error {
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

// Extract receives the dump of a struct inet_sock* and scans it for the
// random local and remote IPs and ports. Will return lists of all the
// offsets were each value was found.
func (g *guessInetSockIPv4) Extract(ev interface{}) (mapstr.M, bool) {
	data := ev.([]byte)

	laddr := g.local.Addr[:]
	lport := make([]byte, 2)
	binary.BigEndian.PutUint16(lport, uint16(g.local.Port))
	raddr := g.remote.Addr[:]
	rport := make([]byte, 2)
	binary.BigEndian.PutUint16(rport, uint16(g.remote.Port))
	var laddrHits []int
	var lportHits []int
	var raddrHits []int
	var rportHits []int

	off := indexAligned(data, laddr, 0, 4)
	for off != -1 {
		laddrHits = append(laddrHits, off)
		off = indexAligned(data, laddr, off+4, 4)
	}

	off = indexAligned(data, lport, 0, 2)
	for off != -1 {
		lportHits = append(lportHits, off)
		off = indexAligned(data, lport, off+2, 2)
	}

	off = indexAligned(data, raddr, 0, 4)
	for off != -1 {
		raddrHits = append(raddrHits, off)
		off = indexAligned(data, raddr, off+4, 4)
	}

	off = indexAligned(data, rport, 0, 2)
	for off != -1 {
		rportHits = append(rportHits, off)
		off = indexAligned(data, rport, off+2, 2)
	}

	if len(laddrHits) == 0 || len(lportHits) == 0 || len(raddrHits) == 0 || len(rportHits) == 0 {
		return nil, false
	}

	return mapstr.M{
		"INET_SOCK_LADDR": laddrHits,
		"INET_SOCK_LPORT": lportHits,
		"INET_SOCK_RADDR": raddrHits,
		"INET_SOCK_RPORT": rportHits,
	}, true
}

// NumRepeats makes this guess to be repeated to avoid collisions.
func (g *guessInetSockIPv4) NumRepeats() int {
	return 4
}

// Reduce receives the output from multiple runs (list of offsets for each field)
// and for every field it returns the first offset that appeared all the runs.
func (g *guessInetSockIPv4) Reduce(results []mapstr.M) (result mapstr.M, err error) {
	if result, err = consolidate(results); err != nil {
		return nil, err
	}

	for _, key := range []string{
		"INET_SOCK_LADDR", "INET_SOCK_LPORT",
		"INET_SOCK_RADDR", "INET_SOCK_RPORT",
	} {
		list, err := getListField(result, key)
		if err != nil {
			return nil, err
		}
		result[key+"_LIST"] = list
		result[key] = list[0]
	}
	return result, nil
}
