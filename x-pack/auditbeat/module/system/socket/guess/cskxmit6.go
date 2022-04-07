// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build (linux && 386) || (linux && amd64)
// +build linux,386 linux,amd64

package guess

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/unix"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/x-pack/auditbeat/module/system/socket/helper"
	"github.com/elastic/beats/v8/x-pack/auditbeat/tracing"
)

/*
	Guess for inet6_csk_xmit.
    This function is called to transmit data on a connected IPv6 socket. It has
	different signatures across Linux kernel versions:
		2.6+  : int inet6_csk_xmit(struct sk_buff *skb, int ipfragok)
		3.16+ : int inet6_csk_xmit(struct sock *sk, struct sk_buff *skb, struct flowi *fl_unused)

	It discovers how to get the sk_buff (either 1st or 2nd argument)
	And how to get the struct sock* (1st or indirect via sk_buff->sk)

	The output of this probe is:
		INET6_CSK_XMIT_SKBUFF: %si
		INET6_CSK_XMIT_SOCK": %di
*/

func init() {
	if err := Registry.AddGuess(func() Guesser { return &guessInet6CskXmit{} }); err != nil {
		panic(err)
	}
}

type guessInet6CskXmit struct {
	ctx                    Context
	loopback               helper.IPv6Loopback
	clientAddr, serverAddr unix.SockaddrInet6
	client, server         int
	sock                   uintptr
	acceptedFd             int
}

// Name of this guess.
func (g *guessInet6CskXmit) Name() string {
	return "guess_inet6_csk_xmit"
}

// Provides returns the names of discovered variables.
func (g *guessInet6CskXmit) Provides() []string {
	return []string{
		"INET6_CSK_XMIT_SOCK",
		"INET6_CSK_XMIT_SKBUFF",
	}
}

// Requires declares the variables required to run this guess.
func (g *guessInet6CskXmit) Requires() []string {
	return []string{
		"RET",
		"P1",
	}
}

// Condition allows this probe to run only when IPv6 is enabled.
func (g *guessInet6CskXmit) Condition(ctx Context) (bool, error) {
	return isIPv6Enabled(ctx.Vars)
}

// Probes returns 2 probes:
//   - kretprobe on inet_csk_accept, which returns a struct sock*
//   - kprobe on inet6_csk_xmit, returning 1st argument as pointer and dump.
func (g *guessInet6CskXmit) Probes() ([]helper.ProbeDef, error) {
	return []helper.ProbeDef{
		{
			Probe: tracing.Probe{
				Type:      tracing.TypeKRetProbe,
				Name:      "inet_csk_accept_guess",
				Address:   "inet_csk_accept",
				Fetchargs: "sock={{.RET}}",
			},
			Decoder: helper.NewStructDecoder(func() interface{} { return new(sockArgumentGuess) }),
		},
		{
			Probe: tracing.Probe{
				Name:      "inet6_csk_xmit_guess",
				Address:   "inet6_csk_xmit",
				Fetchargs: "arg={{.P1}} dump=" + helper.MakeMemoryDump("{{.P1}}", 0, skbuffDumpSize),
			},
			Decoder: helper.NewStructDecoder(func() interface{} { return new(skbuffSockGuess) }),
		},
	}, nil
}

// Prepare setups an IPv6 TCP client/server where the server is listening
// and the client is connecting to it.
func (g *guessInet6CskXmit) Prepare(ctx Context) (err error) {
	g.ctx = ctx
	g.acceptedFd = -1
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
	if err = unix.Connect(g.client, &g.serverAddr); err != nil {
		return fmt.Errorf("connect failed: %w", err)
	}
	return nil
}

// Terminate cleans up the sockets.
func (g *guessInet6CskXmit) Terminate() error {
	if g.acceptedFd != -1 {
		unix.Close(g.acceptedFd)
	}
	unix.Close(g.client)
	unix.Close(g.server)
	return g.loopback.Cleanup()
}

// Trigger accepts the client connection on the server, triggering a call
// to inet_csk_accept, and then writes on the returned description, triggering
// a inet_csk_xmit on the socket returned by accept.
func (g *guessInet6CskXmit) Trigger() error {
	fd, _, err := unix.Accept(g.server)
	if err != nil {
		return fmt.Errorf("accept failed: %w", err)
	}
	_, err = unix.Write(fd, []byte("hello world"))
	return err
}

// Extract receives first the sock* from inet_csk_accept, then the arguments
// from inet6_csk_xmit.
func (g *guessInet6CskXmit) Extract(event interface{}) (common.MapStr, bool) {
	switch msg := event.(type) {
	case *sockArgumentGuess:
		g.sock = msg.Sock
		return nil, false

	case *skbuffSockGuess:
		if g.sock == 0 {
			return nil, false
		}
		if msg.Arg == g.sock {
			// struct sock * is the first argument
			return common.MapStr{
				"INET6_CSK_XMIT_SOCK":   g.ctx.Vars["P1"],
				"INET6_CSK_XMIT_SKBUFF": g.ctx.Vars["P2"],
			}, true
		}
		// struct sk_buff* is the first argument. Obtain sock* from sk_buff
		off := indexAligned(msg.Dump[:], ((*[sizeOfPtr]byte)(unsafe.Pointer(&g.sock)))[:], 0, int(sizeOfPtr))
		if off != -1 {
			return common.MapStr{
				"INET6_CSK_XMIT_SOCK":   fmt.Sprintf("+%d(%s)", off, g.ctx.Vars["P1"]),
				"INET6_CSK_XMIT_SKBUFF": g.ctx.Vars["P1"],
			}, true
		}
	}
	return nil, false
}
