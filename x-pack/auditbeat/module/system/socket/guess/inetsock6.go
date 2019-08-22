// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package guess

import (
	"bytes"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/x-pack/auditbeat/module/system/socket/helper"
	"github.com/elastic/beats/x-pack/auditbeat/tracing"
)

/*
	Guess the offset of local and remote IPv6 addresses on a struct inet_sock*.
	This is an easy one as the addresses appear consecutive in memory.

	Output:

	INET_SOCK_V6_RADDR_A: 56
	INET_SOCK_V6_RADDR_B: 64
	INET_SOCK_V6_LADDR_A: 72
	INET_SOCK_V6_LADDR_B: 80
*/

const inetSockDumpSize = 8 * 256

func init() {
	if err := Registry.AddGuess(&guessInetSockIPv6{}); err != nil {
		panic(err)
	}
}

type guessInetSockIPv6 struct {
	ctx                    Context
	loopback               ipv6loopback
	clientAddr, serverAddr unix.SockaddrInet6
	client, server         int
}

// Name of this guess.
func (g *guessInetSockIPv6) Name() string {
	return "guess_inet_sock_ipv6"
}

// Provides returns the names of variables provided by this guess.
func (g *guessInetSockIPv6) Provides() []string {
	return []string{
		"INET_SOCK_V6_RADDR_A",
		"INET_SOCK_V6_RADDR_B",
		"INET_SOCK_V6_LADDR_A",
		"INET_SOCK_V6_LADDR_B",
	}
}

// Requires returns the list of required variables.
func (g *guessInetSockIPv6) Requires() []string {
	return []string{
		"RET",
	}
}

// Probes returns a kretprobe in inet_csk_accept that dumps the memory pointed
// to by the return value (an inet_sock*).
func (g *guessInetSockIPv6) Probes() ([]helper.ProbeDef, error) {
	return []helper.ProbeDef{
		{
			Probe: tracing.Probe{
				Type:      tracing.TypeKRetProbe,
				Name:      "inet_sock_ipv6_guess",
				Address:   "inet_csk_accept",
				Fetchargs: helper.MakeMemoryDump("{{.RET}}", 0, inetSockDumpSize),
			},
			Decoder: tracing.NewDumpDecoder,
		},
	}, nil
}

// Prepare creates an IPv6 client/server bound to loopback.
// Unlike with IPv4, it's not possible to bind to a random IP in the IPv6
// loopback as it is ::1/128 by default. Thus it's necessary to add temporary
// random addresses in the fd00::/8 reserved network to the loopback device.
func (g *guessInetSockIPv6) Prepare(ctx Context) (err error) {
	g.ctx = ctx
	g.loopback, err = newIPv6Loopback()
	if err != nil {
		return errors.Wrap(err, "detect IPv6 loopback failed")
	}
	defer func() {
		if err != nil {
			g.loopback.Cleanup()
		}
	}()
	clientIP, err := g.loopback.addRandomAddress()
	if err != nil {
		return errors.Wrap(err, "failed adding first device address")
	}
	serverIP, err := g.loopback.addRandomAddress()
	if err != nil {
		return errors.Wrap(err, "failed adding second device address")
	}
	copy(g.clientAddr.Addr[:], clientIP)
	copy(g.serverAddr.Addr[:], serverIP)

	if g.client, g.clientAddr, err = createSocket6WithProto(unix.SOCK_STREAM, g.clientAddr); err != nil {
		return errors.Wrap(err, "error creating server")
	}
	if g.server, g.serverAddr, err = createSocket6WithProto(unix.SOCK_STREAM, g.serverAddr); err != nil {
		return errors.Wrap(err, "error creating client")
	}
	if err = unix.Listen(g.server, 1); err != nil {
		return errors.Wrap(err, "error in listen")
	}
	return nil
}

// Trigger connects the client to the server, causing an inet_csk_accept call.
func (g *guessInetSockIPv6) Trigger() error {
	if err := unix.Connect(g.client, &g.serverAddr); err != nil {
		return errors.Wrap(err, "connect failed")
	}
	fd, _, err := unix.Accept(g.server)
	if err != nil {
		return errors.Wrap(err, "accept failed")
	}
	unix.Close(fd)
	return nil
}

// Validate scans the returned memory dump for the remote address followed
// by the local address.
func (g *guessInetSockIPv6) Validate(event interface{}) (common.MapStr, bool) {
	raw := event.([]byte)
	var expected []byte
	expected = append(expected, g.clientAddr.Addr[:]...) // sck_v6_daddr
	expected = append(expected, g.serverAddr.Addr[:]...) // sck_v6_rcv_saddr
	offset := bytes.Index(raw, expected)
	if offset == -1 {
		return nil, false
	}
	return common.MapStr{
		"INET_SOCK_V6_RADDR_A": offset,
		"INET_SOCK_V6_RADDR_B": offset + 8,
		"INET_SOCK_V6_LADDR_A": offset + 16,
		"INET_SOCK_V6_LADDR_B": offset + 24,
	}, true
}

// Terminate closes the client/server and releases the random IPs from loopback.
func (g *guessInetSockIPv6) Terminate() error {
	unix.Close(g.client)
	unix.Close(g.server)
	if err := g.loopback.Cleanup(); err != nil {
		return err
	}
	return nil
}
