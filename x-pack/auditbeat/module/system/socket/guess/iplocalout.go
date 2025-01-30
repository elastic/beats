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

	"github.com/elastic/beats/v7/auditbeat/tracing"
	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/helper"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// Guess how to get a struct sock* and an sk_buff* from an ip_local_out() call.
// This function has three forms depending on kernel version:
// - ip_local_out(struct sk_buff *skb) // 2.x//<3.13
// - ip_local_out_sk(struct sock *sk, struct sk_buff *skb) // 3.13..4.3
// - ip_local_out(struct net *net, struct sock *sk, struct sk_buff *skb) // 4.4+
//
// To make things more complicated, in some 5.x+ kernels, ip_local_out is never
// triggered although it exists, but __ip_local_out always works, so
// this guess expects the template variable IP_LOCAL_OUT to be set to the
// first of these functions that is available for tracing:
// [ "ip_local_out_sk", "__ip_local_out", "ip_local_out" ]
//
// ----
//
// What it guess does is set a probe on tcp_sendmsg (guaranteed to have a *sock)
// and in .IP_LOCAL_OUT, which will be called by tcp_sendmsg.
// It dumps the first param (which can be a struct net* or a struct sk_buff)
// and gets the second param. Either the second param is the sock, or is it
// found at some point in the dumped first param.
//
// Output:
//  IP_LOCAL_OUT_SOCK    : +16(%ax)
//  IP_LOCAL_OUT_SK_BUFF : %ax

const (
	sizeOfPtr      = unsafe.Sizeof(uintptr(0))
	skbuffDumpSize = 960
)

func init() {
	if err := Registry.AddGuess(func() Guesser { return &guessIPLocalOut{} }); err != nil {
		panic(err)
	}
}

type sockArgumentGuess struct {
	Sock uintptr `kprobe:"sock"`
}

type skbuffSockGuess struct {
	Arg  uintptr              `kprobe:"arg"`
	Dump [skbuffDumpSize]byte `kprobe:"dump,greedy"`
}

type guessIPLocalOut struct {
	ctx  Context
	cs   inetClientServer
	sock uintptr
}

// Name of this guess.
func (g *guessIPLocalOut) Name() string {
	return "guess_ip_local_out"
}

// Provides returns the list of variables discovered.
func (g *guessIPLocalOut) Provides() []string {
	return []string{
		"IP_LOCAL_OUT_SOCK",
		"IP_LOCAL_OUT_SK_BUFF",
	}
}

// Requires declares the variables required to run this guess.
func (g *guessIPLocalOut) Requires() []string {
	return []string{
		"IP_LOCAL_OUT",
		"P1",
		"P2",
		"TCP_SENDMSG_SOCK",
		"TCP_SENDMSG_LEN",
		"INET_SOCK_LADDR",
		"INET_SOCK_LPORT",
		"INET_SOCK_RADDR",
		"INET_SOCK_RPORT",
	}
}

// Probes returns two probes:
// - ip_local_out(_sk). fetches:
//   - arg2 (arg1 if this system has ip_local_out_sk)
//   - dump of arg1 (arg2 if this system has ip_local_out_sk)
//
// - tcp_sendmsg, returning the sock* argument.
func (g *guessIPLocalOut) Probes() ([]helper.ProbeDef, error) {
	return []helper.ProbeDef{
		{
			Probe: tracing.Probe{
				Name:    "ip_local_out_sock_guess",
				Address: "{{.IP_LOCAL_OUT}}",
				Fetchargs: "arg={{if ne .IP_LOCAL_OUT \"ip_local_out_sk\"}}{{.P2}}{{else}}{{.P1}}{{end}} dump=" +
					helper.MakeMemoryDump("{{if ne .IP_LOCAL_OUT \"ip_local_out_sk\"}}{{.P1}}{{else}}{{.P2}}{{end}}", 0, skbuffDumpSize),
			},
			Decoder: helper.NewStructDecoder(func() interface{} { return new(skbuffSockGuess) }),
		},
		{
			Probe: tracing.Probe{
				Name:      "tcp_sendmsg_in",
				Address:   "tcp_sendmsg",
				Fetchargs: "sock={{.TCP_SENDMSG_SOCK}}",
			},
			Decoder: helper.NewStructDecoder(func() interface{} { return new(sockArgumentGuess) }),
		},
	}, nil
}

// Prepare sets up a connected TCP client/server.
func (g *guessIPLocalOut) Prepare(ctx Context) error {
	g.ctx = ctx
	return g.cs.SetupTCP()
}

// Terminate cleans up the TCP client/server.
func (g *guessIPLocalOut) Terminate() error {
	return g.cs.Cleanup()
}

// Trigger writes from the client to the server, causing both probes to fire.
func (g *guessIPLocalOut) Trigger() error {
	buf := []byte("Hello World!\n")
	_, err := unix.Write(g.cs.client, buf)
	if err != nil {
		return err
	}
	unix.Read(g.cs.accepted, buf)
	return nil
}

// Extract first receives and saves the sock* from tcp_sendmsg.
// Once ip_local_out is called, it analyses the captured arguments to determine
// their layout.
func (g *guessIPLocalOut) Extract(ev interface{}) (mapstr.M, bool) {
	switch v := ev.(type) {
	case *sockArgumentGuess:
		g.sock = v.Sock

	case *skbuffSockGuess:
		if g.sock == 0 {
			// No tcp_sendmsg received?
			return nil, false
		}
		// Special handling for ip_local_out_sk
		isIpLocalOut := g.ctx.Vars["IP_LOCAL_OUT"] != "ip_local_out_sk"
		if v.Arg == g.sock {
			if isIpLocalOut {
				return mapstr.M{
					// Second argument to ip_local_out is the struct sock*
					"IP_LOCAL_OUT_SOCK":    g.ctx.Vars["P2"],
					"IP_LOCAL_OUT_SK_BUFF": g.ctx.Vars["P3"],
				}, true
			}
			return mapstr.M{
				// Second argument to ip_local_out is the struct sock*
				"IP_LOCAL_OUT_SOCK":    g.ctx.Vars["P1"],
				"IP_LOCAL_OUT_SK_BUFF": g.ctx.Vars["P2"],
			}, true
		}
		off := indexAligned(v.Dump[:], ((*[sizeOfPtr]byte)(unsafe.Pointer(&g.sock)))[:], 0, int(sizeOfPtr))
		if off != -1 {
			return mapstr.M{
				// struct sock* is a field of struct pointed to by first argument
				"IP_LOCAL_OUT_SOCK":    fmt.Sprintf("+%d(%s)", off, g.ctx.Vars["P1"]),
				"IP_LOCAL_OUT_SK_BUFF": g.ctx.Vars["P1"],
			}, true
		}

	}
	return nil, false
}
